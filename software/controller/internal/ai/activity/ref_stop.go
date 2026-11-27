package activity

import (
	"fmt"
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/controller/internal/rules"
)

type RefStop struct {
	GenericComposition
	keepDistance float64
	move         *MoveToPosition // inner move of the last frame
}

// ContinueFrom keeps the previous frame's inner move.
func (m *RefStop) ContinueFrom(previous Activity) {
	if prev, ok := previous.(*RefStop); ok && prev != nil && prev != m && prev.team == m.team && prev.id == m.id {
		m.move = prev.move
	}
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
		keepDistance: rules.StopBallKeepout,
	}

}

func (m *RefStop) GetAction(gi *info.GameInfo) action.Action {
	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		logger.Logger.Error("Ball position retrieval failed: %v", err)
		return NewStop(m.id).GetAction(gi)
	}

	robotPos, err := gi.State.GetRobot(m.id, m.team).GetPosition()

	if err != nil {
		logger.Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return NewStop(m.id).GetAction(gi)
	}

	// Clear the announced destination before the simulator places the ball.
	if ge := gi.Status.GetGameEvent(); rules.For(gi, m.team).Stopped && ge.GetDesignatedPosition() != nil {
		p := ge.GetDesignatedPosition()
		target := info.Position{X: p.AtVec(0), Y: p.AtVec(1)}
		if robotPos.Dist2d(target) < robotPos.Dist2d(ballPos) {
			ballPos = target
		}
	}
	ballDist := robotPos.Distance(ballPos)
	act := NewStop(m.id).GetAction(gi)
	if ballDist < m.keepDistance {

		direction := ballPos.AngleToPosition(robotPos)
		targetPos := ballPos.OnRadius(m.keepDistance, direction)
		m.move = continueMove(m.move, m.team, m.id, targetPos)
		act = m.move.GetAction(gi)
	}

	return act
}

func (m *RefStop) Achieved(gi *info.GameInfo) bool {
	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		logger.Logger.Error("Ball position retrieval failed: %v", err)
	}

	robotPos, err := gi.State.GetRobot(m.id, m.team).GetPosition()

	if err != nil {
		logger.Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
	}

	ballDist := robotPos.Distance(ballPos)

	return ballDist > m.keepDistance
}

func (m *RefStop) GetID() info.ID {
	return m.id
}
