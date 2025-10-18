package info

import (
	"container/list"
	"github.com/LiU-SeeGoals/controller/internal/tracker"
)

type Ball struct {
	rawBall
	possessor         *Robot
	tracker           *tracker.BallTracker
	estimatedPosition Position
}

func NewBall(historyCapacity int) *Ball {
	tracker := tracker.NewBallTracker()
	return &Ball{
		rawBall: rawBall{
			history:         list.New(),
			historyCapacity: historyCapacity,
		},
		tracker:   tracker,
		possessor: nil,
	}
}

// get position
func (b *Ball) GetEstimatedPosition() (Position, error) {
	return b.estimatedPosition, nil
}

// set position
func (b *Ball) SetEstimatedPosition(pos Position) {
	b.estimatedPosition = pos
}

func (b *Ball) SetPossessor(robot *Robot) {
	b.possessor = robot
}

func (b *Ball) GetPossessor() *Robot {
	return b.possessor
}

func (b *Ball) GetVelocity() Position {

	if b.history.Len() < 2 {
		return Position{0, 0, 0, 0}
	}

	element := b.history.Front()
	ball := element.Value.(*rawBallPos)

	sum_deltas := Position{}

	for e := b.history.Front().Next(); e != nil; e = e.Next() {
		ball2 := e.Value.(*rawBallPos)
		dPos := ball2.pos.Sub(&ball.pos)
		dt := float64(ball2.time - ball.time)
		scaled := dPos.Scale(1 / dt)
		sum_deltas = sum_deltas.Add(&scaled)
	}
	return sum_deltas.Scale(1 / float64(b.history.Len()-1))

}

type BallDTO struct {
	PosX float64
	PosY float64
	PosZ float64
	VelX float64
	VelY float64
	VelZ float64
}

func (b *Ball) ToDTO() BallDTO {
	pos, _ := b.GetPosition()
	vel := b.GetVelocity()
	return BallDTO{
		PosX: pos.X,
		PosY: pos.Y,
		PosZ: pos.Z,
		VelX: vel.X,
		VelY: vel.Y,
		VelZ: vel.Z,
	}
}
