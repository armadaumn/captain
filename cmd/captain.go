package main

import (
  "github.com/armadanet/captain"
  //"flag"
  //"net/url"
  "log"
)

type SpinConfig struct {
  SelfSpin bool
  SpinnerSelected struct {
    Url string
    RetryTimes int
  }
}

func main() {
  dialurl := ""
  //TODO query beacon
  //dialurl := Beacon


  // log.Println(os.Args[1])
  // con, err := url.Parse(os.Args[1])
  // if err != nil {panic(err)}

  log.Println("Creating")
  // log.Println(con.String())
  // log.Println(con.Scheme)
  // log.Println(con.User)
  cap, err := captain.New()
  if err != nil {panic(err)}

  cap.Run(dialurl)
}
