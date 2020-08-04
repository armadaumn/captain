// Package captain leads and manages the containers on a single machine
package captain

import (
  "context"
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/spinner/spincomm"
  "google.golang.org/grpc"
  "io"
  "log"
  "time"
)

// Captain holds state information and an exit mechanism.
type Captain struct {
  state    *dockercntrl.State
  storage  bool
  name     string
  resource *Resource
}

type Resource struct {
  totalResource      dockercntrl.Limits
  unassignedResource dockercntrl.Limits
}

// Constructs a new captain.
func New(name string) (*Captain, error) {
  state, err := dockercntrl.New()
  if err != nil {return nil, err}
  res, err := initResource(state)
  if err != nil {return nil, err}
  return &Captain{
    state: state,
    storage: false,
    name: name,
    resource: res,
  }, nil
}

// Connects to a given spinner and runs an infinite loop.
// This loop is because the dial runs a goroutine, which
// stops if the main thread closes.
func (c *Captain) Run(dialurl string) error {
  var opts []grpc.DialOption
  opts = append(opts, grpc.WithInsecure())
  conn, err := grpc.Dial(dialurl, opts...)
  if err != nil {return err}
  defer conn.Close()
  log.Println("Connected")
  client := spincomm.NewSpinnerClient(conn)
  // c.state.GetNetwork()
  // c.ConnectStorage()

  request := &spincomm.JoinRequest{
    CaptainId: &spincomm.UUID{
      Value: c.name,
    },
  }
  ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
  defer cancel()
  stream, err := client.Attach(ctx, request)
  if err != nil {return err}
  log.Println("Attached")
  // Send running status
  //go c.SendStatus(ctx, client)
  for {
    task, err := stream.Recv()
    if err == io.EOF {
      log.Println("EOF")
      return nil
    }
    if err != nil {
      return err
    }
    log.Println(task)
  }
  return nil
  // var wg sync.WaitGroup
  // for {
  //   task, err := stream.Recv()
  //   if err == io.EOF {
  //     wg.Wait()
  //     return nil
  //   }
  //   if err != nil {return err}
  //   clientstream, err := client.Run(ctx)
  //   if err != nil {return err}
  //   wg.Add(1)
  //   go func() {
  //     defer wg.Done()
  //     c.ExecuteTask(task, clientstream)
  //   }()
  //   log.Println(task)
  // }
}

func (c *Captain) SendStatus(ctx context.Context, client spincomm.SpinnerClient) {
  ctx, cancel := context.WithCancel(ctx)
  defer cancel()

  //start here
  for {
    res := c.resource
    containers, err := c.state.List(false, false)
    if err != nil {
      log.Fatalln(err)
    }
    var (
      cpuUsage         float64
      memUsage         float64
      activeContainers []string
      //images           []string
      usedPorts        []string
    )
    // Get Status of each active container
    for _, container := range containers {
      cpuPercent, memPercent, err := c.state.RealtimeRC(container.ID)
      if err != nil {
        log.Fatalln(err)
      }
      cpuUsage = cpuUsage + cpuPercent
      memUsage = memUsage + memPercent
      activeContainers = append(activeContainers, container.Image)

      ports, err := c.state.UsedPorts(container)
      if err != nil {
        log.Fatalln(err)
      }
      usedPorts = append(usedPorts, ports[:]...)
    }

    cpu := spincomm.ResourceStatus{
      Total:      res.totalResource.CPUShares,
      Unassigned: res.unassignedResource.CPUShares,
      Assigned:   res.totalResource.CPUShares - res.unassignedResource.CPUShares,
      Available:  100.0 - cpuUsage,
    }

    mem := spincomm.ResourceStatus{
      Total:      res.totalResource.Memory,
      Unassigned: res.unassignedResource.Memory,
      Assigned:   res.totalResource.Memory - res.unassignedResource.Memory,
      Available:  100.0 - memUsage,
    }
    hostResource := make(map[string]*spincomm.ResourceStatus)
    hostResource["CPU"] = &cpu
    hostResource["Memory"] = &mem

    nodeInfo := spincomm.NodeInfo{
      CaptainId: &spincomm.UUID{
        Value: c.name,
      },
      HostResource: hostResource,
      UsedPorts: usedPorts,
      ContainerStatus: &spincomm.ContainerStatus{
        ActiveContainer: activeContainers,
        Images:          activeContainers,
      },
    }

    // calling grpc
    r, err := client.Update(ctx, &nodeInfo)
    if err != nil {
      log.Fatalln(err)
    }
    log.Println(r)
    time.Sleep(10 * time.Second)
  }
}

func initResource(state *dockercntrl.State) (*Resource, error) {
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
  return &Resource{
    totalResource:      total,
    unassignedResource: avail,
  }, nil
}

// func (c *Captain) ExecuteTask(task *spincomm.TaskRequest, stream spincomm.Spinner_RunClient) {
//   config, err := dockercntrl.TaskRequestConfig(task)
//   if err != nil {
//     log.Fatal(err)
//     return
//   }
// }


// Executes a given config, waiting to print output.
// Should be changed to logging or a logging system.
// Kubeedge uses Mosquito for example.
// func (c *Captain) ExecuteConfig(config *dockercntrl.Config, write chan interface{}) {
//   container, err := c.state.Create(config)
//   if err != nil {
//     log.Println(err)
//     return
//   }
//   // For debugging
//   config.Storage = true
//   // ^^ Remove
//   if config.Storage {
//     log.Println("Storage in Config")
//     if !c.storage {
//       log.Println("Establishing Storage")
//       c.storage = true
//       c.ConnectStorage()
//     } else {
//       log.Println("Storage already exists")
//     }
//     err = c.state.NetworkConnect(container)
//     if err != nil {
//       log.Println(err)
//       return
//     }
//   } else {
//     log.Println("No storage in config")
//   }
//   s, err := c.state.Run(container)
//   if err != nil {
//     log.Println(err)
//     return
//   }
//   log.Println("Container Output: ")
//   log.Println(*s)
//   if write != nil {
//     write <- &spincomm.Response{
//       Id: config.Id,
//       Code: spincomm.Success,
//       Data: *s,
//     }
//   }
// }
