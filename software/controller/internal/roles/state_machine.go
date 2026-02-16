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
		fmt.Println("States have not been initialized!!")
	}
	event := sm.currentState.Update()
	sm.TriggerEvent(event)
}

type TargetContext interface{
	getTargetPosition() info.Position
	getFromPosition() info.Position
}

type AlignState struct {
	gi *info.GameInfo
	robotId info.ID
	team info.Team
	name StateName
	activityHandler *ai.ActivityHandler
	ctx TargetContext
}

func (s *AlignState) Initialize() {
}

func (s *AlignState) GetName() StateName {
	return s.name
}

func (s *AlignState) Update() EventName{

	activity := act.NewAlign(s.team, s.robotId, s.ctx.getTargetPosition(), s.ctx.getFromPosition())
	s.activityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.gi)
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

	activity := act.NewMoveToPosition(s.team, s.robotId, s.ctx.getFromPosition())
	s.activityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.gi)
	if achieved {
		return "WAITING"
	}
	return "NONE"
}

type KickState struct {
	name StateName
	robotId info.ID
	team info.Team
	kickAct *act.KickBall
	gi *info.GameInfo
	handle *ai.ActivityHandler
	ctx TargetContext
}

func (s *KickState) Initialize() {

	s.kickAct = act.NewKickBall(s.team, s.robotId, s.ctx.getTargetPosition(), s.ctx.getFromPosition())
	s.handle.AddActivity(s.kickAct)
}

func (s *KickState) GetName() StateName {
	return s.name
}

func (s *KickState) Update() EventName {
	if(s.kickAct.Achieved(s.gi)){
		return "KICKED"
	}
	return "NONE"
}
