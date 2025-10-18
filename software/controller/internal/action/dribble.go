package action

import (
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

type Dribble struct {
	Id int
	// set Dribbling, useless right now
	Dribble bool
}

func (d *Dribble) TranslateSim() *simulation.RobotCommand {
	id := uint32(d.Id)
	dribblerSpeed := float32(0)
	if d.Dribble {
		dribblerSpeed = 100 // in rpm, adjust as needed
	}

	return &simulation.RobotCommand{
		Id:            &id,
		DribblerSpeed: &dribblerSpeed,
	}
}
