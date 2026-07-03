package referee

import (
	"math"
	"sync"
	"testing"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
	"google.golang.org/protobuf/proto"
)

func TestFreeKickDefenseMirrorsAssignedHalfAndKeepsGoalie(t *testing.T) {
	tests := []struct {
		name             string
		bluePositiveHalf bool
		wantDefenseX     float64
	}{
		{name: "negative half", bluePositiveHalf: false, wantDefenseX: -freeKickWallDistanceMM},
		{name: "positive half", bluePositiveHalf: true, wantDefenseX: freeKickWallDistanceMM},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gi := info.NewGameInfo(int(info.TEAM_SIZE))
			gi.Status.SetGameStatus(0, 0, 0, 0, 0, tc.bluePositiveHalf, "")
			activities := [info.TEAM_SIZE]act.Activity{}
			handler := &coreai.ActivityHandler{
				Activities:    &activities,
				Activity_lock: &sync.Mutex{},
			}

			moveRobotsToDefensePosition(gi, []info.ID{1, 2, 6}, info.Blue, handler)

			if _, ok := activities[6].(*act.Goalie); !ok {
				t.Fatalf("goalie activity = %T, want *activity.Goalie", activities[6])
			}
			for _, robotID := range []info.ID{1, 2} {
				if _, ok := activities[robotID].(*act.MoveToPosition); !ok {
					t.Errorf("defender %d activity = %T, want *activity.MoveToPosition", robotID, activities[robotID])
				}
			}

			slots := freeKickDefenseSlots(gi, info.Blue, info.Position{}, 2)
			if slots[0].X != tc.wantDefenseX || slots[1].X != tc.wantDefenseX {
				t.Fatalf("defense X positions = %.0f, %.0f, want %.0f", slots[0].X, slots[1].X, tc.wantDefenseX)
			}
			if slots[0].Y != -150 || slots[1].Y != 150 {
				t.Fatalf("defense Y positions = %.0f, %.0f, want -150, 150", slots[0].Y, slots[1].Y)
			}
		})
	}
}

func TestFreeKickDefenseUsesThreeRobotWallAcrossShotLine(t *testing.T) {
	gi := newFreeKickDefenseGameInfo(false)
	ball := info.Position{X: 1200, Y: 800}
	slots := freeKickDefenseSlots(gi, info.Blue, ball, 5)
	if len(slots) != freeKickWallRobotCount {
		t.Fatalf("wall size = %d, want %d", len(slots), freeKickWallRobotCount)
	}

	center := info.Position{}
	for _, slot := range slots {
		center.X += slot.X / float64(len(slots))
		center.Y += slot.Y / float64(len(slots))
		if math.Abs(info.NormalizeAngleDelta(slot.AngleToPosition(ball), slot.Angle)) > 1e-9 {
			t.Fatalf("slot %+v does not face the ball", slot)
		}
	}
	if math.Abs(center.Dist2d(ball)-freeKickWallDistanceMM) > 1e-6 {
		t.Fatalf("wall center distance = %.3f, want %.3f", center.Dist2d(ball), freeKickWallDistanceMM)
	}

	goal := ownGoalCenter(gi, info.Blue)
	shotX, shotY := goal.X-ball.X, goal.Y-ball.Y
	wallX := slots[len(slots)-1].X - slots[0].X
	wallY := slots[len(slots)-1].Y - slots[0].Y
	if math.Abs(shotX*wallX+shotY*wallY) > 1e-6 {
		t.Fatalf("wall is not perpendicular to shot line: dot = %.3f", shotX*wallX+shotY*wallY)
	}
}

func TestFreeKickDefenseStaysClearOfDefenseAreaAndFieldLines(t *testing.T) {
	gi := newFreeKickDefenseGameInfo(false)
	// A valid free-kick placement exactly 1 m in front of our defense area.
	ball := info.Position{X: -2500, Y: 0}
	slots := freeKickDefenseSlots(gi, info.Blue, ball, 3)

	for _, slot := range slots {
		if !freeKickSlotLegal(gi, slot) {
			t.Fatalf("illegal wall slot: %+v", slot)
		}
		if slot.Dist2d(ball) < freeKickBallClearanceMM {
			t.Fatalf("slot distance to ball = %.1f, want at least %.1f", slot.Dist2d(ball), freeKickBallClearanceMM)
		}
	}

	// Near a touchline, the complete angled wall must remain inside the
	// 200 mm field-line margin.
	ball = info.Position{X: 0, Y: 2800}
	slots = freeKickDefenseSlots(gi, info.Blue, ball, 3)
	for _, slot := range slots {
		if !freeKickSlotLegal(gi, slot) {
			t.Fatalf("touchline wall slot is illegal: %+v", slot)
		}
	}
}

func newFreeKickDefenseGameInfo(bluePositiveHalf bool) *info.GameInfo {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.Status.SetGameStatus(0, 0, 0, 0, 0, bluePositiveHalf, "")
	gi.SetField(&ssl_vision.SSL_GeometryFieldSize{
		FieldLength:      proto.Int32(9000),
		FieldWidth:       proto.Int32(6000),
		GoalWidth:        proto.Int32(1000),
		PenaltyAreaDepth: proto.Int32(1000),
		PenaltyAreaWidth: proto.Int32(2000),
	})
	return gi
}
