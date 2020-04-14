package main

import (
  "github.com/armadanet/captain"
  "gopkg.in/yaml.v2"
  "io/ioutil"
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
  //spinnerSelected := flag.String("spinner", "spinner", "The spinner url to connect to.")
  //selfSpin := flag.Bool("selfspin", false, "Become a spinner.")
  //flag.Parse()

  file, err := ioutil.ReadFile("config.yaml")
  if err != nil {
    panic(err)
  }
  spinConfig := SpinConfig{}
  err = yaml.Unmarshal(file, &spinConfig)

  if err != nil {
    panic(err)
  }

  // log.Println(os.Args[1])
  // con, err := url.Parse(os.Args[1])
  // if err != nil {panic(err)}

  log.Println("Creating")
  // log.Println(con.String())
  // log.Println(con.Scheme)
  // log.Println(con.User)
  cap, err := captain.New()
  if err != nil {panic(err)}

  if spinConfig.SelfSpin {
   cap.SelfSpin(spinConfig.SpinnerSelected.RetryTimes)
  } else {
   cap.Run(spinConfig.SpinnerSelected.Url, spinConfig.SpinnerSelected.RetryTimes)
  }
}
