package roles

import (
	"testing"
	"time"
)

func TestUpdateAlignConfirmationRequiresStableWindow(t *testing.T) {
	var alignedSince time.Time
	now := time.Now()

	if updateAlignConfirmation(&alignedSince, true, now) {
		t.Fatal("expected confirmation window to start without completing")
	}
	if updateAlignConfirmation(&alignedSince, true, now.Add(alignTransitionConfirmTime-time.Millisecond)) {
		t.Fatal("expected alignment to remain unconfirmed before the stable window")
	}
	if !updateAlignConfirmation(&alignedSince, true, now.Add(alignTransitionConfirmTime)) {
		t.Fatal("expected alignment to confirm after the stable window")
	}
}

func TestUpdateAlignConfirmationResetsWhenAlignmentIsLost(t *testing.T) {
	now := time.Now()
	alignedSince := now.Add(-alignTransitionConfirmTime)

	if updateAlignConfirmation(&alignedSince, false, now) {
		t.Fatal("expected lost alignment to remain unconfirmed")
	}
	if !alignedSince.IsZero() {
		t.Fatal("expected lost alignment to reset the confirmation window")
	}
}
