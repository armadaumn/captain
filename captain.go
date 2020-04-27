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
  totalResource dockercntrl.Limits
  availResource dockercntrl.Limits
  exit    chan interface{}
}

// Constructs a new captain.
func New() (*Captain, error) {
  state, err := dockercntrl.New()
  if err != nil {return nil, err}
  res, err := state.MachineInfo()
  if err != nil {
    log.Println(err)
  }
  total := dockercntrl.Limits{
    CPUShares: int64(res.NCPU),
    Memory:    res.MemTotal,
  }
  avail := total
  list, err := state.List()
  if err != nil {
    log.Println(err)
  }
  for _, container := range list {
    resp, err := state.ContainerInspect(container)
    if err != nil {
      log.Println(err)
    }
    avail.CPUShares -= resp.HostConfig.CPUShares
    avail.Memory -= resp.HostConfig.Memory
  }

  return &Captain{
    state: state,
    totalResource: total,
    availResource: avail,
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
  if config.Limits.CPUShares > c.availResource.CPUShares || config.Limits.Memory > c.availResource.Memory {
    errInfo := "The container can't be created because it exceeds the limitation of the current machine."
    log.Println(errInfo)
    return &spinresp.Response{
      Id:   config.Id,
      Code: -8, // An temporary code representing insufficient machine resources
      Data: errInfo,
    }
  }
  c.availResource.CPUShares -= config.Limits.CPUShares
  c.availResource.Memory -= c.availResource.Memory

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
      Memory: 973741824,
    },
  }

  if config.Limits.CPUShares > c.availResource.CPUShares || config.Limits.Memory > c.availResource.Memory {
    log.Println("The spinner can't be created because it exceeds the limitation of the current machine.")
    return
  }
  c.availResource.CPUShares -= config.Limits.CPUShares
  c.availResource.Memory -= config.Limits.Memory

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
