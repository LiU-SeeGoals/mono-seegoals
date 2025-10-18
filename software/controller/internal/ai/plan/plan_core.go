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

const (
	RUNNING int = iota
	COMPLETE
	TIME_EXPIRED
	ERROR
	FAILED
	REFEREE
)

type plannerCore struct {
	team             info.Team
	incomingGameInfo <-chan info.GameInfo
	scale            float64
	run_scenario     int // -1 for all
	start            time.Time
	activities       *[info.TEAM_SIZE]ai.Activity // <-- pointer to the slice
	activity_lock    *sync.Mutex                  // shared mutex for synchronization
	prev_ref         bool
	waiting_for_kick bool
}

func (m *plannerCore) ClearActivities() {
	m.activity_lock.Lock()
	defer m.activity_lock.Unlock()
	*m.activities = [info.TEAM_SIZE]ai.Activity{}
}

func (m *plannerCore) AddActivity(activity ai.Activity) {
	// m.activity_lock.Lock()
	// defer m.activity_lock.Unlock()
	idx := activity.GetID()
	Logger.Infof("Adding activity %v", activity)
	m.activities[idx] = activity
}

func (m *plannerCore) ReplaceActivities(activities *[info.TEAM_SIZE]ai.Activity) {
	m.activity_lock.Lock()
	defer m.activity_lock.Unlock()
	m.activities = activities
}

func (m *plannerCore) HandleRef(gi *info.GameInfo, active_robots []int) bool {
	kicker := info.ID(1)
	m.AddActivity(ai.NewGoalie(m.team, 0))

	switch gi.Status.GetGameEvent().GetCurrentState() {
	case info.STATE_FREE_KICK:
		if gi.Status.GetGameEvent().GetTeamWithPossession() == m.team { // We are kicker
			targetPos := info.Position{X: 300, Y: 0, Z: 0, Angle: 0} // Position for negative half
			if m.team == info.Blue && gi.Status.GetBlueTeamOnPositiveHalf() || m.team == info.Yellow && !gi.Status.GetBlueTeamOnPositiveHalf() {
				// We have the positive half
				targetPos = info.Position{X: -300, Y: 0, Z: 0, Angle: math.Pi}
			}
			m.AddActivity(ai.NewRefStop(m.team, 3))
			m.AddActivity(ai.NewRamAtPosition(m.team, 1, targetPos))
		}

		m.AddActivity(ai.NewRefStop(m.team, 3))

		m.prev_ref = true
		return true
	case info.STATE_STOPPED, info.STATE_BALL_PLACEMENT:
		if gi.Status.GetGameEvent().NextCommand == info.PREPARE_KICKOFF_YELLOW {

			m.AddActivity(ai.NewPrepareKicker(m.team, kicker))

			m.AddActivity(ai.NewRefKickoff(3, m.team))
			m.prev_ref = true
			return true

		} else if gi.Status.GetGameEvent().NextCommand == info.PREPARE_KICKOFF_BLUE {
			m.AddActivity(ai.NewRefKickoff(kicker, m.team))
			m.AddActivity(ai.NewRefKickoff(3, m.team))

			m.prev_ref = true
			return true
		} else {
			
			m.AddActivity(ai.NewStop(3))
			m.AddActivity(ai.NewStop(1))
			m.prev_ref = true
			return true
		}
	case info.STATE_HALTED, info.STATE_PENALTY_PREPARATION, info.STATE_TIMEOUT:
		for _, value := range active_robots {
			m.AddActivity(ai.NewStop(info.ID(value)))
		}
		m.prev_ref = true
		return true

	case info.STATE_KICKOFF_PREPARATION:

		// if m.waiting_for_kick {
		// 	if gi.State.Ball.GetVelocity().Norm() > 0.4 {
		// 		fmt.Println("yooyoyooy")

		// 		gi.Status.GetGameEvent().RefCommand = info.RefCommand(info.STATE_PLAYING)
		// 		m.waiting_for_kick = false
		// 		m.prev_ref = false
		// 		return false
		// 	} else {
		// 		m.prev_ref = true
		// 		return true
		// 	}
		// }

		m.AddActivity(ai.NewRefKickoff(3, m.team))
		if gi.Status.GetGameEvent().GetTeamWithPossession() == m.team { // We are kicker
			if gi.Status.GetGameEvent().RefCommand != info.NORMAL_START {
				m.AddActivity(ai.NewPrepareKicker(m.team, kicker))

			} else if m.activities[kicker] == nil { // We are preparing for kickoff but arent allowed to kick ball yet
				targetPos := info.Position{X: 300, Y: 0, Z: 0, Angle: 0} // Position for negative half
				if m.team == info.Blue && gi.Status.GetBlueTeamOnPositiveHalf() || m.team == info.Yellow && !gi.Status.GetBlueTeamOnPositiveHalf() {
					// We have the positive half
					targetPos = info.Position{X: -300, Y: 0, Z: 0, Angle: math.Pi}
				}
				m.AddActivity(ai.NewRamAtPosition(m.team, kicker, targetPos))

			}
			// m.AddActivity(ai.NewRefKickoff(info.ID(0), m.team))
		} else { // We are not kicker
			m.AddActivity(ai.NewRefKickoff(kicker, m.team))
			m.AddActivity(ai.NewRefKickoff(3, m.team))
		}
		m.waiting_for_kick = true
		m.prev_ref = true
		return true

	default:

		// If we are exiting ref activity
		if m.prev_ref == true {
			m.ClearActivities()
		}
		m.prev_ref = false
		return false
	}
}

// GetActionTypeName returns the name of the concrete type that implements the Action interface
func (m *plannerCore) GetActionTypeName(activity ai.Activity) string {
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
