package roles

import (
	"math"
	"sync"
	"testing"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
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
	opponent, position, _, found := state.closestOpponentToOwnGoal()

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

	if _, _, _, found := state.closestOpponentToOwnGoal(); found {
		t.Fatal("found an opponent without game state")
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
	want := state.coverParkingPos(ballPos)
	if got := state.coverTarget(ballPos); got != want {
		t.Fatalf("cover target = %+v, want parking position %+v", got, want)
	}
}

func TestCoverTargetParksWithoutOpponentNearBall(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	state := &DefensePressState{gi: gi, team: info.Blue, robotId: 7}
	ballPos := info.Position{X: -500, Y: 725}

	want := state.coverParkingPos(ballPos)
	if got := state.coverTarget(ballPos); got != want {
		t.Fatalf("cover target = %+v, want parking position %+v", got, want)
	}
	if got := want.X; got != -3000 {
		t.Fatalf("Blue parking X = %.0f, want -3000", got)
	}
	if got := want.Y; got != ballPos.Y {
		t.Fatalf("parking Y = %.0f, want ball Y %.0f", got, ballPos.Y)
	}
}

func TestCoverParkingPositionUsesTeamHalf(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	ballPos := info.Position{X: 250, Y: -640}
	blueState := &DefensePressState{gi: gi, team: info.Blue}
	yellowState := &DefensePressState{gi: gi, team: info.Yellow}

	if got := blueState.coverParkingPos(ballPos); got.X != -3000 || got.Y != ballPos.Y {
		t.Fatalf("Blue parking position = %+v, want X=-3000 and Y=%.0f", got, ballPos.Y)
	}
	if got := yellowState.coverParkingPos(ballPos); got.X != 3000 || got.Y != ballPos.Y {
		t.Fatalf("Yellow parking position = %+v, want X=3000 and Y=%.0f", got, ballPos.Y)
	}
}

func TestLowDefenderTargetsProjectedBallPath(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBall(0, 0, 0, 1000)
	gi.State.SetBall(-1000, 200, 0, 1020)
	ballPos := info.Position{X: -1000, Y: 200}

	state := &DefensePressState{gi: gi, team: info.Blue}
	target := LowDefenderTarget(state.gi, state.team, ballPos, 0)
	if target.X != -3000 || target.Y != 600 {
		t.Fatalf("Blue Low Def target = %+v, want X=-3000 Y=600", target)
	}
}

func TestLowDefenderUsesPositiveHalfForYellow(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBall(0, 0, 0, 1000)
	gi.State.SetBall(1000, 200, 0, 1020)
	ballPos := info.Position{X: 1000, Y: 200}

	state := &DefensePressState{gi: gi, team: info.Yellow}
	target := LowDefenderTarget(state.gi, state.team, ballPos, 0)
	if target.X != 3000 || target.Y != 600 {
		t.Fatalf("Yellow Low Def target = %+v, want X=3000 Y=600", target)
	}
}

func TestLowDefenderRetreatsPastMinimumDepth(t *testing.T) {
	tests := []struct {
		name  string
		team  info.Team
		ballX float64
		wantX float64
	}{
		{name: "blue", team: info.Blue, ballX: -3600, wantX: -3600},
		{name: "yellow", team: info.Yellow, ballX: 3800, wantX: 3800},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gi := info.NewGameInfo(int(info.TEAM_SIZE))
			ballPos := info.Position{X: test.ballX, Y: 250}
			gi.State.GetBall().SetEstimatedPosition(ballPos)

			target := LowDefenderTarget(gi, test.team, ballPos, 0)
			if target.X != test.wantX {
				t.Fatalf("Low Def target X = %.0f, want %.0f", target.X, test.wantX)
			}
		})
	}
}

func TestLowDefenderClampsProjectedBallPath(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBall(0, 0, 0, 1000)
	gi.State.SetBall(-1000, 1000, 0, 1020)
	ballPos := info.Position{X: -1000, Y: 1000}

	state := &DefensePressState{gi: gi, team: info.Blue}
	target := LowDefenderTarget(state.gi, state.team, ballPos, 0)
	if target.X != -3000 || target.Y != coverYClamp {
		t.Fatalf("clamped Low Def target = %+v, want X=-3000 Y=%.0f", target, coverYClamp)
	}
}

