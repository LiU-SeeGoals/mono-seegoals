package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/plt"
)

const DEBUG = false

func debugf(format string, args ...interface{}) {
	if DEBUG {
		fmt.Printf(format, args...)
	}
}

const (
	KICKER_FETCH_BALL = 0
	KICKER_PASS_BALL  = 1

	RECEIVER_TURN_TO_GOAL = 0
	RECEIVER_KICK_GOAL    = 1
	RECEIVER_POSITION   = 2
	RECEIVER_WATCH_BALL = 3

	STATE_START     = 0
	STATE_EXECUTING = 1
)

var KickerStateMachineNames = map[int]string{
	KICKER_FETCH_BALL: "FETCH_BALL",
	KICKER_PASS_BALL:  "PASS_BALL",
}

var ReceiverStateMachineNames = map[int]string{
	RECEIVER_TURN_TO_GOAL: "TURN_TO_GOAL",
	RECEIVER_KICK_GOAL:    "KICK_GOAL",
	RECEIVER_POSITION:     "POSITION",
	RECEIVER_WATCH_BALL:   "WATCH_BALL",
}

type RobotStateMachine struct {
	robotID      ID
	role         string
	stateMachine int
	state        int
	gameScenario *GameScenario
}

func NewRobotStateMachine(robotID ID, role string, gs *GameScenario) *RobotStateMachine {
	return &RobotStateMachine{
		robotID:      robotID,
		role:         role,
		stateMachine: 0,
		state:        0,
		gameScenario: gs,
	}
}

func (sm *RobotStateMachine) GetStateName() string {
	switch sm.role {
	case "kicker":
		if name, exists := KickerStateMachineNames[sm.stateMachine]; exists {
			return name
		}
	case "receiver":
		if name, exists := ReceiverStateMachineNames[sm.stateMachine]; exists {
			return name
		}
	}
	return fmt.Sprintf("SM_%d", sm.stateMachine)
}

func (sm *RobotStateMachine) ToStateMachine(newSM int) {
	if sm.stateMachine != newSM {
		oldName := sm.GetStateName()
		sm.stateMachine = newSM
		sm.state = STATE_START
		newName := sm.GetStateName()
		fmt.Printf("Robot %d (%s): %s -> %s, State -> START\n", sm.robotID, sm.role, oldName, newName)
	}
}

func (sm *RobotStateMachine) NextState() {
	if sm.state != STATE_EXECUTING {
		fmt.Printf("Robot %d (%s): %s State START -> EXECUTING\n", sm.robotID, sm.role, sm.GetStateName())
		sm.state = STATE_EXECUTING
	}
}

func (sm *RobotStateMachine) ResetState() {
	if sm.state != STATE_START {
		fmt.Printf("Robot %d (%s): %s State EXECUTING -> START\n", sm.robotID, sm.role, sm.GetStateName())
		sm.state = STATE_START
	}
}

func (sm *RobotStateMachine) TakeBallOwnership(reason string) {
	sm.gameScenario.changeBallOwner(sm.robotID, reason)
	sm.ToStateMachine(sm.getInitialStateMachine())
}

func (sm *RobotStateMachine) ResetIfPreviousOwner() {
	if sm.gameScenario.previousBallOwner == sm.robotID {
		sm.ToStateMachine(sm.getInitialStateMachine())
	}
}

func (sm *RobotStateMachine) getInitialStateMachine() int {
	switch sm.role {
	case "kicker":
		return KICKER_FETCH_BALL
	case "receiver":
		if sm.gameScenario.ballOwner == sm.robotID {
			return RECEIVER_TURN_TO_GOAL
		} else {
			return RECEIVER_POSITION
		}
	default:
		return 0
	}
}

type GameScenario struct {
	plannerCore
	start             time.Time
	ballOwner         ID
	previousBallOwner ID
	ballOwnerTeam     Team
}

func (g *GameScenario) changeBallOwner(newOwner ID, reason string) {
	if g.ballOwner != newOwner {
		fmt.Printf("Ball Owner: %d -> %d (%s)\n", g.ballOwner, newOwner, reason)
		g.previousBallOwner = g.ballOwner
		g.ballOwner = newOwner
	}
}

