package ai

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

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
	if !kickBallShouldFire(60.4, false, true) {
		t.Fatal("expected settled held ball to fire even if it is not perfectly centered")
	}
}

func TestKickBallShouldNotFireWhenBallIsTooFar(t *testing.T) {
	if kickBallShouldFire(GetKickConfig().kickContactDist+1, true, true) {
		t.Fatal("expected ball outside contact distance to stay unready")
	}
}

func TestKickBallCenterToleranceIsStricterThanMouthTolerance(t *testing.T) {
	if !(info.KickCenterTolerance < kickMouthLateralTolerance) {
		t.Fatal("expected mouth tolerance fallback to cover slightly off-center dribbled balls")
	}
}
