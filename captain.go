// Package captain leads and manages the containers on a single machine
package captain

import (
  "log"
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/spinner/spinresp"
)

// Captain holds state information and an exit mechanism.
type Captain struct {
  state   *dockercntrl.State
  storage bool
  name    string
}

// Constructs a new captain.
func New(name string) (*Captain, error) {
  state, err := dockercntrl.New()
  if err != nil {return nil, err}
  return &Captain{
    state: state,
    storage: false,
    name: name,
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
  client := spinresp.NewSpinnerClient(conn)
  c.state.GetNetwork()
  c.ConnectStorage()

  request := &spinresp.JoinRequest{
    CaptianId: &spinresp.UUID{
      Value: c.name,
    },
  }
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  stream, err := client.Attach(ctx, request)
  if err != nil {return err}
  var wg sync.WaitGroup
  for {
    task, err := stream.Recv()
    if err == io.EOF {
      wg.Wait()
      return nil
    }
    if err != nil {return err}
    clientstream, err := client.Run(ctx)
    if err != nil {return err}
    wg.Add(1)
    go func() {
      defer wg.Done()
      c.ExecuteTask(task, clientstream)
    }()
    log.Println(task)
  }
}

func (c *Captian) ExecuteTask(task *spinresp.TaskRequest, stream spinresp.Spinner_RunClient) {
  config, err := dockercntrl.TaskRequestConfig(task)
  if err != nil {
    log.Fatal(err)
    return
  }
  

}


// Executes a given config, waiting to print output.
// Should be changed to logging or a logging system.
// Kubeedge uses Mosquito for example.
func (c *Captain) ExecuteConfig(config *dockercntrl.Config, write chan interface{}) {
  container, err := c.state.Create(config)
  if err != nil {
    log.Println(err)
    return
  }
  // For debugging
  config.Storage = true
  // ^^ Remove
  if config.Storage {
    log.Println("Storage in Config")
    if !c.storage {
      log.Println("Establishing Storage")
      c.storage = true
      c.ConnectStorage()
    } else {
      log.Println("Storage already exists")
    }
    err = c.state.NetworkConnect(container)
    if err != nil {
      log.Println(err)
      return
    }
  } else {
    log.Println("No storage in config")
  }
  s, err := c.state.Run(container)
  if err != nil {
    log.Println(err)
    return
  }
  log.Println("Container Output: ")
  log.Println(*s)
  if write != nil {
    write <- &spinresp.Response{
      Id: config.Id,
      Code: spinresp.Success,
      Data: *s,
    }
  }
}
