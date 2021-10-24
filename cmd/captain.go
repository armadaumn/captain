package main

import (
	"log"
	"os"

	"github.com/armadanet/captain"
)

func main() {

	serverType := os.Args[1]
	location := os.Args[2]
	tags := []string{os.Args[3]}
	url := os.Args[4]
	localIP := os.Args[5]

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	//For evaluation, registryURL is used as BeaconURL
	// Spinner will register itself to Beacon
	//url := connectBeacon(spinnerUrl)

	cap, err := captain.New("captain", serverType)
	if err != nil {
		log.Fatalln(err)
	}
	err = cap.Run(url, location, tags, localIP)
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
