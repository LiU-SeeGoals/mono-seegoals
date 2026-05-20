package roles

import (
	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type GoalieDefendState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            StateName
	activityHandler *ai.ActivityHandler
}

func (s *GoalieDefendState) Initialize() {
}

func (s *GoalieDefendState) GetName() StateName {
	return s.name
}

func (s *GoalieDefendState) Update() EventName {
	s.activityHandler.AddActivity(act.NewGoalie(s.team, s.robotId))
	return "NONE"
}
