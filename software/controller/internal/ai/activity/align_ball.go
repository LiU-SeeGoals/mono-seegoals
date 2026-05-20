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
	doneDist           float64
	angleError         float64
}

func GetAlignConfig() AlignConfig {
	return AlignConfig{
		robotBallClearence: 200,
		doneDist:           50,
		angleError:         3.0 * math.Pi / 180,
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

	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()
	ballV2 := info.Vec2{X: m.from.X, Y: m.from.Y}
	if ok && ballVel.Norm2d() > 0.3 {
		lookahead := 0.8
		ballPos.X += ballVel.X * 1000 * lookahead
		ballPos.Y += ballVel.Y * 1000 * lookahead
		ballV2 = info.Vec2{X: ballPos.X, Y: ballPos.Y}

		// if we are in line with the ball don't lookahead, it just makes us miss the ball
		if math.Abs(info.NormalizeAngleDelta(ballPos.AngleToPosition(m.to), ballPos.Angle)) < 10*math.Pi/180 {
			ballV2 = info.Vec2{X: ballPos.X, Y: ballPos.Y}
		}

	}

	
	

	goalPos := info.Vec2{X: m.to.X, Y: m.to.Y}

	ballGoalTangent := info.Sub(goalPos, ballV2)
	ballGoalTangent.DivNorm()

	alignPos := ballGoalTangent.Mult(GetAlignConfig().robotBallClearence)
	robotXY := info.Sub(ballV2, alignPos)
	robotTargetPos := info.Position{X: robotXY.X, Y: robotXY.Y, Z: 0, Angle: ballGoalTangent.Angle()}
	m.AlignAngle = ballGoalTangent.Angle()
	return robotTargetPos
}

func (m *AlignBall) GetAction(gi *info.GameInfo) action.Action {

	robotTargetPos := m.getTargetPos(gi)
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

	moveTo := NewMoveToPosition(m.team, m.id, robotTargetPos)
	moveTo.SetUseRRT(m.useRRT)
	moveTo.AvoidBall(m.avoidBall)
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
        // Ball is slow align to kick direction
        act.Dest.Angle = robotTargetPos.Angle
    }

	// act := action.MoveTo{}
	// act.Id = int(m.id)
	// act.Team = m.team
	// act.Pos = myRobotPos
	// act.Dest = robotTargetPos
	// act.Dribble = false

	return act
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
	val := dist < GetAlignConfig().doneDist && math.Abs(angle_error) < GetAlignConfig().angleError
	if m.id == 3 {
		// fmt.Println("angle error", math.Abs(angle_error), "threshold: ",GetAlignConfig().angleError, "dist:", dist, "threshold: ", GetAlignConfig().doneDist, val)
	}
	return val
}

func (m *AlignBall) GetID() info.ID {

	return m.id
}
