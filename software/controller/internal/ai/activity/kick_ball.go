package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type KickBall struct {
	team info.Team
	id   info.ID
	orignalBallPos info.Position
	inited bool
}

type KickConfig struct {
	driveThrough float64
	doneDist float64
	ballAbortRadius float64
}

func GetKickConfig() KickConfig{
	return KickConfig{
		driveThrough: 200,
		doneDist: 50,
		ballAbortRadius: 200,
	}
}

func (m *KickBall) String() string {
	return fmt.Sprintf("KickBall(%d)", m.id)
}

func NewKickBall(team info.Team, id info.ID) *KickBall {
	return &KickBall{
		team,
		id,
		info.Position{},
		false,
	}
}

func (m *KickBall) init(gi *info.GameInfo) {
	if (m.inited){
		return
	}
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	m.orignalBallPos = ballPos
}

func (m *KickBall) GetTargetPos(gi *info.GameInfo) info.Position{

	m.init(gi)
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	ballV2 := info.Vec2{X: ballPos.X, Y: ballPos.Y}

	robotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	if err != nil{
		fmt.Println(err)
	}

	robotV2 := info.Vec2{X: robotPos.X, Y: robotPos.Y}

	ballRobotTangent := info.Sub(ballV2, robotV2)
	ballRobotTangent.DivNorm()

	robotXY := info.Add(ballV2, ballRobotTangent.Mult(GetKickConfig().driveThrough))
	robotTargetPos := info.Position{X: robotXY.X, Y:robotXY.Y, Z:0, Angle: ballRobotTangent.Angle()}
	return robotTargetPos
}

func (m *KickBall) GetAction(gi *info.GameInfo) action.Action {
	robotTargetPos := m.GetTargetPos(gi)

	robotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	if err != nil{
		fmt.Println(err)
	}

	ball := gi.State.GetBall()
	balVel := math.Sqrt(ball.GetVelocity().X * ball.GetVelocity().X + ball.GetVelocity().Y * ball.GetVelocity().Y)

	if balVel > 1{
		robotTargetPos = robotPos
	}

	act := action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = robotPos
	act.Dest = robotTargetPos
	act.Dribble = true
	act.KickSpeed = 50

	return &act
}

func (m *KickBall) Achieved(gi *info.GameInfo) bool {
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	// myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	// if err != nil{
	// 	fmt.Println(err)
	// }
	// robotTargetPos := m.GetTargetPos(gi)

	// robotDist := robotTargetPos.Norm2d(myRobotPos)
	ballDist := ballPos.Norm2d(m.orignalBallPos)

	// return false
	return ballDist > GetKickConfig().ballAbortRadius
}

func (m *KickBall) GetID() info.ID {

	return m.id
}

