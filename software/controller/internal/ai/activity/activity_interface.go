package activity

import (
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type Activity interface {
	GetAction(*info.GameInfo) action.Action
	Achieved(*info.GameInfo) bool
	String() string
	GetID() info.ID
}

// Continuer is implemented by activities that keep state between frames, such
// as look-ahead hysteresis or alignment latches. Planners build a new activity
// every frame; when it replaces one of the same kind for the same robot,
// ai.ActivityHandler.AddActivity calls ContinueFrom so that state carries over
// instead of resetting each frame. Implementations ignore activities of
// another type or robot.
type Continuer interface {
	ContinueFrom(previous Activity)
}