func NewGameScenario(team Team) *GameScenario {
	return &GameScenario{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *GameScenario) Init(
	incoming <-chan GameInfo,
	activities *[TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team Team,
) {
	m.incomingGameInfo = incoming
	m.activities = activities
	m.activity_lock = lock
	m.team = team

	go m.run()
}

func (g *GameScenario) kicker(myID ID, receiver ID, gi GameInfo, statemachine int, state int) (int, int) {
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	var activity ai.Activity = nil
	receiverPos, _ := gi.State.GetTeam(g.team)[receiver].GetPosition()

	sm := &RobotStateMachine{
		robotID:      myID,
		role:         "kicker",
		stateMachine: statemachine,
		state:        state,
		gameScenario: g,
	}

	sm.ResetIfPreviousOwner()

	if g.ballOwner == myID {
		switch sm.stateMachine {
		case KICKER_FETCH_BALL:
			switch sm.state {
			case STATE_START:
				activity = ai.NewAlign(g.team, myID, receiverPos, ballPos)
				sm.NextState()
			case STATE_EXECUTING:
				existingActivity := g.GetActivity(myID)
				if existingActivity != nil {
					achieved := existingActivity.Achieved(&gi)
					debugf("Robot %d FETCH_BALL check: achieved=%v\n", myID, achieved)
					if achieved {
						sm.ToStateMachine(KICKER_PASS_BALL)
					}
				} else {
					debugf("Robot %d FETCH_BALL: activity is nil!\n", myID)
				}
			}

		case KICKER_PASS_BALL:
			switch sm.state {
			case STATE_START:
				activity = ai.NewKickBall(g.team, myID, ballPos)
				sm.NextState()
			case STATE_EXECUTING:
				existingActivity := g.GetActivity(myID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					sm.ResetState()
				}
			}
		}
	} else { // not ball owner - stay in place
		switch sm.state {
		case STATE_START:
			robotPos, _ := gi.State.GetTeam(g.team)[myID].GetPosition()
			activity = ai.NewMoveToPosition(g.team, myID, robotPos)
			sm.NextState()
		case STATE_EXECUTING:
			existingActivity := g.GetActivity(myID)
			if existingActivity != nil && existingActivity.Achieved(&gi) {
				sm.ResetState()
			}
		}
	}

	if activity != nil {
		activityID := activity.GetID()
		debugf("Robot %d (kicker): Adding activity %T - activity.GetID()=%d, myID=%d\n", myID, activity, activityID, myID)
		g.AddActivity(activity)

		retrieved := g.GetActivity(myID)
		debugf("Robot %d (kicker): Retrieved activity after adding: %T (nil=%v)\n", myID, retrieved, retrieved == nil)
	}

	return sm.stateMachine, sm.state
}

func (g *GameScenario) receiver(myID ID, gi GameInfo, statemachine int, state int) (int, int) {
	var activity ai.Activity = nil

	//ball := gi.State.GetTrackedBall()
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	opponentGoal := Position{X: 1500, Y: 1000, Z: 0, Angle: 0}

	sm := &RobotStateMachine{
		robotID:      myID,
		role:         "receiver",
		stateMachine: statemachine,
		state:        state,
		gameScenario: g,
	}

	sm.ResetIfPreviousOwner()

	if g.ballOwner == myID {
		switch sm.stateMachine {
		case RECEIVER_TURN_TO_GOAL:
			switch sm.state {
			case STATE_START:
				activity = ai.NewAlign(g.team, myID, ballPos, opponentGoal)
				sm.NextState()
			case STATE_EXECUTING:
				existingActivity := g.GetActivity(myID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					sm.ToStateMachine(RECEIVER_KICK_GOAL)
				}
			}

		case RECEIVER_KICK_GOAL:
			switch sm.state {
			case STATE_START:
				activity = ai.NewKickBall(g.team, myID, ballPos)
				sm.NextState()
			case STATE_EXECUTING:
				existingActivity := g.GetActivity(myID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					sm.ResetState()
				}
			}
		}
	} else { // not ball owner
		switch sm.stateMachine {
		case RECEIVER_POSITION:
			switch sm.state {
			case STATE_START:
				wantedPos := Position{X: 2000, Y: 1500, Z: 0, Angle: 0}
				activity = ai.NewMoveToPosition(g.team, myID, wantedPos)
				debugf("Robot %d POSITION: Created new MoveToPosition activity to (%.0f,%.0f)\n", myID, wantedPos.X, wantedPos.Y)
				sm.NextState()
			case STATE_EXECUTING:
				existingActivity := g.GetActivity(myID)
				if existingActivity != nil {
					achieved := existingActivity.Achieved(&gi)
					myPos, _ := gi.State.GetTeam(g.team)[myID].GetPosition()
					targetPos := Position{X: 2000, Y: 1500, Z: 0, Angle: 0}
					distance := myPos.Dist2d(targetPos)
					debugf("Robot %d POSITION check: achieved=%v, distance=%.2f, myPos=(%.0f,%.0f), target=(%.0f,%.0f)\n",
						myID, achieved, distance, myPos.X, myPos.Y, targetPos.X, targetPos.Y)
					if achieved {
						sm.ToStateMachine(RECEIVER_WATCH_BALL)
					}
				} else {
					debugf("Robot %d POSITION: activity is nil!\n", myID)
				}
			}

		case RECEIVER_WATCH_BALL:
			switch sm.state {
			case STATE_START:
				myPos, _ := gi.State.GetTeam(g.team)[myID].GetPosition()
				activity = ai.NewAlign(g.team, myID, ballPos, myPos)
				sm.NextState()
			case STATE_EXECUTING:
				myPos, _ := gi.State.GetTeam(g.team)[myID].GetPosition()
				if ballPos.Dist2d(myPos) < 800 {
					sm.TakeBallOwnership("receiver close to ball")
				}
			}
		}
	}

	if activity != nil {
		activityID := activity.GetID()
		debugf("Robot %d (receiver): Adding activity %T - activity.GetID()=%d, myID=%d\n", myID, activity, activityID, myID)
		g.AddActivity(activity)

		retrieved := g.GetActivity(myID)
		debugf("Robot %d (receiver): Retrieved activity after adding: %T (nil=%v)\n", myID, retrieved, retrieved == nil)
	}

	return sm.stateMachine, sm.state
}

func (g *GameScenario) run() {
	plt.Init()

	kicker := info.ID(3)
	receiver := info.ID(1)
	// active_robots := []int{int(kicker), int(receiver)}

	kicker_SM := KICKER_FETCH_BALL
	kicker_S := STATE_START

	receiver_SM := RECEIVER_POSITION
	receiver_S := STATE_START

	g.changeBallOwner(kicker, "start of game")

	gi := <-g.incomingGameInfo
	fmt.Println(gi.Status)

	for {
		gi = <-g.incomingGameInfo
		time.Sleep(100 * time.Millisecond)
		// if g.HandleRef(&gi, active_robots) {
		// 	continue
		// }

		kicker_SM, kicker_S = g.kicker(kicker, receiver, gi, kicker_SM, kicker_S)
		receiver_SM, receiver_S = g.receiver(receiver, gi, receiver_SM, receiver_S)
	}

}
