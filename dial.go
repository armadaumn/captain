package captain

import (
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/comms"
  "log"
  "google.golang.org/grpc"
  "github.com/armadanet/spinner/spinresp"
  "time"
  "context"
)

// Dial a socket connection to a given url. Listen for reads and writes
func (c *Captain) Dial(dailurl string) error {
  var opts []grpc.DialOption
  opts = append(opts, grpc.WithInsecure())
  conn, err := grpc.Dial(dialurl, opts...)
  if err != nil {return err}
  
  socket, err := comms.EstablishSocket(dailurl)
  if err != nil {return err}
  var config dockercntrl.Config
  socket.Start(config)
  go c.connect(socket.Reader(), socket.Writer())
  return nil
}

// Read in a container config from the socket and write the
// execution output back. Should be adjusted for logging.
func (c *Captain) connect(read chan interface{}, write chan interface{}) {
  for {
    select {
    case data, ok := <- read:
      if !ok {break}
      config, ok := data.(*dockercntrl.Config)
      if !ok {break}
      log.Println(config)
      go c.ExecuteConfig(config, write)
    }
  }
}
