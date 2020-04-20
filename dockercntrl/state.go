package dockercntrl

import (
  "github.com/docker/docker/client"
  "golang.org/x/net/context"
  "github.com/docker/docker/api/types"
  "github.com/docker/docker/api/types/filters"
  "github.com/docker/docker/api/types/volume"
  "bytes"
  "strings"
  "log"
)

// State holds the structs required to manipulate the docker daemon
type State struct {
  Context context.Context
  Client  *client.Client
}

// Construct a new State
func New() (*State, error) {
  ctx := context.Background()
  cli, err := client.NewEnvClient()
  return &State{Context: ctx, Client: cli}, err
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
  if _, err := s.Pull(configuration); err != nil {return nil, err}
  config, hostConfig, err := configuration.convert()
  if err != nil {return nil, err}

  resp, err := s.Client.ContainerCreate(s.Context, config, hostConfig, nil, configuration.Name)
  if err != nil {return nil, err}

  return &Container{ID: resp.ID, State: s}, nil
}

// Run runs a built docker container. It follows the execution to display
// logs at the end of execution.
func (s *State) Run(c *Container) (*string, error) {
  if err := s.Client.ContainerStart(s.Context, c.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}
	_, err := s.Client.ContainerWait(s.Context, c.ID)
	if err != nil {return nil, err}

	out, err := s.Client.ContainerLogs(s.Context, c.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return nil, err
	}
  buf := new(bytes.Buffer)
  buf.ReadFrom(out)
  logs := strings.TrimSuffix(strings.TrimSuffix(buf.String(), "\n"), "\r")
  return &logs, nil
}

// List returns all nebula-specific docker containers, determined by
// docker label
func (s *State) List() ([]*Container, error) {
  result := []*Container{}
  nebulaFilter := filters.NewArgs()
  nebulaFilter.Add("label", LABEL)
  resp, err := s.Client.ContainerList(s.Context, types.ContainerListOptions{
    All: true,
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
func (s *State) VolumeCreateIdempotent(name string) error {
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
