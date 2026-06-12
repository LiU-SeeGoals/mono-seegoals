package ai

import (
	"sort"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
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

				_ = activity.Achieved(&gameInfo)
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
			actions = append(actions, clampMoveActionToField(result.action, &gameInfo))
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

func clampMoveActionToField(act action.Action, gi *info.GameInfo) action.Action {
	move, ok := act.(*action.MoveTo)
	if !ok || gi == nil || !gi.HasField() {
		return act
	}

	margin := 0.0
	if move.AllowOutsideField {
		margin = -gi.FieldBoundaryWidth()
	}

	move.Dest = gi.ClampToField(move.Dest, margin)
	if len(move.Path) > 0 {
		clamped := make([]info.Position, len(move.Path))
		for i, waypoint := range move.Path {
			clamped[i] = gi.ClampToField(waypoint, margin)
		}
		move.Path = clamped
	}
	return move
}
