package ai

import (
	"fmt"
	"math"

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
		angleError:          3.0 * math.Pi / 180,
		maxContactLineError: info.KickCenterTolerance,
		turnToKickDist:      180,
		minBehindBall:       120,
		maxLineError:        captureLineTolerance,
	}
}

type AlignBall struct {
	team       info.Team
	id         info.ID
	to         info.Position
	from       info.Position
	AlignAngle float64
	useRRT     bool
	avoidBall  bool
}

func (m *AlignBall) String() string {
	return fmt.Sprintf("AlignBall(%d)", m.id)
}

func NewAlign(team info.Team, id info.ID, to info.Position, from info.Position) *AlignBall {
	return &AlignBall{
		team,
		id,
		to,
		from,
		0,
		true,
		true,
	}
}
func NewDirectAlign(team info.Team, id info.ID, to info.Position, from info.Position) *AlignBall {
	align := NewAlign(team, id, to, from)
	align.useRRT = false
	align.avoidBall = false
	return align
}
func (m *AlignBall) getTargetPos(gi *info.GameInfo) info.Position {
	return m.getTargetPosWithClearance(gi, GetAlignConfig().robotBallClearence)
}

func (m *AlignBall) getStagingPos(gi *info.GameInfo) info.Position {
	return m.getTargetPosWithClearance(gi, GetAlignConfig().stagingClearance)
}

func (m *AlignBall) getTargetPosWithClearance(gi *info.GameInfo, clearance float64) info.Position {
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()
	alignBallPos := ballPos
	useLookahead := ok && ballVel.Norm2d() > minRollingBallSpeed
	if useLookahead {
		if myPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition(); err == nil && myPos.Dist2d(ballPos) <= kickFarApproachDist {
			useLookahead = false
		}
	}
	if useLookahead {
		lookahead := 0.8
		alignBallPos.X += ballVel.X * 1000 * lookahead
		alignBallPos.Y += ballVel.Y * 1000 * lookahead

		// if we are in line with the ball don't lookahead, it just makes us miss the ball
		if math.Abs(info.NormalizeAngleDelta(ballPos.AngleToPosition(m.to), ballPos.Angle)) < 10*math.Pi/180 {
			alignBallPos = ballPos
		}

	}

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
	act := moveTo.GetMoveToAction(gi)
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()
	if ok && ballVel.Norm2d() > 0.3 {
		// Ball is moving face the ball to receive it
		ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
		myPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
		if err == nil {
			act.Dest.Angle = myPos.AngleToPosition(ballPos)
		}
	} else {
		if !m.readyToFaceKick(myPos, robotTargetPos, gi) {
			// Drive to the behind-ball staging/final point before rotating toward the pass angle.
			act.Dest.Angle = myPos.AngleToPosition(act.Dest)
		} else {
			// Ball is slow and we are in the approach corridor, align to kick direction.
			act.Dest.Angle = robotTargetPos.Angle
		}
	}

	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
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
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return false
	}
	dist := myPos.Dist2d(ballPos)
	if dist <= nearBallOrbitRetainDist {
		return true
	}
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()
	if ok && ballVel.Norm2d() > minRollingBallSpeed {
		return false
	}
	return dist <= kickFarApproachDist
}

// aroundBallAction walks around the ball onto the kick line, the standoff
// margin closing as the heading aligns, so the robot keeps the ball at its
// kicker instead of backing off to the clearance points.
func (m *AlignBall) aroundBallAction(myPos info.Position, gi *info.GameInfo) action.Action {
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	robot := gi.State.GetTeam(m.team)[m.id]

	finalOrientation := ballPos.AngleToPosition(m.to)
	m.AlignAngle = finalOrientation
	headingErr := math.Abs(info.NormalizeAngleDelta(finalOrientation, myPos.Angle))
	dribblerPos := robot.DribblerPos()
	ballCentered := m.contactPointCentered(robot, ballPos)
	captureReady := capturePoseReady(myPos, ballPos, m.to, headingErr)

	minMargin := alignmentMargin(headingErr)
	if !captureReady {
		minMargin = math.Max(minMargin, captureMarginToBall)
	} else if !behindBallHalfPlane(ballPos, myPos, m.to) || !ballCentered {
		minMargin = math.Max(minMargin, 0)
	}

	lineup := behindBallDest(ballPos, m.to, minMargin)
	carrot := aroundBallDest(ballPos, myPos, lineup, minMargin)
	carrot.Angle = steppedOrientation(myPos, ballPos, finalOrientation)

	dribble := dribblerPos.Dist2d(ballPos) < 120 &&
		headingErr < 2*roughAngleTolerance &&
		captureReady &&
		ballCentered

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
		Id:      int(m.id),
		Team:    m.team,
		Pos:     myPos,
		Dest:    carrot,
		Dribble: dribble,
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
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		fmt.Println(err)
		return 0, 0, false
	}
	return lineErrorToTarget(pos, ballPos, m.to)
}
func (m *AlignBall) Achieved(gi *info.GameInfo) bool {

	robotTargetPos := m.getTargetPos(gi)

	myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()

	if err != nil {
		fmt.Println(err)
	}

	// At a lying ball the clearance-point check below would force a robot
	// already in possession to back off before ALIGNED could fire; count it
	// aligned once it is behind the ball on the pass line, close in and
	// facing the kick direction.
	if m.nearLyingBall(myRobotPos, gi) {
		ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
		robot := gi.State.GetTeam(m.team)[m.id]
		headingErr := info.NormalizeAngleDelta(ballPos.AngleToPosition(m.to), myRobotPos.Angle)
		return myRobotPos.Dist2d(ballPos) < kickerStandoffDist(maxMarginToBall) &&
			captureApproachReady(myRobotPos, ballPos, m.to, math.Abs(headingErr)) &&
			m.contactPointCentered(robot, ballPos) &&
			math.Abs(headingErr) < GetAlignConfig().angleError
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
