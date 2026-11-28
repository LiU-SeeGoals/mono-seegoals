package plan

import (
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
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

func TestRoleAssignmentFillsLargeTeams(t *testing.T) {
	for _, mode := range []tacticalMode{tacticalModeAttack, tacticalModeDefend} {
		for total := 0; total <= 10; total++ {
			if got := len(roleAssignmentForMode(total, mode).slots); got != total {
				t.Fatalf("%s with %d robots: %d slots", mode, total, got)
			}
		}
	}
	extra := roleAssignmentForMode(7, tacticalModeDefend).slots[5:]
	if extra[0] != tacticalSlotDefenderWall || extra[1] != tacticalSlotDefenderWall {
		t.Fatalf("defend extras should join the wall: %v", extra)
	}
}

func TestGoalieClearTargetPointsAwayFromOwnGoal(t *testing.T) {
	gi := info.NewGameInfo(10)
	for _, team := range []info.Team{info.Blue, info.Yellow} {
		target := NewCombinedPlan(team).goalieClearTarget(gi)
		if target.X*gi.OwnHalfXSign(team) >= 0 {
			t.Fatalf("team %v: clear target %v is in own half", team, target)
		}
	}
}
