// Package captain leads and manages the containers on a single machine
package captain

import (
  "github.com/google/uuid"
  "log"
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/spinner/spinresp"
)

// Captain holds state information and an exit mechanism.
type Captain struct {
  state   *dockercntrl.State
  exit    chan interface{}
}

// Constructs a new captain.
func New() (*Captain, error) {
  state, err := dockercntrl.New()
  if err != nil {return nil, err}
  return &Captain{
    state: state,
  }, nil
}

// Connects to a given spinner and runs an infinite loop.
// This loop is because the dial runs a goroutine, which
// stops if the main thread closes.
func (c *Captain) Run(dialurl string, retryTimes int) {
  // Try to connect to the spinner with a limited times
  // If the captain can't make a connection, then jump to self-spin
  err := c.Dial(dialurl, retryTimes)
  if err != nil {
    log.Println(err)
    c.SelfSpin(retryTimes)
    return
  }
  select {
  case <- c.exit:
  }
}

// Executes a given config, waiting to print output.
// Should be changed to logging or a logging system.
// Kubeedge uses Mosquito for example.
func (c *Captain) ExecuteConfig(config *dockercntrl.Config) *spinresp.Response {
  container, err := c.state.Create(config)
  if err != nil {
    log.Println(err)
    return nil
  }
  s, err := c.state.Run(container)
  if err != nil {
    log.Println(err)
    return nil
  }
  log.Println("Container Output: ")
  log.Println(*s)
  return &spinresp.Response{
    Id: config.Id,
    Code: spinresp.Success,
    Data: *s,
  }
}

// Create a config for spinner
// After starting a spinner container, get its IP
// Transform the IP into ws url, make container connect to the dialurl
func (c *Captain) SelfSpin(retryTimes int) {
  config := dockercntrl.Config{
    Image: "docker.io/codyperakslis/spinner",
    Cmd:   nil,
    Tty:   true,
    Name:  uuid.New().String(),
    Env:   []string{},
    Port:  0,
    Limits: &dockercntrl.Limits{
      CPUShares: 2,
    },
  }
  container, err := c.state.Create(&config)
  if err != nil {
    log.Println(err)
  }
  _, err = c.state.Run(container)
  if err != nil {
    log.Println(err)
  }

  resp, err := c.state.ContainerInspect(container)
  if err != nil {
    log.Println(err)
  }
  ip := resp.NetworkSettings.IPAddress
  url := "ws://" + ip + "/spin"
  err = c.Dial(url, retryTimes)
  if err != nil {
    log.Println(err)
    return
  }

  //captainConfig := dockercntrl.Config{
  //  Image: "docker.io/codyperakslis/captain",
  //  Cmd:   nil,
  //  Tty:   true,
  //  Name:  uuid.New().String(),
  //  Env:   []string{},
  //  Port:  0,
  //  Limits: &dockercntrl.Limits{
  //    CPUShares: 2,
  //  },
  //}
  //container, err = c.state.Create(&captainConfig)
  //if err != nil {
  //  log.Println(err)
  //}
  //_, err = c.state.Run(container)
  //if err != nil {
  //  log.Println(err)
  //}
}
