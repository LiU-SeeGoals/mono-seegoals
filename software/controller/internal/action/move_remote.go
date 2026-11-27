package action

import (
	"github.com/LiU-SeeGoals/controller/internal/info"
	"gonum.org/v1/gonum/mat"
)

// MoveRemote is Move with an explicit speed, used by the gamepad remote
// control (internal/ui). Real robots only; the simulator ignores it.
type MoveRemote struct {
	Id        int
	Direction *mat.VecDense // (x, y)
	Dest      info.Position
	Speed     int
	Dribble   bool
}

func (m *MoveRemote) RobotID() int { return m.Id }
