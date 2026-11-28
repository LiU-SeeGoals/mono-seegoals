package rules

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestForRobotLimitsBallToRestartKicker(t *testing.T) {
	ge := &info.GameEvent{CurrentState: info.STATE_FREE_KICK, TeamWithPossession: info.Blue}
	team := FromEvent(ge, info.Blue)
	if !team.MayTouchBall {
		t.Fatal("team rules should allow our free kick")
	}
	if r := team.ForRobot(3, 3, true); !r.MayTouchBall || r.BallKeepout != 0 {
		t.Fatalf("kicker restricted: %+v", r)
	}
	if r := team.ForRobot(4, 3, true); r.MayTouchBall || r.BallKeepout < RestartBallKeepout {
		t.Fatalf("non-kicker may touch the ball: %+v", r)
	}
	if r := team.ForRobot(4, 0, false); !r.MayTouchBall {
		t.Fatal("unknown kicker should keep the team rules")
	}
	ge.BallInPlay = true
	if r := FromEvent(ge, info.Blue).ForRobot(4, 3, true); !r.MayTouchBall {
		t.Fatal("ball in play: every robot may touch it")
	}
}
