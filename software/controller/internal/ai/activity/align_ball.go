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
	doneDist float64
	angleError float64
}

func GetAlignConfig() AlignConfig{
	return AlignConfig{
		robotBallClearence: 200,
		doneDist: 50,
		angleError: 2.0 * math.Pi/180,
	}
}

type AlignBall struct {
	team info.Team
	id   info.ID
	to   info.Position
	from   info.Position
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
	}
}

func (m *AlignBall) getTargetPos(gi *info.GameInfo) info.Position{

	ballV2 := info.Vec2{X: m.from.X, Y: m.from.Y}

	goalPos := info.Vec2{X: m.to.X, Y: m.to.Y}

	ballGoalTangent := info.Sub(goalPos, ballV2)
	ballGoalTangent.DivNorm()

	alignPos := ballGoalTangent.Mult(GetAlignConfig().robotBallClearence)
	robotXY := info.Sub(ballV2, alignPos)
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

	val := dist < GetAlignConfig().doneDist && math.Abs(angle_error) < GetAlignConfig().angleError

	return val
}

func (m *AlignBall) GetID() info.ID {

	return m.id
}

