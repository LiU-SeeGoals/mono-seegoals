package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/gonum/mat"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/eiannone/keyboard"
)

const (
	gsim           = iota
	basestation    = iota
	remote_control = iota
)

var (
	clientType   int = 0
	commands     map[rune]command
	speed        int   = 0
	robotStopped bool  = false
	angle        int32 = 0
)

type command struct {
	message string
	run     func() action.Action
}

func main() {
	clientHost := "127.0.0.1:20011"
	client := askForClient(clientHost)
	client.Init()
	initCommands(askForRobotId())
	listenKeyboard(client)
}

func askForClient(port string) client.Client {
	var userChoice string
	var clientHost string = port
	var clientBaseStation client.Client

	fmt.Println("Enter <ip>:<port> for the basestation (port defaults to 6001): ")
	fmt.Scanln(&clientHost)
	if !strings.Contains(clientHost, ":") {
		clientHost = clientHost + ":6001"
	}

	if userChoice == "b" {
		clientType = basestation
		fmt.Println("Creating base station client.")
	} else {
		clientType = remote_control
		fmt.Println("Creating base station client for remote control.")
	}

	clientBaseStation = client.NewBaseStationClient(clientHost)
	return clientBaseStation
}

func askForRobotId() int {
	var robotId string

	fmt.Println("Please enter the robot ID (defaults to 0): ")
	fmt.Scanln(&robotId)
	id, err := strconv.Atoi(robotId)
	if err != nil {
		fmt.Println("Setting Robot ID to 0.")
		id = 0
	}
	fmt.Println("Robot ID is: ", id)
	return id
}

func initCommands(robotId int) {
	// remote control
	// In remote control mode, we've got some additional commands and expect
	// some special handling of other commands.
	commands = map[rune]command{
		'w': {
			message: "Moving forward",
			run: func() action.Action {
				robotStopped = false
				return &action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{1.0, 0.0}),
					Speed:     speed,
					Angle:     angle,
				}
			},
		},
		'a': {
			message: "Moving left",
			run: func() action.Action {
				robotStopped = false
				return &action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{0.0, 1.0}),
					Speed:     speed,
					Angle:     angle,
				}
			},
		},
		's': {
			message: "Moving backward",
			run: func() action.Action {
				robotStopped = false
				return &action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{-1.0, 0.0}),
					Speed:     speed,
					Angle:     angle,
				}
			},
		},
		'd': {
			message: "Moving right",
			run: func() action.Action {
				robotStopped = false
				return &action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{0.0, -1.0}),
					Speed:     speed,
					Angle:     angle,
				}
			},
		},
		'l': {
			message: "Stopping robot",
			run: func() action.Action {
				robotStopped = true
				return &action.Stop{
					Id: robotId,
				}
			},
		},
		'k': {
			message: "Kicking",
			run: func() action.Action {
				return &action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{0, 0}),
					Speed:     1,
					Angle:     angle,
				}
			},
		},
		'q': {
			message: "Rotating left",
			run: func() action.Action {
				robotStopped = false
				angle -= 10
				return &action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{0, 0}),
					Angle:     angle,
				}
			},
		},
		'e': {
			message: "Rotating right",
			run: func() action.Action {
				robotStopped = false
				angle += 10
				return &action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{0, 0}),
					Angle:     angle,
				}
			},
		},
	}
}

func sendCommand(char rune, client client.Client) {
	if cmd, exists := commands[char]; exists {
		fmt.Println(cmd.message)
		client.SendActions([]action.Action{cmd.run()})
	} else {
		fmt.Println("Bad command: ", char)
	}
}

func listenKeyboard(client client.Client) {
	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()

	fmt.Println("Use WASD to control the robot, <space> to stop all movement, K to kick, O/P to decrease/increase speed.")
	fmt.Println("Pings are sent continually unless <space> is pressed.")
	fmt.Println("Press <ESC> to exit.")

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		if key == keyboard.KeyEsc {
			break
		} else if key == keyboard.KeySpace {
			// translate space to "stop" command
			char = 'l'
		}

		sendCommand(char, client)

		time.Sleep(time.Millisecond)
	}
}
