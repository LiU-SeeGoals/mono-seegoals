package roles

import (
	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	sm "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type TargetContext interface {
	GetTargetPosition() info.Position
	GetFromPosition() info.Position
}

type FreezableTargetContext interface {
	FreezeTarget()
	ResetTarget()
}

type AlignState struct {
	Gi              *info.GameInfo
	RobotId         info.ID
	Team            info.Team
	Name            sm.StateName
	ActivityHandler *ai.ActivityHandler
	Ctx             TargetContext
}

func (s *AlignState) Initialize() {
	if ctx, ok := s.Ctx.(FreezableTargetContext); ok {
		ctx.FreezeTarget()
	}
}

func (s *AlignState) GetName() sm.StateName {
	return s.Name
}
func enemyCloseToBall(gi *info.GameInfo, team info.Team, ballPos info.Position, radius float64) bool {
	enemies := gi.State.GetOtherTeam(team)

	for _, enemy := range enemies {
		enemyPos, err := enemy.GetPosition()

		if err != nil {
			continue
		}

		if ballPos.Dist2d(enemyPos) < radius {
			return true
		}
	}

	return false
}
func (s *AlignState) Update() sm.EventName {

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
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            sm.StateName
	activityHandler *ai.ActivityHandler
	ctx             TargetContext
}

func (s *SupportState) Initialize() {
	if ctx, ok := s.ctx.(FreezableTargetContext); ok {
		ctx.ResetTarget()
	}
}

func (s *SupportState) GetName() sm.StateName {
	return s.name
}

func (s *SupportState) Update() sm.EventName {

	activity := act.NewMoveToPosition(s.team, s.robotId, s.ctx.GetFromPosition())
	s.activityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.gi)
	if achieved {
		return "WAITING"
	}
	return "NONE"
}

type ReceivePassState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            sm.StateName
	target          *info.Position
	activityHandler *ai.ActivityHandler
}

func (s *ReceivePassState) Initialize() {
}

func (s *ReceivePassState) GetName() sm.StateName {
	return s.name
}

func (s *ReceivePassState) Update() sm.EventName {
	target := *s.target
	ballPos, err := s.gi.State.GetBall().GetEstimatedPosition()
	if err == nil {
		target.Angle = target.AngleToPosition(ballPos)
	}

	activity := act.NewMoveToPosition(s.team, s.robotId, target)
	activity.SetUseRRT(false)
	activity.AvoidBall(false)
	activity.SetDribble(true)
	s.activityHandler.AddActivity(activity)

	possessor := s.gi.State.GetBall().GetPossessor()
	if possessor != nil && possessor.GetTeam() == s.team && possessor.GetID() == s.robotId {
		return "BALL_RECEIVED"
	}

	return "NONE"
}

type InterceptBallState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            sm.StateName
	activityHandler *ai.ActivityHandler
	ctx             TargetContext
}

func (s *InterceptBallState) Initialize() {
}

func (s *InterceptBallState) GetName() sm.StateName {
	return s.name
}

func (s *InterceptBallState) Update() sm.EventName {
	ballVel, ok := s.gi.State.GetTrackedBall().GetTrackedVelocity()
	if ok && ballVel.Norm2d() > 0.3 {
		activity := act.NewMoveToBall(s.team, s.robotId)
		s.activityHandler.AddActivity(activity)

		if activity.Achieved(s.gi) {
			return "BALL_OWNER"
		}

		return "NONE"
	}

	targetPos := s.ctx.GetTargetPosition()
	fromPos := s.ctx.GetFromPosition()
	activity := act.NewAlign(s.team, s.robotId, targetPos, fromPos)
	s.activityHandler.AddActivity(activity)

	if activity.Achieved(s.gi) {
		return "BALL_OWNER"
	}

	return "NONE"
}

type KickState struct {
	Name            sm.StateName
	RobotId         info.ID
	Team            info.Team
	KickAct         *act.KickBall
	Gi              *info.GameInfo
	ActivityHandler *ai.ActivityHandler
	Ctx             TargetContext
}

func (s *KickState) Initialize() {

	s.KickAct = act.NewKickBall(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.ActivityHandler.AddActivity(s.KickAct)
}

func (s *KickState) GetName() sm.StateName {
	return s.Name
}

func (s *KickState) Update() sm.EventName {
	if s.KickAct.Achieved(s.Gi) {
		if ctx, ok := s.Ctx.(FreezableTargetContext); ok {
			ctx.ResetTarget()
		}
		return "KICKED"
	}
	return "NONE"
}
