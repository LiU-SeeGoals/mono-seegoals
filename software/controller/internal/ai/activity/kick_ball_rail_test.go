package ai

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func kickBallDest(gi *info.GameInfo, kick *KickBall, now int64, robot, ball info.Position) info.Position {
	gi.State.SetBall(ball.X, ball.Y, 0, now)
	visionFrameAt(gi, now, blueRobot(robot))
	return kick.GetAction(gi).(*action.MoveTo).Dest
}

func TestKickBallReachesKickLineBeforeContact(t *testing.T) {
	gi := alignScene()
	kick := NewKickBall(info.Blue, 1, alignKickTarget, info.Position{})
	contact := -(info.Center2DribblerDist + info.BallRadius)
	startSide := 20.0
	robot := info.Position{X: -136.5, Y: startSide}

	for i := 0; i < 200; i++ {
		dest := kickBallDest(gi, kick, int64(1000+16*i), robot, info.Position{})
		robot.X += 0.2 * (dest.X - robot.X)
		robot.Y += 0.2 * (dest.Y - robot.Y)
		robot.Angle = dest.Angle
		if robot.X >= contact {
			if math.Abs(robot.Y) > 0.4*startSide {
				t.Fatalf("touched the ball %.1f mm beside the kick line, started %.0f mm off", robot.Y, startSide)
			}
			return
		}
	}
	t.Fatal("robot never reached the ball")
}

func TestKickBallLineIgnoresBallJitter(t *testing.T) {
	jitter := []info.Position{{X: 0, Y: 0}, {X: 10, Y: -11}, {X: -12, Y: -24}}
	gi := alignScene()
	kick := NewKickBall(info.Blue, 1, alignKickTarget, info.Position{})
	robot := info.Position{X: -130}

	minY, maxY := math.Inf(1), math.Inf(-1)
	for i := 0; i < 30; i++ {
		dest := kickBallDest(gi, kick, int64(1000+16*i), robot, jitter[i%len(jitter)])
		minY = math.Min(minY, dest.Y)
		maxY = math.Max(maxY, dest.Y)
	}
	if maxY-minY > 5 {
		t.Fatalf("destination swung %.1f mm sideways on ball detection jitter", maxY-minY)
	}
}

func TestKickBallKeepsDrivingWhilePushingBall(t *testing.T) {
	gi := alignScene()
	kick := NewKickBall(info.Blue, 1, alignKickTarget, info.Position{})
	robot := info.Position{X: -130}

	gi.State.SetTrackedBall(info.Position{}, info.Position{X: 0.2}, 0)
	if dest := kickBallDest(gi, kick, 1000, robot, info.Position{}); dest.X <= robot.X+1 {
		t.Fatalf("destination x=%.1f, want ahead of the robot at %.1f while pushing a slow ball", dest.X, robot.X)
	}

	gi.State.SetTrackedBall(info.Position{}, info.Position{X: 1.0}, 0)
	if dest := kickBallDest(gi, kick, 1016, robot, info.Position{}); dest.Dist2d(robot) > 1e-6 {
		t.Fatalf("destination (%.1f, %.1f), want the robot to hold once the ball is kicked away", dest.X, dest.Y)
	}
}
