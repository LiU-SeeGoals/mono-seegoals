package ai

import (
	"reflect"
	"strings"
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/logger"
)

// Planner decides what each robot of a team should do. Tick runs once per
// frame and writes the team's activities into the shared array given to Init.
type Planner interface {
	Init(activities *[info.TEAM_SIZE]activity.Activity, lock *sync.Mutex, team info.Team)
	Tick(gi *info.GameInfo)
	Kill()
}

// Ai runs one team's decision pipeline synchronously on the caller's
// goroutine: planner -> activities -> safety -> actions. Nothing runs in the
// background, so GetActions returns once every action for the frame is ready.
type Ai struct {
	team       info.Team
	planner    Planner
	executor   *activityExecutor
	activities *[info.TEAM_SIZE]activity.Activity // written by the planner, read by the executor
	lock       *sync.Mutex                        // guards activities; scripts write them from their own goroutine
}

func NewAi(team info.Team, planner Planner) *Ai {
	activities := &[info.TEAM_SIZE]activity.Activity{}
	lock := &sync.Mutex{}

	planner.Init(activities, lock, team)
	activity.SetPathService(team, pathplanner.New())

	return &Ai{
		team:       team,
		planner:    planner,
		executor:   newActivityExecutor(team, activities, lock),
		activities: activities,
		lock:       lock,
	}
}

// HotswapPlanner replaces the planner, keeping the shared activity array.
func (m *Ai) HotswapPlanner(team info.Team, planner Planner) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.planner.Kill()
	planner.Init(m.activities, m.lock, team)
	m.planner = planner
}

// GetActions plans and executes one frame.
func (m *Ai) GetActions(gi *info.GameInfo) []action.Action {
	m.planner.Tick(gi)
	return m.executor.execute(gi)
}

type ActivityHandler struct {
	Activities    *[info.TEAM_SIZE]activity.Activity // <-- pointer to the slice
	Activity_lock *sync.Mutex                        // shared mutex for synchronization
}

func (m *ActivityHandler) ClearActivities() {
	m.Activity_lock.Lock()
	defer m.Activity_lock.Unlock()
	*m.Activities = [info.TEAM_SIZE]activity.Activity{}
}

func (m *ActivityHandler) ClearActivity(id info.ID) {
	m.Activity_lock.Lock()
	defer m.Activity_lock.Unlock()
	m.Activities[id] = nil
}

func (m *ActivityHandler) AddActivity(act activity.Activity) {
	m.Activity_lock.Lock()
	idx := act.GetID()
	previous := m.Activities[idx]
	m.Activities[idx] = act
	m.Activity_lock.Unlock()

	previousName, name := m.GetActionTypeName(previous), m.GetActionTypeName(act)
	if previousName != name {
		if previousName == "" {
			previousName = "none"
		}
		logger.Logger.Infof("Robot %d activity %s -> %v", idx, previousName, act)
	}
}

func (m *ActivityHandler) GetActivity(id info.ID) activity.Activity {
	m.Activity_lock.Lock()
	defer m.Activity_lock.Unlock()
	return m.Activities[id]
}

func (m *ActivityHandler) GetActionTypeName(act activity.Activity) string {
	// Check if activity is nil
	if act == nil {
		return ""
	}

	// Get the type using reflection
	t := reflect.TypeOf(act)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Get the full name (including package)
	fullName := t.String()

	// just the type name without the package
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}
