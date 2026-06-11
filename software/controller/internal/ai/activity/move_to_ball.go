package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type MoveToBall struct {
	GenericComposition
	// MovementComposition
}

func (m *MoveToBall) String() string {
	return fmt.Sprintf("(Robot %d, MoveToBall()", m.id)
}

func NewMoveToBall(team info.Team, id info.ID) *MoveToBall {
	return &MoveToBall{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
	}
}

func (m *MoveToBall) GetAction(gi *info.GameInfo) action.Action {
	robot := gi.State.GetRobot(m.id, m.team)
	robotPos, err := robot.GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return NewStop(m.id).GetAction(gi)
	}

	ballPos := predictedBallPos(gi, ballLookaheadSec)
	angleToBall := robotPos.AngleToPosition(ballPos)
	headingErr := math.Abs(info.NormalizeAngleDelta(angleToBall, robotPos.Angle))

	// Target the standoff point in front of the ball on our side, never the
	// ball itself; the margin only closes once the kicker points at the ball.
	dist := kickerStandoffDist(alignmentMargin(headingErr))
	target := info.Position{
		X:     ballPos.X - dist*math.Cos(angleToBall),
		Y:     ballPos.Y - dist*math.Sin(angleToBall),
		Angle: angleToBall,
	}

	dribble := false
	dribblerPos := robot.DribblerPos()
	if dribblerPos.Dist2d(ballPos) < 120 && headingErr < 2*roughAngleTolerance {
		dribble = true
	}

	move := NewMoveToPosition(m.team, m.id, target)
	move.AvoidBall(false)
	moveAction := move.GetMoveToAction(gi)
	moveAction.Dest.Angle = angleToBall
	act := action.MoveTo{
		Id:   int(m.id),
		Team: m.team,
		Pos:  robotPos,
		Dest: moveAction.Dest,

		Dribble: dribble,
		// Visualization only: keep the planned path without changing motion behavior.
		Path: moveAction.Path,
	}

	return &act
}

func (m *MoveToBall) Achieved(gi *info.GameInfo) bool {
	return gi.State.GetBall().GetPossessor() == gi.State.GetRobot(m.id, m.team)
}

func (m *MoveToBall) GetID() info.ID {
	return m.id
}
