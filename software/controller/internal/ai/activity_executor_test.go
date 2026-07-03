package ai

import (
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
	"google.golang.org/protobuf/proto"
)

func TestStoppedRobotMovesOutOfDefenseArea(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.State.SetBlueRobot(1, 4000, 0, 0.25, time.Now().UnixMilli())

	got := moveStoppedRobotOutOfEnemyDefenseArea(&action.Stop{Id: 1}, info.Blue, gi)
	move, ok := got.(*action.MoveTo)
	if !ok {
		t.Fatalf("action = %T, want *action.MoveTo", got)
	}

	clearance := defenseAreaClearance(goalLineBaseClearanceMM, gi)
	if positionInGoalArea(move.Dest, getGoalAreaBounds(gi), clearance) {
		t.Fatalf("retreat destination remains inside a defense area: %+v", move.Dest)
	}
	if move.Dest.X >= move.Pos.X {
		t.Fatalf("robot in right defense area did not retreat toward the field: from %.1f to %.1f", move.Pos.X, move.Dest.X)
	}
	if move.Dest.Angle != move.Pos.Angle {
		t.Fatalf("retreat changed heading from %.3f to %.3f", move.Pos.Angle, move.Dest.Angle)
	}
}

func TestStoppedRobotMovesOutOfEnemyDefenseAreaAfterHalfSwap(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.Status.SetGameStatus(0, 0, 0, 0, 0, true, "")
	gi.State.SetBlueRobot(1, -4000, 0, 0, time.Now().UnixMilli())

	got := moveStoppedRobotOutOfEnemyDefenseArea(&action.Stop{Id: 1}, info.Blue, gi)
	move, ok := got.(*action.MoveTo)
	if !ok {
		t.Fatalf("action = %T, want *action.MoveTo", got)
	}
	if move.Dest.X <= move.Pos.X {
		t.Fatalf("robot in left enemy defense area did not retreat toward the field: from %.1f to %.1f", move.Pos.X, move.Dest.X)
	}
}

func TestStoppedRobotOutsideDefenseAreaRemainsStopped(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.State.SetBlueRobot(1, 0, 0, 0, time.Now().UnixMilli())

	got := moveStoppedRobotOutOfEnemyDefenseArea(&action.Stop{Id: 1}, info.Blue, gi)
	if _, ok := got.(*action.Stop); !ok {
		t.Fatalf("action = %T, want *action.Stop", got)
	}
}

func TestStoppedRobotInOwnDefenseAreaRemainsStopped(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.State.SetBlueRobot(1, -4000, 0, 0, time.Now().UnixMilli())

	got := moveStoppedRobotOutOfEnemyDefenseArea(&action.Stop{Id: 1}, info.Blue, gi)
	if _, ok := got.(*action.Stop); !ok {
		t.Fatalf("action = %T, want *action.Stop in own defense area", got)
	}
}

func TestHaltedRobotInDefenseAreaRemainsStopped(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.Status.GetGameEvent().CurrentState = info.STATE_HALTED
	gi.State.SetBlueRobot(1, 4000, 0, 0, time.Now().UnixMilli())

	got := moveStoppedRobotOutOfEnemyDefenseArea(&action.Stop{Id: 1}, info.Blue, gi)
	if _, ok := got.(*action.Stop); !ok {
		t.Fatalf("action = %T, want *action.Stop during HALT", got)
	}
}

func TestStoppedDefenseAreaExitUsesNearestLegalEdge(t *testing.T) {
	area := goalAreaBounds{frontX: 3500, backX: 4500, minY: -1000, maxY: 1000}
	pos := info.Position{X: 4000, Y: 950}

	exit, inside := nearestDefenseAreaExit(pos, area, 390)
	if !inside {
		t.Fatal("expected robot to be inside the inflated defense area")
	}
	if exit.X != pos.X || exit.Y <= 1390 {
		t.Fatalf("exit = %+v, want shortest exit through upper side", exit)
	}
}

func newStoppedDefenseAreaTestGameInfo() *info.GameInfo {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.SetField(&ssl_vision.SSL_GeometryFieldSize{
		FieldLength:      proto.Int32(9000),
		FieldWidth:       proto.Int32(6000),
		GoalWidth:        proto.Int32(1000),
		BoundaryWidth:    proto.Int32(300),
		PenaltyAreaDepth: proto.Int32(1000),
		PenaltyAreaWidth: proto.Int32(2000),
	})
	gi.Status.GetGameEvent().CurrentState = info.STATE_STOPPED
	gi.Status.GetGameEvent().BallInPlay = false
	return gi
}
