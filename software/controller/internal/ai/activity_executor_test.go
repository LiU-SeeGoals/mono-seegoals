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

	escape := defenseAreaEscapeState{}
	got := escape.apply(&action.Stop{Id: 1}, info.Blue, gi)
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

	escape := defenseAreaEscapeState{}
	got := escape.apply(&action.Stop{Id: 1}, info.Blue, gi)
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

	escape := defenseAreaEscapeState{}
	got := escape.apply(&action.Stop{Id: 1}, info.Blue, gi)
	if _, ok := got.(*action.Stop); !ok {
		t.Fatalf("action = %T, want *action.Stop", got)
	}
}

func TestStoppedRobotInOwnDefenseAreaRemainsStopped(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.State.SetBlueRobot(1, -4000, 0, 0, time.Now().UnixMilli())

	escape := defenseAreaEscapeState{}
	got := escape.apply(&action.Stop{Id: 1}, info.Blue, gi)
	if _, ok := got.(*action.Stop); !ok {
		t.Fatalf("action = %T, want *action.Stop in own defense area", got)
	}
}

func TestHaltedRobotInDefenseAreaRemainsStopped(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.Status.GetGameEvent().CurrentState = info.STATE_HALTED
	gi.State.SetBlueRobot(1, 4000, 0, 0, time.Now().UnixMilli())

	escape := defenseAreaEscapeState{}
	got := escape.apply(&action.Stop{Id: 1}, info.Blue, gi)
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

func TestFinalStopMovementPreservesBallAndDefenseClearance(t *testing.T) {
	for _, sign := range []float64{-1, 1} {
		gi := newStoppedDefenseAreaTestGameInfo()
		ball := info.Position{X: sign * 3000}
		gi.State.SetBall(ball.X, ball.Y, 0, 1)
		gi.State.SetBlueRobot(1, sign*2500, 0, 0, time.Now().UnixMilli())
		move := &action.MoveTo{Id: 1, Team: info.Blue,
			Pos: info.Position{X: sign * 2500}, Dest: info.Position{X: sign * 3800}}
		safe := stoppedPlaySafetyAction(move, info.Blue, gi)
		escape := defenseAreaEscapeState{}
		safe = escape.apply(safe, info.Blue, gi)
		safe = clampMoveActionToField(safe, gi)
		safe = finalStoppedPlayAction(safe, gi)
		got, ok := safe.(*action.MoveTo)
		if !ok {
			t.Fatalf("expected reachable escape, got %T", safe)
		}
		if got.Dest.Dist2d(ball) < 700 || positionInGoalArea(got.Dest, getGoalAreaBounds(gi), 390) {
			t.Fatalf("unsafe final destination: %+v", got.Dest)
		}
		if (got.Dest.X-got.Pos.X)*(got.Pos.X-ball.X) < 0 {
			t.Fatal("escape initially moves toward the ball")
		}
	}
}

func TestFinalStopMovementDoesNotCrossBall(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	gi.State.SetBall(0, 0, 0, 1)
	move := &action.MoveTo{Team: info.Blue, Id: 1, Pos: info.Position{X: -1000}, Dest: info.Position{X: 1000}}
	safe := finalStoppedPlayAction(move, gi)
	got, ok := safe.(*action.MoveTo)
	if !ok {
		t.Fatalf("expected a reachable waypoint around the ball, got %T", safe)
	}
	if got.Dest.X == move.Dest.X && got.Dest.Y == move.Dest.Y {
		t.Fatal("retained a destination that drives through the ball")
	}
}

func TestDefenseEscapeHoldsInitialHeadingAcrossFrames(t *testing.T) {
	gi := newStoppedDefenseAreaTestGameInfo()
	escape := defenseAreaEscapeState{}
	gi.State.SetBlueRobot(1, 4000, 0, 0.25, 1)
	escape.apply(&action.Stop{Id: 1}, info.Blue, gi)
	gi.State.SetBlueRobot(1, 3900, 0, 0.3, 2)
	got := escape.apply(&action.MoveTo{Id: 1, Team: info.Blue}, info.Blue, gi).(*action.MoveTo)
	if got.Dest.Angle != 0.25 {
		t.Fatalf("escape heading drifted to %f", got.Dest.Angle)
	}
}
