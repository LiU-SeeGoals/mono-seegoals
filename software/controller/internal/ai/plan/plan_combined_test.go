package ai

import "testing"

func TestDesiredOffenseCountKeepsTwoAttackersForTwoRobotAttack(t *testing.T) {
	got := desiredOffenseCount(2, tacticalModeAttack, attackModeAttackerRatio)
	if got != 2 {
		t.Fatalf("desiredOffenseCount(2, attack) = %d, want 2", got)
	}
}

func TestDesiredOffenseCountKeepsOneAttackerForTwoRobotDefense(t *testing.T) {
	got := desiredOffenseCount(2, tacticalModeDefend, defendModeAttackerRatio)
	if got != 1 {
		t.Fatalf("desiredOffenseCount(2, defend) = %d, want 1", got)
	}
}
