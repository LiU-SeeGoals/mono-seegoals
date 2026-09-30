package ai

import (
	"reflect"
	"strings"
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/action"
	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type planner interface {
	Init(activities *[info.TEAM_SIZE]ai.Activity, lock *sync.Mutex, team info.Team)
	Tick(gi *info.GameInfo)
	Kill()
}

type executor interface {
	Init(incoming <-chan info.GameInfo,
		activities *[info.TEAM_SIZE]ai.Activity,
		lock *sync.Mutex,
		outgoing chan<- []action.Action,
		team info.Team,
	)
}

type Ai struct {
	team               info.Team
	planner            planner
	executor           executor
	gameInfoSenderFB   chan<- info.GameInfo
	gameInfoRecieverFB <-chan info.GameInfo // Save game reciever to pass it to hotswapped ais
	actionReceiver     chan []action.Action
	activities         *[info.TEAM_SIZE]ai.Activity // Shared slice of Activity
	activity_lock      *sync.Mutex                  // Shared mutex for synchronization
}

func (m *Ai) HotswapPlanner(team info.Team, planner planner) {

	m.activity_lock.Lock()
	defer m.activity_lock.Unlock()

	m.planner.Kill()
	planner.Init(m.activities, m.activity_lock, team)
	m.planner = planner
}

func NewAi(team info.Team, planner planner, executor executor) *Ai {
	activities := &[info.TEAM_SIZE]ai.Activity{}
	lock := &sync.Mutex{}

	gameInfoSenderFB, gameInfoReceiverFB := helper.NB_KeepLatestChan[info.GameInfo]()
	actionReceiver := make(chan []action.Action)

	// Initialize plan and executor with the shared resources
	planner.Init(activities, lock, team)
	executor.Init(gameInfoReceiverFB, activities, lock, actionReceiver, team)

	ai.SetPathService(team, pathplanner.New())

	// Construct the AI object
	ai := &Ai{
		team:               team,
		planner:            planner,
		executor:           executor,
		activities:         activities,
		gameInfoSenderFB:   gameInfoSenderFB,
		gameInfoRecieverFB: gameInfoReceiverFB, // Save game reciever to pass it to hotswapped ais
		activity_lock:      lock,
		actionReceiver:     actionReceiver,
	}
	return ai
}

// Decides on new actions for the robots
func (ai *Ai) GetActions(gi *info.GameInfo) []action.Action {
	// Planner and executor share state, so they must not run concurrently
	ai.planner.Tick(gi)

	// Send the game state to the executor so it can execute gamestate aware activities (e.g. avoid obstacles)
	ai.gameInfoSenderFB <- *gi

	// Get the actions from the executor, this will block until it has decided on actions
	return <-ai.actionReceiver
}

type ActivityHandler struct {
	Activities    *[info.TEAM_SIZE]ai.Activity // <-- pointer to the slice
	Activity_lock *sync.Mutex                  // shared mutex for synchronization
}

func (m *ActivityHandler) ClearActivities() {
	m.Activity_lock.Lock()
	defer m.Activity_lock.Unlock()
	*m.Activities = [info.TEAM_SIZE]ai.Activity{}
}

func (m *ActivityHandler) ClearActivity(id info.ID) {
	m.Activity_lock.Lock()
	defer m.Activity_lock.Unlock()
	m.Activities[id] = nil
}

func (m *ActivityHandler) AddActivity(activity ai.Activity) {
	m.Activity_lock.Lock()
	idx := activity.GetID()
	previous := m.Activities[idx]
	m.Activities[idx] = activity
	m.Activity_lock.Unlock()

	previousName, name := m.GetActionTypeName(previous), m.GetActionTypeName(activity)
	if previousName != name {
		if previousName == "" {
			previousName = "none"
		}
		Logger.Infof("Robot %d activity %s -> %v", idx, previousName, activity)
	}
}

func (m *ActivityHandler) GetActivity(id info.ID) ai.Activity {
	return m.Activities[id]
}

func (m *ActivityHandler) ReplaceActivities(activities *[info.TEAM_SIZE]ai.Activity) {
	m.Activity_lock.Lock()
	defer m.Activity_lock.Unlock()
	m.Activities = activities
}

func (m *ActivityHandler) GetActionTypeName(activity ai.Activity) string {
	// Check if activity is nil
	if activity == nil {
		return ""
	}

	// Get the type using reflection
	t := reflect.TypeOf(activity)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Get the full name (including package)
	fullName := t.String()

	// just the type name without the package
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}
