package info

import (
	"testing"
	"time"
)

func TestTrackedSamplesUseGameClock(t *testing.T) {
	gi := NewGameInfo(4)
	clock := NewManualClock(time.Unix(1000, 0))
	gi.SetClock(clock)
	gi.State.SetTrackedBall(Position{X: 10}, Position{X: 1}, 0)
	gi.State.SetTrackedRobot(Blue, 1, Position{}, Position{X: 2}, 0, 0, 0)

	if _, ok := gi.State.GetTrackedBall().GetFreshTrackedVelocity(gi.Now(), 100*time.Millisecond); !ok {
		t.Fatal("fresh sample rejected under ManualClock")
	}
	if _, ok := gi.State.GetTrackedRobot(Blue, 1).GetFreshTrackedVelocity(gi.Now(), 100*time.Millisecond); !ok {
		t.Fatal("fresh robot sample rejected under ManualClock")
	}
	clock.Advance(TrackedStaleAfter + time.Millisecond)
	if _, ok := gi.State.GetTrackedBall().GetFreshTrackedPosition(gi.Now(), TrackedStaleAfter); ok {
		t.Fatal("stale tracked position still returned")
	}
}

func TestEstimatedPositionFailsBeforeFirstSighting(t *testing.T) {
	b := NewBall(4)
	if _, err := b.GetEstimatedPosition(); err == nil {
		t.Fatal("unseen ball reported a position")
	}
	b.SetEstimatedPosition(Position{X: 5})
	if pos, err := b.GetEstimatedPosition(); err != nil || pos.X != 5 {
		t.Fatalf("got %v, %v", pos, err)
	}
}
