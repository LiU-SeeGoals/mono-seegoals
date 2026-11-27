package roles

import (
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type GoalieDefendState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            state_machine.StateName
	activityHandler *ai.ActivityHandler
}

func (s *GoalieDefendState) Initialize() {
}

func (s *GoalieDefendState) GetName() state_machine.StateName {
	return s.name
}

func (s *GoalieDefendState) Update() state_machine.EventName {
	s.activityHandler.AddActivity(activity.NewGoalie(s.team, s.robotId))
	return "NONE"
}
