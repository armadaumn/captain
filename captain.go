// Package captain leads and manages the containers on a single machine
package captain

import (
	"context"
	"github.com/armadanet/captain/dockercntrl"
	"github.com/armadanet/captain/internal/utils"
	"github.com/armadanet/spinner/spincomm"
	"google.golang.org/grpc"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"
)

// Captain holds state information and an exit mechanism.
type Captain struct {
	state      *dockercntrl.State
	storage    bool
	name       string
	rm         *ResourceManager
	serverType spincomm.Type
}

// Constructs a new captain.
func New(name string, serverType string) (*Captain, error) {
	state, err := dockercntrl.New()
	if err != nil {
		return nil, err
	}
	res, err := initResourceManager(state)
	if err != nil {
		return nil, err
	}

	// random name
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomName := make([]rune, 10)
    rand.Seed(time.Now().UnixNano())
	for i := range randomName {
		randomName[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	var tp spincomm.Type
	if serverType == "server" {
		tp = spincomm.Type_LocalServer
	} else {
		tp = spincomm.Type_Volunteer
	}

	return &Captain{
		state:      state,
		storage:    false,
		name:       string(randomName),
		rm:         res,
		serverType: tp,
	}, nil
}

// Connects to a given spinner and runs an infinite loop.
// This loop is because the dial runs a goroutine, which
// stops if the main thread closes.
func (c *Captain) Run(dialurl string, loc string, tags []string) error {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(dialurl, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Println("Connected")
	client := spincomm.NewSpinnerClient(conn)
	// c.state.GetNetwork()
	// c.ConnectStorage()
	log.Println(c.name)

	synth := false
	ip := "0.0.0.0"
	lat := 0.0
	lon := 0.0
	if !synth {
		ip = utils.GetPublicIP()
	}
	isClose := 1
	if loc == "close" {
		isClose = 0
	} else if loc == "far" {
		isClose = 2
	}
	lat, lon = utils.GetLocationInfo(ip, isClose)

	request := &spincomm.JoinRequest{
		CaptainId: &spincomm.UUID{
			Value: c.name,
		},
		IP: ip,
		Lat: lat,
		Lon: lon,
		Type: c.serverType,
		Tags: tags,
	}
	log.Println(request)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.Attach(ctx, request)
	if err != nil {
		return err
	}
	log.Println("Attached")
	// Send running status
	go c.UpdateRealTimeResource()
	go c.PeriodicalUpdate(ctx, client)

	var wg sync.WaitGroup
	for {
		task, err := stream.Recv()
		if err == io.EOF {
			log.Println("EOF")
			wg.Wait()
			c.RemoveTask()
			return nil
		}
		if err != nil {
			wg.Wait()
			return err
		}
		log.Println("Task:", task)
		logstream, err := client.Run(ctx)
		if err != nil {
			wg.Wait()
			c.RemoveTask()
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.ExecuteTask(task, logstream)
		}()
	}
	return nil
}

func (c *Captain) ExecuteTask(task *spincomm.TaskRequest, stream spincomm.Spinner_RunClient) {
	cargoSpec := task.GetTaskspec().GetCargoSpec()
	if cargoSpec != nil {
		cargoIP := cargoSpec.GetIPs()[0]
		cargoPort := cargoSpec.GetPorts()[0]
		appID := task.GetAppId().GetValue()
		task.Command = append(task.Command, cargoIP)
		task.Command = append(task.Command, cargoPort)
		task.Command = append(task.Command, appID)
		task.Command = append(task.Command, "1")
	}
	log.Println("Task:", task)
	config, err := dockercntrl.TaskRequestConfig(task)
	if err != nil {
		log.Fatal(err)
		return
	}
	c.RequestResource(config)

	// Start the task
	container, err := c.state.Create(config)
	if err != nil {
		log.Println(err)
		return
	}

	c.appendTask(task.GetAppId().GetValue(), config.Id, container)

	logReader, err := c.state.Run(container)
	if err != nil {
		log.Println(err)
		c.removeTask(task.GetAppId().GetValue(), config.Id)
		c.ReleaseResource(config)
		return
	}

	buf := make([]byte, 128)
	for {
		_, err := logReader.Read(buf)
		if err != nil {
			log.Println(err)
			c.removeTask(task.GetAppId().GetValue(), config.Id)
			c.ReleaseResource(config)
			//c.state.Kill(container)
			//c.state.Remove(container)
			//stream.CloseAndRecv()
			return
		}
		//if n > 0 {
		//	logValue := string(buf[:n])
		//	log.Println("Log:", logValue)
		//	stream.Send(&spincomm.TaskLog{
		//		TaskId: &spincomm.UUID{Value: config.Id},
		//		//Log: logValue,
		//	})
		//}
	}
}

func (c *Captain) RemoveTask() {
	taskTable := c.getTaskTable()
	for _, container := range taskTable {
		c.state.Kill(container)
		c.state.Remove(container)
	}
}

// Executes a given config, waiting to print output.
// Should be changed to logging or a logging system.
// Kubeedge uses Mosquito for example.
// func (c *Captain) ExecuteConfig(config *dockercntrl.Config, write chan interface{}) {
//   container, err := c.state.Create(config)
//   if err != nil {
//     log.Println(err)
//     return
//   }
//   // For debugging
//   config.Storage = true
//   // ^^ Remove
//   if config.Storage {
//     log.Println("Storage in Config")
//     if !c.storage {
//       log.Println("Establishing Storage")
//       c.storage = true
//       c.ConnectStorage()
//     } else {
//       log.Println("Storage already exists")
//     }
//     err = c.state.NetworkConnect(container)
//     if err != nil {
//       log.Println(err)
//       return
//     }
//   } else {
//     log.Println("No storage in config")
//   }
//   s, err := c.state.Run(container)
//   if err != nil {
//     log.Println(err)
//     return
//   }
//   log.Println("Container Output: ")
//   log.Println(*s)
//   if write != nil {
//     write <- &spincomm.Response{
//       Id: config.Id,
//       Code: spincomm.Success,
//       Data: *s,
//     }
//   }
// }
