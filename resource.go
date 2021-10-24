package captain

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/armadanet/captain/dockercntrl"
	"github.com/armadanet/spinner/spincomm"
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
	//images               []string
	layers               map[string]string
	usedPorts            map[string]string
	containerAssignedCpu map[string]int64
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
			totalResource:        total,
			unassignedResource:   avail,
			cpuUsage:             initQueue(),
			memUsage:             initQueue(),
			activeContainers:     make([]string, 0),
			layers:               make(map[string]string),
			usedPorts:            make(map[string]string),
			containerAssignedCpu: make(map[string]int64),
		},
		tasksTable: make(map[string]*dockercntrl.Container),
		appIDs:     make(map[string]struct{}),
	}, nil
}

func (c *Captain) RequestResource(config *spincomm.TaskRequest) *spincomm.TaskRequest {
	resourceMap := config.GetTaskspec().GetResourceMap()
	c.rm.mutex.Lock()
	if val, ok := resourceMap["CPU"]; ok {
		if val.Requested > c.rm.resource.unassignedResource.CPUShares {
			config.Taskspec.ResourceMap["CPU"].Requested = c.rm.resource.unassignedResource.CPUShares
		}
		c.rm.resource.unassignedResource.CPUShares -= val.Requested
	}
	if val, ok := resourceMap["memory"]; ok {
		if val.Requested > c.rm.resource.unassignedResource.Memory {
			config.Taskspec.ResourceMap["Memory"].Requested = c.rm.resource.unassignedResource.Memory
		}
		c.rm.resource.unassignedResource.Memory -= val.Requested
	}
	c.rm.resource.containerAssignedCpu[config.TaskId.Value] = config.Taskspec.ResourceMap["CPU"].Requested
	c.rm.mutex.Unlock()

	nodeInfo := c.GenNodeInfo()
	c.SendStatus(&nodeInfo)
	return config
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
		//containers, err := c.state.List(false, false)
		//if err != nil {
		//	return err
		//}

		activeContainers := make([]string, 0)
		cpuUsage := HistoryLog{
			containers: make(map[string]float64),
			sum:        0.0,
		}
		memUsage := HistoryLog{
			containers: make(map[string]float64),
			sum:        0.0,
		}

		//for _, container := range containers {
		for taskID, container := range c.rm.tasksTable {
			cpuPercent, memPercent, err := c.state.RealtimeRC(container.ID)
			if err != nil {
				return err
			}

			// Update usage
			cpuUsage.containers[taskID] = cpuPercent
			memUsage.containers[taskID] = memPercent
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
				log.Printf("updating used port for taskID: %s\n", taskID)
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
		err := c.SendStatus(&nodeInfo)
		if err != nil {
			c.RemoveTask()
			log.Fatal(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (c *Captain) GenNodeInfo() spincomm.NodeInfo {
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
		UsedPorts:    c.rm.resource.usedPorts,
		ContainerStatus: &spincomm.ContainerStatus{
			ActiveContainer: c.rm.resource.activeContainers,
			Images:          c.rm.resource.activeContainers,
		},
		AppIDs:               appList,
		TaskIDs:              taskList,
		Layers:               c.rm.resource.layers,
		ContainerUtilization: c.rm.resource.cpuUsage.GetRecentUpdate(),
		AssignedCpu:          c.rm.resource.containerAssignedCpu,
	}
	return nodeInfo
}

func (c *Captain) SendStatus(nodeInfo *spincomm.NodeInfo) error {
	//c.rm.mutex.Lock()
	//defer c.rm.mutex.Unlock()

	log.Printf("Total CPU: %d, Assignged CPU: %d, Available: %f", nodeInfo.HostResource["CPU"].Total, nodeInfo.HostResource["CPU"].Assigned, nodeInfo.HostResource["CPU"].Available)
	_, err := c.rm.client.Update(c.rm.context, nodeInfo)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
	//log.Println(r)
}

func (c *Captain) appendTask(appID string, taskID string, container *dockercntrl.Container) {
	c.rm.mutex.Lock()

	//log.Println("append task")
	c.rm.appIDs[appID] = struct{}{}
	c.rm.tasksTable[taskID] = container
	c.rm.mutex.Unlock()

	nodeInfo := c.GenNodeInfo()

	c.SendStatus(&nodeInfo)
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
		json.Unmarshal([]byte(l), &f)
		m := f.(map[string]interface{})
		if m["id"] != nil {
			layerID := m["id"].(string)
			status := m["status"].(string)
			if layerID == "latest" || (status != "Already exists" && status != "Pulling fs layer") {
				continue
			}
			c.rm.resource.layers[layerID] = ""
		}
	}
}
