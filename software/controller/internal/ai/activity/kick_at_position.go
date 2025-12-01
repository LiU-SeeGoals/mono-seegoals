package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type KickAtPosition struct {
	GenericComposition
	targetPosition info.Position
	retrievingBall bool
}

func (k *KickAtPosition) String() string {
	return fmt.Sprintf("(Robot %d, KickAtPosition(%v))", k.id, k.targetPosition)
}

func NewKickAtPosition(team info.Team, id info.ID, targetPosition info.Position) *KickAtPosition {
	return &KickAtPosition{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition: targetPosition,
		retrievingBall: true,
	}
}

func (kp *KickAtPosition) GetAction(gi *info.GameInfo) action.Action {
	const accuracy = 0.1 // Magic number warning

	robot := gi.State.GetRobot(kp.id, kp.team)
	ball := gi.State.GetBall()

	ballPos, _ := ball.GetEstimatedPosition()
	unitVector := kp.targetPosition.Sub(&ballPos).Normalize()
	unitVector.Angle = 0
	// We can now get a position that is behind the ball, opposite of the targetposition
	possessor := ball.GetPossessor()
	inPossession := possessor == robot

	kp.retrievingBall = !inPossession

	facingBall := robot.Facing(ballPos, accuracy*2)
	facingTarget := robot.Facing(kp.targetPosition, accuracy*2)

	if kp.retrievingBall && facingBall && facingTarget {
		fmt.Printf("[KickAtPosition] robot %d retrieving: aligned with ball and target -> MoveToBall\n", kp.id)
		return NewMoveToBall(kp.team, kp.id).GetAction(gi)
	} else if kp.retrievingBall {
		ballMargin := unitVector.Scale(350) // MagicNumber (100mm behind ball)
		lineUpPos := ballPos.Sub(&ballMargin)

		currPos, _ := robot.GetPosition()
		targetAngle := lineUpPos.AngleToPosition(kp.targetPosition)
		lineUpPos.Angle = currPos.Angle + info.NormalizeAngleDelta(targetAngle, currPos.Angle)

		move := NewMoveToPosition(kp.team, kp.id, lineUpPos)
		move.AvoidBall(true)
		moveAction := move.GetMoveToAction(gi)
		fmt.Printf("[KickAtPosition] robot %d retrieving: lining up at (%.1f, %.1f) angle %.1f° (facingBall=%v facingTarget=%v)\n",
			kp.id, lineUpPos.X, lineUpPos.Y, radToDeg(lineUpPos.Angle), facingBall, facingTarget)
		return &moveAction
	}

	// Robot is in possesion of the ball, but is not facing the target
	if inPossession && !robot.Facing(kp.targetPosition, accuracy) {
		// Robust Turn: Command the robot to be in the shooting position.
		// Shooting position = Robot center is ~90-100mm behind the ball, aligned with target.
		// By commanding this position, the path planner will handle the orbiting/turning.

		vecToTarget := kp.targetPosition.Sub(&ballPos).Normalize()
		// 150mm offset: roughly robot radius (90mm) + small push margin
		offset := vecToTarget.Scale(100)
		shootingPos := ballPos.Sub(&offset)
		shootingPos.Angle = ballPos.AngleToPosition(kp.targetPosition)

		move := NewMoveToPosition(kp.team, kp.id, shootingPos)
		moveAction := move.GetMoveToAction(gi)
		moveAction.Dribble = true

		// fmt.Printf("[KickAtPosition] robot %d aligning: target angle %.1f°\n", kp.id, radToDeg(shootingPos.Angle))
		return &moveAction
	}

	// If we get here, it means we are in possession of the ball and that the robot is facing the target.
	// So we move forward and shoot
	runUpDistance := unitVector.Scale(100)
	destination := ballPos.Add(&runUpDistance)
	// FIX: Set the correct facing angle toward the target
	destination.Angle = ballPos.AngleToPosition(kp.targetPosition)

	moveAction := action.MoveTo{
		Id:   int(kp.id),
		Dest: destination,
	}
	KickSpeed := float32(5)
	moveAction.KickSpeed = int(KickSpeed)
	fmt.Printf("[KickAtPosition] robot %d shooting: run-up target (%.1f, %.1f) angle %.1f° angleToTarget %.1f° kickSpeed=%d\n",
		kp.id, destination.X, destination.Y, radToDeg(possessor.DribblerPos().Angle), radToDeg(destination.AngleToPosition(kp.targetPosition)), moveAction.KickSpeed)
	return &moveAction
}

func (kp *KickAtPosition) Achieved(gi *info.GameInfo) bool {
	if kp.retrievingBall {
		return false
	}

	robot := gi.State.GetRobot(kp.id, kp.team)
	return gi.State.LostBall(robot)
}

func (kp *KickAtPosition) GetID() info.ID {
	return kp.id
}

// Helper
func radToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}
