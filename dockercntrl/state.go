package dockercntrl

import (
  "encoding/json"
  "github.com/docker/docker/client"
  "golang.org/x/net/context"
  "github.com/docker/docker/api/types"
  "github.com/docker/docker/api/types/filters"
  "github.com/docker/docker/api/types/volume"
  "bytes"
  "io/ioutil"
  // "strings"
  "log"
  "net/http"
  "net"
  "io"
)

// State holds the structs required to manipulate the docker daemon
type State struct {
  Context context.Context
  Client  *client.Client
  // TODO: switch to sdk
  HttpUnix  *http.Client
}

// Construct a new State
func New() (*State, error) {
  ctx := context.Background()
  cli, err := client.NewEnvClient()
  // initiate unix requester TODO: switch to sdk
  httpUnix := http.Client{
    Transport: &http.Transport{
      DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
        return net.Dial("unix", "/var/run/docker.sock")
      },
    },
  }
  return &State{Context: ctx, Client: cli, HttpUnix: &httpUnix}, err
}

// Pull pulls the associated image into cache
func (s *State) Pull(config *Config) (*string, error) {
  reader, err := s.Client.ImagePull(s.Context, config.Image, types.ImagePullOptions{})
  if err != nil {
    return nil, err
  }
  buf := new(bytes.Buffer)
  buf.ReadFrom(reader)
  logs := buf.String()
  return &logs, err
}

// Create builds a docker container
func (s *State) Create(configuration *Config) (*Container, error) {
  // if _, err := s.Pull(configuration); err != nil {return nil, err}
  config, hostConfig, err := configuration.convert()
  if err != nil {return nil, err}

  resp, err := s.Client.ContainerCreate(s.Context, config, hostConfig, nil, configuration.Name)
  if err != nil {return nil, err}

  return &Container{ID: resp.ID, State: s}, nil
}

// Run runs a built docker container. It follows the execution to display
// logs at the end of execution.
func (s *State) Run(c *Container) (io.ReadCloser, error) {
  if err := s.Client.ContainerStart(s.Context, c.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}
	_, err := s.Client.ContainerWait(s.Context, c.ID)
  log.Println(err)
	if err != nil {return nil, err}

	out, err := s.Client.ContainerLogs(s.Context, c.ID, types.ContainerLogsOptions{
    ShowStdout: true,
    ShowStderr: true,
    Follow: true,
  })
	if err != nil {
		return nil, err
  }
  return out, nil
  // buf := new(bytes.Buffer)
  // buf.ReadFrom(out)
  // logs := strings.TrimSuffix(strings.TrimSuffix(buf.String(), "\n"), "\r")
  //return &logs, nil
}

// List returns all nebula-specific docker containers, determined by
// docker label
func (s *State) List(allFilter bool, nebulaOnly bool) ([]*Container, error) {
  result := []*Container{}
  nebulaFilter := filters.NewArgs()
  if nebulaOnly {
    nebulaFilter.Add("label", "nebula-id=captain")
  }

  resp, err := s.Client.ContainerList(s.Context, types.ContainerListOptions{
    All: allFilter,
    Filters: nebulaFilter,
  })
  if err != nil {
    return result, err
  }
  for _, c := range resp {
    result = append(result, &Container{
      ID: c.ID,
      State: s,
      Names: c.Names,
      Image: c.Image,
      Command: c.Command,
    })
  }

  return result, nil
}

// Kill immediately ends a docker container
func (s *State) Kill(cont *Container) error {
  // Sends SIGTERM followed by SIGKILL after a graceperio
  // Change last value from nil to give custom graceperiod
  err := s.Client.ContainerStop(s.Context, cont.ID, nil)
  if err != nil {
    return err
  }
  // TODO: ContainerRemove to clean from system
  return nil
}

// Remove clears a docker container from the docker deamon
func (s *State) Remove(cont *Container) error {
  err := s.Client.ContainerRemove(s.Context, cont.ID, types.ContainerRemoveOptions{
    RemoveVolumes: false,
    RemoveLinks: false,
    Force: true,
  })
  return err
}

// Creates a Volume
func (s *State) VolumeCreate(name string) error {
  // Check if overwrites
  v := volume.VolumesCreateBody{
    Driver: "local",
    DriverOpts: map[string]string{},
    Labels: map[string]string{
      LABEL: "default-storage",
    },
    Name: name,
  }
  vol, err := s.Client.VolumeCreate(s.Context, v)
  log.Println(vol)
  return err
}

// Get container detailed information, equivalent to docker inspect
func (s *State) ContainerInspect(c *Container) (types.ContainerJSON, error) {
  resp, err := s.Client.ContainerInspect(s.Context, c.ID)
  if err != nil {
    return resp, err
  }
  return resp, nil
}

// Get information of machine (docker server)
func (s *State) MachineInfo() (types.Info, error) {
  resp, err := s.Client.Info(s.Context)
  if err != nil {
    return resp, err
  }
  return resp, nil
}

func (s *State) Stats(cID string) (types.ContainerStats, error){
  stat, err := s.Client.ContainerStats(s.Context, cID, false)
  if err != nil {
    return stat, err
  }
  return stat, err
}

func (s *State) UsedPorts(c *Container) ([]string, error) {
  var ports []string
  resp, err := s.ContainerInspect(c)
  if err != nil {
    return nil, err
  }
  for _, portMap := range resp.NetworkSettings.Ports {
    for _, portArray := range portMap {
      ports = append(ports, portArray.HostPort)
    }
  }
  return ports, nil
}

func (s *State) RealtimeRC(cID string) (float64, float64, error) {
  resp, err := s.Stats(cID)
  if err != nil {
    return 0, 0, err
  }
  buf, _ := ioutil.ReadAll(resp.Body)
  var stats types.Stats
  json.Unmarshal(buf, &stats)

  var cpuPercent, memPercent float64
  containerDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
  systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
  if containerDelta > 0.0 && systemDelta > 0.0 {
    numCPUs := float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
    cpuPercent = (containerDelta/systemDelta) * numCPUs * 100.0
  } else {
    cpuPercent = 0
  }

  memLimit := float64(stats.MemoryStats.Limit)
  if memLimit != 0 {
    memPercent = float64(stats.MemoryStats.Usage) / memLimit * 100.0
  } else {
    memPercent = 0
  }
  return cpuPercent, memPercent, nil
}