package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/gonum/mat"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/simulatedsimian/joystick"
)

const (
	gsim           = iota
	basestation    = iota
	remote_control = iota
)

var (
	clientType   int = 0
	speed        int   = 0
	robotStopped bool  = false
	angle        int32 = 0
	
	// Xbox One controller constants
	deadzone     = 0.15  // Ignore small stick movements
	maxAxisValue = 32767.0
)

// Xbox One button mappings (may vary by OS)
const (
	ButtonA     = 0
	ButtonB     = 1
	ButtonX     = 2
	ButtonY     = 3
	ButtonLB    = 4
	ButtonRB    = 5
	ButtonBack  = 6
	ButtonStart = 7
	ButtonLS    = 9  // Left stick click
	ButtonRS    = 10 // Right stick click
	
	// Axis mappings for Xbox One X pad (8 axes total)
	AxisLeftX  = 0  // Left stick horizontal
	AxisLeftY  = 1  // Left stick vertical
	AxisLT     = 2  // Left trigger (starts at -32767, goes to +32767)
	AxisRightX = 3  // Right stick horizontal
	AxisRightY = 4  // Right stick vertical
	AxisRT     = 5  // Right trigger (starts at -32767, goes to +32767)
	AxisDPadX  = 6  // D-pad horizontal
	AxisDPadY  = 7  // D-pad vertical
)

func main() {
	clientHost := "127.0.0.1:20011"
	client := askForClient(clientHost)
	client.Init()
	robotId := askForRobotId()
	
	// Find and connect to controller
	js := connectController()
	if js == nil {
		fmt.Println("No controller found. Exiting.")
		return
	}
	defer js.Close()
	
	listenController(client, js, robotId)
}

