package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type ReceiveBallAtPosition struct {
	team            info.Team
	id              info.ID
	target_position info.Position
	dribble_to_ball bool
}

func (rbap *ReceiveBallAtPosition) String() string {
	return fmt.Sprintf("Robot %d, ReceiveBallAtPosition(%v)", rbap.id, rbap.target_position)
}

func NewReceiveBallAtPosition(team info.Team, id info.ID, dest info.Position) *ReceiveBallAtPosition {
	return &ReceiveBallAtPosition{
		team:            team,
		id:              id,
		target_position: dest,
		dribble_to_ball: false,
	}
}

func (rbap *ReceiveBallAtPosition) GetAction(gi *info.GameInfo) action.Action {

	myTeam := gi.State.GetTeam(rbap.team)
	robot := myTeam[rbap.id]

	if !robot.IsActive() {
		return nil
	}

	robotPos, err1 := robot.GetPosition()
	
	if err1 != nil {
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err1)
		return NewStop(rbap.id).GetAction(gi)
	}

	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Ball: %v\n", err)
		return NewStop(rbap.id).GetAction(gi)
	}

	dx := float64(robotPos.X - ballPos.X)
	dy := float64(robotPos.Y - ballPos.Y)

	fmt.Println("Ball pos X: ", ballPos.X, " Y: ", ballPos.Y)
	fmt.Println("Robot Pos X: ", robotPos.X, " Y: ", robotPos.Y)

	distance := math.Sqrt(math.Pow(dx, 2) + math.Pow(dy, 2))
	fmt.Println("Distance is: ", distance)

	act := action.MoveTo{}
	act.Id = int(robot.GetID())
	act.Team = rbap.team
	act.Pos = robotPos
	act.Dest = rbap.target_position

	if distance > 100 {
		return &act
	}
	act.Dribble = true
	act.Dest = robotPos
	rbap.dribble_to_ball = true

	return &act
}

func (rbap *ReceiveBallAtPosition) Achieved(gs *info.GameInfo) bool {
	// Need to be implemented
	return rbap.dribble_to_ball
}

func (rbap *ReceiveBallAtPosition) SetTargetPosition(dest info.Position) {
	rbap.target_position = dest
}
 
