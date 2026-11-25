package info

import (
	"testing"

	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
	"google.golang.org/protobuf/proto"
)

func TestOwnGoalCenterUsesVisionGeometry(t *testing.T) {
	gi := NewGameInfo(int(TEAM_SIZE))
	if _, ok := gi.OwnGoalCenter(Blue); ok {
		t.Fatal("goal center was fabricated without Vision geometry")
	}

	gi.SetField(&ssl_vision.SSL_GeometryFieldSize{
		FieldLength: proto.Int32(12000),
		FieldWidth:  proto.Int32(8000),
		GoalWidth:   proto.Int32(1200),
	})
	blueGoal, blueOK := gi.OwnGoalCenter(Blue)
	yellowGoal, yellowOK := gi.OwnGoalCenter(Yellow)
	if !blueOK || blueGoal.X != -6000 || blueGoal.Y != 0 {
		t.Fatalf("Blue goal = %+v, ok=%t; want X=-6000", blueGoal, blueOK)
	}
	if !yellowOK || yellowGoal.X != 6000 || yellowGoal.Y != 0 {
		t.Fatalf("Yellow goal = %+v, ok=%t; want X=6000", yellowGoal, yellowOK)
	}
}

func TestFieldGeometryDerivesOptionalPenaltyDimensionsFromVisionLines(t *testing.T) {
	line := func(name string, x1, y1, x2, y2 float32) *ssl_vision.SSL_FieldLineSegment {
		return &ssl_vision.SSL_FieldLineSegment{
			Name:      proto.String(name),
			P1:        &ssl_vision.Vector2F{X: proto.Float32(x1), Y: proto.Float32(y1)},
			P2:        &ssl_vision.Vector2F{X: proto.Float32(x2), Y: proto.Float32(y2)},
			Thickness: proto.Float32(10),
		}
	}

	gi := NewGameInfo(int(TEAM_SIZE))
	gi.SetField(&ssl_vision.SSL_GeometryFieldSize{
		FieldLength: proto.Int32(9000),
		FieldWidth:  proto.Int32(6000),
		GoalWidth:   proto.Int32(1000),
		FieldLines: []*ssl_vision.SSL_FieldLineSegment{
			line("LeftGoalLine", -4500, -3000, -4500, 3000),
			line("LeftPenaltyStretch", -3500, -1000, -3500, 1000),
		},
	})

	geometry, ok := gi.FieldGeometry()
	if !ok {
		t.Fatal("expected valid field geometry")
	}
	if got, want := geometry.PenaltyAreaDepth, 1000.0; got != want {
		t.Fatalf("penalty depth = %.0f, want %.0f from Vision lines", got, want)
	}
	if got, want := geometry.PenaltyAreaWidth, 2000.0; got != want {
		t.Fatalf("penalty width = %.0f, want %.0f from Vision lines", got, want)
	}
}
