package roles

import (
	"math"
	"testing"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestClosestOpponentToOwnGoal(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponents := gi.State.GetOtherTeam(info.Blue)

	opponents[1].SetPositionTime(-4000, 100, 0, 1)
	opponents[1].SetActive(true)
	opponents[2].SetPositionTime(-2000, 0, 0, 1)
	opponents[2].SetActive(true)
	// This robot is closer to the goal, but inactive and must be ignored.
	opponents[3].SetPositionTime(-4490, 0, 0, 1)
	opponents[3].SetActive(false)

	state := &DefensePressState{gi: gi, team: info.Blue}
	opponent, position, found := state.closestOpponentToOwnGoal()

	if !found {
		t.Fatal("expected to find an active opponent")
	}
	if got, want := opponent.GetID(), info.ID(1); got != want {
		t.Fatalf("closest opponent ID = %d, want %d", got, want)
	}
	if got, want := position, (info.Position{X: -4000, Y: 100}); got != want {
		t.Fatalf("closest opponent position = %+v, want %+v", got, want)
	}
}

func TestClosestOpponentToOwnGoalReturnsFalseWithoutGameState(t *testing.T) {
	state := &DefensePressState{}

	if _, _, found := state.closestOpponentToOwnGoal(); found {
		t.Fatal("found an opponent without game state")
	}
}

func TestOpponentWithBallReturnsClosestOpponentWithinDistance(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponents := gi.State.GetOtherTeam(info.Blue)
	opponents[4].SetPositionTime(1200, -350, 0, 1)
	opponents[5].SetPositionTime(1125, -350, 0, 1)
	ballPos := info.Position{X: 1100, Y: -350}

	state := &DefensePressState{gi: gi, team: info.Blue}
	possessor, position, found := state.opponentWithBall(ballPos)

	if !found {
		t.Fatal("expected to find the opponent with the ball")
	}
	if got, want := possessor.GetTeam(), info.Yellow; got != want {
		t.Fatalf("possessor team = %s, want %s", got, want)
	}
	if got, want := possessor.GetID(), info.ID(5); got != want {
		t.Fatalf("possessor ID = %d, want %d", got, want)
	}
	if got, want := position, (info.Position{X: 1125, Y: -350}); got != want {
		t.Fatalf("possessor position = %+v, want %+v", got, want)
	}
}

func TestOpponentWithBallIgnoresNearbyTeammate(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	teammate := gi.State.GetTeam(info.Blue)[2]
	teammate.SetPositionTime(50, 0, 0, 1)

	state := &DefensePressState{gi: gi, team: info.Blue}
	if _, _, found := state.opponentWithBall(info.Position{}); found {
		t.Fatal("reported a teammate as the opponent with the ball")
	}
}

func TestOpponentWithBallRejectsOpponentBeyondDistance(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponent := gi.State.GetOtherTeam(info.Blue)[4]
	opponent.SetPositionTime(defensePossessionDistance+1, 0, 0, 1)

	state := &DefensePressState{gi: gi, team: info.Blue}
	if _, _, found := state.opponentWithBall(info.Position{}); found {
		t.Fatal("reported an opponent farther than the possession distance")
	}
}

func TestCoverTargetBlocksPassingLaneCloserToDangerousOpponent(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponents := gi.State.GetOtherTeam(info.Blue)
	ballPos := info.Position{X: -1000, Y: 300}

	// The possessor is near the ball but not exactly at its position. The
	// blocking line must start at the ball, not at the robot center.
	opponents[4].SetPositionTime(-1100, 350, 0, 1)
	opponents[1].SetPositionTime(-3500, -300, 0, 1)

	state := &DefensePressState{gi: gi, team: info.Blue, robotId: 7}
	target := state.coverTarget(ballPos)
	want := info.Position{
		X: ballPos.X + (-3500-ballPos.X)*coverReceiverWeight,
		Y: ballPos.Y + (-300-ballPos.Y)*coverReceiverWeight,
	}
	want.Angle = want.AngleToPosition(ballPos)

	if math.Abs(target.X-want.X) > 1e-9 || math.Abs(target.Y-want.Y) > 1e-9 {
		t.Fatalf("cover target = %+v, want %+v", target, want)
	}
	if math.Abs(info.NormalizeAngleDelta(target.Angle, want.Angle)) > 1e-9 {
		t.Fatalf("cover angle = %.6f, want %.6f", target.Angle, want.Angle)
	}
	if target.Dist2d(info.Position{X: -3500, Y: -300}) >= target.Dist2d(ballPos) {
		t.Fatal("cover target is not closer to the dangerous opponent than the ball possessor")
	}
}

func TestCoverTargetParksWhenReceiverIsNotDangerous(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponents := gi.State.GetOtherTeam(info.Blue)
	ballPos := info.Position{}

	opponents[4].SetPositionTime(ballPos.X, ballPos.Y, 0, 1)
	// Blue defends the negative half by default, so this opponent is 3500 mm
	// from the own goal center at X=-4500.
	opponents[1].SetPositionTime(-1000, 0, 0, 1)

	state := &DefensePressState{gi: gi, team: info.Blue, robotId: 7}
	if got := state.coverTarget(ballPos); got != coverParkingPos {
		t.Fatalf("cover target = %+v, want parking position %+v", got, coverParkingPos)
	}
}

func TestCoverTargetParksWithoutOpponentNearBall(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	state := &DefensePressState{gi: gi, team: info.Blue, robotId: 7}

	if got := state.coverTarget(info.Position{}); got != coverParkingPos {
		t.Fatalf("cover target = %+v, want parking position %+v", got, coverParkingPos)
	}
}

func TestCoverTargetBlocksDangerousOpponentWithoutBallPossessor(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponents := gi.State.GetOtherTeam(info.Blue)
	ballPos := info.Position{X: 0, Y: 400}
	dangerousPos := info.Position{X: -3500, Y: -200}

	// The opponent is dangerous but much farther than 200 mm from the ball, so
	// there is deliberately no local ball possessor.
	opponents[1].SetPositionTime(dangerousPos.X, dangerousPos.Y, 0, 1)

	state := &DefensePressState{gi: gi, team: info.Blue, robotId: 7}
	target := state.coverTarget(ballPos)
	want := info.Position{
		X: ballPos.X + (dangerousPos.X-ballPos.X)*coverReceiverWeight,
		Y: ballPos.Y + (dangerousPos.Y-ballPos.Y)*coverReceiverWeight,
	}
	want.Angle = want.AngleToPosition(ballPos)

	if math.Abs(target.X-want.X) > 1e-9 || math.Abs(target.Y-want.Y) > 1e-9 {
		t.Fatalf("cover target without possessor = %+v, want %+v", target, want)
	}
	if math.Abs(info.NormalizeAngleDelta(target.Angle, want.Angle)) > 1e-9 {
		t.Fatalf("cover angle without possessor = %.6f, want %.6f", target.Angle, want.Angle)
	}
}

func TestAttackerNearActivatesCoverState(t *testing.T) {
	dr := NewDefenseRole(7, ai.ActivityHandler{}, info.NewGameInfo(int(info.TEAM_SIZE)), info.Blue)
	dr.Init()
	dr.SetRoleKind(DefenseRoleHigh)

	dr.TriggerEvent("ATTACKER_NEAR")
	if got, want := string(dr.sm.CurrentStateName()), "DefenseCover ID 7"; got != want {
		t.Fatalf("state after ATTACKER_NEAR = %q, want %q", got, want)
	}

	dr.TriggerEvent("ATTACKER_FAR")
	if got, want := string(dr.sm.CurrentStateName()), "DefensePress ID 7"; got != want {
		t.Fatalf("state after ATTACKER_FAR = %q, want %q", got, want)
	}
	if got := dr.pressState.slot.Kind; got != DefenseRoleHigh {
		t.Fatalf("assigned defense kind after cover = %q, want %q", got, DefenseRoleHigh)
	}
}
