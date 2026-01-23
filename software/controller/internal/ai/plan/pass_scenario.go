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
	KICKER_STAY       = 0

	RECEIVER_TURN_TO_GOAL = 0
	RECEIVER_KICK_GOAL    = 1
	RECEIVER_POSITION     = 0
	RECEIVER_WATCH_BALL   = 1

	STATE_START     = 0
	STATE_EXECUTING = 1
)

func KickerGetStateMachineName(sm *RobotStateMachine) string {
	if sm.isBallOwner {
		switch sm.stateMachine {
		case KICKER_FETCH_BALL:
			return "FETCH_BALL"
		case KICKER_PASS_BALL:
			return "PASS_BALL"
		}
	} else {
		switch sm.stateMachine {
		case KICKER_STAY:
			return "WATCH_BALL"
		}
	}

	return "UNKNOWN"
}

func ReceiverGetStateMachineName(sm *RobotStateMachine) string {
	if sm.isBallOwner {
		switch sm.stateMachine {
		case RECEIVER_TURN_TO_GOAL:
			return "TURN_TO_GOAL"
		case RECEIVER_KICK_GOAL:
			return "KICK_GOAL"
		}
	} else {
		switch sm.stateMachine {
		case RECEIVER_POSITION:
			return "POSITION"
		case RECEIVER_WATCH_BALL:
			return "WATCH_BALL"
		}
	}

	return "UNKNOWN"
}

type RobotStateMachine struct {
	robotID      ID
	role         string
	stateMachine int
	state        int
	gameScenario *GameScenario
	isBallOwner  bool
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
		return KickerGetStateMachineName(sm)
	case "receiver":
		return ReceiverGetStateMachineName(sm)
	}
	return fmt.Sprintf("SM_%d", sm.stateMachine)
}

func (sm *RobotStateMachine) ToStateMachine(newSM int) {
	oldName := sm.GetStateName()
	sm.stateMachine = newSM
	sm.state = STATE_START
	newName := sm.GetStateName()
	fmt.Printf("Robot %d (%s): %s -> %s, State -> START\n", sm.robotID, sm.role, oldName, newName)
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

func (sm *RobotStateMachine) ResetIfPreviousOwner() {
	if sm.gameScenario.previousBallOwner == sm.robotID {
		fmt.Printf("Reset since previous owner: ", sm.robotID)
		sm.ToStateMachine(0)
		sm.gameScenario.previousBallOwner = 50
		sm.isBallOwner = false
	}
}

type GameScenario struct {
	plannerCore
	start             time.Time
	ballOwner         ID
	previousBallOwner ID
	ballOwnerTeam     Team
}

func (g *GameScenario) changeBallOwner(newOwner *RobotStateMachine, reason string) {
	if g.ballOwner != newOwner.robotID {
		fmt.Printf("Ball Owner: %d -> %d (%s)\n", g.ballOwner, newOwner, reason)
		g.previousBallOwner = g.ballOwner
		g.ballOwner = newOwner.robotID
		newOwner.isBallOwner = true
	}
}

func (sm *RobotStateMachine) TakeOwnership(g *GameScenario, reason string) {
	g.changeBallOwner(sm, reason)
	sm.ToStateMachine(0)
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
	receiverPos := Position{X: 2000, Y: 1500, Z: 0, Angle: 0}

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
				sm.TakeOwnership(g, "start of game")
				activity = ai.NewAlign(g.team, myID, receiverPos, ballPos)
				achieved := activity.Achieved(&gi)
				debugf("Robot %d FETCH_BALL check: achieved=%v\n", myID, achieved)
				if achieved {
					sm.ToStateMachine(KICKER_PASS_BALL)
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
	} else { // not ball owner
		switch sm.stateMachine {
		case KICKER_STAY:
			switch sm.state {
			case STATE_START:
				robotPos, _ := gi.State.GetTeam(g.team)[myID].GetPosition()
				activity = ai.NewMoveToPosition(g.team, myID, robotPos)
				sm.NextState()
			case STATE_EXECUTING:
				if ballPos.X < 0 {
					sm.TakeOwnership(g, "ball on my side")
				}
				existingActivity := g.GetActivity(myID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					sm.ResetState()
				}
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
	opponentGoal := Position{X: 10000, Y: 0, Z: 0, Angle: 0}

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
				activity = ai.NewAlign(g.team, myID, opponentGoal, ballPos)
				if activity.Achieved(&gi) {
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
					sm.ToStateMachine(RECEIVER_TURN_TO_GOAL)
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
				if ballPos.Dist2d(myPos) < 2000 {
					sm.TakeOwnership(g, "receiver close to ball")
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

	gi := <-g.incomingGameInfo
	fmt.Println(gi.Status)

	for {
		gi = <-g.incomingGameInfo
		time.Sleep(5 * time.Millisecond)
		// if g.HandleRef(&gi, active_robots) {
		// 	continue
		// }

		kicker_SM, kicker_S = g.kicker(kicker, receiver, gi, kicker_SM, kicker_S)
		receiver_SM, receiver_S = g.receiver(receiver, gi, receiver_SM, receiver_S)
	}
}
