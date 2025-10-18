package ai

import (
	"fmt"
	"sync"

	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type activityExecutor struct {
	team             info.Team
	incomingGameInfo <-chan info.GameInfo
	outgoingActions  chan<- []action.Action
	activities       *[info.TEAM_SIZE]ai.Activity // <-- pointer to a slice
	activity_lock    *sync.Mutex                  // shared mutex for synchronization
}

func NewActivityExecutor() *activityExecutor {
	return &activityExecutor{}
}

func (fb *activityExecutor) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	outgoing chan<- []action.Action,
	team info.Team,
) {
	fb.incomingGameInfo = incoming
	fb.outgoingActions = outgoing
	fb.team = team
	fb.activity_lock = lock

	// Store the pointer directly
	fb.activities = activities

	go fb.Run()
}

func (fb *activityExecutor) Run() {
	for {
		// For example, throttle the loop slightly to avoid busy-loop:
		time.Sleep(1 * time.Millisecond) // or read from fb.incomingGameInfo if event-driven

		gameInfo := <-fb.incomingGameInfo

		// Make a snapshot of current activities under lock
		fb.activity_lock.Lock()
		var activitiesCopy [info.TEAM_SIZE]ai.Activity
		copy(activitiesCopy[:], (*fb.activities)[:])
		fb.activity_lock.Unlock()

		var actions []action.Action
		var i info.ID
		for i = 0; i < info.TEAM_SIZE; i++ { // Loop through all activities
			if activitiesCopy[i] == nil {
				continue
			} // Skip nil activities

			if activitiesCopy[i].Achieved(&gameInfo) { // If achieved, remove it

				Logger.Info(fmt.Sprintf("Activity achieved: %v ", activitiesCopy[i]))
				fb.activity_lock.Lock()
				fb.activities[i] = nil
				fb.activity_lock.Unlock()
			} else { // Otherwise, get an action

				Logger.Info(fmt.Sprintf("Activity running: %v", activitiesCopy[i]))
				actions = append(actions, activitiesCopy[i].GetAction(&gameInfo))
			}
		}

		for _, action := range actions {
			if action != nil {
				// fmt.Println(fmt.Sprintf("Action: %v", action))
			}
		}

		// Send actions
		fb.outgoingActions <- actions
	}
}
