package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type StateName string
type EventName string


type State interface {
	Update() EventName
	Initialize()
	GetName() StateName
}

type StateMachine struct {
	currentState    State
	StateTransitions map[StateName]map[EventName]State
}

func NewStateMachine(initial State) *StateMachine {

	sm := &StateMachine{
		StateTransitions: make(map[StateName]map[EventName]State),
	}
	sm.ChangeState(initial)
	return sm
}

func (sm *StateMachine) AddTransition(from StateName, event EventName, to State) {
	if sm.StateTransitions[from] == nil {
		sm.StateTransitions[from] = make(map[EventName]State)
	}
	sm.StateTransitions[from][event] = to
}

func (sm *StateMachine) TriggerEvent(event EventName) {

	stateName := sm.currentState.GetName()

	transitions := sm.StateTransitions[stateName]
	newState, ok := transitions[event]
	if !ok {
		return
	}
	fmt.Println(event, sm.currentState.GetName(), "->", newState.GetName())
	sm.ChangeState(newState)
}

func (sm *StateMachine) ChangeState(newState State) {
	sm.currentState = newState
	newState.Initialize()
}

func (sm *StateMachine) Update() {
	if sm.currentState == nil{
		panic("States have not been initialized!!")
	}
	event := sm.currentState.Update()
	sm.TriggerEvent(event)
}

type TargetContext interface{
	GetTargetPosition() info.Position
	GetFromPosition() info.Position
}

type AlignState struct {
	Gi *info.GameInfo
	RobotId info.ID
	Team info.Team
	Name StateName
	ActivityHandler *ai.ActivityHandler
	Ctx TargetContext
}

func (s *AlignState) Initialize() {
}

func (s *AlignState) GetName() StateName {
	return s.Name
}

func (s *AlignState) Update() EventName{

	activity := act.NewAlign(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
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
	name StateName
	activityHandler *ai.ActivityHandler
	ctx TargetContext
}

func (s *SupportState) Initialize() {
}

func (s *SupportState) GetName() StateName {
	return s.name
}

func (s *SupportState) Update() EventName{

	activity := act.NewMoveToPosition(s.team, s.robotId, s.ctx.GetFromPosition())
	s.activityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.gi)
	if achieved {
		return "WAITING"
	}
	return "NONE"
}

type KickState struct {
	Name StateName
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

func (s *KickState) GetName() StateName {
	return s.Name
}

func (s *KickState) Update() EventName {
	if(s.KickAct.Achieved(s.Gi)){
		return "KICKED"
	}
	return "NONE"
}
