// Package captain leads and manages the containers on a single machine
package captain

import (
  "log"
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/spinner/spincomm"
  "google.golang.org/grpc"
  "context"
  // "time"
  "sync"
  "io"
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
  log.Println("Connected")
  client := spincomm.NewSpinnerClient(conn)
  // c.state.GetNetwork()
  // c.ConnectStorage()

  request := &spincomm.JoinRequest{
    CaptainId: &spincomm.UUID{
      Value: c.name,
    },
  }
  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()
  stream, err := client.Attach(ctx, request)
  if err != nil {return err}
  log.Println("Attached")
  var wg sync.WaitGroup
  for {
    task, err := stream.Recv()
    if err == io.EOF {
      log.Println("EOF")
      wg.Wait()
      return nil
    }
    if err != nil {
      wg.Wait()
      return err
    }
    log.Println("Task:", task)
    logstream, err := client.Run(ctx)
    if err != nil {
      wg.Wait()
      return err
    }
    wg.Add(1)
    go func() {
      defer wg.Done()
      c.ExecuteTask(task, logstream)
    }()
  }
  return nil
}

func (c *Captain) ExecuteTask(task *spincomm.TaskRequest, stream spincomm.Spinner_RunClient) {
  config, err := dockercntrl.TaskRequestConfig(task)
  if err != nil {
    log.Fatal(err)
    return
  }
  c.ExecuteConfig(config, stream)
}


func (c *Captain) ExecuteConfig(config *dockercntrl.Config, stream spincomm.Spinner_RunClient) {
  container, err := c.state.Create(config)
  if err != nil {
    log.Println(err)
    return
  }
  logReader, err := c.state.Run(container)
  if err != nil {
    log.Println(err)
    return 
  }
  buf := make([]byte, 128)
  for {
    n, err := logReader.Read(buf)
    if err != nil {
      log.Println(err)
      stream.CloseAndRecv()
      return
    }
    if n > 0 {
      logValue := string(buf[:n])
      log.Println("Log:", logValue)
      stream.Send(&spincomm.TaskLog{
        TaskId: &spincomm.UUID{Value: config.Id},
        Log: logValue,
      })
    }
  }
}

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
