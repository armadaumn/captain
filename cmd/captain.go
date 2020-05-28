package main

import (
  "github.com/armadanet/captain"
  "os"
)

func main() {

  cap, err := captain.New(os.Args[2])
  if err != nil {panic(err)}

  // (1) beacon query url, (2) if selfSpin
  cap.Run(os.Args[1], false)
}
