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
)

// This test starts a client and a sever and then sends a action to the server
// and then checks if the response matches what was sent.
// Only checks commandID and robotID in the message.
// TestSocketCommunication sends every supported action through a loopback
// socket and checks the received frame decodes to the same command.
func TestSocketCommunication(t *testing.T) {
	actions := []action.Action{
		&action.Stop{Id: 2},
		&action.Kick{Id: 6, KickSpeed: 5},
		&action.Init{Id: 3},
		&action.MoveTo{Id: 1, Pos: info.Position{X: 100, Y: 200, Angle: math.Pi}, Dest: info.Position{X: 300, Y: 400, Angle: -math.Pi}},
		&action.Move{Id: 9, Direction: mat.NewVecDense(2, []float64{100, 200})},
		&action.Rotate{Id: 3, AngularVel: 5},
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
	for _, a := range actions {
		client.SendActions([]action.Action{a})
		if len(client.queue) != 1 {
			t.Fatalf("expected one queued command, got %d", len(client.queue))
		}
		want := client.queue[0]
		client.queue = nil
		data, err := want.encode()
		if err != nil {
			t.Fatal(err)
		}
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
		got, err := decodeBaseStationCommand(buffer[:n])
		if err != nil {
			t.Fatal(err)
		}
		if got.robotID != want.robotID || got.kind != want.kind {
			t.Fatalf("%T: expected robot %d command %v, got %+v", a, want.robotID, want.kind, got)
		}
	}
}

func TestStopReplacesQueuedCommandsForItsRobot(t *testing.T) {
	client := &BaseStationClient{hasBeenInited: true}
	client.SendActions([]action.Action{&action.Kick{Id: 1, KickSpeed: 3}, &action.MoveTo{Id: 2}, &action.MoveTo{Id: 1}})
	client.SendActions([]action.Action{&action.Stop{Id: 1}})
	if len(client.queue) != 2 || client.queue[0].robotID != 2 ||
		client.queue[1].robotID != 1 || client.queue[1].kind != robot_action.ActionType_STOP_ACTION {
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
	if client.queue[0].robotID != 2 ||
		client.queue[1].kind != robot_action.ActionType_KICK_ACTION ||
		client.queue[2].robotID != 1 ||
		client.queue[2].destX != 300 {
		t.Fatalf("unexpected command order or latest destination: %v", client.queue)
	}
}
