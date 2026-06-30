package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const goalieDeadBallStandoff = 180.0

type GoalieCollectDeadBall struct {
	GenericComposition
	team info.Team
	id   info.ID
}

func NewGoalieCollectDeadBall(team info.Team, id info.ID) *GoalieCollectDeadBall {
	return &GoalieCollectDeadBall{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		team: team,
		id:   id,
	}
}

func (g *GoalieCollectDeadBall) String() string {
	return fmt.Sprintf("GoalieCollectDeadBall(%d, %d)", g.team, g.id)
}

func (g *GoalieCollectDeadBall) GetAction(gi *info.GameInfo) action.Action {
	robot := gi.State.GetRobot(g.id, g.team)
	robotPos, err := robot.GetPosition()
	if err != nil {
		return NewStop(g.id).GetAction(gi)
	}

	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return NewStop(g.id).GetAction(gi)
	}

	xSign := goalieDefenseXSign(gi, g.team)
	target := info.Position{
		X: ballPos.X + xSign*goalieDeadBallStandoff,
		Y: ballPos.Y,
		Z: 0,
	}
	target.Angle = target.AngleToPosition(ballPos)

	dribblerPos := robot.DribblerPos()
	headingErr := math.Abs(info.NormalizeAngleDelta(robotPos.AngleToPosition(ballPos), robotPos.Angle))

	return &action.MoveTo{
		Id:                  int(g.id),
		Team:                g.team,
		Pos:                 robotPos,
		Dest:                target,
		AllowOutsideField:   true,
		AllowBehindGoalLine: true,
		AllowGoalArea:       true,
		Dribble:             dribblerPos.Dist2d(ballPos) < 160 && headingErr < 0.35,
	}
}

func (g *GoalieCollectDeadBall) Achieved(gi *info.GameInfo) bool {
	robot := gi.State.GetRobot(g.id, g.team)
	if robot == nil {
		return false
	}
	return gi.State.GetBall().GetPossessor() == robot
}

func (g *GoalieCollectDeadBall) GetID() info.ID {
	return g.id
}

func goalieDefenseXSign(gi *info.GameInfo, team info.Team) float64 {
	isBlueTeam := team == info.Blue
	isBlueOnPositiveHalf := gi.Status.GetBlueTeamOnPositiveHalf()
	if (isBlueTeam && isBlueOnPositiveHalf) || (!isBlueTeam && !isBlueOnPositiveHalf) {
		return 1.0
	}
	return -1.0
}
