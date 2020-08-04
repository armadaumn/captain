package dockercntrl

import (
  "github.com/docker/docker/api/types/container"
  "github.com/docker/docker/api/types/mount"
  "github.com/docker/go-connections/nat"
  "github.com/phayes/freeport"
  // "github.com/google/uuid"
  "strconv"
  "github.com/armadanet/spinner/spincomm"
)

// Limits hold the set of limits for a given container.
type Limits struct {
  CPUShares int64     `json:"cpushares"`
  Memory    int64     `json:"Memory limit (in bytes)"`
}

// Config represents the configuration to build a new container.
type Config struct {
  Id        string      `json:"nebula_id,omitempty"`
  Image     string      `json:"image"`
  Cmd       []string    `json:"command"`
  Tty       bool        `json:"tty"`
  Name      string      `json:"name"`
  Limits    *Limits     `json:"limits"`
  Env       []string    `json:"env"`
  Port      int         `json:"port"`
  Storage   bool        `json:"storage"`
  mounts    []mount.Mount
}

const (
  LABEL = "nebula-id"
)

func TaskRequestConfig(task *spincomm.TaskRequest) (*Config, error) {
  config := &Config{
    Id: task.GetTaskId().GetValue(),
  }
  return config, nil
}

func (c *Config) AddMount(name string) {
  c.mounts = []mount.Mount{
    {
      Type: mount.TypeVolume,
      Source: name,
      Target: "/data",
    },
  }
}

// Converts a dockercntrl.Config into the necessary docker-go-sdk configs
func (c *Config) convert() (*container.Config, *container.HostConfig, error) {
  var id string
  if c.Id != "" {id = c.Id}
  config := &container.Config{
    Image: c.Image,
    Cmd: c.Cmd,
    Tty: c.Tty,
    Env: c.Env,
    Labels: map[string]string{
      LABEL: id, // To identify as belonging to nebula
    },
  }

  hostConfig := &container.HostConfig{
    Resources: container.Resources{
      CPUShares: c.Limits.CPUShares,
    },
    Mounts: c.mounts,
  }

  // If port is supplied, open that port on the container thru
  // a random open port on the host machine.
  if c.Port != 0 {
    port, err := nat.NewPort("tcp", strconv.Itoa(c.Port))
    if err != nil {return config, hostConfig, err}
    config.ExposedPorts = nat.PortSet{port: struct{}{}}
    openPort, err := freeport.GetFreePort()
    if err != nil {return config, hostConfig, err}
    hostConfig.PortBindings = nat.PortMap{
      port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(openPort)}},
    }
  }

  return config, hostConfig, nil
}
