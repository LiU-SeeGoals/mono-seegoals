package info

import "fmt"

type InstructionType int

const (
	MoveToPosition InstructionType = 0 // Move to a specific position (static)
	MoveToBall     InstructionType = 1 // Move to the ball (dynamic)

	MoveWithBallToPosition InstructionType = 2 // Move with the ball to a specific position (static)

	KickToPlayer   InstructionType = 3 // Kick the ball to a specific player (dynamic)
	KickToGoal     InstructionType = 4 // Kick the ball to some open space in the goal (dynamic)
	KickToPosition InstructionType = 5 // Kick the ball to a specific position (static)

	ReceiveBallFromPlayer InstructionType = 6 // Receive the ball from a specific player (dynamic). Before a kick is made, adjust the position to have a better chance of receiving the ball.
	ReceiveBallAtPosition InstructionType = 7 // Receive the ball at a expected position (dynamic). Make to be at the expected position when the ball is kicked. But adjust the position to have a better chance of receiving the ball after the kick.

	BlockEnemyPlayerFromPosition InstructionType = 8  // Body block an enemy player from a specific position (dynamic)
	BlockEnemyPlayerFromBall     InstructionType = 9  // Body block an enemy player from the ball (dynamic)
	BlockEnemyPlayerFromGoal     InstructionType = 10 // Body block an enemy player from the goal (dynamic). Make sure that the enemy does not have a clear shot at the goal.
	BlockEnemyPlayerFromPlayer   InstructionType = 11 // Body block an enemy player from a specific player (dynamic)

	Goalie	InstructionType = 12 //Goalie instruction
)

type Instruction struct {
	Type     InstructionType
	Id       ID
	OtherId  ID
	Position Position
}

func (inst *Instruction) ToDTO() string {
	dto := fmt.Sprintf("Instruction{Id: %d, Position: %s}", inst.Id, inst.Position.ToDTO())
	return dto
}

type GamePlan struct {
	Valid        bool
	Team         Team
	Instructions []*Instruction
}

func NewGamePlan() *GamePlan {
	gp := GamePlan{}
	return &gp
}

func (gp *GamePlan) ToDTO() string {
	dto := fmt.Sprintf("GamePlan{Valid: %t, Team: %d, Instructions: [", gp.Valid, gp.Team)
	for _, instruction := range gp.Instructions {
		dto += instruction.ToDTO() + ", "
	}

	return dto
}
