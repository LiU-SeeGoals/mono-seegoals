package client

import (
	"math"
	"net"
	"testing"
	"time"

	"gonum.org/v1/gonum/mat"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"google.golang.org/protobuf/proto"
)

// This test starts a client and a sever and then sends a action to the server
// and then checks if the response matches what was sent.
// Only checks commandID and robotID in the message.
func TestSocketCommunication(t *testing.T) {

	// Define stop action
	stopAction := &action.Stop{Id: 2}
	stopCommand := &robot_action.Command{CommandId: robot_action.ActionType_STOP_ACTION, RobotId: 2}

	// Define kick action
	kickAction := &action.Kick{Id: 6, KickSpeed: 5}
	kickCommand := &robot_action.Command{CommandId: robot_action.ActionType_KICK_ACTION, RobotId: 6, KickSpeed: 5}

	// Define init action
	initAction := &action.Init{Id: 3}
	initCommand := &robot_action.Command{CommandId: robot_action.ActionType_INIT_ACTION, RobotId: 3}

	// Define move action
	moveToAction := &action.MoveTo{
		Id:   1,
		Pos:  info.Position{X: 100, Y: 200, Angle: math.Pi},
		Dest: info.Position{X: 300, Y: 400, Angle: -math.Pi},
	}
	moveToCommand := &robot_action.Command{
		CommandId: robot_action.ActionType_MOVE_TO_ACTION,
		RobotId:   1,
		Pos:       &robot_action.Vector3D{X: int32(100), Y: int32(200), W: float32(math.Pi)},
		Dest:      &robot_action.Vector3D{X: int32(300), Y: int32(400), W: float32(-math.Pi)},
	}

	// Define set navigation direction action.
	moveAction := &action.Move{
		Id:        9,
		Direction: mat.NewVecDense(2, []float64{100, 200}),
	}
	moveCommand := &robot_action.Command{
		CommandId: robot_action.ActionType_MOVE_ACTION,
		RobotId:   9,
		Direction: &robot_action.Vector2D{X: int32(100), Y: int32(200)},
	}

	// Define rotate.
	rotateAction := &action.Rotate{
		Id:         3,
		AngularVel: 5,
	}
	rotateCommand := &robot_action.Command{
		CommandId:  robot_action.ActionType_ROTATE_ACTION,
		RobotId:    3,
		AngularVel: 5,
	}

	// Test cases
	testCases := []struct {
		input    action.Action
		expected *robot_action.Command
	}{
		{stopAction, stopCommand},
		{kickAction, kickCommand},
		{initAction, initCommand},
		{moveToAction, moveToCommand},
		{moveAction, moveCommand},
		{rotateAction, rotateCommand},
	}

	// Bind before sending and use an ephemeral loopback port. Construct the
	// transport directly so this test never reads deployment config or multicasts.
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	connection, err := net.DialUDP("udp", nil, server.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	client := &BaseStationClient{connections: []Connection{connection}, hasBeenInited: true}
	for _, tc := range testCases {
		client.SendActions([]action.Action{tc.input})
		if len(client.queue) != 1 {
			t.Fatalf("expected one queued command, got %d", len(client.queue))
		}
		data, err := proto.Marshal(client.queue[0])
		if err != nil {
			t.Fatal(err)
		}
		client.queue = nil
		if err := client.sendMessage(data); err != nil {
			t.Fatal(err)
		}
		if err := server.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		buffer := make([]byte, MAX_SEND_SIZE+1)
		n, _, err := server.ReadFromUDP(buffer)
		if err != nil {
			t.Fatal(err)
		}
		command := &robot_action.Command{}
		if err := proto.Unmarshal(buffer[:n], command); err != nil {
			t.Fatal(err)
		}
		if command.GetRobotId() != tc.expected.GetRobotId() || command.GetCommandId() != tc.expected.GetCommandId() {
			t.Fatalf("expected robot %d command %v, got %v", tc.expected.GetRobotId(), tc.expected.GetCommandId(), command)
		}
	}
}

func TestStopReplacesQueuedCommandsForItsRobot(t *testing.T) {
	client := &BaseStationClient{hasBeenInited: true}
	client.SendActions([]action.Action{&action.Kick{Id: 1, KickSpeed: 3}, &action.MoveTo{Id: 2}, &action.MoveTo{Id: 1}})
	client.SendActions([]action.Action{&action.Stop{Id: 1}})
	if len(client.queue) != 2 || client.queue[0].GetRobotId() != 2 ||
		client.queue[1].GetRobotId() != 1 || client.queue[1].GetCommandId() != robot_action.ActionType_STOP_ACTION {
		t.Fatalf("STOP did not replace stale commands: %v", client.queue)
	}
}

func TestNewMotionDropsOnlyOlderUnsentMotionForSameRobot(t *testing.T) {
	client := &BaseStationClient{hasBeenInited: true}
	client.SendActions([]action.Action{
		&action.MoveTo{Id: 1, Dest: info.Position{X: 100}},
		&action.MoveTo{Id: 2, Dest: info.Position{X: 200}},
		&action.Kick{Id: 1, KickSpeed: 3},
	})
	client.SendActions([]action.Action{&action.MoveTo{Id: 1, Dest: info.Position{X: 300}}})

	if len(client.queue) != 3 {
		t.Fatalf("expected other robot motion, kick, and latest motion; got %v", client.queue)
	}
	if client.queue[0].GetRobotId() != 2 ||
		client.queue[1].GetCommandId() != robot_action.ActionType_KICK_ACTION ||
		client.queue[2].GetRobotId() != 1 ||
		client.queue[2].GetDest().GetX() != 300 {
		t.Fatalf("unexpected command order or latest destination: %v", client.queue)
	}
}
