package ai

import (
	"math"
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func newKickTestGameInfo() (*info.GameInfo, info.Position) {
	gi := info.NewGameInfo(2)
	ballPos := info.Position{X: info.Center2DribblerDist + info.BallRadius}
	gi.State.SetBlueRobot(3, 0, 0, 0, 1)
	gi.State.SetBall(ballPos.X, ballPos.Y, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(ballPos)
	gi.State.SetTrackedBall(ballPos, info.Position{}, 1)
	return gi, ballPos
}

func TestKickBallHeldInMouthAcceptsLogStallCase(t *testing.T) {
	if !kickBallHeldInMouth(60.4, 98.9, 55.5, 17.7*math.Pi/180) {
		t.Fatal("expected close dribbled ball from log stall case to count as held in the mouth")
	}
}

func TestKickBallHeldInMouthRejectsWideLateralOffset(t *testing.T) {
	if kickBallHeldInMouth(60.4, 98.9, kickMouthLateralTolerance+1, 17.7*math.Pi/180) {
		t.Fatal("expected wide lateral offset to stay unready")
	}
}

func TestKickBallHeldInMouthRejectsLargeHeadingError(t *testing.T) {
	if kickBallHeldInMouth(60.4, 98.9, 55.5, kickHeldHeadingTolerance+0.01) {
		t.Fatal("expected large heading error to stay unready")
	}
}

func TestKickBallShouldFireAfterHeldSettle(t *testing.T) {
	if !kickBallShouldFire(60.4, true) {
		t.Fatal("expected settled held ball to fire")
	}
}

func TestKickBallShouldNotFireWhenBallIsTooFar(t *testing.T) {
	if kickBallShouldFire(GetKickConfig().kickContactDist+1, true) {
		t.Fatal("expected ball outside contact distance to stay unready")
	}
}

func TestKickBallShouldNotFireBeforeDribblerSettles(t *testing.T) {
	if kickBallShouldFire(60.4, false) {
		t.Fatal("expected close ball to wait for the dribbler to settle")
	}
}

func TestKickBallCenterToleranceIsStricterThanMouthTolerance(t *testing.T) {
	if !(info.KickCenterTolerance < kickMouthLateralTolerance) {
		t.Fatal("expected mouth tolerance fallback to cover slightly off-center dribbled balls")
	}
}

func TestKickBallDribblesBeforeAndDuringKick(t *testing.T) {
	gi, ballPos := newKickTestGameInfo()
	kick := NewKickBall(info.Blue, 3, info.Position{X: 1000}, ballPos)

	settling, ok := kick.GetAction(gi).(*action.MoveTo)
	if !ok {
		t.Fatal("expected move-to action while settling the dribbler")
	}
	if !settling.Dribble || settling.KickSpeed != 0 {
		t.Fatalf("expected dribbler-only settling action, got dribble=%t kick=%d", settling.Dribble, settling.KickSpeed)
	}

	kick.dribbleSince = time.Now().Add(-kickDribbleSettleTime)
	firing, ok := kick.GetAction(gi).(*action.MoveTo)
	if !ok {
		t.Fatal("expected move-to action while firing")
	}
	if !firing.Dribble || firing.KickSpeed == 0 {
		t.Fatalf("expected dribbler to remain enabled while firing, got dribble=%t kick=%d", firing.Dribble, firing.KickSpeed)
	}
}

func TestKickAtPositionDribblesWhileFiring(t *testing.T) {
	gi, _ := newKickTestGameInfo()
	kick := NewKickAtPosition(info.Blue, 3, info.Position{X: 1000})
	kick.alignedSince = time.Now().Add(-kickAlignConfirmTime)

	firing, ok := kick.GetAction(gi).(*action.MoveTo)
	if !ok {
		t.Fatal("expected move-to action while firing")
	}
	if !firing.Dribble || firing.KickSpeed == 0 {
		t.Fatalf("expected dribbler to remain enabled while firing, got dribble=%t kick=%d", firing.Dribble, firing.KickSpeed)
	}
}
