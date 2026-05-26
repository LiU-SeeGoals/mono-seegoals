package ai

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func gameInfoWithBallAtDribbler() *info.GameInfo {
	gi := info.NewGameInfo(4)
	gi.State.SetYellowRobot(1, 0, 0, 0, 1)
	gi.State.SetBall(220, 0, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{X: 220, Y: 0, Z: 0, Angle: 0})
	return gi
}

func TestMoveToBallAchievedWithBallControl(t *testing.T) {
	gi := gameInfoWithBallAtDribbler()
	activity := NewMoveToBall(info.Yellow, 1)

	if !activity.Achieved(gi) {
		t.Fatal("MoveToBall did not finish even though the ball is at the dribbler")
	}
}

func TestKickBallUsesKickerWithBallControl(t *testing.T) {
	gi := gameInfoWithBallAtDribbler()
	activity := NewKickBall(
		info.Yellow,
		1,
		info.Position{X: 4500, Y: 0, Z: 0, Angle: 0},
		info.Position{X: 220, Y: 0, Z: 0, Angle: 0},
	)

	act, ok := activity.GetAction(gi).(*action.MoveTo)
	if !ok {
		t.Fatalf("KickBall action type = %T, want *action.MoveTo", activity.GetAction(gi))
	}
	if act.KickSpeed == 0 {
		t.Fatal("KickBall did not set KickSpeed even though the ball is at the dribbler")
	}
	if act.Dribble {
		t.Fatal("KickBall set dribble instead of kicking even though the ball is at the dribbler")
	}
}
