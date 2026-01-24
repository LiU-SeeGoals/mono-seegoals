package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

const DEBUG = false

func debugf(format string, args ...interface{}) {
	if DEBUG {
		fmt.Printf(format, args...)
	}
}

type GameScenarioInterface interface {
	AddActivity(ai.Activity)
	GetActivity(ID) ai.Activity

	GetBallOwner() ID
	GetPreviousBallOwner() ID
	ChangeBallOwner(ID, string)

	GetTeam() Team
}

type RobotRole struct {
	robotID       ID
	role          string
	stateMachine  int
	state         int
	gameScenario  interface{}
	isBallOwner   bool
	hasBeenReset  bool
}

type RoleStateMachineNamer interface {
	GetStateMachineName() string
}

func NewRobotRole(robotID ID, role string, gameScenario interface{}) *RobotRole {
	return &RobotRole{
		robotID:      robotID,
		role:         role,
		stateMachine: 0,
		state:        0,
		gameScenario: gameScenario,
	}
}

func (sm *RobotRole) ToStateMachine(newSM int, namer RoleStateMachineNamer) {
	if sm.stateMachine != newSM {
		oldName := namer.GetStateMachineName()
		sm.stateMachine = newSM
		sm.state = 0
		newName := namer.GetStateMachineName()
		fmt.Printf("Robot %d (%s): %s -> %s, State -> START\n", sm.robotID, sm.role, oldName, newName)
	}
}

func (sm *RobotRole) NextState(namer RoleStateMachineNamer) {
	stateName := namer.GetStateMachineName()
	debugf("Robot %d (%s): %s State %d -> %d\n", sm.robotID, sm.role, stateName, sm.state, sm.state+1)
	sm.state++
}

func (sm *RobotRole) ResetState(namer RoleStateMachineNamer) {
	if sm.state != 0 {
		stateName := namer.GetStateMachineName()
		sm.state = 0
		debugf("Robot %d (%s): %s State %d -> 0\n", sm.robotID, sm.role, stateName, sm.state)
	}
}

func (sm *RobotRole) ResetIfPreviousOwner(namer RoleStateMachineNamer, g GameScenarioInterface) {
	currentOwner := g.GetBallOwner()
	previousOwner := g.GetPreviousBallOwner()
	debugf("[DEBUG] ResetIfPreviousOwner: Robot %d, CurrentOwner: %d, PreviousOwner: %d, HasBeenReset: %v\n",
		sm.robotID, currentOwner, previousOwner, sm.hasBeenReset)

	if previousOwner == sm.robotID && currentOwner != sm.robotID && !sm.hasBeenReset {
		debugf("[DEBUG] Robot %d WAS previous owner and is NO LONGER current owner, triggering reset\n", sm.robotID)
		fmt.Printf("Robot %d (%s): Reset from being ballowner since previous owner\n", sm.robotID, sm.role)
		sm.ToStateMachine(0, namer)
		sm.hasBeenReset = true
	}

	if currentOwner == sm.robotID {
		sm.hasBeenReset = false
	}
}

func (sm *RobotRole) TakeOwnership(namer RoleStateMachineNamer, g GameScenarioInterface, reason string) {
	debugf("[DEBUG] Robot %d taking ownership: %s\n", sm.robotID, reason)
	g.ChangeBallOwner(sm.robotID, reason)
	sm.isBallOwner = true
	sm.ToStateMachine(0, namer)
}

func (sm *RobotRole) GetRobotID() ID {
	return sm.robotID
}

func (sm *RobotRole) GetRole() string {
	return sm.role
}

func (sm *RobotRole) GetStateMachine() int {
	return sm.stateMachine
}

func (sm *RobotRole) GetState() int {
	return sm.state
}

func (sm *RobotRole) GetGameScenario() interface{} {
	return sm.gameScenario
}

func (sm *RobotRole) IsBallOwner() bool {
	return sm.isBallOwner
}

func (sm *RobotRole) SetBallOwner(owner bool) {
	sm.isBallOwner = owner
}
