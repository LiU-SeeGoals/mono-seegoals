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

// SSL Vision receiver
type SimClient struct {
	// Connection
	conn *net.UDPConn

	// UDP address
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

	// Create the SimClient instance
	sim_client := &SimClient{
		conn: nil,
		addr: udpAddr,
	}

	// Set gamestate if provided
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
	go client.sendActionThread()
}

func (client *SimClient) CloseConnection() {
	// Do nothing, only implemented to satisfy interface
}

// sends all the actions to the simulator
// OBS make sure gamestate is provided in the constructor when sending moveTo actions
func (client *SimClient) SendActions(actions []action.Action) {
	// To simulate how robot acts, we send action until action changed
	// Ex. We send moveTo, then this action will be submitted all the time until
	//     another action (Ex. Kick) is sent
	if client.gameState == nil {
		fmt.Println("Please provide gamestate in the sim_client constructor")
		Logger.Panic("Please provide gamestate in the sim_client constructor")
	}
	client.actionListMutex.Lock()

	// If robot have active action --> replace that one, otherwise add as new one
	for _, action := range actions {
		found := false // Flag to track if we found a match
		for i, savedAction := range client.savedActions {
			if action.ToDTO().Id == savedAction.ToDTO().Id {
				// Update the existing action
				client.savedActions[i] = action
				found = true // Mark as found
				break        // Exit inner loop since we've found and updated the action
			}
		}

		// Only append if no match was found
		if !found {
			client.savedActions = append(client.savedActions, action)
		}
	}

	client.actionListMutex.Unlock()
}

func (client *SimClient) sendActionThread() {
	for {
		client.actionListMutex.Lock()

		// Send all the commands to the simulator
		robotCommands := make([]*simulation.RobotCommand, 0)
		for _, action := range client.savedActions {
			robotCommands = append(robotCommands, action.TranslateSim())
		}
		// wrap the commands in a RobotControl message
		RobotControl := &simulation.RobotControl{
			RobotCommands: robotCommands,
		}

		client.Send(RobotControl)

		// Make sure all the MoveTo actions is updated with current data
		for i, act := range client.savedActions {
			if a, ok := act.(*action.MoveTo); ok {
				pos, err := client.gameState.GetTeam(a.Team)[a.Id].GetPosition()
				if err != nil {
					Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
					continue
				}
				a.Pos = pos
				client.savedActions[i] = a
			}
		}
		client.actionListMutex.Unlock()
	}
}

func (client *SimClient) Send(msg proto.Message) (int, error) {
	// fmt.Println("Sending message")

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
	// fmt.Println("Sending message")
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
