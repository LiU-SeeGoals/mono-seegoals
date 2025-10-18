package ai

import (
	"fmt"
	"math"

	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type KickToPlayer struct {
	GenericComposition
	// MovementComposition
	other_id info.ID
}

func (k *KickToPlayer) String() string {
	return fmt.Sprintf("(Robot %d, KickToPlayer(%d))", k.id, k.other_id)
}

func NewKickToPlayer(team info.Team, id info.ID, other_id info.ID) *KickToPlayer {
	return &KickToPlayer{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		other_id: other_id,
	}
}

func (fb *KickToPlayer) GetAction(gi *info.GameInfo) action.Action {
	myTeam := gi.State.GetTeam(fb.team)
	robotKicker := myTeam[fb.id]
	if !robotKicker.IsActive() {
		return nil
	}
	robotReciever := myTeam[fb.other_id]

	kickerPos, err1 := robotKicker.GetPosition()
	recieverPos, err2 := robotReciever.GetPosition()

	if err1 != nil || err2 != nil {
		Logger.Errorf("Position retrieval failed - Kicker: %v, Receiver: %v\n", err1, err2)
		return NewStop(fb.id).GetAction(gi)
	}
	

	dx := float64(kickerPos.X - recieverPos.X)
	dy := float64(kickerPos.Y - recieverPos.Y)
	distance := math.Sqrt(dx*dx + dy*dy)

	targetAngle := math.Atan2(math.Abs(dy), math.Abs(dx))
	if dx > 0 {
		targetAngle = math.Pi - targetAngle
	}
	if dy > 0 {
		targetAngle = -targetAngle
	}

	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Ball: %v\n", err)
		return NewStop(fb.id).GetAction(gi)
	}
	dxBall := float64(kickerPos.X - ballPos.X)
	dyBall := float64(kickerPos.Y - ballPos.Y)
	distanceBall := math.Sqrt(math.Pow(dxBall, 2) + math.Pow(dyBall, 2))

	// Rotate to target
	if math.Abs(float64(kickerPos.Angle)-float64(targetAngle)) > 0.05 {
		pos := info.Position{X: kickerPos.X, Y: kickerPos.Y, Z: kickerPos.Z, Angle: float64(targetAngle)}
		move := NewMoveWithBallToPosition(fb.team, fb.id, pos)
		return move.GetAction(gi)
	}

	// kick
	if distanceBall > 90 {
		move := NewMoveToBall(fb.team, fb.id)
		return move.GetAction(gi)
	} else {
		kickAct := &action.Kick{}
		kickAct.Id = int(robotKicker.GetID())

		// Compute the kick speed as a function of the distance to target
		normDistance := float64(distance) / 10816
		kickSpeed := 1 + int(4*normDistance)
		kickAct.KickSpeed = int(math.Min(math.Max(float64(kickSpeed), 1), 5))
		return kickAct
	}

	//Needs to add that is doesn't kick if there is an obsicle

}

func (k *KickToPlayer) Achieved(gi *info.GameInfo) bool {
	ballPos, err := gi.State.GetBall().GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Ball: %v\n", err)
		return false
	}
	receiverPos, err := gi.State.GetTeam(k.team)[k.other_id].GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Receiver: %v\n", err)
		return false
	}
	distance := ballPos.Distance(receiverPos)
	const distance_threshold = 10
	ballRecived := distance <= distance_threshold
	return ballRecived
}

func (k *KickToPlayer) GetID() info.ID {
	return k.id
}

