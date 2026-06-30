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

type goalAreaBounds struct {
	frontX float64
	backX  float64
	minY   float64
	maxY   float64
}

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

	originalDestX := move.Dest.X
	move.Dest = gi.ClampToField(move.Dest, margin)
	if !move.AllowBehindGoalLine {
		move.Dest = clampToGoalLines(move.Dest, gi, goalClearance)
	}
	goalAreaAdjusted := false
	var goalAreas []goalAreaBounds
	if !move.AllowGoalArea {
		goalAreas = getGoalAreaBounds(gi)
		move.Dest, goalAreaAdjusted = clampGoalAreaMotion(move.Pos, move.Dest, goalAreas, goalClearance)
	}

	// Kicks are a distinct firmware action which continues position control
	// while charging. If safety changed its drive-through destination, or the
	// robot is already inside a forbidden goal area, use a normal MoveTo command
	// so it brakes/retreats without arming the kicker.
	if move.KickSpeed != 0 && (move.Dest.X != originalDestX || goalAreaAdjusted ||
		(!move.AllowGoalArea && positionInGoalArea(move.Pos, goalAreas, goalLineBaseClearanceMM))) {
		move.KickSpeed = 0
	}

	if len(move.Path) > 0 {
		clamped := make([]info.Position, len(move.Path))
		for i, waypoint := range move.Path {
			clamped[i] = gi.ClampToField(waypoint, margin)
			if !move.AllowBehindGoalLine {
				clamped[i] = clampToGoalLines(clamped[i], gi, goalClearance)
			}
			if !move.AllowGoalArea {
				clamped[i], _ = clampOutsideGoalAreas(clamped[i], goalAreas, goalClearance)
			}
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

func clampOutsideGoalAreas(pos info.Position, areas []goalAreaBounds, clearance float64) (info.Position, bool) {
	for _, area := range areas {
		minX := math.Min(area.frontX, area.backX) - clearance
		maxX := math.Max(area.frontX, area.backX) + clearance
		minY := area.minY - clearance
		maxY := area.maxY + clearance
		if pos.X < minX || pos.X > maxX || pos.Y < minY || pos.Y > maxY {
			continue
		}

		if area.backX < area.frontX {
			pos.X = area.frontX + clearance
		} else {
			pos.X = area.frontX - clearance
		}
		return pos, true
	}
	return pos, false
}

// clampGoalAreaMotion also catches a direct kick approach whose destination is
// outside a goal area but whose straight drive-through segment cuts a corner.
func clampGoalAreaMotion(from, dest info.Position, areas []goalAreaBounds, clearance float64) (info.Position, bool) {
	if clamped, adjusted := clampOutsideGoalAreas(dest, areas, clearance); adjusted {
		return clamped, true
	}
	if positionInGoalArea(from, areas, clearance) {
		// Do not block a robot which is already inside from retreating.
		return dest, false
	}

	for _, area := range areas {
		minX := math.Min(area.frontX, area.backX) - clearance
		maxX := math.Max(area.frontX, area.backX) + clearance
		minY := area.minY - clearance
		maxY := area.maxY + clearance
		if t, ok := segmentRectangleEntry(from, dest, minX, maxX, minY, maxY); ok {
			dx := dest.X - from.X
			dy := dest.Y - from.Y
			length := math.Hypot(dx, dy)
			if length > 0 {
				t = math.Max(0, t-1.0/length) // remain 1 mm outside the inflated area
			}
			dest.X = from.X + t*dx
			dest.Y = from.Y + t*dy
			return dest, true
		}
	}
	return dest, false
}

func segmentRectangleEntry(from, to info.Position, minX, maxX, minY, maxY float64) (float64, bool) {
	tEnter, tExit := 0.0, 1.0
	dx := to.X - from.X
	dy := to.Y - from.Y
	for _, edge := range [][2]float64{
		{-dx, from.X - minX},
		{dx, maxX - from.X},
		{-dy, from.Y - minY},
		{dy, maxY - from.Y},
	} {
		p, q := edge[0], edge[1]
		if p == 0 {
			if q < 0 {
				return 0, false
			}
			continue
		}
		r := q / p
		if p < 0 {
			if r > tExit {
				return 0, false
			}
			tEnter = math.Max(tEnter, r)
		} else {
			if r < tEnter {
				return 0, false
			}
			tExit = math.Min(tExit, r)
		}
	}
	return tEnter, tEnter <= tExit
}

func positionInGoalArea(pos info.Position, areas []goalAreaBounds, clearance float64) bool {
	_, adjusted := clampOutsideGoalAreas(pos, areas, clearance)
	return adjusted
}

func getGoalAreaBounds(gi *info.GameInfo) []goalAreaBounds {
	if gi == nil || !gi.HasField() {
		return nil
	}

	areas := make([]goalAreaBounds, 0, 2)
	for _, names := range [][2]string{
		{"LeftPenaltyStretch", "LeftGoalLine"},
		{"RightPenaltyStretch", "RightGoalLine"},
	} {
		front := gi.GetFieldLine(names[0])
		back := gi.GetFieldLine(names[1])
		if front == nil || back == nil || front.GetP1() == nil || front.GetP2() == nil || back.GetP1() == nil {
			continue
		}
		areas = append(areas, goalAreaBounds{
			frontX: float64(front.GetP1().GetX()),
			backX:  float64(back.GetP1().GetX()),
			minY:   math.Min(float64(front.GetP1().GetY()), float64(front.GetP2().GetY())),
			maxY:   math.Max(float64(front.GetP1().GetY()), float64(front.GetP2().GetY())),
		})
	}
	return areas
}
