package captain

import (
	"context"
	"encoding/json"
	"github.com/armadanet/captain/dockercntrl"
	"github.com/armadanet/spinner/spincomm"
	"log"
	"sync"
	"time"
)

type ResourceManager struct {
	mutex      *sync.Mutex
	context    context.Context
	client     spincomm.SpinnerClient
	resource   *Resource
	tasksTable map[string]*dockercntrl.Container
	appIDs     map[string]struct{}
}

type Resource struct {
	totalResource      dockercntrl.Limits
	unassignedResource dockercntrl.Limits
	cpuUsage           ResQueue
	memUsage           ResQueue
	activeContainers   []string
	//images           []string
	layers			   map[string]string
	usedPorts          map[string]string
}

func initResourceManager(state *dockercntrl.State) (*ResourceManager, error) {
	res, err := state.MachineInfo()
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	total := dockercntrl.Limits{
		CPUShares: int64(res.NCPU),
		Memory:    res.MemTotal,
	}

	avail := total
	list, err := state.List(false, false)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	for _, container := range list {
		resp, err := state.ContainerInspect(container)
		if err != nil {
			log.Fatalln(err)
		}
		avail.CPUShares -= resp.HostConfig.CPUShares
		avail.Memory -= resp.HostConfig.Memory
	}
	return &ResourceManager{
		mutex: &sync.Mutex{},
		resource: &Resource{
			totalResource:      total,
			unassignedResource: avail,
			cpuUsage:           initQueue(),
			memUsage:           initQueue(),
			activeContainers:   make([]string, 0),
			layers:             make(map[string]string),
			usedPorts:          make(map[string]string),
		},
		tasksTable: make(map[string]*dockercntrl.Container),
		appIDs: make(map[string]struct{}),
	}, nil
}

func (c *Captain) RequestResource(config *dockercntrl.Config) {
	c.rm.mutex.Lock()
	c.rm.resource.unassignedResource.CPUShares -= config.Limits.CPUShares
	c.rm.resource.unassignedResource.Memory -= config.Limits.Memory
	c.rm.mutex.Unlock()

	nodeInfo := c.GenNodeInfo()
	c.SendStatus(&nodeInfo)
}

func (c *Captain) ReleaseResource(config *dockercntrl.Config) {
	c.rm.mutex.Lock()
	c.rm.resource.unassignedResource.CPUShares += config.Limits.CPUShares
	c.rm.resource.unassignedResource.Memory += config.Limits.Memory
	c.rm.mutex.Unlock()

	nodeInfo := c.GenNodeInfo()
	c.SendStatus(&nodeInfo)
}

func (c *Captain) UpdateRealTimeResource() error {
	for {
		containers, err := c.state.List(false, false)
		if err != nil {
			return err
		}

		activeContainers := make([]string, 0)
		cpuUsage := HistoryLog{
			containers: make(map[string]float64),
			sum:        0.0,
		}
		memUsage := HistoryLog{
			containers: make(map[string]float64),
			sum:        0.0,
		}

		for _, container := range containers {
			cpuPercent, memPercent, err := c.state.RealtimeRC(container.ID)
			if err != nil {
				return err
			}

			// Update usage
			cpuUsage.containers[container.ID] = cpuPercent
			memUsage.containers[container.ID] = memPercent
			cpuUsage.sum += cpuPercent
			memUsage.sum += memPercent

			// Update container list
			activeContainers = append(activeContainers, container.Image)
		}
		c.rm.mutex.Lock()
		c.rm.resource.cpuUsage.Push(cpuUsage)
		c.rm.resource.memUsage.Push(memUsage)
		c.rm.resource.activeContainers = activeContainers
		c.rm.mutex.Unlock()
	}
}

