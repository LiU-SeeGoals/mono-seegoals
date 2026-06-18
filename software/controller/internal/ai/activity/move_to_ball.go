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
	captureTarget := info.Position{
		X: ballPos.X + 1000*math.Cos(angleToBall),
		Y: ballPos.Y + 1000*math.Sin(angleToBall),
	}
	approachReady := captureApproachReady(robotPos, ballPos, captureTarget, headingErr)
	captureReady := capturePoseReady(robotPos, ballPos, captureTarget, headingErr)

	// Target the standoff point in front of the ball on our side, never the
	// ball itself; the margin only closes once the kicker points at the ball.
	margin := alignmentMargin(headingErr)
	if !approachReady {
		margin = math.Max(margin, captureMarginToBall)
	}
	dist := kickerStandoffDist(margin)
	target := info.Position{
		X:     ballPos.X - dist*math.Cos(angleToBall),
		Y:     ballPos.Y - dist*math.Sin(angleToBall),
		Angle: angleToBall,
	}

	dribble := false
	dribblerPos := robot.DribblerPos()
	_, lateral, ok := robot.BallLocalOffset(ballPos)
	ballCentered := ok && math.Abs(lateral) < info.KickCenterTolerance
	if dribblerPos.Dist2d(ballPos) < 120 && headingErr < 2*roughAngleTolerance && approachReady {
		dribble = true
	}

	move := NewMoveToPosition(m.team, m.id, target)
	move.AvoidBall(false)
	moveAction := move.GetMoveToAction(gi)
	moveAction.Dest.Angle = angleToBall
	printCaptureDebug(
		"move-to-ball",
		m.team,
		m.id,
		robot,
		robotPos,
		ballPos,
		captureTarget,
		moveAction.Dest,
		headingErr,
		captureReady,
		ballCentered,
		dribble,
		0,
		margin,
	)
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
