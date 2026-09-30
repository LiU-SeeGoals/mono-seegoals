package ai

import (
	"testing"
	"time"
)

const frameTime = 16 * time.Millisecond

func TestAlignLatchIgnoresBallSpeedNoiseAroundOldCutoff(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	for i, speed := range []float64{0.28, 0.32, 0.27, 0.35, 0.29, 0.31, 0.38, 0.25} {
		if latch.ballRolling(speed, true, now.Add(time.Duration(i)*frameTime)) {
			t.Fatalf("frame %d: speed %.2f m/s latched a lying ball as rolling", i, speed)
		}
	}
}

func TestAlignLatchKeepsRollingUntilBallHasSlowed(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	if !latch.ballRolling(0.6, true, now) {
		t.Fatal("expected a 0.6 m/s ball to count as rolling")
	}
	now = now.Add(alignModeMinHold)
	if !latch.ballRolling(0.3, true, now) {
		t.Fatal("expected a ball slowing through 0.3 m/s to stay rolling")
	}
	if latch.ballRolling(0.15, true, now.Add(frameTime)) {
		t.Fatal("expected a ball below the exit speed to count as lying")
	}
}

func TestAlignLatchHoldsModeAfterSwitch(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	latch.ballRolling(0.6, true, now)
	if !latch.ballRolling(0, true, now.Add(alignModeMinHold-time.Millisecond)) {
		t.Fatal("expected the rolling mode to be held for the minimum time")
	}
	if latch.ballRolling(0, true, now.Add(alignModeMinHold)) {
		t.Fatal("expected the rolling mode to be released after the minimum time")
	}
}

func TestAlignLatchTreatsUnknownVelocityAsLying(t *testing.T) {
	var latch AlignLatch
	if latch.ballRolling(5, false, time.Now()) {
		t.Fatal("expected an unknown velocity to count as a lying ball")
	}
}

func TestAlignLatchReleasesOrbitOnlyBeyondMargin(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	enter := kickFarApproachDist
	steps := []struct {
		dist float64
		want bool
	}{
		{enter + 10, false},
		{enter, true},
		{enter + alignOrbitReleaseMargin, true},
		{enter + alignOrbitReleaseMargin + 1, false},
		{enter + 10, false},
		{enter, true},
	}
	for i, step := range steps {
		now = now.Add(alignModeMinHold)
		if got := latch.orbitBall(step.dist, enter, now); got != step.want {
			t.Fatalf("step %d: orbitBall(%.0f mm) = %v, want %v", i, step.dist, got, step.want)
		}
	}
}

func TestAlignLatchKeepsDirectApproachUntilReleaseRadius(t *testing.T) {
	var latch AlignLatch
	now := time.Now()
	steps := []struct {
		inEnter, inRelease, want bool
	}{
		{false, true, false},
		{true, true, true},
		{false, true, true},
		{false, false, false},
	}
	for i, step := range steps {
		now = now.Add(alignModeMinHold)
		if got := latch.DirectApproach(step.inEnter, step.inRelease, now); got != step.want {
			t.Fatalf("step %d: DirectApproach = %v, want %v", i, got, step.want)
		}
	}
}
