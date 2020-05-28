package dockercntrl

import(
  "encoding/json"
  "bytes"
  "io/ioutil"
  "log"
)

type SwarmInfo struct {
  Id        string            `json:"ID"`
  JoinTokens map[string]string `json:"JoinTokens"`
}

// initialize the swarm
func (s *State) CreateSwarm(myIp string) (int, error) {
  requestBody, err := json.Marshal(map[string]string{
    "ListenAddr": "0.0.0.0:2377",
    "AdvertiseAddr":myIp,
  })
  if err != nil {
		log.Println(err)
    return 0, err
	}
  response, err := s.HttpUnix.Post("http://unix/swarm/init", "application/json", bytes.NewBuffer(requestBody))
  if err != nil {
		log.Println(err)
    return 0, err
	}
  return response.StatusCode, nil
}

// get swarm info
func (s *State) GetSwarmInfo() (int, *SwarmInfo, error) {
  response, err := s.HttpUnix.Get("http://unix/swarm")
  if err != nil {
		log.Println(err)
    return 0, nil, err
	}
  if response.StatusCode == 200 {
    body, err := ioutil.ReadAll(response.Body)
  	if err != nil {
  		log.Println(err)
      return 0, nil, err
  	}
    var swarmInfo SwarmInfo
    err = json.Unmarshal(body, &swarmInfo)
  	if err != nil {
  		log.Println(err)
      return 0, nil, err
  	}
    response.Body.Close()
    return 200, &swarmInfo, nil
  } else {
    return response.StatusCode, nil, nil
  }
}

// join the swarm
func (s *State) JoinSwarm(myIp, token, managerIp string) (int, error) {
  requestBody, err := json.Marshal(map[string]interface{}{
    "ListenAddr": "0.0.0.0:2377",
    "AdvertiseAddr": myIp,
    "RemoteAddrs": []string {
      managerIp+":2377",
    },
    "JoinToken": token,
  })
  if err != nil {
		log.Println(err)
    return 0, err
	}
  response, err := s.HttpUnix.Post("http://unix/swarm/join", "application/json", bytes.NewBuffer(requestBody))
  if err != nil {
		log.Println(err)
    return 0, err
	}
  return response.StatusCode, nil
}