func (c *Captain) PeriodicalUpdate(ctx context.Context, client spincomm.SpinnerClient) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c.rm.context = ctx
	c.rm.client = client
	time.Sleep(1 * time.Second)

	for {
		for taskID, container := range c.rm.tasksTable {
			if _, ok := c.rm.resource.usedPorts[taskID]; !ok {
				//Update used ports
				ports, err := c.state.UsedPorts(container)
				if err != nil {
					log.Println(err)
					ports = []string{}
				}
				if len(ports) == 0 {
					c.rm.resource.usedPorts[taskID] = ""
				} else {
					c.rm.resource.usedPorts[taskID] = ports[0]
				}
			}
		}
		nodeInfo := c.GenNodeInfo()
		c.SendStatus(&nodeInfo)
		time.Sleep(5 * time.Second)
	}
}

func (c *Captain) GenNodeInfo() spincomm.NodeInfo{
	c.rm.mutex.Lock()
	defer c.rm.mutex.Unlock()

	// Resources
	cpu := spincomm.ResourceStatus{
		Total:      c.rm.resource.totalResource.CPUShares,
		Unassigned: c.rm.resource.unassignedResource.CPUShares,
		Assigned:   c.rm.resource.totalResource.CPUShares - c.rm.resource.unassignedResource.CPUShares,
		Available:  100.0 - c.rm.resource.cpuUsage.Average(),
	}

	mem := spincomm.ResourceStatus{
		Total:      c.rm.resource.totalResource.Memory,
		Unassigned: c.rm.resource.unassignedResource.Memory,
		Assigned:   c.rm.resource.totalResource.Memory - c.rm.resource.unassignedResource.Memory,
		Available:  100.0 - c.rm.resource.memUsage.Average(),
	}
	hostResource := make(map[string]*spincomm.ResourceStatus)
	hostResource["CPU"] = &cpu
	hostResource["Memory"] = &mem

	// Task Info
	appList := make([]string, len(c.rm.appIDs))
	for id, _ := range c.rm.appIDs {
		appList = append(appList, id)
	}
	taskList := make([]string, len(c.rm.tasksTable))
	for id, _ := range c.rm.tasksTable {
		taskList = append(taskList, id)
	}

	nodeInfo := spincomm.NodeInfo{
		CaptainId: &spincomm.UUID{
			Value: c.name,
		},
		HostResource: hostResource,
		UsedPorts: c.rm.resource.usedPorts,
		ContainerStatus: &spincomm.ContainerStatus{
			ActiveContainer: c.rm.resource.activeContainers,
			Images:          c.rm.resource.activeContainers,
		},
		AppIDs: appList,
		TaskIDs: taskList,
		Layers: c.rm.resource.layers,
	}
	return nodeInfo
}

func (c *Captain) SendStatus(nodeInfo *spincomm.NodeInfo) {
	//c.rm.mutex.Lock()
	//defer c.rm.mutex.Unlock()

	log.Println(nodeInfo)
	r, err := c.rm.client.Update(c.rm.context, nodeInfo)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(r)
}

func (c *Captain) appendTask(appID string, taskID string, container *dockercntrl.Container) {
	c.rm.mutex.Lock()
	defer c.rm.mutex.Unlock()

	log.Println("append task")
	c.rm.appIDs[appID] = struct{}{}
	c.rm.tasksTable[taskID] = container
}

func (c *Captain) removeTask(appID string, taskID string) {
	c.rm.mutex.Lock()
	defer c.rm.mutex.Unlock()

	delete(c.rm.appIDs, appID)
	delete(c.rm.tasksTable, taskID)
	delete(c.rm.resource.usedPorts, taskID)
}

func (c *Captain) getTaskTable() map[string]*dockercntrl.Container {
	c.rm.mutex.Lock()
	defer c.rm.mutex.Unlock()

	return c.rm.tasksTable
}

func (c *Captain) updateLayers(logs []string) {
	for _, l := range logs {
		if l == "" {
			continue
		}
		var f interface{}
		log.Println(l)
		json.Unmarshal([]byte(l), &f)
		m := f.(map[string]interface{})
		if m["id"] != nil {
			layerID := m["id"].(string)
			status := m["status"].(string)
			if layerID == "latest" || (status != "Already exists" && status != "Pulling fs layer") {
				continue
			}
			log.Println("after continue")
			c.rm.resource.layers[layerID] = ""
			log.Println(len(c.rm.resource.layers))
		}
	}

	//TODO: for testing
	log.Println(len(c.rm.resource.layers))
}