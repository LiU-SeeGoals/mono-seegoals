package ai

import "github.com/LiU-SeeGoals/controller/internal/info"

// StaticPlan uses the CombinedPlan game logic while keeping every robot in a
// fixed tactical slot. Missing robots leave their slot empty; another robot is
// never reassigned to replace them.
type StaticPlan struct {
	*CombinedPlan
}

func NewStaticPlan(team info.Team) *StaticPlan {
	return &StaticPlan{
		CombinedPlan: &CombinedPlan{
			plannerCore: plannerCore{team: team},
			staticSlots: map[info.ID]tacticalSlotKind{
				1: tacticalSlotBallChaser,
				2: tacticalSlotSupportShooter,
				3: tacticalSlotSupportShooter,
				4: tacticalSlotDefenderLow,
				5: tacticalSlotDefenderWall,
			},
		},
	}
}
