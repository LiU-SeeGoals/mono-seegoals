package ai

import (
	"math"
	"reflect"
	"strings"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type plannerCore struct {
	team             info.Team
	incomingGameInfo <-chan info.GameInfo
	start            time.Time
	ActivityHandler  ActivityHandler
}

type ActivityHandler struct {
	activities       *[info.TEAM_SIZE]ai.Activity // <-- pointer to the slice
	activity_lock    *sync.Mutex                  // shared mutex for synchronization
}

func (m *ActivityHandler) ClearActivities() {
	m.activity_lock.Lock()
	defer m.activity_lock.Unlock()
	*m.activities = [info.TEAM_SIZE]ai.Activity{}
}

func (m *ActivityHandler) AddActivity(activity ai.Activity) {
	// m.activity_lock.Lock()
	// defer m.activity_lock.Unlock()
	idx := activity.GetID()
	Logger.Infof("Adding activity %v", activity)
	m.activities[idx] = activity
}

func (m *ActivityHandler) GetActivity(id info.ID) ai.Activity {
	return m.activities[id]
}

func (m *ActivityHandler) ReplaceActivities(activities *[info.TEAM_SIZE]ai.Activity) {
	m.activity_lock.Lock()
	defer m.activity_lock.Unlock()
	m.activities = activities
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
