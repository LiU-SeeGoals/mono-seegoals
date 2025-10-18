package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type ReceiveBallFromPlayer struct {
	GenericComposition
	team     info.Team
	id       info.ID
	other_id info.ID
}

func (fb *ReceiveBallFromPlayer) String() string {
	return fmt.Sprintf("Robot %d, ReceiveBallFromPlayer(%d)", fb.id, fb.other_id)
}

func NewReceiveBallFromPlayer(team info.Team, id info.ID, other_id info.ID) *ReceiveBallFromPlayer {
	return &ReceiveBallFromPlayer{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		other_id: other_id,
	}
}

func (fb *ReceiveBallFromPlayer) GetAction(gi *info.GameInfo) action.Action {

	myTeam := gi.State.GetTeam(fb.team)
	robotReceiver := myTeam[fb.id]
	if !robotReceiver.IsActive() {
		return nil
	}
	robotKicker := myTeam[fb.other_id]
	receiverPos, err1 := robotReceiver.GetPosition()
	kickerPos, err2 := robotKicker.GetPosition()

	if err1 != nil || err2 != nil {
		Logger.Errorf("Position retrieval failed - Receiver: %v, Kicker: %v\n", err1, err2)
		return NewStop(fb.id).GetAction(gi)
	}

	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Ball: %v\n", err)
		return NewStop(fb.id).GetAction(gi)
	}

	dxBall := float64(receiverPos.X - ballPos.X)
	dyBall := float64(receiverPos.Y - ballPos.Y)
	distanceBall := math.Sqrt(math.Pow(dxBall, 2) + math.Pow(dyBall, 2))

	dx := float64(kickerPos.X - receiverPos.X)
	dy := float64(kickerPos.Y - receiverPos.Y)
	distance := math.Sqrt(math.Pow(dx, 2) + math.Pow(dy, 2))

	if distanceBall < (distance / 3) {
		move := NewMoveToBall(fb.team, fb.id)
		moveAction := move.GetAction(gi)
		moveAction.(*action.MoveTo).Dribble = true
		return moveAction
	}

	targetAngle := math.Atan2(math.Abs(dyBall), math.Abs(dxBall))
	if dx > 0 {
		targetAngle = math.Pi - targetAngle
	}
	if dy > 0 {
		targetAngle = -targetAngle
	}

	//because opposit angle
	if targetAngle > 0 {
		targetAngle -= math.Pi
	} else {
		targetAngle += math.Pi
	}

	pos := info.Position{X: receiverPos.X, Y: receiverPos.Y, Z: receiverPos.Z, Angle: float64(targetAngle)}
	move := NewMoveWithBallToPosition(fb.team, fb.id, pos)
	return move.GetAction(gi)//.MoveWithBallToPosition(pos, gi)

	//Also needs to fix so that it moves out of the way if there is an obsticle

}

func (fb *ReceiveBallFromPlayer) Achieved(gi *info.GameInfo) bool {
	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Ball: %v\n", err)
		return false
	}

	receiverPos, err := gi.State.GetTeam(fb.team)[fb.id].GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return false
	}
	distance := ballPos.Distance(receiverPos)
	const distance_threshold = 10
	ballRecived := distance <= distance_threshold
	return ballRecived
}

// get id
func (fb *ReceiveBallFromPlayer) GetID() info.ID {
	return fb.id
}

