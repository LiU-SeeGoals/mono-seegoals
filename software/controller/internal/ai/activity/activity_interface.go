package ai

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
