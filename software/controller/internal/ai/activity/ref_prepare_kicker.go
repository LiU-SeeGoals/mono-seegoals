package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type PrepareKicker struct {
	GenericComposition
	done bool
}

func (m *PrepareKicker) String() string {
	return fmt.Sprintf("PrepareKicker(%d)", m.id)
}

func NewPrepareKicker(team info.Team, id info.ID) *PrepareKicker {
	return &PrepareKicker{
		GenericComposition: GenericComposition{
			id:   id,
			team: team,
		},
		done: false,
	}

}

func (m *PrepareKicker) GetAction(gi *info.GameInfo) action.Action {
	var act action.Action

	// Prepare kickoff, get kicker in position
	targetPos := info.Position{X: -300, Y: 0, Z: 0, Angle: 0} // Position for negative half

	// We have the positive half
	if m.team == info.Blue && gi.Status.GetBlueTeamOnPositiveHalf() || m.team == info.Yellow && !gi.Status.GetBlueTeamOnPositiveHalf() {
		targetPos = info.Position{X: 300, Y: 0, Z: 0, Angle: math.Pi}
	}


	move := NewMoveToPosition(m.team, m.id, targetPos)
	move.AvoidBall(true)
	act = move.GetAction(gi)

	if move.Achieved(gi) {
		m.done = true
	}

	return act
}

func (m *PrepareKicker) Achieved(gi *info.GameInfo) bool {
	return m.done
}

func (m *PrepareKicker) GetID() info.ID {
	return m.id

}
