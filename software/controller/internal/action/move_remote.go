package action

import (
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
	"gonum.org/v1/gonum/mat"
)

// Same as move but with the speed embedded, should only be usable
// when remote controlling the robot
type MoveRemote struct {
	Id        int
	Direction *mat.VecDense // 2D vector, first value is x, second is y
	Dest      info.Position
	Speed     int
	Dribble   bool
}

// Do nothing, only implemented to satisfy interface
func (i *MoveRemote) TranslateSim() *simulation.RobotCommand {
	id := uint32(i.Id)
	return &simulation.RobotCommand{
		Id: &id,
	}
}

func (s *MoveRemote) TranslateReal() *robot_action.Command {

	dribble := int32(0)
	if s.Dribble {
		dribble = 1
	}

	command := &robot_action.Command{
		CommandId: robot_action.ActionType_MOVE_ACTION,
		RobotId:   int32(s.Id),
		Direction: &robot_action.Vector2D{
			X: int32(s.Direction.AtVec(1)),
			Y: int32(s.Direction.AtVec(0)),
		},
		Dest: &robot_action.Vector3D{
			X: int32(s.Dest.Y),
			Y: int32(s.Dest.X),
			W: float32(s.Dest.Z),
		},
		KickSpeed:  int32(s.Speed),
		AngularVel: dribble,
	}
	return command
}

// Do nothing, only implemented to satisfy interface
func (s *MoveRemote) ToDTO() ActionDTO {
	return ActionDTO{
		Id: s.Id,
	}
}
