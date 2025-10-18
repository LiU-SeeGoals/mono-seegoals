package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type RefKickoff struct {
	GenericComposition
	targetPos info.Position
}

func (m *RefKickoff) String() string {
	return fmt.Sprintf("RefKickoff(%d)", m.id)
}

func NewRefKickoff(id info.ID, team info.Team) *RefKickoff {
	return &RefKickoff{
		GenericComposition: GenericComposition{
			id:   id,
			team: team,
		},
	}

}

func (m *RefKickoff) GetAction(gi *info.GameInfo) action.Action {
	yMultiplier := float64(1)
	if m.id == 1 {
		yMultiplier = float64(-1)
	
	}

	targetPos := info.Position{X: -2000, Y: 750, Z: 0, Angle: 0}
	if m.team == info.Blue && gi.Status.GetBlueTeamOnPositiveHalf() || m.team == info.Yellow && !gi.Status.GetBlueTeamOnPositiveHalf() {
		// We have the positive half
		targetPos = info.Position{X: 2000, Y: 750, Z: 0, Angle: math.Pi}
	} 	
	m.targetPos = targetPos
	m.targetPos.Y = yMultiplier * m.targetPos.Y
	fmt.Println("RefKickoff: ", m.id, m.targetPos)

	return NewMoveToPosition(m.team, m.id, m.targetPos).GetAction(gi)
}

func (m *RefKickoff) Achieved(gi *info.GameInfo) bool {
	return gi.State.GetRobot(m.id, m.team).At(m.targetPos, 100)
}

func (m *RefKickoff) GetID() info.ID {
	return m.id
}
