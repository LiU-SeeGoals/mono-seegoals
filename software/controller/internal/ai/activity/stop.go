package ai

import (
	"fmt"
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type Stop struct {
	GenericComposition
}

func (m *Stop) String() string {
	return fmt.Sprintf("Stop(%d)", m.id)
}

func NewStop(id info.ID) *Stop {
	return &Stop{
		GenericComposition: GenericComposition{
			id: id,
		},
	}
}

func (m *Stop) GetAction(gi *info.GameInfo) action.Action {
	act := action.Stop{}
	act.Id = int(m.id)
	return &act
}

func (m *Stop) Achieved(gi *info.GameInfo) bool {
	// In stop position untill slow brain tells it to move
	// Will never automically be achieved
	return false
}

func (m *Stop) GetID() info.ID {
	return m.id
}

