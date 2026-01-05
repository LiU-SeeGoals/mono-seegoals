package ai

import (
	"fmt"
	"math"

	// "gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/plt"
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
		angleError: 1.0 * math.Pi/180,
	}
}

type AlignBall struct {
	team info.Team
	id   info.ID
	target   info.Position
}

func (m *AlignBall) String() string {
	return fmt.Sprintf("AlignBall(%d)", m.id)
}

func NewAlignBall(team info.Team, id info.ID, target info.Position) *AlignBall {
	return &AlignBall{
		team,
		id,
		target,
	}
}
var saved int

func (m *AlignBall) getTargetPos(gi *info.GameInfo) info.Position{

	// plt.Init()

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()


	ballV2 := info.Vec2{X: ballPos.X, Y: ballPos.Y}

	goalPos := info.Vec2{X: m.target.X, Y: m.target.Y}

	ballGoalTangent := info.Sub(goalPos, ballV2)
	ballGoalTangent.DivNorm()

	alignPos := ballGoalTangent.Mult(GetAlignConfig().robotBallClearence)
	robotXY := info.Sub(ballV2, alignPos)
	robotTargetPos := info.Position{X: robotXY.X, Y:robotXY.Y, Z:0, Angle: ballGoalTangent.Angle()}

	points := plotter.XYs{}
	points = append(points, plotter.XY{X: ballPos.X, Y: ballPos.Y})
	points = append(points, plotter.XY{X: goalPos.X, Y: goalPos.Y})

	plt.Scatter(points)
	plt.Line(plotter.XY{X:ballGoalTangent.X, Y: ballGoalTangent.Y}, plotter.XY{X:alignPos.X, Y: alignPos.Y})
	plt.Line(plotter.XY{X:robotXY.X, Y: robotXY.Y}, plotter.XY{X:alignPos.X, Y: alignPos.Y})
	saved += 1
	// go plt.SaveFig(fmt.Sprintf("robobitch%d.png",saved))


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

