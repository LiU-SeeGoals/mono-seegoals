package ai

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
	"google.golang.org/protobuf/proto"
)

func newDefenseGeometryGameInfo() *info.GameInfo {
	return newDefenseGeometryGameInfoWithSize(9000, 6000, 1000, 1000, 2000)
}

func newDefenseGeometryGameInfoWithSize(length, width, goalWidth, penaltyDepth, penaltyWidth int32) *info.GameInfo {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.SetField(&ssl_vision.SSL_GeometryFieldSize{
		FieldLength:      proto.Int32(length),
		FieldWidth:       proto.Int32(width),
		GoalWidth:        proto.Int32(goalWidth),
		GoalDepth:        proto.Int32(180),
		PenaltyAreaDepth: proto.Int32(penaltyDepth),
		PenaltyAreaWidth: proto.Int32(penaltyWidth),
	})
	return gi
}

func TestPassReceiverThreatRewardsCentralOpenGoalPosition(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	ballPos := info.Position{}

	central := scorePassReceiver(
		gi,
		info.Blue,
		3,
		ballPos,
		info.Position{X: -3200, Y: 0},
	)
	wide := scorePassReceiver(
		gi,
		info.Blue,
		3,
		ballPos,
		info.Position{X: -3200, Y: 1800},
	)

	if central.Centrality <= wide.Centrality {
		t.Fatalf("centrality: central %.3f, wide %.3f", central.Centrality, wide.Centrality)
	}
	if central.OpenGoalAngle <= wide.OpenGoalAngle {
		t.Fatalf("goal angle: central %.3f, wide %.3f", central.OpenGoalAngle, wide.OpenGoalAngle)
	}
	if central.Total <= wide.Total {
		t.Fatalf("total threat: central %.3f, wide %.3f", central.Total, wide.Total)
	}
}

func TestCoverMarkingRequiresVisionFieldGeometry(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, -2500, 0, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	gi.State.GetOtherTeam(info.Blue)[2].SetPositionTime(-3200, 0, 0, 1)

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if len(assignments) != 0 {
		t.Fatalf("assignments without Vision geometry = %v, want none", assignments)
	}
}

func TestGoalAndThreatScoreUseNonStandardVisionField(t *testing.T) {
	gi := newDefenseGeometryGameInfoWithSize(12000, 8000, 1200, 1500, 3000)
	goal, ok := gi.OwnGoalCenter(info.Blue)
	if !ok || goal.X != -6000 {
		t.Fatalf("Blue goal center = %+v, ok=%t; want X=-6000 from Vision", goal, ok)
	}

	score := scorePassReceiver(gi, info.Blue, 3, info.Position{}, info.Position{X: -3000})
	if math.Abs(score.GoalProximity-0.5) > 1e-9 {
		t.Fatalf("goal proximity = %.3f, want 0.5 on 12 m field", score.GoalProximity)
	}
}

func TestPassReceiverThreatPenalizesBlockedPassLine(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	ballPos := info.Position{}
	receiverPos := info.Position{X: -3200, Y: 800}

	open := scorePassReceiver(gi, info.Blue, 3, ballPos, receiverPos)
	// Robot 4 is directly on the ball-to-receiver segment. Robot 3 is the
	// prospective High defender and is deliberately ignored by the score.
	gi.State.SetBlueRobot(4, receiverPos.X/2, receiverPos.Y/2, 0, 1)
	blocked := scorePassReceiver(gi, info.Blue, 3, ballPos, receiverPos)

	if math.Abs(open.PassLineOpenness-1) > 1e-9 {
		t.Fatalf("open lane score = %.3f, want 1", open.PassLineOpenness)
	}
	if blocked.PassLineOpenness != 0 {
		t.Fatalf("blocked lane score = %.3f, want 0", blocked.PassLineOpenness)
	}
	if blocked.Total >= open.Total {
		t.Fatalf("blocked total %.3f is not below open total %.3f", blocked.Total, open.Total)
	}
}

func TestCoverMarkingSelectsHighestRankedPassReceiver(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	gi.State.SetBlueRobot(3, -2500, 0, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[1].SetPositionTime(-3500, 1800, 0, 1)
	opponents[2].SetPositionTime(-3200, 0, 0, 1)

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 2 {
		t.Fatalf("marked opponent = %d, want central receiver 2", got)
	}
}

func TestCoverMarkingConsidersReceiverBeforeFinalDefensiveZone(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	gi.State.SetBlueRobot(3, -2500, 0, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	// This receiver is 3500 mm from our goal: outside the legacy Cover zone,
	// but central and already in our defensive half.
	gi.State.GetOtherTeam(info.Blue)[2].SetPositionTime(-1000, 0, 0, 1)

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 2 {
		t.Fatalf("marked opponent = %d, want early receiver 2", got)
	}
}

func TestCoverMarkingKeepsLockWhenAnotherOpponentBecomesMoreDangerous(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
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

func TestCoverMarkingSwitchesToSubstantiallyMoreDangerousReceiver(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	gi.State.SetBlueRobot(3, -2500, 1200, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[1].SetPositionTime(-3300, 1800, 0, 1)

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 1 {
		t.Fatalf("initial marked opponent = %d, want 1", got)
	}

	// A central receiver with a much wider goal angle should overcome the
	// switch margin, unlike the small score change in the lock test above.
	opponents[2].SetPositionTime(-3500, 0, 0, 2)
	assignments = tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 2 {
		t.Fatalf("marked opponent after major threat change = %d, want 2", got)
	}
}

func TestCoverMarkingUsesHysteresisBeforeReleasingTarget(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	gi.State.SetBlueRobot(3, -2500, 0, 0, 1)
	gi.State.GetBall().SetEstimatedPosition(info.Position{})
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[1].SetPositionTime(-200, 1000, 0, 1) // Inside receiver acquisition boundary.

	tracker := coverMarkingTracker{}
	assignments := tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 1 {
		t.Fatalf("initial marked opponent = %d, want 1", got)
	}

	opponents[1].SetPositionTime(100, 1000, 0, 2) // Across midfield but inside the release margin.
	assignments = tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 1 {
		t.Fatalf("jitter released opponent: got %d, want retained opponent 1", got)
	}

	opponents[1].SetPositionTime(400, 1000, 0, 3) // Outside release threshold.
	opponents[2].SetPositionTime(-3500, 1000, 0, 3)
	assignments = tracker.update(gi, info.Blue, []info.ID{3})
	if got := assignments[3]; got != 2 {
		t.Fatalf("replacement marked opponent = %d, want 2", got)
	}
}

func TestCoverMarkingBalancesThreatAndDefenderDistance(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
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
	gi := newDefenseGeometryGameInfo()
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
