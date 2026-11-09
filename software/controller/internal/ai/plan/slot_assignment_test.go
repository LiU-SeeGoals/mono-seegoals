package ai

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
