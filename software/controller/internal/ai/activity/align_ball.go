package ai

import (
	"fmt"
	"math"
	"time"

	// "gonum.org/v1/plot"
	// "gonum.org/v1/plot/plotter"
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	// "github.com/LiU-SeeGoals/controller/internal/plt"
)

type AlignConfig struct {
	robotBallClearence  float64
	stagingClearance    float64
	doneDist            float64
	angleError          float64
	maxContactLineError float64
	turnToKickDist      float64
	minBehindBall       float64
	maxLineError        float64
}

func GetAlignConfig() AlignConfig {
	return AlignConfig{
		robotBallClearence:  300,
		stagingClearance:    500,
		doneDist:            90,
		angleError:          roughAngleTolerance,
		maxContactLineError: info.KickCenterTolerance,
		turnToKickDist:      180,
		minBehindBall:       120,
		maxLineError:        captureLineTolerance,
	}
}

type AlignBall struct {
	team              info.Team
	id                info.ID
	to                info.Position
	from              info.Position
	AlignAngle        float64
	useRRT            bool
	avoidBall         bool
	allowGoalArea     bool
	allowOutsideField bool
	allowBehindGoal   bool
	latch             *AlignLatch
}

func (m *AlignBall) String() string {
	return fmt.Sprintf("AlignBall(%d)", m.id)
}

func NewAlign(team info.Team, id info.ID, to info.Position, from info.Position) *AlignBall {
	return &AlignBall{
		team:              team,
		id:                id,
		to:                to,
		from:              from,
		AlignAngle:        0,
		useRRT:            true,
		avoidBall:         true,
		allowOutsideField: true,
		latch:             &AlignLatch{},
	}
}

func (m *AlignBall) SetLatch(latch *AlignLatch) {
	if latch != nil {
		m.latch = latch
	}
}

func NewDirectAlign(team info.Team, id info.ID, to info.Position, from info.Position) *AlignBall {
	align := NewAlign(team, id, to, from)
	align.useRRT = false
	align.avoidBall = false
	return align
}

func (m *AlignBall) AllowGoalArea(allow bool) {
	m.allowGoalArea = allow
}

func (m *AlignBall) AllowOutsideField(allow bool) {
	m.allowOutsideField = allow
}

func (m *AlignBall) AllowBehindGoalLine(allow bool) {
	m.allowBehindGoal = allow
}

func (m *AlignBall) getTargetPos(gi *info.GameInfo) info.Position {
	return m.getTargetPosWithClearance(gi, GetAlignConfig().robotBallClearence)
}

func (m *AlignBall) getStagingPos(gi *info.GameInfo) info.Position {
	return m.getTargetPosWithClearance(gi, GetAlignConfig().stagingClearance)
}

func (m *AlignBall) getTargetPosWithClearance(gi *info.GameInfo, clearance float64) info.Position {
	alignBallPos := m.ballPos(gi)

	ballV2 := info.Vec2{X: alignBallPos.X, Y: alignBallPos.Y}
	goalPos := info.Vec2{X: m.to.X, Y: m.to.Y}

	ballGoalTangent := info.Sub(goalPos, ballV2)
	if ballGoalTangent.Norm() < 1 {
		return alignBallPos
	}
	ballGoalTangent.DivNorm()

	alignPos := ballGoalTangent.Mult(clearance)
	robotXY := info.Sub(ballV2, alignPos)
	robotTargetPos := info.Position{X: robotXY.X, Y: robotXY.Y, Z: 0, Angle: ballGoalTangent.Angle()}
	m.AlignAngle = ballGoalTangent.Angle()
	return robotTargetPos
}

