package ai

import (
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
		3: tacticalSlotDefenderWall,
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
	gi := info.NewGameInfo(int(info.TEAM_SIZE))

	if goalie, found := plan.chooseGoalie(gi, []info.ID{1, 2, 3, 4, 5, 6}); !found || goalie != 6 {
		t.Fatalf("goalie = %d, found = %t; want robot 6", goalie, found)
	}
	if goalie, found := plan.chooseGoalie(gi, []info.ID{1, 2, 3, 4, 5}); found {
		t.Fatalf("replacement goalie = %d; want no goalie when robot 6 is missing", goalie)
	}
}

func TestLowDefenderAssignmentTargetUsesBallTrajectory(t *testing.T) {
	plan := NewCombinedPlan(info.Blue)
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBall(0, 0, 0, 1000)
	gi.State.SetBall(-1000, 200, 0, 1020)

	target := plan.targetForTacticalSlot(gi, tacticalSlotDefenderLow)
	if target.X != -3000 || target.Y != 600 {
		t.Fatalf("Low Def assignment target = %+v, want X=-3000 Y=600", target)
	}
}

func TestClosestReadyDefenderWinsLooseBallAssignment(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
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