func TestLowDefenderPredictsWhereStationaryShooterAims(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponent := gi.State.GetOtherTeam(info.Blue)[2]
	opponentPos := info.Position{X: 500, Y: 200}
	wantedHit := info.Position{X: -3000, Y: 900}
	opponentPos.Angle = opponentPos.AngleToPosition(wantedHit)
	opponent.SetPositionTime(opponentPos.X, opponentPos.Y, opponentPos.Angle, 1)

	// Put the stationary ball at the opponent's dribbler. There is deliberately
	// no ball movement history, so the target must come from the shooter's aim.
	ballPos := opponent.DribblerPos()
	gi.State.GetBall().SetEstimatedPosition(ballPos)

	state := &DefensePressState{gi: gi, team: info.Blue}
	target := LowDefenderTarget(state.gi, state.team, ballPos, 0)
	if math.Abs(target.X-wantedHit.X) > 1e-9 || math.Abs(target.Y-wantedHit.Y) > 1e-9 {
		t.Fatalf("Low Def target from shooter aim = %+v, want %+v", target, wantedHit)
	}
}

func TestLowDefenderIgnoresShooterAimingAwayFromDefenseLine(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponent := gi.State.GetOtherTeam(info.Blue)[2]
	opponent.SetPositionTime(0, 200, 0, 1)
	ballPos := opponent.DribblerPos()
	gi.State.GetBall().SetEstimatedPosition(ballPos)

	state := &DefensePressState{gi: gi, team: info.Blue}
	target := LowDefenderTarget(state.gi, state.team, ballPos, 0)
	if target.X != -3000 || target.Y != ballPos.Y {
		t.Fatalf("Low Def fallback target = %+v, want X=-3000 Y=%.3f", target, ballPos.Y)
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

func TestCoverTargetUsesAssignedOpponentInsteadOfClosestToGoal(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponents := gi.State.GetOtherTeam(info.Blue)
	ballPos := info.Position{X: 0, Y: 0}
	opponents[1].SetPositionTime(-3900, -900, 0, 1)
	assignedPos := info.Position{X: -3400, Y: 900}
	opponents[2].SetPositionTime(assignedPos.X, assignedPos.Y, 0, 1)

	state := &DefensePressState{
		gi:   gi,
		team: info.Blue,
		coverOpponent: &coverOpponentSelection{
			id:       2,
			assigned: true,
			managed:  true,
		},
	}
	target := state.coverTarget(ballPos)
	want := info.Position{
		X: ballPos.X + (assignedPos.X-ballPos.X)*coverReceiverWeight,
		Y: ballPos.Y + (assignedPos.Y-ballPos.Y)*coverReceiverWeight,
	}
	want.Angle = want.AngleToPosition(ballPos)

	if math.Abs(target.X-want.X) > 1e-9 || math.Abs(target.Y-want.Y) > 1e-9 {
		t.Fatalf("assigned cover target = %+v, want %+v", target, want)
	}
}

func TestCoverTargetParksWhenDangerousOpponentIsNearBall(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	opponents := gi.State.GetOtherTeam(info.Blue)
	ballPos := info.Position{X: -3500, Y: 0}

	opponents[1].SetPositionTime(
		ballPos.X+CoverDangerousBallDistance,
		ballPos.Y,
		0,
		1,
	)

	state := &DefensePressState{gi: gi, team: info.Blue, robotId: 7}
	want := state.coverParkingPos(ballPos)
	if got := state.coverTarget(ballPos); got != want {
		t.Fatalf("cover target = %+v, want parking position %+v", got, want)
	}
}

func TestCoverRoleCanKickNearbySlowBall(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, 0, 0, 0, 1)
	ballPos := info.Position{X: defenseKickBallDistance - 1, Y: 0}
	gi.State.GetBall().SetEstimatedPosition(ballPos)
	gi.State.SetTrackedBall(ballPos, info.Position{X: defenseKickMaxBallSpeed}, 1)

	dr := NewDefenseRole(3, ai.ActivityHandler{}, gi, info.Blue)
	dr.Init()
	dr.SetRoleKind(DefenseRoleCover)
	dr.SetMayPlayBall(true)

	if !dr.defenderBallReadyToKick() {
		t.Fatal("expected Cover to prepare a kick for a nearby slow ball")
	}
	if got := dr.kickIntent.GetTargetPosition(); got != defenseKickTarget {
		t.Fatalf("kick target = %+v, want %+v", got, defenseKickTarget)
	}
}

func TestCoverRoleStartsAligningBeforeKick(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, 0, 0, 0, 1)
	ballPos := info.Position{X: defenseKickBallDistance - 1}
	gi.State.GetBall().SetEstimatedPosition(ballPos)
	gi.State.SetTrackedBall(ballPos, info.Position{}, 1)

	activities := [info.TEAM_SIZE]act.Activity{}
	handler := ai.ActivityHandler{
		Activities:    &activities,
		Activity_lock: &sync.Mutex{},
	}
	dr := NewDefenseRole(3, handler, gi, info.Blue)
	dr.Init()
	dr.SetRoleKind(DefenseRoleCover)
	dr.SetMayPlayBall(true)
	dr.Run()

	if got, want := string(dr.sm.CurrentStateName()), "DefenseAlignKick ID 3"; got != want {
		t.Fatalf("Cover state = %q, want %q", got, want)
	}
	if _, ok := activities[3].(*act.AlignBall); !ok {
		t.Fatalf("Cover activity = %T, want *activity.AlignBall", activities[3])
	}
}

