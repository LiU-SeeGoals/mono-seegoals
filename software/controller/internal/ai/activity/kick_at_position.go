package ai

import (
	"fmt"
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	// Beyond this distance to the ball [mm] we path-plan to the lineup point.
	kickFarApproachDist = 600.0
	// Time the aim must be held before the kick is armed.
	kickAlignConfirmTime = 30 * time.Millisecond
	// Drive-through distance past the ball when kicking [mm].
	kickRunUpDist = 0.0
	kickSpeed     = 3
	// Far-approach goal must sit outside the planner's no-go band around the
	// ball, otherwise RRT can never reach it and orbits the ball instead.
	kickStagingClearance = pathplanner.BallSafetyRadius + pathplanner.MotionRadius + 50.0
)

type KickAtPosition struct {
	GenericComposition
	targetPosition info.Position
	retrievingBall bool
	// Wall-clock confirmation that the aim has been held (Sumatra's
	// TargetAngleReachedChecker); zero while unaligned.
	alignedSince time.Time
	dribbleSince time.Time
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
	// Bearing error around the ball: 0 when the robot sits on the kick line
	// behind the ball.
	lineErr := math.Abs(info.NormalizeAngleDelta(ballPred.AngleToPosition(robotPos), finalOrientation+math.Pi))

	// Far away: path-plan to a staging point behind the ball, outside the
	// planner's ball no-go zone.
	if robotPos.Dist2d(ballNow) > kickFarApproachDist {
		kp.alignedSince = time.Time{}
		kp.dribbleSince = time.Time{}
		lineup := behindBallDest(ballPred, kp.targetPosition, kickStagingClearance-kickerStandoffDist(0))
		move := NewMoveToPosition(kp.team, kp.id, lineup)
		move.AvoidBall(true)
		moveAction := move.GetMoveToAction(gi)
		moveAction.Dest.Angle = robotPos.AngleToPosition(ballPred)
		return moveAction
	}

	dribblerPos := robot.DribblerPos()
	dribblerDist := dribblerPos.Dist2d(ballNow)
	forward, lateral, ok := robot.BallLocalOffset(ballNow)
	ballCentered := ok && math.Abs(lateral) < info.KickCenterTolerance
	ballReachable := ok && kickBallReachableByMouth(forward, lateral, headingErr)
	ballHeldInMouth := ok && kickBallHeldInMouth(dribblerDist, forward, lateral, headingErr)
	approachReady := captureApproachReady(robotPos, ballPred, kp.targetPosition, headingErr)
	captureReady := capturePoseReady(robotPos, ballPred, kp.targetPosition, headingErr)
	firmwareLeadDist := kickFirmwareLeadDist(robot)

	if kp.armed(robotPos, ballNow, headingErr, lineErr, ballReachable, firmwareLeadDist) {
		// Aim held: drive through the ball toward the target, but only kick
		// once the ball is at the front of the robot or has settled in the mouth.
		dest := info.Position{
			X:     ballPred.X + kickRunUpDist*math.Cos(finalOrientation),
			Y:     ballPred.Y + kickRunUpDist*math.Sin(finalOrientation),
			Angle: finalOrientation,
		}
		kickAfterSettle := updateKickDribbleSettle(&kp.dribbleSince, ballHeldInMouth)
		impactReady := approachReady && ok && kickBallImpactReady(forward, lateral, headingErr, firmwareLeadDist)
		kickSpeedCmd := 0
		if kickBallShouldFire(dribblerDist, kickAfterSettle, impactReady) {
			kickSpeedCmd = kickSpeed
		}
		label := "kick-at-position-drive"
		if kickSpeedCmd != 0 {
			label = "kick-at-position-fire"
		}
		printCaptureDebug(
			label,
			kp.team,
			kp.id,
			robot,
			robotPos,
			ballNow,
			kp.targetPosition,
			dest,
			headingErr,
			captureReady,
			ballCentered,
			true,
			kickSpeedCmd,
			firmwareLeadDist,
		)
		return &action.MoveTo{
			Id:                int(kp.id),
			Team:              kp.team,
			Pos:               robotPos,
			Dest:              dest,
			AllowOutsideField: true,
			Dribble:           true,
			KickSpeed:         kickSpeedCmd,
		}
	}
	kp.dribbleSince = time.Time{}

	// Near the ball: orbit around it onto the kick line. The standoff margin
	// only closes once heading and position are lined up, so we cannot bump
	// the ball away while still rotating around it.
	// Never push toward the ball while still off the kick line or while the
	// ball is visibly offset in the dribbler mouth.
	minMargin := captureOrbitMargin(headingErr, approachReady, lineErr > 0.3 || !ballCentered)
	lineup := behindBallDest(ballPred, kp.targetPosition, minMargin)
	carrot := aroundBallDest(ballPred, robotPos, lineup, minMargin)
	carrot.Angle = steppedOrientation(robotPos, ballPred, finalOrientation)

	dribble := ballReachable && (ballHeldInMouth || (dribblerDist < 120 &&
		headingErr < 2*roughAngleTolerance &&
		lineErr < roughAngleTolerance &&
		approachReady))
	printCaptureDebug(
		"kick-at-position",
		kp.team,
		kp.id,
		robot,
		robotPos,
		ballNow,
		kp.targetPosition,
		carrot,
		headingErr,
		captureReady,
		ballCentered,
		dribble,
		0,
		minMargin,
	)

	return &action.MoveTo{
		Id:                int(kp.id),
		Team:              kp.team,
		Pos:               robotPos,
		Dest:              carrot,
		AllowOutsideField: true,
		MinLinearSpeed:    aroundBallLinearSpeed(robotPos, carrot),
		Dribble:           dribble,
	}
}

// armed gates the drive-through: heading and position on the kick line must be
// held for kickAlignConfirmTime with the ball reachable by the dribbler mouth.
func (kp *KickAtPosition) armed(
	robotPos, ballPos info.Position,
	headingErr, lineErr float64,
	ballReachable bool,
	firmwareLeadDist float64,
) bool {
	nearBall := robotPos.Dist2d(ballPos) < kickFirmwareArmingDist(firmwareLeadDist)
	aligned := headingErr < roughAngleTolerance &&
		lineErr < 2*roughAngleTolerance &&
		nearBall &&
		ballReachable
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
