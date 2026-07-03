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

func TestAlignAchievedAcceptsReachableOffCenterBall(t *testing.T) {
	gi := info.NewGameInfo(2)
	ball := info.Position{}
	robot := info.Position{X: -120, Y: 30, Angle: 0}
	gi.State.SetBlueRobot(3, robot.X, robot.Y, robot.Angle, 1)
	gi.State.SetBall(ball.X, ball.Y, ball.Z, 1)
	gi.State.GetBall().SetEstimatedPosition(ball)
	gi.State.SetTrackedBall(ball, info.Position{}, 1)

	align := NewAlign(info.Blue, 3, info.Position{X: 1000}, ball)
	if !align.Achieved(gi) {
		t.Fatal("expected reachable off-center ball to transition to KickBall")
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

func TestAroundBallDestEnforcesMinimumMoveDistance(t *testing.T) {
	ball := info.Position{X: 0, Y: 0}
	bot := info.Position{X: -137, Y: 0}
	lineup := info.Position{X: -126.5, Y: 35}

	dest := aroundBallDest(ball, bot, lineup, offCenterOrbitMargin)
	moveDist := bot.Dist2d(dest)

	if math.Abs(moveDist-minAroundBallMoveDist) > 1e-6 {
		t.Fatalf("expected %.1f mm minimum move, got %.3f mm", minAroundBallMoveDist, moveDist)
	}
	wantRadius := kickerStandoffDist(offCenterOrbitMargin)
	if got := ball.Dist2d(dest); math.Abs(got-wantRadius) > 1e-6 {
		t.Fatalf("expected destination to remain on %.1f mm orbit, got %.3f mm", wantRadius, got)
	}
}

func TestAroundBallDestDoesNotExtendTinyCorrection(t *testing.T) {
	ball := info.Position{X: 0, Y: 0}
	radius := kickerStandoffDist(offCenterOrbitMargin)
	bot := info.Position{X: -radius, Y: 0}
	lineup := info.Position{X: -radius, Y: 1}

	dest := aroundBallDest(ball, bot, lineup, offCenterOrbitMargin)

	if got := bot.Dist2d(dest); got >= minAroundBallMoveTrigger {
		t.Fatalf("expected tiny correction to remain below trigger, got %.3f mm", got)
	}
}

func TestAroundBallDestDoesNotTurnRadialCorrectionIntoOrbit(t *testing.T) {
	ball := info.Position{X: 0, Y: 0}
	bot := info.Position{X: -137, Y: 0}
	radius := kickerStandoffDist(offCenterOrbitMargin)
	lineup := info.Position{X: -radius, Y: 0}

	dest := aroundBallDest(ball, bot, lineup, offCenterOrbitMargin)

	if got := dest.Dist2d(lineup); got > 1e-6 {
		t.Fatalf("expected radial correction to target lineup, offset %.6f mm", got)
	}
}

func TestAroundBallDestLeavesLargeMoveUnchanged(t *testing.T) {
	ball := info.Position{X: 0, Y: 0}
	botRadius := kickerStandoffDist(offCenterOrbitMargin)
	bot := info.Position{X: -botRadius, Y: 0}
	lineup := info.Position{X: 0, Y: botRadius}

	dest := aroundBallDest(ball, bot, lineup, offCenterOrbitMargin)
	wantBearing := math.Pi - aroundBallShiftAngle
	wantRadius := kickerStandoffDist(offCenterOrbitMargin + maxMarginToBall/2)
	want := info.Position{
		X: wantRadius * math.Cos(wantBearing),
		Y: wantRadius * math.Sin(wantBearing),
	}

	if got := dest.Dist2d(want); got > 1e-6 {
		t.Fatalf("expected large move destination to remain unchanged, offset %.6f mm", got)
	}
}
