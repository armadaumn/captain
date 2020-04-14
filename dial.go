package captain

import (
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/comms"
  "log"
)

// Dial a socket connection to a given url. Listen for reads and writes
func (c *Captain) Dial(dailurl string, retryTimes int) error {
  var socket comms.Socket
  for i := 0; i < retryTimes; i++ {
    var err error
    socket, err = comms.EstablishSocket(dailurl)
    if err == nil {
      break
    }
    if i == retryTimes - 1 {
      return err
    }
  }
  //socket, err = comms.EstablishSocket(dailurl)
  //if err != nil {return err}
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
      write <- c.ExecuteConfig(config)
    }
  }
}
