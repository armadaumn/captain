// Package captain leads and manages the containers on a single machine
package captain

import (
  "log"
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/spinner/spinresp"
  "github.com/armadanet/comms"
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
func (c *Captain) Run(beaconURL string, selfSpin bool) {
  // query beacon for a spinner
  spinner_name, err := c.QueryBeacon(beaconURL, selfSpin)
  if err != nil {
    log.Println(err)
    return
  }

  // Register to selected spinner and start acting as a worker
  // TODO: dial fails -> repeat the above operation
  // TODO: connected spinner failed -> find a new spinner to connect
  err = c.Dial("wss://"+spinner_name+":5912/join")
  if err != nil {
    log.Println(err)
    return
  }
  //c.state.GetNetwork()
  //c.ConnectStorage()
  select {
  case <- c.exit:
  }
}

type BeaconResponse struct {
  Valid         bool    `json:"Valid"`  // true if find a spinner
  Token         string  `json:"Token"`
  Ip            string  `json:"Ip"`
  OverlayName   string  `json:"OverlayName"`
  ContainerName string  `json:"ContainerName"`
}

func (c *Captain) QueryBeacon(beaconURL string, selfSpin bool) (string, error) {
  var res BeaconResponse
  // query beacon for spinner
  err := comms.SendGetRequest(beaconURL, &res)
  if err != nil {return "",err}

  // selfSpin
  if selfSpin || !res.Valid {
    // spinner_name, internal_port, err = c.SelfSpin()
    // if err != nil {return "",err}
  }

  // connect the selected spinner
  err = c.state.JoinSwarmAndOverlay(res.Token, res.Ip, res.OverlayName, c.name)
  if err != nil {return "",err}
  return res.ContainerName, nil
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
