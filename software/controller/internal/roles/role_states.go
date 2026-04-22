package roles

import (
	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	sm "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type TargetContext interface{
	GetTargetPosition() info.Position
	GetFromPosition() info.Position
}

type AlignState struct {
	Gi *info.GameInfo
	RobotId info.ID
	Team info.Team
	Name sm.StateName
	ActivityHandler *ai.ActivityHandler
	Ctx TargetContext
}

func (s *AlignState) Initialize() {
}

func (s *AlignState) GetName() sm.StateName {
	return s.Name
}
func enemyCloseToBall(gi *info.GameInfo, team info.Team, ballPos info.Position, radius float64) bool {
    enemies := gi.State.GetOtherTeam(team)

    for _, enemy := range enemies {
        enemyPos,err := enemy.GetPosition()

		if err != nil {
			continue
		}

        if ballPos.Dist2d(enemyPos) < radius {
            return true
        }
    }

    return false
}
func (s *AlignState) Update() sm.EventName{

	targetPos := s.Ctx.GetTargetPosition()
	fromPos := s.Ctx.GetFromPosition()

	var activity act.Activity

	if enemyCloseToBall(s.Gi, s.Team, fromPos, 1000) {
		activity = act.NewDirectAlign(s.Team, s.RobotId, targetPos, fromPos)
	} else {
		activity = act.NewAlign(s.Team, s.RobotId, targetPos, fromPos)
	}
	//activity := act.NewAlign(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.ActivityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.Gi)
	if achieved {
		return "ALIGNED"
	}
	return "NONE"
}

type SupportState struct {
	gi *info.GameInfo
	robotId info.ID
	team info.Team
	name sm.StateName
	activityHandler *ai.ActivityHandler
	ctx TargetContext
}

func (s *SupportState) Initialize() {
}

func (s *SupportState) GetName() sm.StateName {
	return s.name
}

func (s *SupportState) Update() sm.EventName{

	activity := act.NewMoveToPosition(s.team, s.robotId, s.ctx.GetFromPosition())
	s.activityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.gi)
	if achieved {
		return "WAITING"
	}
	return "NONE"
}

type KickState struct {
	Name sm.StateName
	RobotId info.ID
	Team info.Team
	KickAct *act.KickBall
	Gi *info.GameInfo
	ActivityHandler *ai.ActivityHandler
	Ctx TargetContext
}

func (s *KickState) Initialize() {

	s.KickAct = act.NewKickBall(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.ActivityHandler.AddActivity(s.KickAct)
}

func (s *KickState) GetName() sm.StateName {
	return s.Name
}

func (s *KickState) Update() sm.EventName {
	if(s.KickAct.Achieved(s.Gi)){
		return "KICKED"
	}
	return "NONE"
}