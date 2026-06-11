package ai

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/info"
)


const (

	center2DribblerDist = 90.0
	ballRadius          = 21.5
	maxMarginToBall = 70.0
	ballPushMargin = -20.0
	aroundBallShiftAngle = 0.7
	maxTargetOrientationStep = 0.4
	roughAngleTolerance = 0.1
	ballLookaheadSec = 0.3
	minRollingBallSpeed = 0.3
)


func predictedBallPos(gi *info.GameInfo, lookaheadSec float64) info.Position {
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()
	if !ok || ballVel.Norm2d() < minRollingBallSpeed {
		return ballPos
	}
	ballPos.X += ballVel.X * 1000 * lookaheadSec
	ballPos.Y += ballVel.Y * 1000 * lookaheadSec
	return ballPos
}


func kickerStandoffDist(margin float64) float64 {
	return center2DribblerDist + ballRadius + margin
}

func alignmentMargin(headingErr float64) float64 {
	frac := math.Min(1, math.Abs(headingErr)/(math.Pi/4))
	return ballPushMargin + (maxMarginToBall-ballPushMargin)*frac
}

func behindBallHalfPlane(ballPos, robotPos, target info.Position) bool {
	toTargetX := target.X - ballPos.X
	toTargetY := target.Y - ballPos.Y
	toRobotX := robotPos.X - ballPos.X
	toRobotY := robotPos.Y - ballPos.Y
	return toTargetX*toRobotX+toTargetY*toRobotY < 0
}

func behindBallDest(ballPos, target info.Position, margin float64) info.Position {
	dx := target.X - ballPos.X
	dy := target.Y - ballPos.Y
	norm := math.Hypot(dx, dy)
	if norm < 1 {
		return ballPos
	}
	dist := kickerStandoffDist(margin)
	return info.Position{
		X:     ballPos.X - dx/norm*dist,
		Y:     ballPos.Y - dy/norm*dist,
		Angle: math.Atan2(dy, dx),
	}
}


func aroundBallDest(ballPos, botPos, dest info.Position, minMargin float64) info.Position {
	ball2BotAngle := ballPos.AngleToPosition(botPos)
	ball2DestAngle := ballPos.AngleToPosition(dest)
	remaining := info.NormalizeAngleDelta(ball2BotAngle, ball2DestAngle)

	relMargin := maxMarginToBall * math.Abs(remaining) / math.Pi
	if relMargin < 10 {
		relMargin = 0
	}
	distance := kickerStandoffDist(minMargin + relMargin)


	if ballPos.Dist2d(botPos) > 300 {
		abX := dest.X - botPos.X
		abY := dest.Y - botPos.Y
		denom := abX*abX + abY*abY
		if denom > 1e-9 {
			t := ((ballPos.X-botPos.X)*abX + (ballPos.Y-botPos.Y)*abY) / denom
			if t > 0 && t < 1 {
				footX := botPos.X + t*abX
				footY := botPos.Y + t*abY
				norm := math.Hypot(footX-ballPos.X, footY-ballPos.Y)
				if norm > 1e-9 {
					return info.Position{
						X:     ballPos.X + (footX-ballPos.X)/norm*distance,
						Y:     ballPos.Y + (footY-ballPos.Y)/norm*distance,
						Angle: dest.Angle,
					}
				}
			}
		}
	}

	
	shift := -math.Copysign(math.Min(math.Abs(remaining), aroundBallShiftAngle), remaining)
	bearing := ball2BotAngle + shift
	return info.Position{
		X:     ballPos.X + distance*math.Cos(bearing),
		Y:     ballPos.Y + distance*math.Sin(bearing),
		Angle: dest.Angle,
	}
}


func steppedOrientation(botPos, ballPos info.Position, finalOrientation float64) float64 {
	currentDirection := botPos.AngleToPosition(ballPos)
	diff := info.NormalizeAngleDelta(finalOrientation, currentDirection)
	return currentDirection + math.Copysign(math.Min(maxTargetOrientationStep, math.Abs(diff)), diff)
}
