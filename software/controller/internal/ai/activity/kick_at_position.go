package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type KickAtPosition struct {
	GenericComposition
	targetPosition   info.Position
	simpleReposition bool
	retrievingBall   bool
}

func (k *KickAtPosition) String() string {
	return fmt.Sprintf("(Robot %d, KickAtPosition(%v))", k.id, k.targetPosition)
}

func NewKickAtPosition(team info.Team, id info.ID, targetPosition info.Position, simpleReposition bool) *KickAtPosition {
	return &KickAtPosition{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition:   targetPosition,
		simpleReposition: simpleReposition,
		retrievingBall:   true,
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
		return &moveAction
	}

	// Robot is in possesion of the ball, but is not facing the target
	if inPossession && !robot.Facing(kp.targetPosition, accuracy) {
		// Since we cannot move and rotate with the ball reliably,
		// we need to move around the ball, while also rotating towards the ball at the same time
		// until we are lined up with the target, aswell as the ball.

		// We assume possesion means that the ball in proximity to the dribbler
		robotPos, _ := robot.GetPosition()

		angleDelta := info.NormalizeAngleDelta(robotPos.Angle, ballPos.AngleToPosition(kp.targetPosition))

		direction := math.Copysign(1.0, angleDelta)

		vecBallToRobot := robotPos.Sub(&ballPos)
		angleFromBall := vecBallToRobot.Angle

		tangentAngle := angleFromBall + (direction * math.Pi / 2)
		// Positive direction: Turn left
		// Negative direction: Turn right
		stepSize := 150.0 // millimeters
		radius := 115.0   // WARNING MAGIC NUMBER

		var nextPosition info.Position
		// Simple repositioning should only move in straight lines, and then rotate
		if kp.simpleReposition {
			stepSize = 150.0 // millimeters
			nextPosition = robotPos.OnRadius(stepSize, tangentAngle)
			nextPosition.Angle = nextPosition.AngleToPosition(ballPos)
		} else {
			// More advanced arc-like repositioning (Currently only works in sim) that rotates and moves around the ball simultaniously.
			tangentPoint := robotPos.OnRadius(stepSize, tangentAngle)
			vecBallToNextPos := tangentPoint.Sub(&ballPos).Normalize().Scale(radius)
			nextPosition = ballPos.Add(&vecBallToNextPos)
			nextPosition.Angle = nextPosition.AngleToPosition(ballPos)
		}

		// Create move action to the current target
		moveAction := action.MoveTo{
			Id:      int(kp.id),
			Team:    kp.team,
			Dest:    nextPosition,
			Dribble: false,
		}
		moveAction.Dest.Angle = nextPosition.Angle
		return &moveAction

	}

	// If we get here, it means we are in possession of the ball and that the robot is facing the target.
	// So we move forward and shoot
	runUpDistance := unitVector.Scale(100)
	destination := ballPos.Add(&runUpDistance)
	moveAction := action.MoveTo{
		Id:   int(kp.id),
		Dest: destination,
		Team: kp.team,
	}
	KickSpeed := float32(5)
	moveAction.KickSpeed = int(KickSpeed)
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
