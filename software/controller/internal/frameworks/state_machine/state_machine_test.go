package state_machine

import (
	"reflect"
	"testing"
)

type testState struct {
	name  StateName
	next  EventName
	inits int
	exits int
	trace *[]string
}

func (s *testState) Update() EventName { return s.next }
func (s *testState) Initialize() {
	s.inits++
	if s.trace != nil {
		*s.trace = append(*s.trace, "init "+string(s.name))
	}
}
func (s *testState) GetName() StateName { return s.name }

type exitingState struct{ testState }

func (s *exitingState) OnExit() {
	s.exits++
	if s.trace != nil {
		*s.trace = append(*s.trace, "exit "+string(s.name))
	}
}

const (
	evGo   EventName = "GO"
	evBack EventName = "BACK"
	evSelf EventName = "SELF"
)

func TestTransitionsRunExitBeforeInit(t *testing.T) {
	var trace []string
	a := &exitingState{testState{name: "A", trace: &trace}}
	b := &testState{name: "B", trace: &trace}
	sm := NewStateMachine(a)
	sm.AddTransition("A", evGo, b)
	sm.AddTransition("B", evBack, a)

	sm.TriggerEvent(evGo)
	sm.TriggerEvent(evBack)

	want := []string{"init A", "exit A", "init B", "init A"}
	if !reflect.DeepEqual(trace, want) {
		t.Fatalf("trace = %v, want %v", trace, want)
	}
}

func TestSelfTransitionDoesNotReinitialize(t *testing.T) {
	a := &exitingState{testState{name: "A"}}
	sm := NewStateMachine(a)
	sm.AddTransition("A", evSelf, a)

	sm.TriggerEvent(evSelf)
	sm.TriggerEvent(evSelf)

	if a.inits != 1 || a.exits != 0 {
		t.Fatalf("inits=%d exits=%d, want 1 and 0", a.inits, a.exits)
	}
}

func TestReenteringRestartsCurrentState(t *testing.T) {
	a := &exitingState{testState{name: "A"}}
	b := &testState{name: "B"}
	sm := NewStateMachine(a)
	sm.AddTransition("A", evSelf, a)
	sm.AddTransition("A", evGo, b)

	sm.TriggerEventReentering(evSelf)
	if a.inits != 2 || a.exits != 1 {
		t.Fatalf("inits=%d exits=%d, want 2 and 1", a.inits, a.exits)
	}
	sm.TriggerEventReentering(evGo)
	if sm.CurrentStateName() != "B" || a.exits != 2 {
		t.Fatalf("state=%s exits=%d, want B and 2", sm.CurrentStateName(), a.exits)
	}
}

func TestUpdateFollowsReturnedEvent(t *testing.T) {
	a := &testState{name: "A", next: evGo}
	b := &testState{name: "B", next: NoEvent}
	sm := NewStateMachine(a)
	sm.AddTransition("A", evGo, b)

	sm.Update()
	if sm.CurrentStateName() != "B" {
		t.Fatalf("state = %s, want B", sm.CurrentStateName())
	}
	sm.Update()
	if sm.CurrentStateName() != "B" || b.inits != 1 {
		t.Fatalf("NoEvent should stay in B without re-init (inits=%d)", b.inits)
	}
}

func TestUnhandledEventIsReported(t *testing.T) {
	a := &testState{name: "A"}
	sm := NewStateMachine(a)
	var got []EventName
	sm.UnhandledEvent = func(_ StateName, ev EventName) { got = append(got, ev) }

	sm.TriggerEvent(NoEvent)
	sm.TriggerEvent(evGo)

	if !reflect.DeepEqual(got, []EventName{evGo}) {
		t.Fatalf("unhandled = %v, want [GO]", got)
	}
	if sm.HasTransition(evGo) {
		t.Fatal("HasTransition(GO) = true without a transition")
	}
}

func TestAddTransitionRejectsEmptyEvent(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("AddTransition with NoEvent should panic")
		}
	}()
	a := &testState{name: "A"}
	NewStateMachine(a).AddTransition("A", NoEvent, a)
}
