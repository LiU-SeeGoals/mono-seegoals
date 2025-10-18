package ai

import (
	"fmt"
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type RamAtPosition struct {
	GenericComposition
	targetPosition  info.Position
	startWait       int64
	bumpedBall      bool
	reachedStartPos bool
}

func (k *RamAtPosition) String() string {
	return fmt.Sprintf("(Robot %d, RamAtPosition(%d))", k.id, k.targetPosition)
}

func NewRamAtPosition(team info.Team, id info.ID, targetPosition info.Position) *RamAtPosition {
	return &RamAtPosition{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition: targetPosition,
		bumpedBall:     false,
	}
}

func (kp *RamAtPosition) GetAction(gi *info.GameInfo) action.Action {
	if kp.bumpedBall {
		return NewStop(kp.id).GetAction(gi)
	}

	ballPos, err := gi.State.Ball.GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Kicker: %v\n", err)
		return NewStop(kp.id).GetAction(gi)
	}

	// Stop at ball
	if kp.bumpedBall {
		return NewStop(kp.id).GetAction(gi)
	}

	// Get in start position
	angleBallToStartPos := ballPos.AngleToPosition(kp.targetPosition) + math.Pi
	startPos := ballPos.OnRadius(500, angleBallToStartPos)

	move := NewMoveToPosition(kp.team, kp.id, startPos)
	move.AvoidBall(true)

	// In start position, RAM THE BALL
	if move.Achieved(gi) || kp.reachedStartPos {
		kp.reachedStartPos = true
		angleBallToTargetPos := ballPos.AngleToPosition(kp.targetPosition)
		targetPos := ballPos.OnRadius(500, angleBallToTargetPos)
		move = NewMoveToPosition(kp.team, kp.id, targetPos)
		fmt.Println("RamAtPosition: ", targetPos)
	}
	fmt.Println("startPos: ", startPos)
	return move.GetAction(gi)
}

func (k *RamAtPosition) Achieved(gi *info.GameInfo) bool {
	robotPos, err := gi.State.GetRobot(k.id, k.team).GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Kicker: %v\n", err)
		return false
	}

	ballPos, err := gi.State.Ball.GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Kicker: %v\n", err)
		return false
	}

	if robotPos.Distance(ballPos) < 91 {
		k.bumpedBall = true
		k.startWait = time.Now().UnixMilli()
	}

	waited := time.Now().UnixMilli() - k.startWait

	return waited > 5000 && k.bumpedBall

}

func (k *RamAtPosition) GetID() info.ID {
	return k.id
}
