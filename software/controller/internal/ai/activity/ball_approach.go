package ai

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	maxMarginToBall         = 90.0
	ballPushMargin          = -10.0
	captureMarginToBall     = 40.0
	offCenterOrbitMargin    = 30.0
	captureLineTolerance    = info.DribblerHalfWidth + info.BallRadius - 0.5
	capturePoseTolerance    = 35.0
	nearBallOrbitRetainDist = 300.0
	// The real robot's position follower scales velocity with destination
	// distance. Keep close orbit corrections large enough to avoid crawling.
	minAroundBallMoveDist    = 50.0
	minAroundBallMoveTrigger = 1.0
	aroundBallMinLinearSpeed = 0.5
	// Keep the orbit carrot far enough ahead that the position controller
	// commands useful tangential speed while the robot has substantial
	// rotation remaining. Small final corrections are still limited by the
	// remaining angle below.
	aroundBallShiftAngle     = 1.5
	maxTargetOrientationStep = 0.6
	roughAngleTolerance      = 0.05
	ballLookaheadSec         = 0.2
	minRollingBallSpeed      = 0.3
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
	return info.Center2DribblerDist + info.BallRadius + margin
}

func alignmentMargin(headingErr float64) float64 {
	frac := math.Min(1, math.Abs(headingErr)/(math.Pi/4))
	return ballPushMargin + (maxMarginToBall-ballPushMargin)*frac
}

func captureOrbitMargin(headingErr float64, approachReady bool, keepWide bool) float64 {
	margin := alignmentMargin(headingErr)
	if !approachReady {
		return math.Max(margin, captureMarginToBall)
	}
	if keepWide {
		return math.Max(margin, offCenterOrbitMargin)
	}
	return margin
}

func behindBallHalfPlane(ballPos, robotPos, target info.Position) bool {
	toTargetX := target.X - ballPos.X
	toTargetY := target.Y - ballPos.Y
	toRobotX := robotPos.X - ballPos.X
	toRobotY := robotPos.Y - ballPos.Y
	return toTargetX*toRobotX+toTargetY*toRobotY < 0
}

func lineErrorToTarget(pos, ballPos, target info.Position) (float64, float64, bool) {
	targetDir := target.Sub(&ballPos)
	targetDir.Z = 0
	targetDir.Angle = 0
	if targetDir.Norm2d() < 1 {
		return 0, 0, false
	}
	targetDir = targetDir.Normalize2d()
	posFromBall := pos.Sub(&ballPos)
	alongLine := posFromBall.X*targetDir.X + posFromBall.Y*targetDir.Y
	sideError := math.Abs(posFromBall.X*targetDir.Y - posFromBall.Y*targetDir.X)
	return alongLine, sideError, true
}

func captureApproachReady(robotPos, ballPos, target info.Position, headingErr float64) bool {
	alongLine, sideError, ok := lineErrorToTarget(robotPos, ballPos, target)
	if !ok {
		return false
	}
	return sideError < captureLineTolerance &&
		alongLine < -info.Center2DribblerDist &&
		headingErr < 2*roughAngleTolerance
}

func capturePoseReady(robotPos, ballPos, target info.Position, headingErr float64) bool {
	capturePos := behindBallDest(ballPos, target, captureMarginToBall)
	return captureApproachReady(robotPos, ballPos, target, headingErr) &&
		robotPos.Dist2d(capturePos) < capturePoseTolerance
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

	shiftMag := math.Min(math.Abs(remaining), aroundBallShiftAngle)
	botRadius := ballPos.Dist2d(botPos)
	if distance > 1 && botRadius > 1 {
		orbitArc := shiftMag * distance
		moveDist := math.Sqrt(botRadius*botRadius + distance*distance -
			2*botRadius*distance*math.Cos(shiftMag))
		if orbitArc > minAroundBallMoveTrigger && moveDist < minAroundBallMoveDist {
			// Solve the chord equation for the angular shift that places the
			// carrot far enough away while keeping it on the intended orbit.
			cosShift := (botRadius*botRadius + distance*distance -
				minAroundBallMoveDist*minAroundBallMoveDist) / (2 * botRadius * distance)
			cosShift = math.Max(-1, math.Min(1, cosShift))
			minShift := math.Acos(cosShift)
			shiftMag = math.Min(aroundBallShiftAngle, math.Max(shiftMag, minShift))
		}
	}
	shift := -math.Copysign(shiftMag, remaining)
	bearing := ball2BotAngle + shift
	return info.Position{
		X:     ballPos.X + distance*math.Cos(bearing),
		Y:     ballPos.Y + distance*math.Sin(bearing),
		Angle: dest.Angle,
	}
}

func aroundBallLinearSpeed(botPos, dest info.Position) float64 {
	if botPos.Dist2d(dest) < minAroundBallMoveTrigger {
		return 0
	}
	return aroundBallMinLinearSpeed
}

func steppedOrientation(botPos, ballPos info.Position, finalOrientation float64) float64 {
	currentDirection := botPos.AngleToPosition(ballPos)
	diff := info.NormalizeAngleDelta(finalOrientation, currentDirection)
	return currentDirection + math.Copysign(math.Min(maxTargetOrientationStep, math.Abs(diff)), diff)
}
