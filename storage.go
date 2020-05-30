package captain

import (
  "github.com/armadanet/captain/dockercntrl"
)

func (c *Captain) ConnectStorage() {
  storageconfig := &dockercntrl.Config{
    //Image: "docker.io/codyperakslis/armada-cargo",
    Image: "docker.io/geoffreyhl/armada-cargo",
    Cmd: []string{"./main"},
    Tty: false,
    Name: "armada-storage-"+c.name,
    Limits: &dockercntrl.Limits{
      CPUShares: 4,
    },
    Env: []string{},
    Storage: true,
  }
  c.state.VolumeCreate("cargo")
  storageconfig.AddMount("cargo")
  go c.ExecuteConfig(storageconfig, nil)
}
