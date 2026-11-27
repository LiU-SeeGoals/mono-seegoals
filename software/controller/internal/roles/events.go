package roles

import "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"

// Events understood by the role state machines (offense, defense, goalie) and
// by the restart kick sub-machines in the referee package. Planners trigger
// them every frame; a role that has no transition for an event ignores it.
const (
	EventBallOwner       state_machine.EventName = "BALL_OWNER"
	EventBallLost        state_machine.EventName = "BALL_LOST"
	EventBallApproaching state_machine.EventName = "BALL_APPROACHING"
	EventBallReceived    state_machine.EventName = "BALL_RECEIVED"
	EventPassTargeted    state_machine.EventName = "PASS_TARGETED"
	EventAligned         state_machine.EventName = "ALIGNED"
	EventMisaligned      state_machine.EventName = "MISALIGNED"
	EventKicked          state_machine.EventName = "KICKED"
	EventDeadBallTrapped state_machine.EventName = "DEAD_BALL_TRAPPED"
	EventAttackerNear    state_machine.EventName = "ATTACKER_NEAR"
	EventAttackerFar     state_machine.EventName = "ATTACKER_FAR"
)
