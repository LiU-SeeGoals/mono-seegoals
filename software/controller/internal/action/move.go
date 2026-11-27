package action

import "gonum.org/v1/gonum/mat"

// Move drives in a robot-local direction: forward is (0, 1), right is (1, 0).
// The vector's length sets the speed.
type Move struct {
	Id        int
	Direction *mat.VecDense // (x, y)
}

func (m *Move) RobotID() int { return m.Id }
