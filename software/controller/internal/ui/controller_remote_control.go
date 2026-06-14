package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"gonum.org/v1/gonum/mat"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/info"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	remote_control = iota
	deadzone       = 8000
	updateRateHz   = 50
)

var (
	clientType int = remote_control
	speed      int = 3
)

func main() {
	runtime.LockOSThread()

	clientHost := "127.0.0.1:20011"

	c := askForClient(clientHost)
	c.Init()

	robotID := askForRobotId()

	controller := openController()
	defer controller.Close()

	fmt.Println()
	fmt.Println("Xbox Controller Connected")
	fmt.Println("-------------------------")
	fmt.Println("Left Stick  : Translation")
	fmt.Println("Right Stick : Rotation")
	fmt.Println("A           : Kick")
	fmt.Println("B           : Stop")
	fmt.Println("X           : Toggle Dribbler")
	fmt.Println("LT          : Decrease Speed")
	fmt.Println("RT          : Increase Speed")
	fmt.Println("-------------------------")

	runControllerLoop(c, robotID, controller)
}

func askForClient(port string) client.Client {
	fmt.Println("Creating base station client for remote control.")
	return client.NewBaseStationClient(port)
}

func askForRobotId() int {
	var robotId string

	fmt.Println("Please enter robot ID (default 0):")

	fmt.Scanln(&robotId)

	id, err := strconv.Atoi(robotId)
	if err != nil {
		id = 0
	}

	fmt.Println("Robot ID:", id)

	return id
}

func openController() *sdl.GameController {

	if err := sdl.Init(
		sdl.INIT_GAMECONTROLLER,
	); err != nil {
		panic(err)
	}

	for i := 0; i < sdl.NumJoysticks(); i++ {

		if sdl.IsGameController(i) {

			controller := sdl.GameControllerOpen(i)

			if controller != nil {

				fmt.Println("Controller found:")
				fmt.Println(controller.Name())

				return controller
			}
		}
	}

	panic("no game controller detected")
}

func normalizeAxis(v int16) float64 {

	if v > -deadzone && v < deadzone {
		return 0.0
	}

	return float64(v) / 32767.0
}

func runControllerLoop(
	c client.Client,
	robotID int,
	controller *sdl.GameController,
) {

	var (
		 dribble bool

		 lastB bool
		 lastX bool
		 lastY bool

		 lastLB bool
		 lastRB bool

		 lastLT bool
		 lastRT bool
	)

	ticker := time.NewTicker(
		time.Second / updateRateHz,
	)
	defer ticker.Stop()

	for {

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {

			switch event.(type) {

			case *sdl.QuitEvent:
				return
			}
		}

		lx := normalizeAxis(
			controller.Axis(
				sdl.CONTROLLER_AXIS_LEFTX,
			),
		)

		ly := -normalizeAxis(
			controller.Axis(
				sdl.CONTROLLER_AXIS_LEFTY,
			),
		)

		rx := normalizeAxis(
			controller.Axis(
				sdl.CONTROLLER_AXIS_RIGHTX,
			),
		)

		move := &action.MoveRemote{
			Id: robotID,

			Direction: mat.NewVecDense(
				2,
				[]float64{0, 0},
			),

			Dest: info.Position{
				X: lx * 10,
				Y: ly * 10,
				Z: rx * 20,
			},

			Speed:   speed,
			Dribble: dribble,
		}

		c.SendActions(
			[]action.Action{move},
		)

		lbPressed := controller.Button(
			 sdl.CONTROLLER_BUTTON_LEFTSHOULDER,
		) != 0

		if lbPressed && !lastLB {

			 fmt.Println("Kick (speed 0)")

			 c.SendActions(
				  []action.Action{
						&action.Kick{
							 Id:        robotID,
							 KickSpeed: 0,
						},
				  },
			 )
		}

		lastLB = lbPressed

		rbPressed := controller.Button(
			 sdl.CONTROLLER_BUTTON_RIGHTSHOULDER,
		) != 0

		if rbPressed && !lastRB {

			 fmt.Println("Kick (speed 1)")

			 c.SendActions(
				  []action.Action{
						&action.Kick{
							 Id:        robotID,
							 KickSpeed: 1,
						},
				  },
			 )
		}

		lastRB = rbPressed

		yPressed := controller.Button(
			sdl.CONTROLLER_BUTTON_Y,
		) != 0

		if yPressed && !lastY {

			fmt.Println("Kick chip")

			c.SendActions(
				[]action.Action{
					&action.Kick{
						Id:        robotID,
						KickSpeed: 2,
					},
				},
			)
		}

		lastY = yPressed

		bPressed := controller.Button(
			sdl.CONTROLLER_BUTTON_B,
		) != 0

		if bPressed && !lastB {

			fmt.Println("Stop")

			c.SendActions(
				[]action.Action{
					&action.Stop{
						Id: robotID,
					},
				},
			)
		}

		lastB = bPressed

		xPressed := controller.Button(
			sdl.CONTROLLER_BUTTON_X,
		) != 0

		if xPressed && !lastX {

			dribble = !dribble

			fmt.Println(
				"Dribbler:",
				dribble,
			)
		}

		lastX = xPressed

		ltPressed := controller.Axis(
			sdl.CONTROLLER_AXIS_TRIGGERLEFT,
		) > 20000

		if ltPressed && !lastLT {

			if speed > 0 {
				speed--
			}

			fmt.Println(
				"Speed:",
				speed,
			)
		}

		lastLT = ltPressed

		rtPressed := controller.Axis(
			sdl.CONTROLLER_AXIS_TRIGGERRIGHT,
		) > 20000

		if rtPressed && !lastRT {

			speed++

			fmt.Println(
				"Speed:",
				speed,
			)
		}

		lastRT = rtPressed

		<-ticker.C
	}
}
