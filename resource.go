package captain

import (
	"context"
	"github.com/armadanet/captain/dockercntrl"
	"github.com/armadanet/spinner/spincomm"
	"log"
	"sync"
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
	cpuUsage           float64
	memUsage           float64
	activeContainers   []string
	//images           []string
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
			activeContainers:   make([]string, 0),
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
	c.rm.mutex.Lock()
	defer c.rm.mutex.Unlock()

	containers, err := c.state.List(false, false)
	if err != nil {
		return err
	}

	activeContainers := make([]string, 0)
	usedPorts := make(map[string]string)
	cpuUsage := 0.0
	memUsage := 0.0
	for _, container := range containers {
		cpuPercent, memPercent, err := c.state.RealtimeRC(container.ID)
		if err != nil {
			return err
		}
		cpuUsage += cpuPercent
		memUsage += memPercent
		activeContainers = append(activeContainers, container.Image)
		c.rm.resource.activeContainers = activeContainers

		ports, err := c.state.UsedPorts(container)

		if err != nil {
			return err
		}
		if len(ports) == 0{
			usedPorts[container.Names[0][1:]] = ""
		} else {
			usedPorts[container.Names[0][1:]] = ports[0]
		}
		c.rm.resource.usedPorts = usedPorts
	}
	c.rm.resource.cpuUsage = cpuUsage
	c.rm.resource.memUsage = memUsage
	return nil
}

func (c *Captain) PeriodicalUpdate(ctx context.Context, client spincomm.SpinnerClient) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c.rm.context = ctx
	c.rm.client = client

	for {
		err := c.UpdateRealTimeResource()
		if err != nil {
			log.Fatalln(err)
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
		Available:  100.0 - c.rm.resource.cpuUsage,
	}

	mem := spincomm.ResourceStatus{
		Total:      c.rm.resource.totalResource.Memory,
		Unassigned: c.rm.resource.unassignedResource.Memory,
		Assigned:   c.rm.resource.totalResource.Memory - c.rm.resource.unassignedResource.Memory,
		Available:  100.0 - c.rm.resource.memUsage,
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
	for id, _ := range c.rm.appIDs {
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
	}
	return nodeInfo
}

func (c *Captain) SendStatus(nodeInfo *spincomm.NodeInfo) {
	//c.rm.mutex.Lock()
	//defer c.rm.mutex.Unlock()

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
}

func (c *Captain) getTaskTable() map[string]*dockercntrl.Container {
	c.rm.mutex.Lock()
	defer c.rm.mutex.Unlock()

	return c.rm.tasksTable
}