func TestLowDefenseStartsAligningBeforeKick(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(5, 0, 0, 0, 1)
	ballPos := info.Position{X: defenseKickBallDistance - 1}
	gi.State.GetBall().SetEstimatedPosition(ballPos)
	gi.State.SetTrackedBall(ballPos, info.Position{}, 1)

	activities := [info.TEAM_SIZE]act.Activity{}
	handler := ai.ActivityHandler{
		Activities:    &activities,
		Activity_lock: &sync.Mutex{},
	}
	dr := NewDefenseRole(5, handler, gi, info.Blue)
	dr.Init()
	dr.SetRoleKind(DefenseRoleLow)
	dr.SetMayPlayBall(true)
	dr.Run()

	if got, want := string(dr.sm.CurrentStateName()), "DefenseAlignKick ID 5"; got != want {
		t.Fatalf("Low Def state = %q, want %q", got, want)
	}
	if _, ok := activities[5].(*act.AlignBall); !ok {
		t.Fatalf("Low Def activity = %T, want *activity.AlignBall", activities[5])
	}
}

func TestCoverRoleDoesNotKickFarOrFastBall(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, 0, 0, 0, 1)
	dr := NewDefenseRole(3, ai.ActivityHandler{}, gi, info.Blue)
	dr.Init()
	dr.SetRoleKind(DefenseRoleCover)
	dr.SetMayPlayBall(true)

	farBall := info.Position{X: defenseKickBallDistance + 1}
	gi.State.GetBall().SetEstimatedPosition(farBall)
	gi.State.SetTrackedBall(farBall, info.Position{}, 1)
	if dr.defenderBallReadyToKick() {
		t.Fatal("Cover prepared a kick while the ball was too far away")
	}

	nearBall := info.Position{X: defenseKickBallDistance - 1}
	gi.State.GetBall().SetEstimatedPosition(nearBall)
	gi.State.SetTrackedBall(nearBall, info.Position{X: defenseKickMaxBallSpeed + 0.01}, 2)
	if dr.defenderBallReadyToKick() {
		t.Fatal("Cover prepared a kick while the ball was moving too quickly")
	}
}

func TestCoverRoleCannotKickWithoutBallAssignment(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(3, 0, 0, 0, 1)
	ballPos := info.Position{X: defenseKickBallDistance - 1}
	gi.State.GetBall().SetEstimatedPosition(ballPos)
	gi.State.SetTrackedBall(ballPos, info.Position{}, 1)

	dr := NewDefenseRole(3, ai.ActivityHandler{}, gi, info.Blue)
	dr.Init()
	dr.SetRoleKind(DefenseRoleCover)

	if !dr.CanKickBall() {
		t.Fatal("expected Cover to be physically able to kick the ball")
	}
	if dr.defenderBallReadyToKick() {
		t.Fatal("Cover prepared a kick without being selected as ball actor")
	}

	dr.SetMayPlayBall(true)
	if !dr.defenderBallReadyToKick() {
		t.Fatal("selected Cover did not prepare a kick")
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
