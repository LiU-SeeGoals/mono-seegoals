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

func (sm *StateMachine) SetGlobalTransition(event EventName, to State) {
	if sm.StateTransitions["GLOBAL"] == nil {
		sm.StateTransitions["GLOBAL"] = make(map[EventName]State)
	}
	sm.StateTransitions["GLOBAL"][event] = to
}

func (sm *StateMachine) TriggerEvent(event EventName) {
	transitions := sm.StateTransitions[sm.currentState.GetName()]
	newState, ok := transitions[event]
	if !ok {
		return
	}
	fmt.Println(event ,sm.currentState.GetName(), "->", newState.GetName())
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


type AlignState struct {
	sm *StateMachine
	gi *info.GameInfo
	robotId info.ID
	team info.Team
	from info.Position
	name StateName
	activityHandler *ai.ActivityHandler
}

func (s *AlignState) Initialize() {
}

func (s *AlignState) GetName() StateName {
	return s.name
}

func (s *AlignState) Update() EventName{

	ball := s.gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	activity := act.NewAlign(s.team, s.robotId, s.from, ballPos)
	s.activityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.gi)
	if achieved {
		return "ALIGNED"
	}
	return "NONE"
}

type KickState struct {
	name StateName
	robotId info.ID
	team info.Team
	sm *StateMachine
	kickAct *act.KickBall
	gi *info.GameInfo
	handle *ai.ActivityHandler
}

func (s *KickState) Initialize() {
	fmt.Println("Entering Kick State")
	ballPos, _ := s.gi.State.GetBall().GetEstimatedPosition()
	receiverPos, _ := s.gi.State.GetBall().GetEstimatedPosition()

	s.kickAct = act.NewKickBall(s.team, s.robotId, receiverPos, ballPos)
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


func main() {
	position := info.Position{1,1,1,1}
	team := info.Yellow
	robotId := info.ID(3)
	gi := info.NewGameInfo(1)

	align := &AlignState{gi: gi, team: team, robotId: robotId, name: "Align", from: position}
	prepareKick := &AlignState{gi: gi, team: team, robotId: robotId, name: "PrepareKick", from: position}
	kick := &KickState{name: "Kick", gi: gi, team: team, robotId: robotId}

	sm := NewStateMachine(align)

	sm.SetGlobalTransition("BALL_LOST", align)
	sm.AddTransition("Align", "BALL_OWNER", prepareKick)
	sm.AddTransition("KickPrepare", "ALIGNED", kick)
	sm.AddTransition("Kick", "KICKED", align)

	sm.Update()
}
