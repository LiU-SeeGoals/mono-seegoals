package client

import (
	"encoding/hex"
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/simulation"
	"google.golang.org/protobuf/proto"
)

// The expected bytes were recorded from the previous implementation
// (Action.TranslateReal + action.EncodeCommand, Action.TranslateSim) before
// the codecs moved into this package. They pin the robot wire format and the
// default simulator controller output.
func goldenActions() []action.Action {
	return []action.Action{
		&action.Stop{Id: 2},
		&action.Kick{Id: 6, KickSpeed: 5},
		&action.Init{Id: 3},
		&action.MoveTo{Id: 1, Pos: info.Position{X: 100, Y: 200, Angle: math.Pi}, Dest: info.Position{X: 300, Y: 400, Angle: -math.Pi}},
		&action.MoveTo{Id: 4, Dribble: true, Pos: info.Position{X: -1000, Y: 500, Angle: 0.3}, Dest: info.Position{X: 2000, Y: -700, Angle: 1.2}},
		&action.MoveTo{Id: 5, KickSpeed: 3, Dribble: true, SimKickSpeed: 4.5, KickAngle: 30, Pos: info.Position{X: 10, Y: 20, Angle: -2.5}, Dest: info.Position{X: 400, Y: 20, Angle: 2.9}},
		&action.MoveTo{Id: 7, KickSpeed: 2, Pos: info.Position{X: -4000, Y: -2900, Angle: 3.1}, Dest: info.Position{X: -3990, Y: -2890, Angle: -3.1}},
		&action.MoveTo{Id: 8, MinLinearSpeed: 0.5, Pos: info.Position{X: 0, Y: 0, Angle: 1}, Dest: info.Position{X: 50, Y: 0, Angle: 1}},
		&action.Move{Id: 9, Direction: mat.NewVecDense(2, []float64{100, 200})},
		&action.Rotate{Id: 3, AngularVel: 5},
		&action.MoveRemote{Id: 2, Direction: mat.NewVecDense(2, []float64{-30, 70}), Dest: info.Position{X: 11, Y: 22, Z: 0.5}, Speed: 40, Dribble: true},
	}
}

var golden = []struct{ wire, sim string }{
	{"0102000000000000000000000000000000000000000000000000000000000000", "08021211120f0d0000000015000000001d000000001d000000002d00000000"},
	{"0006050000000000000000000000000000000000000000000000000000000000", "08061d0000a040"},
	{"0303000000000000000000000000000000000000000000000000000000000000", "0803"},
	{"020100006400c8002c019001000000000000450cbbf300000000000000000000", "08011211120f0dcdcc4cbe15cdcc4cbe1d171677272d00000000"},
	{"0204000018fcf401d00744fd0000000001002c01b00400000000000000000000", "08041211120f0d22fa463f15611221bf1d000040402d0000c842"},
	{"000503000a001400900114000000000000003cf6540b00000000000000000000", "08051211120f0decf89fbe157e016f3e1d000040c01d00009040250000f0412d0000c842"},
	{"0007020060f0acf46af0b6f40000000000001c0ce4f300000000000000000000", "08071211120f0dbfe21cbc15c9822abc1d6d11953f1d000000402d00000000"},
	{"020800000000000032000000000000000000e803e80300000000000000000000", "08081211120f0d40518a3e15a46ad7be1d000000002d00000000"},
	{"0409000000000000000000006400c80000000000000000000000000000000000", "08091211120f0d0000c84215000048431d00000000"},
	{"0503000000000000000000000000000005000000000000000000000000000000", "08031211120f0d0000000015000000001d0000a040"},
	{"040228000000000016000b004600e2ff0100f401f40100000000000000000000", "0802"},
}

func TestBaseStationWireFormatUnchanged(t *testing.T) {
	for i, a := range goldenActions() {
		cmd, ok := baseStationCommandFor(a)
		if !ok {
			t.Fatalf("case %d (%T): not supported", i, a)
		}
		buf, err := cmd.encode()
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if got := hex.EncodeToString(buf); got != golden[i].wire {
			t.Errorf("case %d (%T):\n got  %s\n want %s", i, a, got, golden[i].wire)
		}
		decoded, err := decodeBaseStationCommand(buf)
		if err != nil || decoded.kind != cmd.kind || decoded.robotID != cmd.robotID || decoded.aux != cmd.aux {
			t.Errorf("case %d: decode round trip = %+v, %v", i, decoded, err)
		}
	}
}

func TestSimCommandsUnchangedWithDefaultLimits(t *testing.T) {
	motion := NewSimMotion(DefaultSimMotionLimits)
	for i, a := range goldenActions() {
		got, ok := motion.Command(a)
		if !ok {
			t.Fatalf("case %d (%T): not supported", i, a)
		}
		raw, _ := hex.DecodeString(golden[i].sim)
		want := &simulation.RobotCommand{}
		if err := proto.Unmarshal(raw, want); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(got, want) {
			t.Errorf("case %d (%T):\n got  %v\n want %v", i, a, got, want)
		}
	}
}

func TestEncodeRejectsOutOfRangeRobotID(t *testing.T) {
	cmd, _ := baseStationCommandFor(&action.Stop{Id: 300})
	if _, err := cmd.encode(); err == nil {
		t.Fatal("expected an error for robot id 300")
	}
}
