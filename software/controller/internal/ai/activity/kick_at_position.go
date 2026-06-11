package ai

import (
	"fmt"
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	// Beyond this distance to the ball [mm] we path-plan to the lineup point.
	kickFarApproachDist = 600.0
	// Time the aim must be held before the kick is armed.
	kickAlignConfirmTime = 300 * time.Millisecond
	// Drive-through distance past the ball when kicking [mm].
	kickRunUpDist = 100.0
	kickSpeed     = 5
)

type KickAtPosition struct {
	GenericComposition
	targetPosition info.Position
	retrievingBall bool
	// Wall-clock confirmation that the aim has been held (Sumatra's
	// TargetAngleReachedChecker); zero while unaligned.
	alignedSince time.Time
}

func (k *KickAtPosition) String() string {
	return fmt.Sprintf("(Robot %d, KickAtPosition(%v))", k.id, k.targetPosition)
}

func NewKickAtPosition(team info.Team, id info.ID, targetPosition info.Position) *KickAtPosition {
	return &KickAtPosition{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition: targetPosition,
		retrievingBall: true,
	}
}

func (kp *KickAtPosition) GetAction(gi *info.GameInfo) action.Action {
	robot := gi.State.GetRobot(kp.id, kp.team)
	robotPos, _ := robot.GetPosition()
	ball := gi.State.GetBall()
	ballNow, _ := ball.GetEstimatedPosition()
	ballPred := predictedBallPos(gi, ballLookaheadSec)

	kp.retrievingBall = ball.GetPossessor() != robot

	finalOrientation := ballPred.AngleToPosition(kp.targetPosition)
	headingErr := math.Abs(info.NormalizeAngleDelta(finalOrientation, robotPos.Angle))
	behindBall := kp.isBehindBall(robotPos, ballPred)

	// Far away: path-plan to the lineup point behind the ball at full standoff.
	if robotPos.Dist2d(ballNow) > kickFarApproachDist {
		kp.alignedSince = time.Time{}
		lineup := behindBallDest(ballPred, kp.targetPosition, maxMarginToBall)
		move := NewMoveToPosition(kp.team, kp.id, lineup)
		move.AvoidBall(true)
		moveAction := move.GetMoveToAction(gi)
		moveAction.Dest.Angle = robotPos.AngleToPosition(ballPred)
		return moveAction
	}

	if kp.armed(robotPos, ballNow, headingErr, behindBall) {
		// Aim held: drive through the ball toward the target and kick.
		dest := info.Position{
			X:     ballPred.X + kickRunUpDist*math.Cos(finalOrientation),
			Y:     ballPred.Y + kickRunUpDist*math.Sin(finalOrientation),
			Angle: finalOrientation,
		}
		return &action.MoveTo{
			Id:        int(kp.id),
			Team:      kp.team,
			Pos:       robotPos,
			Dest:      dest,
			KickSpeed: kickSpeed,
		}
	}

	// Near the ball: orbit around it onto the kick line. The standoff margin
	// only closes once heading and position are lined up, so we cannot bump
	// the ball away while still rotating around it.
	minMargin := alignmentMargin(headingErr)
	if !behindBall {
		minMargin = math.Max(minMargin, 0)
	}
	lineup := behindBallDest(ballPred, kp.targetPosition, minMargin)
	carrot := aroundBallDest(ballPred, robotPos, lineup, minMargin)
	carrot.Angle = steppedOrientation(robotPos, ballPred, finalOrientation)

	dribblerPos := robot.DribblerPos()
	dribble := dribblerPos.Dist2d(ballNow) < 120 && headingErr < 2*roughAngleTolerance

	return &action.MoveTo{
		Id:      int(kp.id),
		Team:    kp.team,
		Pos:     robotPos,
		Dest:    carrot,
		Dribble: dribble,
	}
}

// isBehindBall reports whether the robot is on the far side of the ball seen
// from the kick target, i.e. kicking would send the ball toward the target.
func (kp *KickAtPosition) isBehindBall(robotPos, ballPos info.Position) bool {
	toTargetX := kp.targetPosition.X - ballPos.X
	toTargetY := kp.targetPosition.Y - ballPos.Y
	toRobotX := robotPos.X - ballPos.X
	toRobotY := robotPos.Y - ballPos.Y
	return toTargetX*toRobotX+toTargetY*toRobotY < 0
}

// armed gates the kick: heading and position on the kick line must be held
// for kickAlignConfirmTime with the kicker close to the ball.
func (kp *KickAtPosition) armed(robotPos, ballPos info.Position, headingErr float64, behindBall bool) bool {
	nearBall := robotPos.Dist2d(ballPos) < kickerStandoffDist(maxMarginToBall)
	aligned := headingErr < roughAngleTolerance && behindBall && nearBall
	if !aligned {
		kp.alignedSince = time.Time{}
		return false
	}
	if kp.alignedSince.IsZero() {
		kp.alignedSince = time.Now()
	}
	return time.Since(kp.alignedSince) >= kickAlignConfirmTime
}

func (kp *KickAtPosition) Achieved(gi *info.GameInfo) bool {
	if kp.retrievingBall {
		return false
	}

	robot := gi.State.GetRobot(kp.id, kp.team)
	return gi.State.LostBall(robot)
}

func (kp *KickAtPosition) GetID() info.ID {
	return kp.id
}
