package info

import (
	"container/list"
	// "fmt"

)

type Ball struct {
	rawBall
	possessor         *Robot
	estimatedPosition Position
	estimateKind      BallEstimateKind
	holder            *Robot
}

type BallEstimateKind int8

const (
	BallUnseen BallEstimateKind = iota
	BallObserved
	BallInDribbler
)

const (
	BallUnseenAfterMs    = 100
	dribblerReachForward = Center2DribblerDist + BallRadius + 20
	holderUnseenAfterMs  = 250
)

func NewBall(historyCapacity int) *Ball {
	return &Ball{
		rawBall: rawBall{
			history:         list.New(),
			historyCapacity: historyCapacity,
		},
		possessor: nil,
	}
}

// get position
func (b *Ball) GetEstimatedPosition() (Position, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.estimatedPosition, nil
}

// set position
func (b *Ball) SetEstimatedPosition(pos Position) {
	b.setEstimate(pos, BallObserved, nil)
}

func (b *Ball) setEstimate(pos Position, kind BallEstimateKind, holder *Robot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.estimatedPosition = pos
	b.estimateKind = kind
	b.holder = holder
}

func (b *Ball) GetEstimateKind() (BallEstimateKind, *Robot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.estimateKind, b.holder
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
	count := 0

	for e := b.history.Front().Next(); e != nil; e = e.Next() {
		ball2 := e.Value.(*rawBallPos)
		dt := float64(ball2.time - ball.time)
		if dt == 0 {
			continue
		}
		dPos := ball2.pos.Sub(&ball.pos)
		scaled := dPos.Scale(1 / dt)
		sum_deltas = sum_deltas.Add(&scaled)
		count++
	}
	if count == 0 {
		return Position{0, 0, 0, 0}
	}
	return sum_deltas.Scale(1 / float64(count))
}

func (b *Ball) GetVelocity2() (Vec2, error){

	return b.rawBall.GetVelocity()
	// if b.history.Len() < 2 {
	// 	return Vec2{0,0}, fmt.Errorf("No balls in history")
	// }
	//
	// element := b.history.Front()
	// ball := element.Value.(*rawBallPos)
	//
	// element2 := element.Next()
	// if (element2 == nil){
	// 	return Vec2{0,0}, fmt.Errorf("Ball 2 nil why is it nil?")
	// }
	// ball2 := element2.Value.(*rawBallPos)
	//
	// dt := float64(ball.time) - float64(ball2.time)
	// dPos := ball.pos.Sub(&ball2.pos)
	//
	// return Vec2{dPos.X/dt, dPos.Y/dt}, nil
}

func (b *Ball) GetLatestTwoPositionsTime() (Position, int64, Position, int64, error) {
	return b.rawBall.GetLatestTwoPositionsTime()
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
