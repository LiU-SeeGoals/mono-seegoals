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

var (
	prevChar rune
)

var (
    clientType int = 0
    commands map[rune]command
    speed int = 0
    robotStopped bool = false
)

type command struct {
    message string
    run func() action.Action
}

func main() {
    clientHost := "127.0.0.1:20011"
    client := askForClient(clientHost)
    client.Init()
    initCommands(askForRobotId())
    listenKeyboard(client)
}

// Client for base station or for sim
func askForClient(port string) client.Client {
	fmt.Println("Please enter the client type (g)sim (default) or (b)base_station: ")
	var clientType string
	fmt.Scanln(&clientType)
	switch clientType {
	case "b":
		fmt.Println("Enter <ip>:<port> for the basestation, port defaults to 6001: ")
		var basestationIP string
		fmt.Scanln(&basestationIP)
		if !strings.Contains(basestationIP, ":") {
			basestationIP = basestationIP + ":6001"
		}
		fmt.Println("Creating base station client.")
		return client.NewBaseStationClient(basestationIP)
	}
	fmt.Println("Creating sim client.")
	return client.NewBaseStationClient(port)
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

func sendCommand(robotId int, char rune, client client.Client) {
	actions := []action.Action{}

	if prevChar == char { // same command as last time -> no need to send it again
		return
	} else { // new command
		switch char {
		case 'w':
			fmt.Println("Moving forward")
			action := &action.Move{
				Id:        robotId,
				Direction: mat.NewVecDense(2, []float64{0.0, 1.0}),
			}
			actions = append(actions, action)
		case 'l':
			fmt.Println("Stopping robot")
			action := &action.Stop{
				Id: robotId,
			}
			actions = append(actions, action)
		case 'a':
			fmt.Println("Moving left")
			action := &action.Move{
				Id:        robotId,
				Direction: mat.NewVecDense(2, []float64{1.0, 0.0}),
			}
			actions = append(actions, action)
		case 's':
			fmt.Println("Moving backward")
			action := &action.Move{
				Id:        robotId,
				Direction: mat.NewVecDense(2, []float64{0.0, -1.0}),
			}
			actions = append(actions, action)
		case 'd':
			fmt.Println("Moving right")
			action := &action.Move{
				Id:        robotId,
				Direction: mat.NewVecDense(2, []float64{-1.0, 0.0}),
			}
			actions = append(actions, action)
		case 'k':
			fmt.Println("Kicking")
			action := &action.Kick{
				Id: robotId,
			}
			actions = append(actions, action)
		}
	}

	prevChar = char
	client.SendActions(actions)
}

func listenKeyboard(robotId int, client client.Client) {
	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()

	fmt.Println("Use WASD to control the robot, L to stop all movement, K to kick.")
	fmt.Println("Press <ESC> to exit.")

    // Send continous pings if we're remote controlling
    if clientType == remote_control {
        go sendPing(client)
    }

		if key == keyboard.KeyEsc {
			break
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
