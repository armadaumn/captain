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
func (c *Captain) Run(dialurl string) {
  if dialurl == "" {
    flag := c.SelfSpin()
    if flag == false {
      log.Println("No available spinner")
      return
    } else {
      //TODO query beacon again to get the spinner url
    }
  }
  // Continue the original workflow

  err := c.Dial(dialurl)
  if err != nil {
    log.Println(err)
  }
  select {
  case <- c.exit:
  }
}

// Executes a given config, waiting to print output.
// Should be changed to logging or a logging system.
// Kubeedge uses Mosquito for example.
func (c *Captain) ExecuteConfig(config *dockercntrl.Config) *spinresp.Response {
  // Resource check
  if c.ResourceCheck(*config) {
    return &spinresp.Response{
      Id:   config.Id,
      Code: -8, // An temporary code representing insufficient machine resources
      Data: "The container exceeds the limitation of the current machine.",
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
func (c *Captain) SelfSpin() bool {
  config := dockercntrl.Config{
    Image: "docker.io/codyperakslis/spinner",
    Cmd:   nil,
    Tty:   true,
    Name:  uuid.New().String(),
    Env:   []string{},
    Port:  0,
    Limits: &dockercntrl.Limits{
      CPUShares: 2,
      Memory: 1073741824,
    },
  }

  // Resource check
  if !c.ResourceCheck(config) {
    return false
  }

  // Create spinner container
  container, err := c.state.Create(&config)
  if err != nil {
    log.Println(err)
    return false
  }
  _, err = c.state.Run(container)
  if err != nil {
    log.Println(err)
    return false
  }

  // Connect to spinner based on brideg network
  resp, err := c.state.ContainerInspect(container)
  if err != nil {
    log.Println(err)
  }
  ip := resp.NetworkSettings.IPAddress
  url := "ws://" + ip + "/spin"
  err = c.Dial(url)
  if err != nil {
    log.Println(err)
    return false
  }
  return true
}

// Resource check
func (c *Captain) ResourceCheck(config dockercntrl.Config) bool {
  if config.Limits.CPUShares > c.availResource.CPUShares {
    log.Println("The requirement of CPU exceeds the limitation of the current host.")
    return false
  } else if config.Limits.Memory > c.availResource.Memory {
    log.Println("The requirement of Memory exceeds the limitation of the current host.")
    return false
  }
  c.availResource.CPUShares -= config.Limits.CPUShares
  c.availResource.Memory -= config.Limits.Memory
  return true
}