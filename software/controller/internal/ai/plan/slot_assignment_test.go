package ai

import (
	"math"
	"sync"
	"testing"
	"time"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

func TestSlotSwapPreservesFormationDuringCooldown(t *testing.T) {
	now := time.Now()
	rm := &combinedRoleManager{
		slotByRobot: map[info.ID]tacticalSlotKind{1: tacticalSlotBallChaser, 2: tacticalSlotSupportShooter},
		lastChanged: map[info.ID]time.Time{1: now, 2: now.Add(-time.Second)},
	}
	desired := map[info.ID]tacticalSlotKind{1: tacticalSlotSupportShooter, 2: tacticalSlotBallChaser}
	got := rm.reserveRetainedSlots(desired, now)
	if got[1] != tacticalSlotBallChaser || got[2] != tacticalSlotSupportShooter {
		t.Fatalf("cooldown broke formation: %v", got)
	}
	got = rm.reserveRetainedSlots(desired, now.Add(roleSwitchMinDuration))
	if got[1] != desired[1] || got[2] != desired[2] {
		t.Fatalf("swap did not complete after cooldown: %v", got)
	}
}

func TestRemovedSlotDoesNotSurviveCooldown(t *testing.T) {
	now := time.Now()
	rm := &combinedRoleManager{
		slotByRobot: map[info.ID]tacticalSlotKind{1: tacticalSlotDefenderWall, 2: tacticalSlotBallChaser},
		lastChanged: map[info.ID]time.Time{1: now, 2: now},
	}
	got := rm.reserveRetainedSlots(map[info.ID]tacticalSlotKind{1: tacticalSlotSupportShooter, 2: tacticalSlotBallChaser}, now)
	if got[1] != tacticalSlotSupportShooter || got[2] != tacticalSlotBallChaser {
		t.Fatalf("obsolete slot retained: %v", got)
	}
}

func TestStaticPlanKeepsConfiguredRobotSlots(t *testing.T) {
	plan := NewStaticPlan(info.Blue)
	got := plan.configuredStaticSlots([]info.ID{1, 2, 3, 4, 5})
	want := map[info.ID]tacticalSlotKind{
		1: tacticalSlotBallChaser,
		2: tacticalSlotSupportShooter,
		3: tacticalSlotDefenderHigh,
		4: tacticalSlotSupportShooter,
		5: tacticalSlotDefenderLow,
	}

	if len(got) != len(want) {
		t.Fatalf("static assignments = %v, want %v", got, want)
	}
	for id, slot := range want {
		if got[id] != slot {
			t.Fatalf("robot %d static slot = %q, want %q", id, got[id], slot)
		}
	}
}

func TestStaticPlanDoesNotReplaceMissingRobot(t *testing.T) {
	plan := NewStaticPlan(info.Blue)
	got := plan.configuredStaticSlots([]info.ID{1, 2, 4, 5})

	if _, assigned := got[3]; assigned {
		t.Fatal("missing robot 3 unexpectedly received an assignment")
	}
	if len(got) != 4 {
		t.Fatalf("static assignments = %v, want four assignments", got)
	}
}

func TestStaticPlanOnlyUsesRobotSixAsGoalie(t *testing.T) {
	plan := NewStaticPlan(info.Blue)
	gi := newDefenseGeometryGameInfo()

	if goalie, found := plan.chooseGoalie(gi, []info.ID{1, 2, 3, 4, 5, 6}); !found || goalie != 6 {
		t.Fatalf("goalie = %d, found = %t; want robot 6", goalie, found)
	}
	if goalie, found := plan.chooseGoalie(gi, []info.ID{1, 2, 3, 4, 5}); found {
		t.Fatalf("replacement goalie = %d; want no goalie when robot 6 is missing", goalie)
	}
}

func TestLowDefenderAssignmentTargetUsesBallTrajectory(t *testing.T) {
	plan := NewCombinedPlan(info.Blue)
	gi := newDefenseGeometryGameInfo()
	gi.State.SetBall(0, 0, 0, 1000)
	gi.State.SetBall(-1000, 200, 0, 1020)

	target := plan.targetForTacticalSlot(gi, tacticalSlotDefenderLow)
	if target.X != -3000 || target.Y != 600 {
		t.Fatalf("Low Def assignment target = %+v, want X=-3000 Y=600", target)
	}
}

func TestDefensePositioningUsesWallForBallAndHighForReceiver(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	gi.State.SetBlueRobot(3, -3000, -300, 0, 1)
	gi.State.SetBlueRobot(4, -2500, 500, 0, 1)
	ballPos := info.Position{X: -500, Y: 200}
	gi.State.GetBall().SetEstimatedPosition(ballPos)
	receiverPos := info.Position{X: -3300, Y: 900}
	gi.State.GetOtherTeam(info.Blue)[2].SetPositionTime(receiverPos.X, receiverPos.Y, 0, 1)

	activities := [info.TEAM_SIZE]act.Activity{}
	handler := coreai.ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
	wall := roles.NewDefenseRole(3, handler, gi, info.Blue)
	wall.Init()
	high := roles.NewDefenseRole(4, handler, gi, info.Blue)
	high.Init()

	roleManager := newCombinedRoleManager(&handler, gi, info.Blue)
	roleManager.defenders[3] = wall
	roleManager.defenders[4] = high
	roleManager.setSlot(3, tacticalSlotDefenderWall, time.Now())
	roleManager.setSlot(4, tacticalSlotDefenderHigh, time.Now())

	plan := NewCombinedPlan(info.Blue)
	plan.updateDefenseAndGoaliePositioning(gi, roleManager, 0, nil, false)

	wallMove, ok := activities[3].(*act.MoveToPosition)
	if !ok {
		t.Fatalf("Wall activity = %T, want *activity.MoveToPosition", activities[3])
	}
	wantWall := plan.calcGoalWallPositionsForRobots(gi, ballPos, 1)[0]
	gotWall := wallMove.GetMoveToAction(gi).Dest
	if gotWall.X != wantWall.X || gotWall.Y != wantWall.Y {
		t.Fatalf("Wall destination = %+v, want ball-wall position %+v", gotWall, wantWall)
	}

	highMove, ok := activities[4].(*act.MoveToPosition)
	if !ok {
		t.Fatalf("High Def activity = %T, want *activity.MoveToPosition", activities[4])
	}
	gotHigh := highMove.GetMoveToAction(gi).Dest
	wantHigh := info.Position{
		X: ballPos.X + (receiverPos.X-ballPos.X)*0.8,
		Y: ballPos.Y + (receiverPos.Y-ballPos.Y)*0.8,
	}
	if math.Abs(gotHigh.X-wantHigh.X) > 1e-9 || math.Abs(gotHigh.Y-wantHigh.Y) > 1e-9 {
		t.Fatalf("High Def destination = %+v, want receiver pass block %+v", gotHigh, wantHigh)
	}
}

func TestWallPositionUsesNonStandardVisionField(t *testing.T) {
	gi := newDefenseGeometryGameInfoWithSize(12000, 8000, 1200, 1500, 3000)
	plan := NewCombinedPlan(info.Blue)
	positions := plan.calcGoalWallPositionsForRobots(
		gi,
		info.Position{X: -4500, Y: 3500},
		1,
	)
	if len(positions) != 1 {
		t.Fatalf("wall positions = %v, want one", positions)
	}
	if got, want := positions[0].X, -4800.0; got != want {
		t.Fatalf("wall X = %.0f, want %.0f from Vision goal line", got, want)
	}
	if got, want := positions[0].Y, -2100.0; math.Abs(got) != math.Abs(want) {
		t.Fatalf("wall |Y| = %.0f, want %.0f from Vision penalty width", math.Abs(got), math.Abs(want))
	}
}

func TestClosestReadyDefenderWinsLooseBallAssignment(t *testing.T) {
	gi := newDefenseGeometryGameInfo()
	gi.State.SetBlueRobot(3, 400, 0, 0, 1)
	gi.State.SetBlueRobot(5, 100, 0, 0, 1)
	ballPos := info.Position{}
	gi.State.GetBall().SetEstimatedPosition(ballPos)
	gi.State.SetTrackedBall(ballPos, info.Position{}, 1)

	activities := [info.TEAM_SIZE]act.Activity{}
	handler := coreai.ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
	wall := roles.NewDefenseRole(3, handler, gi, info.Blue)
	wall.Init()
	wall.SetRoleKind(roles.DefenseRoleCover)
	low := roles.NewDefenseRole(5, handler, gi, info.Blue)
	low.Init()
	low.SetRoleKind(roles.DefenseRoleLow)
	defenders := map[info.ID]*roles.DefenseRole{3: wall, 5: low}

	got, found := closestReadyDefenderToBall(gi, info.Blue, defenders)
	if !found || got != 5 {
		t.Fatalf("loose-ball defender = %d, found = %t; want robot 5", got, found)
	}

	setDefenderBallActor(defenders, got, true)
	wall.Run()
	low.Run()
	if _, ok := activities[3].(*act.AlignBall); ok {
		t.Fatal("unselected Wall unexpectedly started aligning with the ball")
	}
	if _, ok := activities[5].(*act.AlignBall); !ok {
		t.Fatalf("selected Low Def activity = %T, want *activity.AlignBall", activities[5])
	}
}
