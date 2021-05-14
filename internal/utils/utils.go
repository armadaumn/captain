package utils

import (
	"encoding/csv"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type GeoLocInfo struct {
	IP  string  `json:"ip"`
	Lat float64 `json:"latitude"`
	Lon float64 `json:"longitude"`
}

// return the public of the calling node
func GetIP() string {
	resp, err := http.Get("https://ipecho.net/plain")
	if err != nil {
		log.Println(err)
		return "0.0.0.0"
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "0.0.0.0"
	}

	publicIP := string(ip)

	return publicIP

	//var ip string
	//interfaces, _ := net.Interfaces()
	//for _, interf := range interfaces {
	//	if interf.Name == "eth0" {
	//		addrs, err := interf.Addrs()
	//		if err != nil {
	//			log.Fatal(err)
	//		}
	//		for _, addr := range addrs {
	//			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
	//				if len(ipnet.IP.String()) < 16 {
	//					ip = ipnet.IP.String()
	//					fmt.Println("Current IP address : ", ip)
	//				}
	//			}
	//		}
	//	}
	//}
	//return ip
}

// return the lat, lon of the calling node
func GetLocationInfo(ip string, loc int) (float64, float64) {

	lat := float64(0)
	lon := float64(0)
	rand.Seed(time.Now().UnixNano())
	var (
		csvFile *os.File
		err     error
	)

	if loc == 0{
		csvFile, err = os.Open("internal/utils/latlon.csv")
	} else if loc == 1 {
		csvFile, err = os.Open("internal/utils/rochester.csv")
	}else {
		csvFile, err = os.Open("internal/utils/farlocation.csv")
	}

	if err != nil {
		log.Fatalln("Error: Can't open file", err)
	}
	defer csvFile.Close()

	r := csv.NewReader(csvFile)
	randLineNumber := rand.Intn(11) + 1
	currLineNum := 1
	for {
		record, err := r.Read()
		if err != nil {
			log.Println(err)
		}
		if err == io.EOF {
			break
		}

		if currLineNum != randLineNumber {
			currLineNum++
			continue
		} else {
			// fmt.Println(record)
			lat, err = strconv.ParseFloat(record[0], 64)
			if err != nil {
				log.Println(err)
			}

			lon, err = strconv.ParseFloat(record[1], 64)
			if err != nil {
				log.Println(err)
			}
		}
		break
	}

	return lat, lon
}
