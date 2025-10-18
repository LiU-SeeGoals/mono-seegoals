package ai

import (
	"fmt"
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type RefStop struct {
	GenericComposition
	keepDistance float64
}

func (m *RefStop) String() string {
	return fmt.Sprintf("RefStop(%d)", m.id)
}

func NewRefStop(team info.Team, id info.ID) *RefStop {
	return &RefStop{
		GenericComposition: GenericComposition{
			id:   id,
			team: team,
		},
		keepDistance: 700,
	}

}

func (m *RefStop) GetAction(gi *info.GameInfo) action.Action {
	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		Logger.Error("Ball position retrieval failed: %v", err)
		return NewStop(m.id).GetAction(gi)
	}

	robotPos, err := gi.State.GetRobot(m.id, m.team).GetPosition()

	if err != nil {
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return NewStop(m.id).GetAction(gi)
	}

	ballDist := robotPos.Distance(ballPos)
	act := NewStop(m.id).GetAction(gi)
	if ballDist < m.keepDistance {

		direction := ballPos.AngleToPosition(robotPos)
		targetPos := ballPos.OnRadius(m.keepDistance, direction)
		act = NewMoveToPosition(m.team, m.id, targetPos).GetAction(gi)
	}

	return act
}

func (m *RefStop) Achieved(gi *info.GameInfo) bool {
	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		Logger.Error("Ball position retrieval failed: %v", err)
	}

	robotPos, err := gi.State.GetRobot(m.id, m.team).GetPosition()

	if err != nil {
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
	}

	ballDist := robotPos.Distance(ballPos)

	return ballDist > m.keepDistance
}

func (m *RefStop) GetID() info.ID {
	return m.id
}
