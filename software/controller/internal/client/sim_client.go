package client

import (
	"fmt"
	"net"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/proto_go/gc"
	"github.com/LiU-SeeGoals/proto_go/simulation"
	"google.golang.org/protobuf/proto"
)

// SimClient sends commands to the simulator. Robot commands go through a
// SimMotion controller; simulator-control messages use Send directly.
type SimClient struct {
	conn   *net.UDPConn
	addr   *net.UDPAddr
	motion *SimMotion
}

// NewSimClient connects to addr ("<ip>:<port>") and drives robots with the
// configured SimMotionLimits.
func NewSimClient(addr string) *SimClient {
	fmt.Println("Creating new SimClient with address: ", addr)
	logger.Logger.Info("Creating new SimClient with address: ", addr)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logger.Logger.Panicf("Unable to resolve UDP address: %v", err)
	}

	sim_client := &SimClient{
		addr:   udpAddr,
		motion: NewSimMotion(SimMotionLimitsFromConfig()),
	}
	sim_client.Init()

	return sim_client
}

// Connect/subscribe receiver to UDP multicast.
// Note, this will NOT block.
func (client *SimClient) Init() {
	conn, err := net.DialUDP("udp", nil, client.addr)
	if err != nil {
		panic(err)
	}
	client.conn = conn
}

func (client *SimClient) CloseConnection() {
	// Do nothing, only implemented to satisfy interface
}

func (client *SimClient) SendActions(actions []action.Action) {
	robotCommands := make([]*simulation.RobotCommand, 0, len(actions))
	for _, a := range actions {
		cmd, ok := client.motion.Command(a)
		if !ok {
			logger.Logger.Warnf("Simulator: unsupported action %T for robot %d", a, a.RobotID())
			continue
		}
		robotCommands = append(robotCommands, cmd)
	}

	robotControl := &simulation.RobotControl{
		RobotCommands: robotCommands,
	}

	client.Send(robotControl)
}

func (client *SimClient) Send(msg proto.Message) (int, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("unable to marshal TeleportRobot data: %w", err)
	}
	writeCount, err := client.conn.Write(data)
	if err != nil {
		return 0, fmt.Errorf("unable to send TeleportRobot data over socket: %w", err)
	}

	return writeCount, nil
}

func (client *SimClient) SendTestMessage() (int, error) {
	idNum := uint32(3)
	team := gc.Team_BLUE

	id := gc.RobotId{
		Id:   &idNum,
		Team: &team,
	}
	x := float32(1.0)           // X-coordinate
	y := float32(1.0)           // Y-coordinate
	orientation := float32(0.0) // Approx. 45 degrees in radians
	vx := float32(0.0)          // Velocity towards x-axis
	vy := float32(0.0)          // Velocity towards y-axis
	vAngular := float32(0.0)    // Angular velocity
	present := false

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

	robotList := []*simulation.TeleportRobot{teleportRobot}

	simControl := &simulation.SimulatorControl{
		TeleportRobot:   robotList,
		TeleportBall:    nil,
		SimulationSpeed: nil,
	}

	simCommand := &simulation.SimulatorCommand{
		Control: simControl,
		Config:  nil,
	}

	// syncReq := &simulation.SimulationSyncRequest{
	// 	SimulatorCommand: simCommand,
	// }

	data, err := proto.Marshal(simCommand)
	if err != nil {
		return 0, fmt.Errorf("unable to marshal TeleportRobot data: %w", err)
	}

	// [libprotobuf ERROR google/protobuf/message_lite.cc:121]
	// Can't parse message of type "sslsim.SimulatorCommand" because it is missing required fields:

	// config.geometry.field,
	// config.geometry.calib[0].camera_id,
	// config.geometry.calib[0].q2,
	// config.geometry.calib[0].q3,
	// config.geometry.calib[0].tx,
	// config.geometry.calib[0].ty,
	// config.geometry.calib[0].tz

	writeCount, err := client.conn.Write(data)
	if err != nil {
		return 0, fmt.Errorf("unable to send TeleportRobot data over socket: %w", err)
	}

	return writeCount, nil
}
