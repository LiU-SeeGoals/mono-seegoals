package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/LiU-SeeGoals/proto_go/gc"
	"github.com/LiU-SeeGoals/proto_go/simulation"
	"google.golang.org/protobuf/proto"
)

const (
	kickerID  uint32 = 3
	blockerID uint32 = 7
	kickerX          = -0.18
	ballX            = -0.09
)

func main() {
	controlAddr := flag.String("control", "127.0.0.1:10300", "simulator control UDP address")
	yellowAddr := flag.String("yellow", "127.0.0.1:10302", "yellow team robot-control UDP address")
	kickSpeed := flag.Float64("speed", 4.0, "kick speed in m/s")
	kickAngle := flag.Float64("angle", 45.0, "vertical kick angle in degrees; 0 is flat")
	settle := flag.Int("settle", 20, "number of dribbler/drive packets before kicking")
	repeat := flag.Int("repeat", 12, "number of kick packets to send")
	flag.Parse()

	controlConn := mustDial(*controlAddr)
	defer controlConn.Close()
	yellowConn := mustDial(*yellowAddr)
	defer yellowConn.Close()

	teleport := setupCommand()
	mustSend(controlConn, teleport)
	time.Sleep(500 * time.Millisecond)

	collect := robotControl(collectCommand(kickerID))
	for i := 0; i < *settle; i++ {
		mustSend(yellowConn, collect)
		time.Sleep(50 * time.Millisecond)
	}

	kick := robotControl(kickCommand(kickerID, float32(*kickSpeed), float32(*kickAngle)))
	for i := 0; i < *repeat; i++ {
		mustSend(yellowConn, kick)
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf(
		"Sent chip probe: yellow %d, speed %.1f m/s, angle %.1f deg, blue blocker %d in front.\n",
		kickerID,
		*kickSpeed,
		*kickAngle,
		blockerID,
	)
}

func collectCommand(id uint32) *simulation.RobotCommand {
	forward := float32(0.25)
	left := float32(0)
	angular := float32(0)
	dribbler := float32(100)

	return &simulation.RobotCommand{
		Id:            &id,
		MoveCommand:   localVelocityCommand(forward, left, angular),
		DribblerSpeed: &dribbler,
	}
}

func kickCommand(id uint32, speed, angle float32) *simulation.RobotCommand {
	forward := float32(0.35)
	left := float32(0)
	angular := float32(0)
	dribbler := float32(0)

	return &simulation.RobotCommand{
		Id:            &id,
		MoveCommand:   localVelocityCommand(forward, left, angular),
		DribblerSpeed: &dribbler,
		KickSpeed:     &speed,
		KickAngle:     &angle,
	}
}

func robotControl(command *simulation.RobotCommand) *simulation.RobotControl {
	return &simulation.RobotControl{
		RobotCommands: []*simulation.RobotCommand{command},
	}
}

func localVelocityCommand(forward, left, angular float32) *simulation.RobotMoveCommand {
	return &simulation.RobotMoveCommand{
		Command: &simulation.RobotMoveCommand_LocalVelocity{
			LocalVelocity: &simulation.MoveLocalVelocity{
				Forward: &forward,
				Left:    &left,
				Angular: &angular,
			},
		},
	}
}

func setupCommand() *simulation.SimulatorCommand {
	return &simulation.SimulatorCommand{
		Control: &simulation.SimulatorControl{
			TeleportRobot: setupRobots(),
			TeleportBall:  teleportBall(ballX, 0),
		},
	}
}

func setupRobots() []*simulation.TeleportRobot {
	robots := make([]*simulation.TeleportRobot, 0, 22)
	for id := uint32(0); id < 11; id++ {
		present := id == kickerID
		x := float32(-2)
		y := float32(0)
		orientation := float32(0)
		if present {
			x = kickerX
		}
		robots = append(robots, teleportRobot(id, gc.Team_YELLOW, x, y, orientation, present))
	}
	for id := uint32(0); id < 11; id++ {
		present := id == blockerID
		x := float32(2)
		y := float32(0)
		orientation := float32(math.Pi)
		if present {
			x = 0.45
		}
		robots = append(robots, teleportRobot(id, gc.Team_BLUE, x, y, orientation, present))
	}
	return robots
}

func teleportRobot(id uint32, team gc.Team, x, y, orientation float32, present bool) *simulation.TeleportRobot {
	zero := float32(0)
	return &simulation.TeleportRobot{
		Id: &gc.RobotId{
			Id:   &id,
			Team: &team,
		},
		X:           &x,
		Y:           &y,
		Orientation: &orientation,
		VX:          &zero,
		VY:          &zero,
		VAngular:    &zero,
		Present:     &present,
	}
}

func teleportBall(x, y float32) *simulation.TeleportBall {
	zero := float32(0)
	return &simulation.TeleportBall{
		X:  &x,
		Y:  &y,
		Z:  &zero,
		Vx: &zero,
		Vy: &zero,
		Vz: &zero,
	}
}

func mustDial(addr string) *net.UDPConn {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		panic(err)
	}
	return conn
}

func mustSend(conn *net.UDPConn, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	if _, err := conn.Write(data); err != nil {
		panic(err)
	}
}
