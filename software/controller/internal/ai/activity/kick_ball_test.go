package ai

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func kickBallGameInfo(robotPos, ballPos, ballVel info.Position) *info.GameInfo {
	gi := info.NewGameInfo(2)
	gi.State.SetYellowRobot(1, robotPos.X, robotPos.Y, robotPos.Angle, 1)
	gi.State.SetBall(ballPos.X, ballPos.Y, ballPos.Z, 1)
	gi.State.SetTrackedBall(ballPos, ballVel, 1)
	gi.State.Update()
	return gi
}

func requireMoveTo(t *testing.T, act action.Action) *action.MoveTo {
	t.Helper()
	move, ok := act.(*action.MoveTo)
	if !ok {
		t.Fatalf("expected *action.MoveTo, got %T", act)
	}
	return move
}

func requireNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %f, want %f", got, want)
	}
}

func TestKickBallUsesCloseApproachBeforeKick(t *testing.T) {
	ballPos := info.Position{X: 0, Y: 0, Z: 0, Angle: 0}
	targetPos := info.Position{X: 1000, Y: 0, Z: 0, Angle: 0}
	kick := NewKickBall(info.Yellow, 1, targetPos, ballPos)

	approachGI := kickBallGameInfo(
		info.Position{X: -200, Y: 0, Z: 0, Angle: 0},
		ballPos,
		info.Position{},
	)
	approachAction := requireMoveTo(t, kick.GetAction(approachGI))
	requireNear(t, approachAction.Dest.X, -GetKickConfig().closeApproachClearance)
	requireNear(t, approachAction.Dest.Y, 0)
	requireNear(t, approachAction.Dest.Angle, 0)
	if approachAction.KickSpeed != 0 {
		t.Fatalf("close approach should not kick, got kick speed %d", approachAction.KickSpeed)
	}
	if !approachAction.Dribble {
		t.Fatalf("close approach should keep the dribbler on")
	}

	kickGI := kickBallGameInfo(
		info.Position{X: -GetKickConfig().closeApproachClearance, Y: 0, Z: 0, Angle: 0},
		ballPos,
		info.Position{},
	)
	kickAction := requireMoveTo(t, kick.GetAction(kickGI))
	requireNear(t, kickAction.Dest.X, GetKickConfig().driveThrough)
	requireNear(t, kickAction.Dest.Y, 0)
	if kickAction.KickSpeed == 0 {
		t.Fatalf("expected kick after close approach is reached")
	}
}

func TestKickBallDoesNotKickMovingBall(t *testing.T) {
	ballPos := info.Position{X: 0, Y: 0, Z: 0, Angle: 0}
	targetPos := info.Position{X: 1000, Y: 0, Z: 0, Angle: 0}
	robotPos := info.Position{X: -GetKickConfig().closeApproachClearance, Y: 0, Z: 0, Angle: 0}
	kick := NewKickBall(info.Yellow, 1, targetPos, ballPos)
	kick.closeApproachReached = true

	gi := kickBallGameInfo(
		robotPos,
		ballPos,
		info.Position{X: GetKickConfig().movingBallHoldSpeed + 0.01, Y: 0, Z: 0, Angle: 0},
	)
	move := requireMoveTo(t, kick.GetAction(gi))
	requireNear(t, move.Dest.X, robotPos.X)
	requireNear(t, move.Dest.Y, robotPos.Y)
	if move.KickSpeed != 0 {
		t.Fatalf("moving ball should not be kicked, got kick speed %d", move.KickSpeed)
	}
}
