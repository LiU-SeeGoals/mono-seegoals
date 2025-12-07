package ai

import (
	"fmt"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type KickAtPosition struct {
	GenericComposition
	targetPosition info.Position
	retrievingBall bool
}

func (k *KickAtPosition) String() string {
	return fmt.Sprintf("(Robot %d, KickAtPosition(%d))", k.id, k.targetPosition)
}

func NewKickAtPosition(team info.Team, id info.ID, targetPosition info.Position) *KickAtPosition {
	return &KickAtPosition{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition: targetPosition,
	}
}

func (kp *KickAtPosition) GetAction(gi *info.GameInfo) action.Action {
	robot := gi.State.GetRobot(kp.id, kp.team)
	robotPos, err := robot.GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Kicker: %v\n", err)
		return NewStop(kp.id).GetAction(gi)
	}
	angleToTarget := robotPos.AngleToPosition(kp.targetPosition)

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


	// return &action

	// Face target
	if !robot.Facing(kp.targetPosition, 0.1) {
		Logger.Debug("Rotating to target")

		angleToTarget = robotPos.AngleToPosition(kp.targetPosition)
		robotPos.Angle = angleToTarget
		return NewMoveWithBallToPosition(kp.team, kp.id, robotPos).GetAction(gi)
	}

	Logger.Debug("Kicking")

	// Kick
	action := action.Kick{
		Id:        int(kp.id),
		KickSpeed: 4,
	}

	return &action
}

func (k *KickAtPosition) Achieved(gi *info.GameInfo) bool {
	robot := gi.State.GetRobot(k.id, k.team)
	return gi.State.LostBall(robot)
}

func (k *KickAtPosition) GetID() info.ID {
	return k.id
}
