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
  exit    chan interface{}
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
func (c *Captain) Run(dialurl string) {
  err := c.Dial(dialurl)
  if err != nil {
    log.Println(err)
    return
  }
  c.state.GetNetwork()
  c.ConnectStorage()
  select {
  case <- c.exit:
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