func (m *AlignBall) GetAction(gi *info.GameInfo) action.Action {

	robotTargetPos := m.getTargetPos(gi)
	stagingPos := m.getStagingPos(gi)
	// myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// ball := gi.State.GetTrackedBall()
	// //ballPos, err := ball.GetTrackedPosition()
	// ballVel, _ := ball.GetTrackedVelocity()

	// speed := ballVel.Norm2d()

	// if speed > 0.3 {
	// 	robotTargetPos = myRobotPos
	// }

	myPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	if err != nil {
		fmt.Println(err)
	}

	// A robot already close to a lying ball (it just took a pass, or a failed
	// pass stopped at its feet) must not retreat to the clearance points;
	// orbit the ball onto the kick line instead.
	if m.nearLyingBall(myPos, gi) {
		return m.aroundBallAction(myPos, gi)
	}

	isBehindBall, _ := m.passLineChecks(myPos, gi)
	moveTarget := robotTargetPos
	useRRT := m.useRRT
	avoidBall := m.avoidBall
	if !isBehindBall {
		moveTarget = stagingPos
	} else {
		useRRT = false
		avoidBall = false
	}

	moveTo := NewMoveToPosition(m.team, m.id, moveTarget)
	moveTo.SetUseRRT(useRRT)
	moveTo.AvoidBall(avoidBall)
	moveTo.AvoidGoallines(!m.allowGoalArea)
	moveTo.AllowOutsideField(m.allowOutsideField)
	moveTo.AllowBehindGoalLine(m.allowBehindGoal)
	act := moveTo.GetMoveToAction(gi)
	if m.ballRolling(gi) {
		// Ball is moving face the ball to receive it
		ballPos := m.ballPos(gi)
		myPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
		if err == nil {
			act.Dest.Angle = myPos.AngleToPosition(ballPos)
		}
	} else {
		if !m.readyToFaceKick(myPos, robotTargetPos, gi) {
			// The robot is omnidirectional, so its travel waypoint must not also
			// dictate its heading. Turn toward the eventual kick direction along
			// the shortest path, with a bounded per-frame target step. This avoids
			// the large spin that occurred when an RRT waypoint sat behind the bot.
			act.Dest.Angle = limitedOrientationStep(
				myPos.Angle,
				robotTargetPos.Angle,
				maxTargetOrientationStep,
			)
		} else {
			// Ball is slow and we are in the approach corridor, align to kick direction.
			act.Dest.Angle = robotTargetPos.Angle
		}
	}

	ballPos := m.ballPos(gi)
	robot := gi.State.GetTeam(m.team)[m.id]
	finalHeadingErr := math.Abs(info.NormalizeAngleDelta(ballPos.AngleToPosition(m.to), myPos.Angle))
	captureReady := capturePoseReady(myPos, ballPos, m.to, finalHeadingErr)
	ballCentered := m.contactPointCentered(robot, ballPos)
	printCaptureDebug(
		"align",
		m.team,
		m.id,
		robot,
		myPos,
		ballPos,
		m.to,
		act.Dest,
		finalHeadingErr,
		captureReady,
		ballCentered,
		act.Dribble,
		act.KickSpeed,
		math.NaN(),
	)

	// act := action.MoveTo{}
	// act.Id = int(m.id)
	// act.Team = m.team
	// act.Pos = myRobotPos
	// act.Dest = robotTargetPos
	// act.Dribble = false

	return act
}

// nearLyingBall reports whether the ball is lying still with the robot close
// enough that the clearance/staging targets would point away from the ball.
func (m *AlignBall) nearLyingBall(myPos info.Position, gi *info.GameInfo) bool {
	ballPos := m.ballPos(gi)
	enterDist := kickFarApproachDist
	if m.ballRolling(gi) {
		enterDist = nearBallOrbitRetainDist
	}
	return m.latch.orbitBall(myPos.Dist2d(ballPos), enterDist, time.Now())
}

func (m *AlignBall) ballPos(gi *info.GameInfo) info.Position {
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	if m.ballRolling(gi) {
		m.latch.releaseBall()
		return ballPos
	}
	return m.latch.holdBall(ballPos, time.Now())
}

func (m *AlignBall) onKickLine(gi *info.GameInfo, robot *info.Robot, myPos, ballPos info.Position, headingErr float64) bool {
	along, sideErr, ok := lineErrorToTarget(myPos, ballPos, m.to)
	if !ok {
		return false
	}
	now := time.Now()
	vel, fresh := gi.State.GetTrackedRobot(m.team, uint32(m.id)).GetFreshTrackedVelocity(now, alignTrackedBallMaxAge)
	if !fresh {
		vel = robot.GetVelocity()
	}
	sideSpeed := sideSpeedToLine(vel, ballPos, m.to)
	return m.latch.onKickLine(along, sideErr, sideSpeed, headingErr, now)
}

func (m *AlignBall) ballRolling(gi *info.GameInfo) bool {
	now := time.Now()
	ballVel, ok := gi.State.GetTrackedBall().GetFreshTrackedVelocity(now, alignTrackedBallMaxAge)
	return m.latch.ballRolling(ballVel.Norm2d(), ok, now)
}

func (m *AlignBall) ballTrusted(gi *info.GameInfo) bool {
	kind, holder := gi.State.GetBall().GetEstimateKind()
	switch kind {
	case info.BallObserved:
		return true
	case info.BallInDribbler:
		return holder == gi.State.GetTeam(m.team)[m.id]
	default:
		return false
	}
}

