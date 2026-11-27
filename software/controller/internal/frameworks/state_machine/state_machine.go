// Package state_machine is a small event-driven finite state machine used by
// the referee handler and the robot roles.
//
// States and events are typed (StateName, EventName). Declare them as named
// constants next to the machine that uses them instead of writing string
// literals at call sites, so a typo is a compile error rather than a silently
// ignored event.
package state_machine

import (
	"fmt"
	"sort"
)

type StateName string
type EventName string

// NoEvent is returned by State.Update when the state wants to stay where it is.
const NoEvent EventName = ""

// State is one node of a StateMachine.
type State interface {
	// Update runs one frame of the state and returns the event to trigger,
	// or NoEvent to stay.
	Update() EventName
	// Initialize runs when the state is entered from a different state.
	Initialize()
	GetName() StateName
}

// Exiter is implemented by states that need to clean up when they are left.
// OnExit runs before the next state's Initialize.
type Exiter interface {
	OnExit()
}

type StateMachine struct {
	currentState     State
	StateTransitions map[StateName]map[EventName]State

	// UnhandledEvent, when set, is called for every non-empty event that has no
	// transition out of the current state. Most of these are expected (roles
	// broadcast events every frame), so the default is to ignore them.
	UnhandledEvent func(from StateName, event EventName)
}

func NewStateMachine(initial State) *StateMachine {
	sm := &StateMachine{
		StateTransitions: make(map[StateName]map[EventName]State),
	}
	sm.currentState = initial
	initial.Initialize()
	return sm
}

func (sm *StateMachine) AddTransition(from StateName, event EventName, to State) {
	if event == NoEvent {
		panic(fmt.Sprintf("state_machine: transition from %q needs a non-empty event", from))
	}
	if to == nil {
		panic(fmt.Sprintf("state_machine: transition %q --%q--> nil", from, event))
	}
	if sm.StateTransitions[from] == nil {
		sm.StateTransitions[from] = make(map[EventName]State)
	}
	sm.StateTransitions[from][event] = to
}

// HasTransition reports whether event leads anywhere from the current state.
func (sm *StateMachine) HasTransition(event EventName) bool {
	_, ok := sm.StateTransitions[sm.currentState.GetName()][event]
	return ok
}

// TriggerEvent follows the transition for event, if the current state has
// one. Events without a transition are ignored (see UnhandledEvent).
func (sm *StateMachine) TriggerEvent(event EventName) {
	if event == NoEvent {
		return
	}
	from := sm.currentState.GetName()
	newState, ok := sm.StateTransitions[from][event]
	if !ok {
		if sm.UnhandledEvent != nil {
			sm.UnhandledEvent(from, event)
		}
		return
	}
	sm.ChangeState(newState)
}

// TriggerEventReentering is TriggerEvent, except that a transition back into
// the current state restarts it (OnExit, then Initialize). Use it for
// external commands that mean "start this over", such as a new referee
// command for the same restart.
func (sm *StateMachine) TriggerEventReentering(event EventName) {
	if event == NoEvent {
		return
	}
	newState, ok := sm.StateTransitions[sm.currentState.GetName()][event]
	if ok && newState == sm.currentState {
		sm.exitCurrent()
		newState.Initialize()
		return
	}
	sm.TriggerEvent(event)
}

// ChangeState switches to newState. Entering the state that is already
// current is a no-op: it neither exits nor re-initializes it.
func (sm *StateMachine) ChangeState(newState State) {
	if newState == sm.currentState {
		return
	}
	sm.exitCurrent()
	sm.currentState = newState
	newState.Initialize()
}

func (sm *StateMachine) exitCurrent() {
	if exiter, ok := sm.currentState.(Exiter); ok {
		exiter.OnExit()
	}
}

// Update runs the current state for one frame and follows the event it returns.
func (sm *StateMachine) Update() {
	if sm.currentState == nil {
		panic("States have not been initialized!!")
	}
	sm.TriggerEvent(sm.currentState.Update())
}

func (sm *StateMachine) CurrentStateName() StateName {
	return sm.currentState.GetName()
}

// Events lists the events with a transition out of from, sorted.
func (sm *StateMachine) Events(from StateName) []EventName {
	events := make([]EventName, 0, len(sm.StateTransitions[from]))
	for event := range sm.StateTransitions[from] {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool { return events[i] < events[j] })
	return events
}
