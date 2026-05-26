package ai

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	dribblerBallControlDistance = 150.0
	centerBallControlDistance   = 180.0
	ballControlFacingTolerance  = 35.0 * math.Pi / 180.0
)

func robotHasBallControl(gi *info.GameInfo, team info.Team, id info.ID) bool {
	robot := gi.State.GetRobot(id, team)
	ball := gi.State.GetBall()

	if ball.GetPossessor() == robot {
		return true
	}

	ballPos, err := ball.GetPosition()
	if err != nil {
		return false
	}

	robotPos, err := robot.GetPosition()
	if err != nil {
		return false
	}

	dribblerPos := robotPos
	dribblerPos.X += 90 * math.Cos(robotPos.Angle)
	dribblerPos.Y += 90 * math.Sin(robotPos.Angle)
	if ballPos.Dist2d(dribblerPos) < dribblerBallControlDistance {
		return true
	}

	return ballPos.Dist2d(robotPos) < centerBallControlDistance &&
		robot.Facing(ballPos, ballControlFacingTolerance)
}
