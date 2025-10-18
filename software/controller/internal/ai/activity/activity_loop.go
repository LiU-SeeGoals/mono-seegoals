package ai

import (
	"fmt"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type ActivityLoop struct {
	GenericComposition
	activities []Activity
	current    int
}

func (l *ActivityLoop) String() string {
	return fmt.Sprintf("ActivityLoop: %s", l.activities[l.current].String())
}

func NewActivityLoop(id info.ID, activities []Activity) *ActivityLoop {
	for _, a := range activities {
		if a.GetID() != id {
			panic("ActivityLoop: Activity ID does not match")
		}
	}
	return &ActivityLoop{
		GenericComposition: GenericComposition{
			id: id,
		},
		activities: activities,
		current:    0,
	}
}

func (l *ActivityLoop) GetAction(gi *info.GameInfo) action.Action {
	if l.activities[l.current].Achieved(gi) {
		l.current = (l.current + 1) % len(l.activities)
	}

	return l.activities[l.current].GetAction(gi)
}

func (l *ActivityLoop) Achieved(gi *info.GameInfo) bool {
	return false
}

func (l *ActivityLoop) GetID() info.ID {
	return l.id
}

