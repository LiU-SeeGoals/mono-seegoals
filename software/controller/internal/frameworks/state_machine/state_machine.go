package state_machine

import (
	"fmt"
)

type StateName string
type EventName string

type State interface {
	Update() EventName
	Initialize()
	GetName() StateName
}

type StateMachine struct {
	currentState     State
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
	if sm.currentState == nil {
		panic("States have not been initialized!!")
	}
	event := sm.currentState.Update()
	sm.TriggerEvent(event)
}

func (sm *StateMachine) CurrentStateName() StateName {
	return sm.currentState.GetName()
}
