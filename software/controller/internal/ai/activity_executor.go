package ai

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
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

type activityResult struct {
	index  int
	action action.Action
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
		gameInfo := <-fb.incomingGameInfo
		tickStart := time.Now()

		// Make a snapshot of current activities under lock
		fb.activity_lock.Lock()
		var activitiesCopy [info.TEAM_SIZE]ai.Activity
		copy(activitiesCopy[:], (*fb.activities)[:])
		fb.activity_lock.Unlock()

		results := make([]activityResult, 0, info.TEAM_SIZE)
		resultCh := make(chan activityResult, info.TEAM_SIZE)
		var wg sync.WaitGroup

		var i info.ID
		for i = 0; i < info.TEAM_SIZE; i++ { // Loop through all activities
			activity := activitiesCopy[i]
			if activity == nil {
				continue
			} // Skip nil activities

			wg.Add(1)
			go func(index int, activity ai.Activity) {
				defer wg.Done()

				if activity.Achieved(&gameInfo) { // If achieved, log it but let planner handle lifecycle
					Logger.Info(fmt.Sprintf("Activity achieved: %v ", activity))
					// Don't clear the activity - let the planner detect achievement and transition states
				} else { // Otherwise, get an action
					Logger.Info(fmt.Sprintf("Activity running: %v", activity))
				}
				resultCh <- activityResult{
					index:  index,
					action: activity.GetAction(&gameInfo),
				}
			}(int(i), activity)
		}

		wg.Wait()
		close(resultCh)

		for result := range resultCh {
			results = append(results, result)
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].index < results[j].index
		})

		actions := make([]action.Action, 0, len(results))
		for _, result := range results {
			actions = append(actions, result.action)
		}

		for _, action := range actions {
			if action != nil {
				// fmt.Println(fmt.Sprintf("Action: %v", action))
			}
		}

		// Send actions
		fb.outgoingActions <- actions

		helper.PaceLoop(tickStart, helper.ExecutorLoopPeriod, "activity_executor")
	}
}
