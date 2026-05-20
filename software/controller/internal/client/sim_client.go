package client

import (
    "fmt"
    "net"
    "sync"

    . "github.com/LiU-SeeGoals/controller/internal/logger"
    "github.com/LiU-SeeGoals/controller/internal/action"
    "github.com/LiU-SeeGoals/controller/internal/info"
    "github.com/LiU-SeeGoals/proto_go/gc"
    "github.com/LiU-SeeGoals/proto_go/simulation"
    "google.golang.org/protobuf/proto"
)

type SimClient struct {
	conn *net.UDPConn

	addr *net.UDPAddr

	savedActions    []action.Action
	actionListMutex sync.Mutex
	gameState       *info.GameState
}

// Create new sim client
// Address should be <ip>:<port>
// gameState is optionaly provided
func NewSimClient(addr string, gameInfo ...*info.GameInfo) *SimClient {
	fmt.Println("Creating new SimClient with address: ", addr)
	Logger.Info("Creating new SimClient with address: ", addr)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		Logger.Panicf("Unable to resolve UDP address: %v", err)
	}

	sim_client := &SimClient{
		conn: nil,
		addr: udpAddr,
	}

	if len(gameInfo) > 0 && gameInfo[0] != nil {
		sim_client.gameState = gameInfo[0].State
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
	if client.gameState == nil {
		fmt.Println("Please provide gamestate in the sim_client constructor")
		Logger.Panic("Please provide gamestate in the sim_client constructor")
	}
	
	robotCommands := make([]*simulation.RobotCommand, 0, len(actions))
	for _, action := range actions {
		robotCommands = append(robotCommands, action.TranslateSim())
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
