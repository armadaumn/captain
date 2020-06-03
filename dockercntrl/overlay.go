package dockercntrl

import(
  "net/http"
  "io/ioutil"
  "log"
  "fmt"
  "time"
  "errors"
  "encoding/json"
)

// Captain/Spinner join the overlay network
// join the swarm first
// given target token, ip, overlay name and self_container_name
func (s *State) JoinSwarmAndOverlay(token string, ip string, containerName, overlayName string) error {
  // 1) get self ip
  ipInfo, err := getIpInfo()
  if err != nil {return err}

  // 2) join the swarm
  respCode, err := s.JoinSwarm(ipInfo.Ip, token, ip)
  if err != nil {return err}
  if respCode != 200 {return errors.New(fmt.Sprintf("Join swarm failed. Response code: %d", respCode))}

  // 3) attach self to overlay
  // respCode, respMessage, err := s.AttachOverlay(containerName, overlayName)
  err = s.AttachNetwork(containerName, overlayName)
  if err != nil {return err}

  // 4) wait for network setup
  time.Sleep(5*time.Second)
  fmt.Println(containerName+" successfully join the overlay "+overlayName)
  return nil
}

// join the overlay (already join the swarm)
func (s *State) JoinOverlay(containerName, overlayName string) error {
  // 1) attach self to overlay
  err := s.AttachNetwork(containerName, overlayName)
  if err != nil {return err}

  // 2) wait for network setup
  time.Sleep(5*time.Second)
  fmt.Println(containerName+" successfully join the overlay "+overlayName)
  return nil
}

/* Beacon create overlay network
for a new joined spinner */
func (s *State) BeaconCreateSpinnerOverlay(overlayName string) error {
  respCode, err := s.CreateOverlay(overlayName)
  if err != nil {
    return err
  }
  if respCode != 201 {
    return errors.New(fmt.Sprintf("Create Overlay network failed. Response code: %d", respCode))
  }
  return nil
}

/* Beacon create overlay network
Input: containerName overlayName
Return: token, beacon_ip, error */
func (s *State) BeaconCreateOverlay(containerName string, overlayName string) (string, string, error) {
  // get self ip info
  ipInfo, err := getIpInfo()
  if err != nil {
    return "", "", err
  }
  // initialize a new swarm
  respCode, err := s.CreateSwarm(ipInfo.Ip)
  if err != nil {
    return "","",err
  }
  if respCode != 200 {
    return "","",errors.New(fmt.Sprintf("Create new swarm failed. Response code: %d", respCode))
  }
  // get the swarm token
  respCode, swarmInfo, err := s.GetSwarmInfo()
  if err != nil {
    return "","",err
  }
  token := ""
  if respCode == 200 {
    token = swarmInfo.JoinTokens["Worker"]
  } else {
    return "","",errors.New(fmt.Sprintf("Failed getting swarm info. Response code: %d", respCode))
  }
  // create overlay network
  respCode, err = s.CreateOverlay(overlayName)
  if err != nil {
    return "","",err
  }
  if respCode != 201 {
    return "","",errors.New(fmt.Sprintf("Create Overlay network failed. Response code: %d", respCode))
  }
  // attach beacon to beacon overlay
  //err = s.AttachOverlay(containerName, overlayName)
  err = s.AttachNetwork(containerName, overlayName)
  if err != nil {
    return "","",err
  }

  return token, ipInfo.Ip, nil
}




// Spinner create overlay network (beacon call this func)
// Input: spinner_Overlay_name
// Output: error
// 1) create overlay network (name)





type IpInfo struct {
	Ip         string `json:"ip"`
	City       string `json:"city"`
	Loc        string `json:"loc"`
}

func getIpInfo() (*IpInfo, error) {
  var info IpInfo
  // ip info loop-up service from ipinfo.io
  response, err := http.Get("http://ipinfo.io/?token=8925eef57c197f")
  if err != nil {
		log.Println(err)
    return nil, err
	}
  body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
    return nil, err
	}
  err = json.Unmarshal(body, &info)
	if err != nil {
		log.Println(err)
    return nil, err
	}
  response.Body.Close()
  return &info, nil
}
