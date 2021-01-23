package main

import (
  "github.com/armadanet/captain"
  "os"

  // "net/url"
  "log"
  // "os"
  // "flag"
)

func main() {

  serverType := os.Args[1]
  location := os.Args[2]
  tags := []string{os.Args[3]}

  cap, err := captain.New("captain", serverType)
  if err != nil {log.Fatalln(err)}
  err = cap.Run("spinner:5912", location, tags)
  if err != nil {
    log.Fatalln(err)
  }
  // spinnerSelected := flag.String("spinner", "spinner", "The spinner url to connect to.")
  // selfSpin := flag.Bool("selfspin", false, "Become a spinner.")
  // flag.Parse()

  // log.Println(os.Args[1])
  // con, err := url.Parse(os.Args[1])
  // if err != nil {panic(err)}

  // log.Println("Creating")
  // log.Println(con.String())
  // log.Println(con.Scheme)
  // log.Println(con.User)
  // cap, err := captain.New(os.Args[2])
  // if err != nil {panic(err)}

  // cap.Run(os.Args[1])
}
