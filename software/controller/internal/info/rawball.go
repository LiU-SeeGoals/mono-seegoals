package info

import (
	"container/list"
	"errors"
	"time"
	"sync"
	"fmt"
)

type rawBallPos struct {
	pos  Position
	time int64
}

type rawBall struct {
	mu sync.Mutex
	history *list.List
	historyCapacity int
}

func (b *rawBall) SetPositionTime(x, y, z float64, time int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.history.Len() >= b.historyCapacity {
		element := b.history.Back()
		b.history.Remove(element)

		ball := element.Value.(*rawBallPos)

		ball.pos.X = x
		ball.pos.Y = y
		ball.pos.Z = z
		ball.time = time

		b.history.PushFront(ball)
	} else {
		pos := Position{x, y, z, 0}
		b.history.PushFront(&rawBallPos{pos, time})
	}
}

func (b *rawBall) GetVelocity() (Vec2, error){
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.history.Len() < 2 {
		return Vec2{0,0}, fmt.Errorf("No balls in history")
	}

	element := b.history.Front()
	ball := element.Value.(*rawBallPos)

	element2 := element.Next()
	if (element2 == nil){
		return Vec2{0,0}, fmt.Errorf("Ball 2 nil why is it nil?")
	}
	ball2 := element2.Value.(*rawBallPos)

	dt := float64(ball.time) - float64(ball2.time)
	dPos := ball.pos.Sub(&ball2.pos)

	return Vec2{dPos.X/dt, dPos.Y/dt}, nil
}

func (b *rawBall) GetPositionTime() (Position, int64, error) {
	if b.history.Len() == 0 {
		return Position{}, 0, errors.New("No position in history for ball")
	}
	ball := b.history.Front().Value.(*rawBallPos)

	return ball.pos, ball.time, nil
}

// get age
func (b *rawBall) GetAge() int64 {
	_, ballTime, err := b.GetPositionTime()
	if err != nil {
		return 0
	}

	return time.Now().UnixMilli() - ballTime
}

func (b *rawBall) GetPosition() (Position, error) {
	pos, _, err := b.GetPositionTime()

	return pos, err
}


func (b *rawBall) GetLatestTwoPositionsTime() (Position, int64, Position, int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.history.Len() < 2 {
		return Position{}, 0, Position{}, 0, errors.New("Not enough ball history")
	}

	currentElement := b.history.Front()
	previousElement := currentElement.Next()
	if previousElement == nil {
		return Position{}, 0, Position{}, 0, errors.New("Missing previous ball position")
	}

	current := currentElement.Value.(*rawBallPos)
	previous := previousElement.Value.(*rawBallPos)

	return current.pos, current.time, previous.pos, previous.time, nil
}