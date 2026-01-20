package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type RecieveBall struct {
	team           info.Team
	id             info.ID
	orignalBallPos info.Position
	inited         bool
}

type RecieveConfig struct {
	driveThrough    float64
	doneDist        float64
	ballAbortRadius float64
}

func GetRecieveConfig() RecieveConfig {
	return RecieveConfig{
		driveThrough:    200,
		doneDist:        50,
		ballAbortRadius: 100,
	}
}

func (m *RecieveBall) String() string {
	return fmt.Sprintf("RecieveBall(%d)", m.id)
}

func NewRecieveBall(team info.Team, id info.ID, ballPos info.Position) *RecieveBall {
	fmt.Println("New kick ball")
	return &RecieveBall{
		team,
		id,
		ballPos,
		false,
	}
}

func (m *RecieveBall) GetTargetPos(gi *info.GameInfo) info.Position {

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	// fmt.Println("new pos", ballPos, m.orignalBallPos, ballPos.Norm2d(m.orignalBallPos))
	ballV2 := info.Vec2{X: ballPos.X, Y: ballPos.Y}

	robotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	if err != nil {
		fmt.Println(err)
	}

	robotV2 := info.Vec2{X: robotPos.X, Y: robotPos.Y}

	ballRobotTangent := info.Sub(ballV2, robotV2)
	ballRobotTangent.DivNorm()

	robotXY := info.Add(ballV2, ballRobotTangent.Mult(GetRecieveConfig().driveThrough))
	robotTargetPos := info.Position{X: robotXY.X, Y: robotXY.Y, Z: 0, Angle: ballRobotTangent.Angle()}
	return robotTargetPos
}

func (m *RecieveBall) GetAction(gi *info.GameInfo) action.Action {
	robotTargetPos := m.GetTargetPos(gi)

	robotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	if err != nil {
		fmt.Println(err)
	}

	ball := gi.State.GetBall()
	balVel := math.Sqrt(ball.GetVelocity().X*ball.GetVelocity().X + ball.GetVelocity().Y*ball.GetVelocity().Y)

	if balVel > 1 {
		robotTargetPos = robotPos
	}

	act := action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = robotPos
	act.Dest = robotTargetPos
	act.Dribble = true
	act.KickSpeed = 1

	return &act
}

func (m *RecieveBall) Achieved(gi *info.GameInfo) bool {
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	// myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	// if err != nil{
	// 	fmt.Println(err)
	// }
	// robotTargetPos := m.GetTargetPos(gi)

	// robotDist := robotTargetPos.Norm2d(myRobotPos)
	ballDist := ballPos.Dist2d(m.orignalBallPos)

	// return false
	if ballDist > GetRecieveConfig().ballAbortRadius {
		fmt.Println("Done")
		fmt.Println(ballDist)
		fmt.Println(ballPos, m.orignalBallPos)
	}

	return ballDist > GetRecieveConfig().ballAbortRadius
}

func (m *RecieveBall) GetID() info.ID {

	return m.id
}
