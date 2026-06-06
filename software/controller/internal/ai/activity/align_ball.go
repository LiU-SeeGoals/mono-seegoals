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
	robotBallClearence float64
	stagingClearance   float64
	doneDist           float64
	angleError         float64
	turnToKickDist     float64
	minBehindBall      float64
	maxLineError       float64
}

func GetAlignConfig() AlignConfig {
	return AlignConfig{
		robotBallClearence: 300,
		stagingClearance:   500,
		doneDist:           90,
		angleError:         3.0 * math.Pi / 180,
		turnToKickDist:     180,
		minBehindBall:      120,
		maxLineError:       40,
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
	if ok && ballVel.Norm2d() > 0.3 {
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

	// act := action.MoveTo{}
	// act.Id = int(m.id)
	// act.Team = m.team
	// act.Pos = myRobotPos
	// act.Dest = robotTargetPos
	// act.Dribble = false

	return act
}

func (m *AlignBall) readyToFaceKick(myRobotPos, robotTargetPos info.Position, gi *info.GameInfo) bool {
	cfg := GetAlignConfig()
	isBehindBall, isOnPassLine := m.passLineChecks(myRobotPos, gi)
	return isBehindBall && (isOnPassLine || myRobotPos.Dist2d(robotTargetPos) < cfg.turnToKickDist)
}

func (m *AlignBall) passLineChecks(myRobotPos info.Position, gi *info.GameInfo) (bool, bool) {
	cfg := GetAlignConfig()
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		fmt.Println(err)
		return false, false
	}
	targetDir := m.to.Sub(&ballPos)
	targetDir.Z = 0
	targetDir.Angle = 0
	if targetDir.Norm2d() < 1 {
		return false, false
	}
	targetDir = targetDir.Normalize2d()
	robotFromBall := myRobotPos.Sub(&ballPos)
	alongLine := robotFromBall.X*targetDir.X + robotFromBall.Y*targetDir.Y
	sideError := math.Abs(robotFromBall.X*targetDir.Y - robotFromBall.Y*targetDir.X)

	return alongLine < -cfg.minBehindBall, sideError < cfg.maxLineError
}
func (m *AlignBall) Achieved(gi *info.GameInfo) bool {

	robotTargetPos := m.getTargetPos(gi)

	myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()

	if err != nil {
		fmt.Println(err)
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
