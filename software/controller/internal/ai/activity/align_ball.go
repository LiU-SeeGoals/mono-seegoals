package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type AlignConfig struct {
	robotBallClearence float64
	doneDist float64
	angleError float64
}

func GetAlignConfig() AlignConfig{
	return AlignConfig{
		robotBallClearence: 200,
		doneDist: 50,
		angleError: 4.0 * math.Pi/180,
	}
}

type AlignBall struct {
	team info.Team
	id   info.ID
}

func (m *AlignBall) String() string {
	return fmt.Sprintf("AlignBall(%d)", m.id)
}

func NewAlignBall(team info.Team, id info.ID) *AlignBall {
	return &AlignBall{
		team,
		id,
	}
}

func (m *AlignBall) getTargetPos(gi *info.GameInfo) info.Position{
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	ballV2 := info.Vec2{X: ballPos.X, Y: ballPos.Y}

	goalPos := info.Vec2{X:4000,Y:0}

	ballGoalTangent := info.Sub(ballV2, goalPos)
	ballGoalTangent.DivNorm()
	fmt.Println(ballGoalTangent)

	robotXY := info.Sub(ballV2, ballGoalTangent.Mult(GetAlignConfig().robotBallClearence))
	fmt.Println("angle",ballGoalTangent.Angle())
	robotTargetPos := info.Position{X: robotXY.X, Y:robotXY.Y, Z:0, Angle: ballGoalTangent.Angle()}

	return robotTargetPos
}

func (m *AlignBall) GetAction(gi *info.GameInfo) action.Action {

	robotTargetPos := m.getTargetPos(gi)
	myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	if err != nil{
		fmt.Println(err)
	}

	ball := gi.State.GetBall()
	balVel := math.Sqrt(ball.GetVelocity().X * ball.GetVelocity().X + ball.GetVelocity().Y * ball.GetVelocity().Y)
	fmt.Println("balell fs fs",balVel)

	if balVel > 1{
		robotTargetPos = myRobotPos
	}

		
	act := action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = myRobotPos
	act.Dest = robotTargetPos
	act.Dribble = false

	return &act
}

func (m *AlignBall) Achieved(gi *info.GameInfo) bool {

	robotTargetPos := m.getTargetPos(gi)

	myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	if err != nil{
		fmt.Println(err)
	}
	xx :=(myRobotPos.X - robotTargetPos.X)*(myRobotPos.X - robotTargetPos.X) 
	yy :=(myRobotPos.Y - robotTargetPos.Y)*(myRobotPos.Y - robotTargetPos.Y) 

	angle_error := info.NormalizeAngleDelta(robotTargetPos.Angle, myRobotPos.Angle)	

	dist := math.Sqrt(xx+yy)

	fmt.Println(dist < GetAlignConfig().doneDist)
	fmt.Println(dist)
	fmt.Println(GetAlignConfig().doneDist)
	fmt.Println(angle_error, robotTargetPos.Angle, myRobotPos.Angle)

	return dist < GetAlignConfig().doneDist && math.Abs(angle_error) < GetAlignConfig().angleError
}

func (m *AlignBall) GetID() info.ID {

	return m.id
}

