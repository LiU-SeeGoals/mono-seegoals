package ai

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/action"
	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	// Keep the robot center this far inside either goal line even when it is
	// stationary. This covers the approximately 90 mm robot radius and leaves
	// additional room for vision and controller error on the real robots.
	goalLineBaseClearanceMM = 250.0

	// The real robot cannot react to a new destination instantaneously. Reserve
	// the distance travelled during that delay and the estimated braking
	// distance when it is moving towards a goal line.
	goalLineReactionTimeS         = 0.10
	goalLineBrakeDecelerationMPS2 = 1.50
	goalLineMaxClearanceMM        = 1200.0
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
	frameMonitor := helper.NewFrameSkipMonitor(fmt.Sprintf("%s activity_executor", fb.team))
	for {
		gameInfo := <-fb.incomingGameInfo
		frameMonitor.Observe(gameInfo.VisionFrame())

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
	goalClearance := goalLineClearance(move, gi)

	move.Dest = gi.ClampToField(move.Dest, margin)
	move.Dest = clampToGoalLines(move.Dest, gi, goalClearance)
	if len(move.Path) > 0 {
		clamped := make([]info.Position, len(move.Path))
		for i, waypoint := range move.Path {
			clamped[i] = gi.ClampToField(waypoint, margin)
			clamped[i] = clampToGoalLines(clamped[i], gi, goalClearance)
		}
		move.Path = clamped
	}
	return move
}

func goalLineClearance(move *action.MoveTo, gi *info.GameInfo) float64 {
	if move == nil || gi == nil || gi.State == nil || move.Id < 0 || move.Id >= int(info.TEAM_SIZE) {
		return goalLineBaseClearanceMM
	}

	robot := gi.State.GetTeam(move.Team)[info.ID(move.Id)]
	if robot == nil {
		return goalLineBaseClearanceMM
	}

	velocity := robot.GetVelocity()
	outwardSpeed := 0.0
	if move.Pos.X > 0 {
		outwardSpeed = math.Max(0, velocity.X)
	} else if move.Pos.X < 0 {
		outwardSpeed = math.Max(0, -velocity.X)
	} else {
		outwardSpeed = math.Abs(velocity.X)
	}

	return goalLineClearanceForSpeed(outwardSpeed)
}

// goalLineClearanceForSpeed expects m/s. Robot.GetVelocity uses mm/ms, which
// has the same numeric value. The result is in millimetres.
func goalLineClearanceForSpeed(outwardSpeed float64) float64 {
	if outwardSpeed <= 0 {
		return goalLineBaseClearanceMM
	}

	reactionDistance := outwardSpeed * goalLineReactionTimeS * 1000
	brakingDistance := outwardSpeed * outwardSpeed / (2 * goalLineBrakeDecelerationMPS2) * 1000
	return math.Min(goalLineMaxClearanceMM, goalLineBaseClearanceMM+reactionDistance+brakingDistance)
}

func clampToGoalLines(pos info.Position, gi *info.GameInfo, clearance float64) info.Position {
	minX, maxX, _, _, ok := gi.FieldBounds(clearance)
	if !ok {
		return pos
	}

	pos.X = math.Max(minX, math.Min(maxX, pos.X))
	return pos
}
