package ai

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestCaptureApproachReadyAcceptsReachableDribblerCorridor(t *testing.T) {
	ball := info.Position{X: 0, Y: 0}
	target := info.Position{X: 1000, Y: 0}
	robot := info.Position{X: -120, Y: info.DribblerHalfWidth + info.BallRadius - 1, Angle: 0}

	if !captureApproachReady(robot, ball, target, 0) {
		t.Fatal("expected approach to be ready while the ball is still within dribbler reach")
	}
}

func TestCaptureApproachReadyRejectsOutsideDribblerCorridor(t *testing.T) {
	ball := info.Position{X: 0, Y: 0}
	target := info.Position{X: 1000, Y: 0}
	robot := info.Position{X: -120, Y: info.DribblerHalfWidth + info.BallRadius + 5, Angle: 0}

	if captureApproachReady(robot, ball, target, 0) {
		t.Fatal("expected approach to stay unready when the ball is outside dribbler reach")
	}
}

func TestCaptureApproachReadyRejectsLargeHeadingError(t *testing.T) {
	ball := info.Position{X: 0, Y: 0}
	target := info.Position{X: 1000, Y: 0}
	robot := info.Position{X: -120, Y: 0, Angle: 0}

	if captureApproachReady(robot, ball, target, 15*math.Pi/180) {
		t.Fatal("expected approach to stay unready with a large heading error")
	}
}

func TestCaptureOrbitMarginStaysWideUntilApproachReady(t *testing.T) {
	if got := captureOrbitMargin(0, false, false); got != captureMarginToBall {
		t.Fatalf("expected capture margin while approach is not ready, got %.1f", got)
	}
}

func TestCaptureOrbitMarginStaysWideWhileOffCenter(t *testing.T) {
	if got := captureOrbitMargin(0, true, true); got != offCenterOrbitMargin {
		t.Fatalf("expected off-center margin while ball is not centered, got %.1f", got)
	}
}

func TestCaptureOrbitMarginAllowsPushOnlyWhenCentered(t *testing.T) {
	if got := captureOrbitMargin(0, true, false); got != ballPushMargin {
		t.Fatalf("expected push margin once approach is ready and centered, got %.1f", got)
	}
}
