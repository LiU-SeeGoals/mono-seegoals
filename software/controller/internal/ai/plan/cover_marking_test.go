package ai

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestCoverMarkingKeepsLockWhenAnotherOpponentBecomesMoreDangerous(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, -2500, -1000, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[1].SetPositionTime(-3300, -1000, 0, 1)

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 1 {
		t.Fatalf("initial marked opponent = %d, want 1", got)
	}

	// Opponent 2 is now slightly more dangerous, but the existing mark must
	// remain stable instead of switching sides.
	opponents[2].SetPositionTime(-3500, 1000, 0, 2)
	assignments = tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 1 {
		t.Fatalf("marked opponent after new threat = %d, want retained opponent 1", got)
	}
}

func TestCoverMarkingUsesHysteresisBeforeReleasingTarget(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, -2500, 0, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[1].SetPositionTime(-2010, 0, 0, 1) // 2490 mm from goal: acquire.

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 1 {
		t.Fatalf("initial marked opponent = %d, want 1", got)
	}

	opponents[1].SetPositionTime(-1950, 0, 0, 2) // 2550 mm: outside acquire threshold but inside release threshold.
	opponents[2].SetPositionTime(-3500, 1000, 0, 2)
	assignments = tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 1 {
		t.Fatalf("jitter released opponent: got %d, want retained opponent 1", got)
	}

	opponents[1].SetPositionTime(-1790, 0, 0, 3) // 2710 mm: outside release threshold.
	assignments = tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 2 {
		t.Fatalf("replacement marked opponent = %d, want 2", got)
	}
}

func TestCoverMarkingBalancesThreatAndDefenderDistance(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, -3000, 1000, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[1].SetPositionTime(-3500, -1000, 0, 1)
	opponents[2].SetPositionTime(-3490, 1000, 0, 1)

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 2 {
		t.Fatalf("marked opponent = %d, want nearby opponent 2", got)
	}
}

func TestCoverMarkingGivesDifferentTargetsToTwoDefenders(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, -3000, -1000, 0, 1)
	gi.State.SetBlueRobot(4, -3000, 1000, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[1].SetPositionTime(-3500, -1000, 0, 1)
	opponents[2].SetPositionTime(-3500, 1000, 0, 1)

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3, 4})
	if got := assignments[3]; got != 1 {
		t.Fatalf("defender 3 marked opponent = %d, want 1", got)
	}
	if got := assignments[4]; got != 2 {
		t.Fatalf("defender 4 marked opponent = %d, want 2", got)
	}
	if assignments[3] == assignments[4] {
		t.Fatal("two Cover defenders were assigned the same opponent")
	}
}
