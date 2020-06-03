package captain

import (
  "log"
  "net/http"
  "github.com/gorilla/mux"
  "github.com/armadanet/captain/dockercntrl"
  "io/ioutil"
  "os"
  "encoding/json"
  "context"
  "time"
  //"github.com/google/uuid"
)

type chanMessage struct {
  Spinner_Overlay string `json:"OverlayName"`
}

func (c *Captain) SelfSpin() (string, string) {
  // get spinner name from env var
  spinner_name := os.Getenv("SPINNER_NAME")
  // start channel listener
  ch := make(chan chanMessage)
  go spinnerNotifyChannel(ch)
  // create and run spinner container
  go c.StartSpinner(spinner_name)

  select {
  case mes := <-ch:
    return mes.Spinner_Overlay, spinner_name
  }
}

func (c *Captain) StartSpinner(spinner_name string) {
  spinnerBeaconQueryUrl := os.Getenv("BEACON_QUERY")
  spinnerconfig := &dockercntrl.Config{
    Image: "docker.io/geoffreyhl/spinner",
    Cmd: []string{"./main"},
    Tty: false,
    // Name: uuid.New().String(),
    Name: spinner_name,
    Limits: &dockercntrl.Limits{
      CPUShares: 4,
    },
    // pass captain name as env var
    Env: []string{
      "CAPTAIN_URL=http://"+c.name+":9999/joinFinished",
      "SPINNERID="+spinner_name,
      "URL="+spinnerBeaconQueryUrl,
      "SELFSPIN=true",
    },
    Storage: false,
  }
  // -v /var/run/docker.sock:/var/run/docker.sock
  spinnerconfig.AddDeamonMount()
  go c.ExecuteConfig(spinnerconfig, nil)
}

func spinnerNotifyChannel(c chan chanMessage) {
  router := mux.NewRouter().StrictSlash(true)
  s := &http.Server{
  	Addr:           ":9999",
  	Handler:        router,
  }
  qs := make(chan int)
  go quitServer(qs, s)
  router.HandleFunc("/joinFinished", func(w http.ResponseWriter, r *http.Request) {
    var res chanMessage
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Println(err)
      return
    }
    err = json.Unmarshal(body, &res)
    if err != nil {
      log.Println(err)
      return
    }
    // get the notice from started spinner
    c <- res
    // notify -> stop the server
    qs <- 0
  })
  //log.Fatal(s.ListenAndServe())
  s.ListenAndServe()
}

// shut down the server after message received
func quitServer(qs chan int, s *http.Server) {
  <- qs
  time.Sleep(1*time.Second)
  log.Println("Shutting down spinner channel...")
  s.Shutdown(context.Background())
}
