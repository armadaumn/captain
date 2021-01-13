package captain

import (
	"context"
	"github.com/armadanet/captain/dockercntrl"
	"github.com/armadanet/spinner/spincomm"
	"log"
	"sync"
)

type ResourceManager struct {
	mutex              *sync.Mutex
	context context.Context
	client spincomm.SpinnerClient
	resource *Resource
}

type Resource struct {
	totalResource      dockercntrl.Limits
	unassignedResource dockercntrl.Limits
	cpuUsage         float64
	memUsage         float64
	activeContainers []string
	//images           []string
	usedPorts        []string
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
	var (
		activeContainers []string
		usedPorts        []string
	)
	for _, container := range containers {
		cpuPercent, memPercent, err := c.state.RealtimeRC(container.ID)
		if err != nil {
			return err
		}
		c.rm.resource.cpuUsage += cpuPercent
		c.rm.resource.memUsage += memPercent
		activeContainers = append(activeContainers, container.Image)
		c.rm.resource.activeContainers = activeContainers

		ports, err := c.state.UsedPorts(container)
		if err != nil {
			return err
		}
		usedPorts = append(usedPorts, ports[:]...)
		c.rm.resource.usedPorts = usedPorts
	}
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
		//time.Sleep(10 * time.Second)
	}
}

func (c *Captain) GenNodeInfo() spincomm.NodeInfo{
	c.rm.mutex.Lock()
	defer c.rm.mutex.Unlock()

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

func initResource(state *dockercntrl.State) (*ResourceManager, error) {
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
		},
	}, nil
}
