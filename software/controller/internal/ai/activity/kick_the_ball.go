package ai

import (
	"fmt"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type KickTheBall struct {
	GenericComposition
	targetPosition info.Position
	retrievingBall bool
}

func (k *KickTheBall) String() string {
	return fmt.Sprintf("(Robot %d, KickTheBall(%d))", k.id, k.targetPosition)
}

func NewKickTheBall(team info.Team, id info.ID, targetPosition info.Position) *KickTheBall {
	return &KickTheBall{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition: targetPosition,
	}
}

func (kp *KickTheBall) GetAction(gi *info.GameInfo) action.Action {
	robot := gi.State.GetRobot(kp.id, kp.team)

	if !kp.retrievingBall { // Check if it lost the ball
		kp.retrievingBall = gi.State.LostBall(robot)
	}

	move := NewMoveToBall(kp.team, kp.id)
	if kp.retrievingBall && move.Achieved(gi) { // We have achivied in retrieving the ball
		Logger.Debug("MoveWithBallToPosition: Ball retrieved")
		kp.retrievingBall = false

	} else if kp.retrievingBall { // We are still working on getting the ball
		Logger.Debug("MoveWithBallToPosition: Retrieving ball")
		return move.GetAction(gi)

	}
	action := action.Kick{
		Id:        int(kp.id),
		KickSpeed: 4,
	}

	return &action
}

func (k *KickTheBall) Achieved(gi *info.GameInfo) bool {
	robot := gi.State.GetRobot(k.id, k.team)
	return gi.State.LostBall(robot)
}

func (k *KickTheBall) GetID() info.ID {
	return k.id
}
