package info

import (
	"math"
	"testing"
)

func visionFrame(gs *GameState, now int64) {
	gs.SetMessageReceivedTime(now)
	gs.Update()
}

func assertBallEstimate(t *testing.T, gs *GameState, wantKind BallEstimateKind, wantHolder *Robot, wantX, wantY float64) {
	t.Helper()
	kind, holder := gs.Ball.GetEstimateKind()
	pos, _ := gs.Ball.GetEstimatedPosition()
	if kind != wantKind || holder != wantHolder {
		t.Fatalf("estimate kind=%d holder=%v, want kind=%d holder=%v", kind, holder, wantKind, wantHolder)
	}
	if math.Abs(pos.X-wantX) > 1e-6 || math.Abs(pos.Y-wantY) > 1e-6 {
		t.Fatalf("estimate at (%.1f, %.1f), want (%.1f, %.1f)", pos.X, pos.Y, wantX, wantY)
	}
}

func TestBallEstimateKeepsRecentSightingAsObserved(t *testing.T) {
	gs := NewGameState(10)
	gs.SetBall(10, 20, 0, 1000)
	visionFrame(gs, 1000)
	assertBallEstimate(t, gs, BallObserved, nil, 10, 20)

	visionFrame(gs, 1000+BallUnseenAfterMs)
	assertBallEstimate(t, gs, BallObserved, nil, 10, 20)
}

func TestBallEstimateMarksLostBallUnseen(t *testing.T) {
	gs := NewGameState(10)
	gs.SetBall(10, 20, 0, 1000)
	visionFrame(gs, 1000)

	visionFrame(gs, 1200)
	assertBallEstimate(t, gs, BallUnseen, nil, 10, 20)
}

func TestBallEstimateFollowsDribblerThatHidesBall(t *testing.T) {
	gs := NewGameState(10)
	robot := gs.GetRobot(1, Blue)
	gs.SetBall(0, 0, 0, 1000)
	gs.SetRobotFromVision(Blue, 1, Position{X: -110}, 1000, 0)
	visionFrame(gs, 1000)

	gs.SetRobotFromVision(Blue, 1, Position{X: -110}, 1200, 0)
	visionFrame(gs, 1200)
	reach := Center2DribblerDist + BallRadius
	assertBallEstimate(t, gs, BallInDribbler, robot, -110+reach, 0)

	gs.SetRobotFromVision(Blue, 1, Position{X: -110, Angle: math.Pi / 2}, 1250, 0)
	visionFrame(gs, 1250)
	assertBallEstimate(t, gs, BallInDribbler, robot, -110, reach)

	gs.SetBall(50, 60, 0, 1300)
	visionFrame(gs, 1300)
	assertBallEstimate(t, gs, BallObserved, nil, 50, 60)
}

func TestBallEstimateIgnoresRobotsThatCannotHoldBall(t *testing.T) {
	cases := []struct {
		name    string
		pos     Position
		seenAt  int64
		frameAt int64
	}{
		{"ball behind robot", Position{X: -110, Angle: math.Pi}, 1200, 1200},
		{"ball beyond dribbler reach", Position{X: -130}, 1200, 1200},
		{"ball beside dribbler", Position{X: -110, Y: 60}, 1200, 1200},
		{"robot not seen recently", Position{X: -110}, 1000, 1000 + holderUnseenAfterMs + 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gs := NewGameState(10)
			gs.SetBall(0, 0, 0, 1000)
			gs.SetRobotFromVision(Blue, 1, c.pos, c.seenAt, 0)
			visionFrame(gs, c.frameAt)
			assertBallEstimate(t, gs, BallUnseen, nil, 0, 0)
		})
	}
}

func TestBallEstimatePrefersClosestDribbler(t *testing.T) {
	gs := NewGameState(10)
	gs.SetBall(0, 0, 0, 1000)
	gs.SetRobotFromVision(Blue, 1, Position{X: -110}, 1200, 0)
	gs.SetRobotFromVision(Yellow, 2, Position{X: 105, Angle: math.Pi}, 1200, 0)
	visionFrame(gs, 1200)

	reach := Center2DribblerDist + BallRadius
	assertBallEstimate(t, gs, BallInDribbler, gs.GetRobot(2, Yellow), 105-reach, 0)
}

func TestBallEstimateDropsHolderVisionLost(t *testing.T) {
	gs := NewGameState(10)
	gs.SetBall(0, 0, 0, 1000)
	gs.SetRobotFromVision(Blue, 1, Position{X: -110}, 1200, 0)
	visionFrame(gs, 1200)
	assertBallEstimate(t, gs, BallInDribbler, gs.GetRobot(1, Blue), -110+Center2DribblerDist+BallRadius, 0)

	visionFrame(gs, 1200+holderUnseenAfterMs+1)
	assertBallEstimate(t, gs, BallUnseen, nil, 0, 0)
}