func askForClient(port string) client.Client {
	var clientHost string = port
	var clientBaseStation client.Client

	fmt.Println("Enter <ip>:<port> for the basestation (port defaults to 6001): ")
	fmt.Scanln(&clientHost)
	if !strings.Contains(clientHost, ":") {
		clientHost = clientHost + ":6001"
	}

	clientType = remote_control
	fmt.Println("Creating base station client for remote control.")

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

func connectController() joystick.Joystick {
	fmt.Println("Scanning for game controllers...")
	
	// Try to find a real game controller (not accelerometer)
	for jsid := 0; jsid < 10; jsid++ {
		js, err := joystick.Open(jsid)
		if err != nil {
			continue
		}
		
		// Xbox controllers have at least 10 buttons and 6+ axes
		// This filters out accelerometers and other non-controller devices
		if js.ButtonCount() >= 10 && js.AxisCount() >= 6 {
			fmt.Printf("✓ Found game controller on ID %d: %s\n", jsid, js.Name())
			fmt.Printf("  Axes: %d, Buttons: %d\n", js.AxisCount(), js.ButtonCount())
			return js
		}
		js.Close()
	}
	
	fmt.Println("✗ No game controller found!")
	fmt.Println("Make sure your Xbox controller is connected.")
	return nil
}

func applyDeadzone(value float64) float64 {
	if math.Abs(value) < deadzone {
		return 0.0
	}
	// Scale the value to account for deadzone
	sign := 1.0
	if value < 0 {
		sign = -1.0
	}
	return sign * (math.Abs(value) - deadzone) / (1.0 - deadzone)
}

func normalizeAxis(rawValue int) float64 {
	normalized := float64(rawValue) / maxAxisValue
	return applyDeadzone(normalized)
}

func listenController(client client.Client, js joystick.Joystick, robotId int) {
	fmt.Println("\n=== Xbox One Controller Mapping ===")
	fmt.Println("Left Stick:    Move robot (WASD equivalent)")
	fmt.Println("Right Stick:   Rotate robot")
	fmt.Println("A Button:      Kick")
	fmt.Println("B Button:      Stop")
	fmt.Println("RT (Right Trigger): Increase speed")
	fmt.Println("LT (Left Trigger):  Decrease speed")
	fmt.Println("Start Button:  Exit")
	fmt.Println("===================================\n")
	
	ticker := time.NewTicker(50 * time.Millisecond) // 20Hz update rate
	defer ticker.Stop()
	
	var previousButtons uint32
	speed = 100 // Start with higher default speed
	
	fmt.Println("Default speed set to 100%")
	
	for range ticker.C {
		state, err := js.Read()
		if err != nil {
			fmt.Println("Error reading controller:", err)
			continue
		}
		
		// Check for exit (Start button)
		if state.Buttons&(1<<ButtonStart) != 0 {
			fmt.Println("Start button pressed. Exiting.")
			break
		}
		
		// A Button - Kick
		if isButtonPressed(state.Buttons, previousButtons, ButtonA) {
			fmt.Println("Kicking!")
			client.SendActions([]action.Action{
				&action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{0, 0}),
					Speed:     255, // Max speed for kick
					Angle:     angle,
				},
			})
		}
		
		// B Button - Stop
		if isButtonPressed(state.Buttons, previousButtons, ButtonB) {
			fmt.Println("Stopping robot")
			robotStopped = true
			angle = 0
			client.SendActions([]action.Action{
				&action.Stop{Id: robotId},
			})
		}
		
		// Update previous button state
		previousButtons = state.Buttons
		
		// Left/Right Triggers for speed control
		// Xbox triggers range from -32767 (not pressed) to +32767 (fully pressed)
		if len(state.AxisData) > AxisRT {
			// Normalize trigger value: -32767 to +32767 becomes 0.0 to 1.0
			rtRaw := float64(state.AxisData[AxisRT])
			rtValue := (rtRaw + 32767.0) / 65534.0
			
			if rtValue > 0.5 { // Only trigger at significant press
				oldSpeed := speed
				speed += 10
				if speed > 255 {
					speed = 255
				}
				if speed != oldSpeed {
					fmt.Printf("Speed: %d\n", speed)
				}
			}
		}
		
		if len(state.AxisData) > AxisLT {
			// Normalize trigger value: -32767 to +32767 becomes 0.0 to 1.0
			ltRaw := float64(state.AxisData[AxisLT])
			ltValue := (ltRaw + 32767.0) / 65534.0
			
			if ltValue > 0.5 { // Only trigger at significant press
				oldSpeed := speed
				speed -= 10
				if speed < 10 {
					speed = 10
				}
				if speed != oldSpeed {
					fmt.Printf("Speed: %d\n", speed)
				}
			}
		}
		
		// Left stick - Movement
		var leftX, leftY float64
		if len(state.AxisData) > AxisLeftY {
			leftX = normalizeAxis(int(state.AxisData[AxisLeftX]))
			leftY = -normalizeAxis(int(state.AxisData[AxisLeftY])) // Invert Y axis
		}
		
		// Right stick - Rotation
		var rightX float64
		if len(state.AxisData) > AxisRightX {
			rightX = normalizeAxis(int(state.AxisData[AxisRightX]))
		}
		
		// Update rotation angle based on right stick
		if math.Abs(rightX) > 0.01 {
			angle += int32(rightX * 20) // Adjust rotation speed
			robotStopped = false
			fmt.Printf("Angle: %d°\n", angle)
		}
		
		// Send movement command if sticks are being used
		isMoving := math.Abs(leftX) > 0.01 || math.Abs(leftY) > 0.01
		isRotating := math.Abs(rightX) > 0.01
		
		if isMoving || isRotating {
			robotStopped = false
			
			// Normalize the direction vector if moving
			if isMoving {
				magnitude := math.Sqrt(leftX*leftX + leftY*leftY)
				if magnitude > 1.0 {
					leftX /= magnitude
					leftY /= magnitude
				}
			}
			
			fmt.Printf("Moving: X=%.2f Y=%.2f Speed=%d Angle=%d\n", leftX, leftY, speed, angle)
			
			client.SendActions([]action.Action{
				&action.MoveRemote{
					Id:        robotId,
					Direction: mat.NewVecDense(2, []float64{leftY, leftX}),
					Speed:     speed,
					Angle:     angle,
				},
			})
		} else if !robotStopped {
			// Send stop command when sticks are released
			robotStopped = true
			fmt.Println("Sticks released - stopping")
			client.SendActions([]action.Action{
				&action.Stop{Id: robotId},
			})
		}
	}
}

func isButtonPressed(current, previous uint32, buttonIndex int) bool {
	// Check if button is currently pressed but wasn't before
	currentPressed := current&(1<<buttonIndex) != 0
	previousPressed := previous&(1<<buttonIndex) != 0
	
	return currentPressed && !previousPressed
}