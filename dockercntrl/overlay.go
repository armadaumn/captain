package dockercntrl

import(
  "net/http"
  "io/ioutil"
  "log"
  "fmt"
  "errors"
  "encoding/json"
)

// Captain/Spinner join the overlay network
// given target token, ip, overlay name, error
// 1) get self ip
// 2) join the swarm
// 3) attach self to overlay
// 4) wait for network setup

// Beacon create overlay network
// Input: containerName overlayName
// Return: token, ip, error
func (s *State) BeaconCreateOverlay(containerName string, overlayName string) (string, string, error) {
  // 1) get self ip (ip)
  ipInfo, err := getIpInfo()
  if err != nil {
    return "", "", err
  }
  // 2) initialize a new swarm
  respCode, err := s.CreateSwarm(ipInfo.Ip)
  if err != nil {
    return "","",err
  }
  if respCode != 200 {
    return "","",errors.New(fmt.Sprintf("Create new swarm failed. Response code: %d", respCode))
  }

  // 3) get the swarm token
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
  // 4) create overlay network
  respCode, err = s.CreateOverlay(overlayName)
  if err != nil {
    return "","",err
  }
  if respCode != 201 {
    return "","",errors.New(fmt.Sprintf("Create Overlay network failed. Response code: %d", respCode))
  }
  // 5) attach self to beacon overlay
  respCode, err = s.AttachOverlay(containerName, overlayName)
  if err != nil {
    return "","",err
  }
  if respCode != 200 {
    return "","",errors.New(fmt.Sprintf("Attach Overlay network failed. Response code: %d", respCode))
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
