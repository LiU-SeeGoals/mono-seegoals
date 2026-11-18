package ai

import (
	"math"
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestBallHoldIgnoresCameraJitterNextToRobot(t *testing.T) {
	jitter := []info.Position{{X: 55, Y: -166}, {X: 65, Y: -177}, {X: 43, Y: -190}}
	var hold ballHold
	now := time.Now()
	prev := hold.update(jitter[0], now)
	for i := 1; i < 60; i++ {
		now = now.Add(frameTime)
		held := hold.update(jitter[i%len(jitter)], now)
		if step := held.Dist2d(prev); step > 6 {
			t.Fatalf("frame %d: held ball moved %.1f mm on detection jitter", i, step)
		}
		prev = held
	}
	centroid := info.Position{X: 54.3, Y: -177.7}
	if d := prev.Dist2d(centroid); d > 8 {
		t.Fatalf("held ball %.1f mm from the jitter centroid", d)
	}
}

func TestBallHoldFollowsBallThatReallyMoved(t *testing.T) {
	var hold ballHold
	now := time.Now()
	hold.update(info.Position{}, now)
	moved := info.Position{X: 100}
	firstSeen := now.Add(frameTime)
	for elapsed := time.Duration(0); elapsed < alignBallMoveConfirm; elapsed += frameTime {
		if held := hold.update(moved, firstSeen.Add(elapsed)); held.Dist2d(info.Position{}) > 1e-9 {
			t.Fatalf("after %v: held ball jumped before the move was confirmed", elapsed)
		}
	}
	if held := hold.update(moved, firstSeen.Add(alignBallMoveConfirm)); held.Dist2d(moved) > 1e-9 {
		t.Fatalf("held ball at %.0f, want the moved ball at %.0f", held.X, moved.X)
	}
}

func TestBallHoldRestartsAfterStaleGap(t *testing.T) {
	var hold ballHold
	now := time.Now()
	hold.update(info.Position{}, now)
	ball := info.Position{X: 20}
	if held := hold.update(ball, now.Add(alignBallHoldStale+time.Millisecond)); held.Dist2d(ball) > 1e-9 {
		t.Fatalf("held ball at %.1f after a stale gap, want %.1f", held.X, ball.X)
	}
}

func TestKickLineLatchClosesOnlyNearLine(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	steps := []struct {
		side float64
		want bool
	}{
		{45, false},
		{alignLineEnterTolerance + 1, false},
		{alignLineEnterTolerance - 1, true},
		{alignLineExitTolerance - 1, true},
		{alignLineExitTolerance + 1, false},
		{alignLineExitTolerance - 1, false},
	}
	for i, step := range steps {
		now = now.Add(frameTime)
		if got := latch.onKickLine(-130, step.side, 0, 0, now); got != step.want {
			t.Fatalf("step %d: side %.0f mm gave onKickLine %v, want %v", i, step.side, got, step.want)
		}
	}
}

func TestKickLineLatchWaitsForSidewaysSlideToStop(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	if latch.onKickLine(-130, 5, 0.3, 0, now) {
		t.Fatal("expected no close-in while sliding sideways across the line")
	}
	if !latch.onKickLine(-130, 5, 0.05, 0, now.Add(frameTime)) {
		t.Fatal("expected close-in once the sideways slide has stopped")
	}
}

func TestKickLineLatchRequiresBehindBallAndFacingKick(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	if latch.onKickLine(-50, 0, 0, 0, now) {
		t.Fatal("expected no close-in with the robot center level with the dribbler")
	}
	if latch.onKickLine(-130, 0, 0, alignLineEnterHeading+0.01, now) {
		t.Fatal("expected no close-in while the heading is off")
	}
	if !latch.onKickLine(-130, 0, 0, 0, now) {
		t.Fatal("expected close-in when behind the ball on the line")
	}
	if !latch.onKickLine(-130, 0, 0, alignLineExitHeading-0.01, now) {
		t.Fatal("expected heading hysteresis to keep the close-in")
	}
}

func TestAlignStaysWideWhileBesideKickLine(t *testing.T) {
	for _, side := range []float64{25, 45} {
		gi := alignScene()
		visionFrameAt(gi, 1000, blueRobot(info.Position{X: -115, Y: side}))
		align := NewAlign(info.Blue, 1, alignKickTarget, info.Position{})

		dest := align.GetAction(gi).(*action.MoveTo).Dest
		if got, min := dest.Dist2d(info.Position{}), kickerStandoffDist(captureMarginToBall); got < min-1e-6 {
			t.Fatalf("side %.0f mm: carrot %.1f mm from the ball, want at least %.1f", side, got, min)
		}
		if align.Achieved(gi) {
			t.Fatalf("side %.0f mm: expected no handoff to KickBall beside the line", side)
		}
	}
}

func TestAlignClosesInOnceOnKickLine(t *testing.T) {
	gi := alignScene()
	visionFrameAt(gi, 1000, blueRobot(info.Position{X: -130}))
	align := NewAlign(info.Blue, 1, alignKickTarget, info.Position{})

	dest := align.GetAction(gi).(*action.MoveTo).Dest
	if got, max := dest.Dist2d(info.Position{}), kickerStandoffDist(0); got >= max {
		t.Fatalf("carrot %.1f mm from the ball, want closer than %.1f on the line", got, max)
	}
	if math.Abs(dest.Y) > 1e-6 {
		t.Fatalf("carrot %.1f mm beside the line, want on it", dest.Y)
	}
}

func TestAroundBallDestDoesNotPassKickLine(t *testing.T) {
	ball := info.Position{}
	radius := kickerStandoffDist(offCenterOrbitMargin)
	bot := info.Position{X: -radius * math.Cos(0.02), Y: radius * math.Sin(0.02)}
	lineup := info.Position{X: -radius}

	dest := aroundBallDest(ball, bot, lineup, offCenterOrbitMargin)
	if dest.Y < -1e-6 {
		t.Fatalf("carrot at y=%.1f mm crossed the kick line from a robot at y=%.1f", dest.Y, bot.Y)
	}
}
