package simulator

import (
	// "fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/proto_go/gc"
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

// The simulator have a lot of things that can be configured.
// This configuration is done with proto messages on port 10300 (not the port for teams).
type SimControl struct {
	client *client.SimClient
}

func NewSimControl() *SimControl {
	simClient := client.NewSimClient(config.GetSimControlAddress())
	simClient.Init()
	return &SimControl{
		client: simClient,
	}
}

func (sc *SimControl) TurnOffCameraRealism() {
	// fmt.Println("Not yet implemented")
	Logger.Error("TurnOffCameraRealism not yet implemented")
}

func (sc *SimControl) TurnOnCameraRealism() {
	// fmt.Println("Not yet implemented")
	Logger.Error("TurnOnCameraRealism not yet implemented")
}

func centerCircle(number int, radius float32) (float32, float32) {
	total_slots := 11 * 2
	angle_step := 2 * math.Pi / float64(total_slots)

	angle := angle_step * float64(number)
	x := float32(math.Cos(angle)) * radius
	y := float32(math.Sin(angle)) * radius
	return x, y
}

func (sc *SimControl) SetPresentRobots(presentYellow []int, presentBlue []int) {
	TOTAL_ROBOTS := 11
	orientation := float32(0.0) // Approx. 45 degrees in radians
	vx := float32(0.0)          // Velocity towards x-axis
	vy := float32(0.0)          // Velocity towards y-axis
	vAngular := float32(0.0)    // Angular velocity

	robotList := []*simulation.TeleportRobot{}

	for i := 0; i < TOTAL_ROBOTS; i++ {
		present := false
		if helper.Contains(presentYellow, i) { // all robots after the number we want --> set to not present
			present = true
		}

		idNum := uint32(i)
		team := gc.Team_YELLOW
		id := gc.RobotId{
			Id:   &idNum,
			Team: &team,
		}

		x, y := centerCircle(i, 1.0)

		teleportRobot := &simulation.TeleportRobot{
			Id:          &id,
			X:           &x,
			Y:           &y,
			Orientation: &orientation,
			VX:          &vx,
			VY:          &vy,
			VAngular:    &vAngular,
			Present:     &present,
		}

		robotList = append(robotList, teleportRobot)
	}

	for i := 0; i < TOTAL_ROBOTS; i++ {
		present := false
		if helper.Contains(presentBlue, i) { // all robots after the number we want --> set to not present
			present = true
		}

		idNum := uint32(i)
		team := gc.Team_BLUE
		id := gc.RobotId{
			Id:   &idNum,
			Team: &team,
		}

		x, y := centerCircle(i+TOTAL_ROBOTS, 1.0)

		teleportRobot := &simulation.TeleportRobot{
			Id:          &id,
			X:           &x,
			Y:           &y,
			Orientation: &orientation,
			VX:          &vx,
			VY:          &vy,
			VAngular:    &vAngular,
			Present:     &present,
		}

		robotList = append(robotList, teleportRobot)
	}
	zero := float32(0.0)
	ball := &simulation.TeleportBall{
		X:  &zero,
		Y:  &zero,
		Z:  &zero,
		Vx: &zero,
		Vy: &zero,
		Vz: &zero,
	}

	SimControl := &simulation.SimulatorControl{
		TeleportRobot:   robotList,
		TeleportBall:    ball,
		SimulationSpeed: nil,
	}

	simCommand := &simulation.SimulatorCommand{
		Control: SimControl,
		Config:  nil,
	}

	sc.client.Send(simCommand)
}

func (sc *SimControl) SetRobotDimentions() {
	// fmt.Println("Not yet implemented")
	Logger.Error("SetRobotDimentions not yet implemented")
}

func (sc *SimControl) RobotStartPositionConfig1(numberOfRobots int) {
	generateCoordinates := func(x, min_y, max_y float32) [][2]float32 {
		coords := make([][2]float32, numberOfRobots)
		step := (max_y - min_y) / float32(numberOfRobots-1)
		for i := 0; i < numberOfRobots; i++ {
			y := min_y + step*float32(i)
			coords[i] = [2]float32{x, y}
		}
		return coords
	}

	blueCoords := generateCoordinates(1, -2, 2)
	yellowCoords := generateCoordinates(-1, -2, 2)

	for robot_id := 0; robot_id < numberOfRobots; robot_id++ {
		x_blue := blueCoords[robot_id][0]
		y_blue := blueCoords[robot_id][1]
		id := uint32(robot_id)
		sc.TeleportRobot(x_blue, y_blue, id, info.Blue)

		x_yellow := yellowCoords[robot_id][0]
		y_yellow := yellowCoords[robot_id][1]
		id = uint32(robot_id)
		sc.TeleportRobot(x_yellow, y_yellow, id, info.Yellow)
	}

}

func (sc *SimControl) RobotStartPositionConfig2(numberOfRobots int) {
	// fmt.Println("Not yet implemented")
	Logger.Error("RobotStartPositionConfig2 not yet implemented")
}

func (sc *SimControl) TeleportRobot(x float32, y float32, id uint32, team info.Team) {
	// We take in x and y as milimeters in float64, simulator need float32 in meters

	// fmt.Println(x, y)
	Logger.Info("Teleporting robot to x: %f, y: %f, id: %d, team: %d", x, y, id, team)
	// Set default values for orientation and velocities
	orientation := float32(0.0) // Approx. 45 degrees in radians
	vx := float32(0.0)          // Velocity towards x-axis
	vy := float32(0.0)          // Velocity towards y-axis
	vAngular := float32(0.0)    // Angular velocity
	present := true             // Teleport indicates the robot is present

	gc_team := gc.Team_BLUE
	if team == info.Yellow {
		gc_team = gc.Team_YELLOW
	}

	// Create the robot ID structure
	robotId := gc.RobotId{
		Id:   &id,
		Team: &gc_team,
	}

	x = x / 1000.0
	y = y / 1000.0
	// Create the TeleportRobot structure with the new position and parameters
	teleportRobot := &simulation.TeleportRobot{
		Id:          &robotId,
		X:           &x,
		Y:           &y,
		Orientation: &orientation,
		VX:          &vx,
		VY:          &vy,
		VAngular:    &vAngular,
		Present:     &present,
	}

	// Prepare the command with the single robot teleportation
	SimControl := &simulation.SimulatorControl{
		TeleportRobot:   []*simulation.TeleportRobot{teleportRobot},
		TeleportBall:    nil,
		SimulationSpeed: nil,
	}

	simCommand := &simulation.SimulatorCommand{
		Control: SimControl,
		Config:  nil,
	}

	// Send the command to teleport the robot
	sc.client.Send(simCommand)
}

func (sc *SimControl) TeleportBall(x float32, y float32) {
	// Set default values for orientation and velocities
	zero := float32(0.0)
	x = x / 1000.0
	y = y / 1000.0
	teleball := &simulation.TeleportBall{
		X:  &x,
		Y:  &y,
		Z:  &zero,
		Vx: &zero,
		Vy: &zero,
		Vz: &zero,
	}

	// Prepare the command with the single robot teleportation
	SimControl := &simulation.SimulatorControl{
		TeleportRobot:   nil,
		TeleportBall:    teleball,
		SimulationSpeed: nil,
	}

	simCommand := &simulation.SimulatorCommand{
		Control: SimControl,
		Config:  nil,
	}

	// Send the command to teleport the robot
	sc.client.Send(simCommand)
}