// aroundBallAction walks around the ball onto the kick line, the standoff
// margin closing as the heading aligns, so the robot keeps the ball at its
// kicker instead of backing off to the clearance points.
func (m *AlignBall) aroundBallAction(myPos info.Position, gi *info.GameInfo) action.Action {
	ballPos := m.ballPos(gi)
	robot := gi.State.GetTeam(m.team)[m.id]

	finalOrientation := ballPos.AngleToPosition(m.to)
	m.AlignAngle = finalOrientation
	headingErr := math.Abs(info.NormalizeAngleDelta(finalOrientation, myPos.Angle))
	dribblerPos := robot.DribblerPos()
	ballCentered := m.contactPointCentered(robot, ballPos)
	approachReady := m.onKickLine(gi, robot, myPos, ballPos, headingErr)
	captureReady := capturePoseReady(myPos, ballPos, m.to, headingErr)

	keepWide := !behindBallHalfPlane(ballPos, myPos, m.to) || !ballCentered
	minMargin := captureOrbitMargin(headingErr, approachReady, keepWide)

	lineup := behindBallDest(ballPos, m.to, minMargin)
	carrot := aroundBallDest(ballPos, myPos, lineup, minMargin)
	carrot.Angle = steppedOrientation(myPos, ballPos, finalOrientation)

	forward, lateral, ok := robot.BallLocalOffset(ballPos)
	ballHeldInMouth := ok && kickBallHeldInMouth(dribblerPos.Dist2d(ballPos), forward, lateral, headingErr)
	dribble := ballCentered && (ballHeldInMouth || (dribblerPos.Dist2d(ballPos) < GetKickConfig().kickContactDist &&
		headingErr < 2*roughAngleTolerance &&
		approachReady))

	printCaptureDebug(
		"align-around",
		m.team,
		m.id,
		robot,
		myPos,
		ballPos,
		m.to,
		carrot,
		headingErr,
		captureReady,
		ballCentered,
		dribble,
		0,
		minMargin,
	)

	return &action.MoveTo{
		Id:                  int(m.id),
		Team:                m.team,
		Pos:                 myPos,
		Dest:                carrot,
		AllowOutsideField:   m.allowOutsideField,
		AllowBehindGoalLine: m.allowBehindGoal,
		AllowGoalArea:       m.allowGoalArea,
		MinLinearSpeed:      aroundBallLinearSpeed(myPos, carrot),
		Dribble:             dribble,
	}
}

func (m *AlignBall) readyToFaceKick(myRobotPos, robotTargetPos info.Position, gi *info.GameInfo) bool {
	cfg := GetAlignConfig()
	isBehindBall, isOnPassLine := m.passLineChecks(myRobotPos, gi)
	return isBehindBall && (isOnPassLine || myRobotPos.Dist2d(robotTargetPos) < cfg.turnToKickDist)
}

func (m *AlignBall) passLineChecks(myRobotPos info.Position, gi *info.GameInfo) (bool, bool) {
	cfg := GetAlignConfig()
	alongLine, sideError, ok := m.passLineError(myRobotPos, gi)
	if !ok {
		return false, false
	}

	return alongLine < -cfg.minBehindBall, sideError < cfg.maxLineError
}

func (m *AlignBall) contactPointCentered(robot *info.Robot, ballPos info.Position) bool {
	_, lateral, ok := robot.BallLocalOffset(ballPos)
	return ok && math.Abs(lateral) < GetAlignConfig().maxContactLineError
}

func (m *AlignBall) passLineError(pos info.Position, gi *info.GameInfo) (float64, float64, bool) {
	return lineErrorToTarget(pos, m.ballPos(gi), m.to)
}
func (m *AlignBall) Achieved(gi *info.GameInfo) bool {
	if !m.ballTrusted(gi) {
		return false
	}

	robotTargetPos := m.getTargetPos(gi)

	myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()

	if err != nil {
		fmt.Println(err)
	}

	// At a lying ball the clearance-point check below would force a robot
	// already in possession to back off before ALIGNED could fire; count it
	// aligned once it is behind the ball, close in, and within the narrower
	// transition corridor. This remains less strict than the kick-center
	// diagnostic while preventing a handoff at the edge of dribbler reach.
	if m.nearLyingBall(myRobotPos, gi) {
		ballPos := m.ballPos(gi)
		robot := gi.State.GetTeam(m.team)[m.id]
		headingErr := math.Abs(info.NormalizeAngleDelta(ballPos.AngleToPosition(m.to), myRobotPos.Angle))
		_, lateral, ballOffsetOK := robot.BallLocalOffset(ballPos)
		return myRobotPos.Dist2d(ballPos) < kickerStandoffDist(maxMarginToBall) &&
			m.onKickLine(gi, robot, myRobotPos, ballPos, headingErr) &&
			ballOffsetOK &&
			math.Abs(lateral) <= alignTransitionLateralTolerance
	}

	xx := (myRobotPos.X - robotTargetPos.X) * (myRobotPos.X - robotTargetPos.X)
	yy := (myRobotPos.Y - robotTargetPos.Y) * (myRobotPos.Y - robotTargetPos.Y)

	angle_error := info.NormalizeAngleDelta(robotTargetPos.Angle, myRobotPos.Angle)

	dist := math.Sqrt(xx + yy)

	isBehindBall, isOnPassLine := m.passLineChecks(myRobotPos, gi)
	val := dist < GetAlignConfig().doneDist &&
		math.Abs(angle_error) < GetAlignConfig().angleError &&
		isBehindBall &&
		isOnPassLine
	if m.id == 3 {
		// fmt.Println("angle error", math.Abs(angle_error), "threshold: ",GetAlignConfig().angleError, "dist:", dist, "threshold: ", GetAlignConfig().doneDist, val)
	}
	return val
}

func (m *AlignBall) GetID() info.ID {

	return m.id
}
