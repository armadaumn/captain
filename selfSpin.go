package captain

// import (
//   "fmt"
//   "net/http"
//   "github.com/gorilla/mux"
//   "io/ioutil"
//   "github.com/google/uuid"
// )
//
//
// func (c *Captain) SelfSpin() (string, int, error) {
//   // channel to receive notice when spinner register finish
//   c := make(chan BeaconResponse)
//   q := make(chan error)
//   go spinnerNotifyChannel(c)
//
//   // create and run spinner container
//   // pass captain self info so that spinner can send info back from bridge network
//   go c.StartSpinner(q)
//
//   select {
//   case res := <-c:  // what is res
//     fmt.Println("Self-spining... Building up connection to spinner...")
//     // connect the selected spinner
//     err := c.state.JoinSwarmAndOverlay(res.Token, res.Ip, res.OverlayName)
//     if err != nil {return "",0,err}
//     return res.ContainerName, res.InternalPort, nil
//   case err := <- q:
//     log.Println(err)
//     return "",0,err
//   }
// }
//
// func (c *Captain) StartSpinner(q chan error) {
//   spinnerconfig := &dockercntrl.Config{
//     Image: "docker.io/codyperakslis/spinner",
//     Cmd: nil,
//     Tty: false,
//     Name: uuid.New().String(),
//     Env: []string{"URL=http://"+c.name+":9999/joinFinished"},
//     Limits: &dockercntrl.Limits{
//       CPUShares: 2,
//     },
//     Storage: false,
//   }
//   container, err := c.state.Create(&config)
//   if err != nil {
//     q <- err
//     return
//   }
//   _, err = c.state.Run(container)  // keep running
//   if err != nil {
//     q <- err
//     return
//   }
// }
//
// func spinnerNotifyChannel(c chan BeaconResponse) {
//   router := mux.NewRouter().StrictSlash(true)
//   s := &http.Server{
//   	Addr:           ":9999",
//   	Handler:        router,
//   }
//   router.HandleFunc("/joinFinished", func(w http.ResponseWriter, r *http.Request) {
//     var res BeaconResponse
//     body, err := ioutil.ReadAll(r.Body)
//     if err != nil {
//       log.Println(err)
//       return
//     }
//     err = json.Unmarshal(body, &res)
//     if err != nil {
//       log.Println(err)
//       return
//     }
//     c <- res
//     s.Shutdown(context.Background())
//   })
//   log.Fatal(s.ListenAndServe())
// }
