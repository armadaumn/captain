package main

import (
  "github.com/armadanet/captain"
  "strconv"
  "os"
)

func main() {

  cap, err := captain.New(os.Args[2])
  if err != nil {panic(err)}

  selfSpin, err := strconv.ParseBool(os.Getenv("SELFSPIN"))
  if err != nil {panic(err)}

  // (1) beacon query url, (2) if selfSpin
  cap.Run(os.Args[1], selfSpin)
}
