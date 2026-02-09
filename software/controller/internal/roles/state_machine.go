package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type StateName string
type EventName string


type State interface {
	Update()
	Initialize(sm *StateMachine)
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
	transitions := sm.StateTransitions[sm.currentState.GetName()]
	newState, ok := transitions[event]
	if !ok {
		return
	}
	sm.ChangeState(newState)
}

func (sm *StateMachine) ChangeState(newState State) {
	sm.currentState = newState
	newState.Initialize(sm)
}

func (sm *StateMachine) Update() {
	sm.currentState.Update()
}


type AlignState struct {
	sm *StateMachine
	gi *info.GameInfo
	robotId info.ID
	team info.Team
	from info.Position
}

func (s *AlignState) Initialize(sm *StateMachine) {
	s.sm = sm
}

func (s *AlignState) GetName() StateName {
	return "Align"
}

func (s *AlignState) Update() {

	ball := s.gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	activity := ai.NewAlign(s.team, s.robotId, s.from, ballPos)
	achieved := activity.Achieved(s.gi)
	if achieved {
		s.sm.TriggerEvent("Aligned")
	}
}

type KickState struct {
	sm *StateMachine
}

func (s *KickState) Initialize(sm *StateMachine) {
	s.sm = sm
	fmt.Println("Entering Kick State")
}

func (s *KickState) GetName() StateName {
	return "Kick"
}

func (s *KickState) Update() {

}


func main() {
	align := &AlignState{gi, team, robotId}
	kick := &KickState{}

	sm := NewStateMachine(align)

	sm.AddTransition("Align", "Aligned", kick)

	sm.Update()
}
