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
	// This should make auto ref happy could be sligtly lower but not a lot.
	goalLineBaseClearanceMM = 150.0
	// During stopped play, keep the robot body 300 mm from either defense area.
	defenseAreaBallOutBodyClearanceMM = 300.0
	sslRobotRadiusMM                  = 90.0

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
	team              info.Team
	incomingGameInfo  <-chan info.GameInfo
	outgoingActions   chan<- []action.Action
	activities        *[info.TEAM_SIZE]ai.Activity // <-- pointer to a slice
	activity_lock     *sync.Mutex                  // shared mutex for synchronization
	defenseAreaEscape defenseAreaEscapeState
}

type defenseAreaEscapeState struct {
	heading    [info.TEAM_SIZE]float64
	headingSet [info.TEAM_SIZE]bool
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
		if gameInfo.Status == nil || gameInfo.Status.GetGameEvent() == nil ||
			gameInfo.Status.GetGameEvent().GetCurrentState() != info.STATE_STOPPED {
			fb.defenseAreaEscape.reset()
		}

		// HALT is a hard safety boundary. Do not depend on the slower planner
		// replacing every activity before this executor snapshots them: an old
		// kick activity must never produce a command from a halted frame.
		if actions, halted := haltSafetyActions(&gameInfo); halted {
			fb.outgoingActions <- actions
			continue
		}

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
			safeAction := fb.defenseAreaEscape.apply(result.action, fb.team, &gameInfo)
			actions = append(actions, clampMoveActionToField(safeAction, &gameInfo))
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

// apply moves a stationary robot out of the opponent's defense area and holds
// the heading it had when the escape began. RefStop can produce either Stop or
// MoveTo depending on ball distance; both paths must suppress rotation.
func (s *defenseAreaEscapeState) apply(act action.Action, team info.Team, gi *info.GameInfo) action.Action {
	id, supported := actionRobotID(act)
	if !supported || id < 0 || id >= int(info.TEAM_SIZE) ||
		gi == nil || gi.State == nil || gi.Status == nil || !gi.HasField() {
		return act
	}

	gameEvent := gi.Status.GetGameEvent()
	if gameEvent == nil || gameEvent.GetCurrentState() != info.STATE_STOPPED {
		s.headingSet[id] = false
		return act
	}

	robot := gi.State.GetTeam(team)[info.ID(id)]
	if robot == nil {
		return act
	}
	robotPos, err := robot.GetPosition()
	if err != nil {
		return act
	}

	clearance := defenseAreaClearance(goalLineBaseClearanceMM, gi)
	enemyGoalSign := -gi.OwnHalfXSign(team)
	for _, area := range getGoalAreaBounds(gi) {
		if area.frontX*enemyGoalSign <= 0 {
			continue
		}
		target, inside := nearestDefenseAreaExit(robotPos, area, clearance)
		if !inside {
			continue
		}
		if !s.headingSet[id] {
			s.heading[id] = robotPos.Angle
			s.headingSet[id] = true
		}
		if move, ok := act.(*action.MoveTo); ok {
			move.Dest.Angle = s.heading[id]
			return move
		}
		target.Angle = s.heading[id]
		return &action.MoveTo{
			Id:   id,
			Team: team,
			Pos:  robotPos,
			Dest: target,
		}
	}
	s.headingSet[id] = false
	return act
}

func actionRobotID(act action.Action) (id int, supported bool) {
	switch typed := act.(type) {
	case *action.Stop:
		return typed.Id, true
	case *action.MoveTo:
		return typed.Id, true
	default:
		return 0, false
	}
}

func (s *defenseAreaEscapeState) reset() {
	for i := range s.headingSet {
		s.headingSet[i] = false
	}
}

// nearestDefenseAreaExit returns the shortest exit through the field-facing
// front or either side. It intentionally excludes the back edge so a robot is
// never directed behind the goal line. This follows Sumatra's nearest-point-
// outside behavior for invalid penalty-area positions.
func nearestDefenseAreaExit(pos info.Position, area goalAreaBounds, clearance float64) (info.Position, bool) {
	minX := math.Min(area.frontX, area.backX) - clearance
	maxX := math.Max(area.frontX, area.backX) + clearance
	minY := area.minY - clearance
	maxY := area.maxY + clearance
	if pos.X < minX || pos.X > maxX || pos.Y < minY || pos.Y > maxY {
		return pos, false
	}

	const outsideEpsilonMM = 1.0
	frontExit := pos
	if area.backX < area.frontX {
		frontExit.X = maxX + outsideEpsilonMM
	} else {
		frontExit.X = minX - outsideEpsilonMM
	}
	candidates := []info.Position{
		frontExit,
		{X: pos.X, Y: minY - outsideEpsilonMM},
		{X: pos.X, Y: maxY + outsideEpsilonMM},
	}

	best := candidates[0]
	bestDistance := pos.Dist2d(best)
	for _, candidate := range candidates[1:] {
		if distance := pos.Dist2d(candidate); distance < bestDistance {
			best = candidate
			bestDistance = distance
		}
	}
	return best, true
}

// haltSafetyActions returns explicit stop commands for every possible robot
// ID. Stopping the complete ID range also covers robots that disappeared from
// vision or became active after the referee handler was initialized.
func haltSafetyActions(gi *info.GameInfo) ([]action.Action, bool) {
	if gi == nil || gi.Status == nil {
		return nil, false
	}

	gameEvent := gi.Status.GetGameEvent()
	if gameEvent == nil || gameEvent.GetCurrentState() != info.STATE_HALTED {
		return nil, false
	}

	actions := make([]action.Action, 0, info.TEAM_SIZE)
	for id := info.ID(0); id < info.TEAM_SIZE; id++ {
		actions = append(actions, &action.Stop{Id: int(id)})
	}
	return actions, true
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
	goalLineMargin := goalLineClearance(move, gi)
	defenseAreaMargin := defenseAreaClearance(goalLineMargin, gi)

	originalDestX := move.Dest.X
	move.Dest = gi.ClampToField(move.Dest, margin)
	if !move.AllowBehindGoalLine {
		move.Dest = clampToGoalLines(move.Dest, gi, goalLineMargin)
	}
	goalAreaAdjusted := false
	var goalAreas []goalAreaBounds
	if !move.AllowGoalArea {
		goalAreas = getGoalAreaBounds(gi)
		move.Dest, goalAreaAdjusted = clampGoalAreaMotion(move.Pos, move.Dest, goalAreas, defenseAreaMargin)
	}

	// Kicks are a distinct firmware action which continues position control
	// while charging. If safety changed its drive-through destination, or the
	// robot is already inside a forbidden goal area, use a normal MoveTo command
	// so it brakes/retreats without arming the kicker.
	if move.KickSpeed != 0 && (move.Dest.X != originalDestX || goalAreaAdjusted ||
		(!move.AllowGoalArea && positionInGoalArea(move.Pos, goalAreas, defenseAreaMargin))) {
		move.KickSpeed = 0
	}

	if len(move.Path) > 0 {
		clamped := make([]info.Position, len(move.Path))
		for i, waypoint := range move.Path {
			clamped[i] = gi.ClampToField(waypoint, margin)
			if !move.AllowBehindGoalLine {
				clamped[i] = clampToGoalLines(clamped[i], gi, goalLineMargin)
			}
			if !move.AllowGoalArea {
				clamped[i], _ = clampOutsideGoalAreas(clamped[i], goalAreas, defenseAreaMargin)
			}
		}
		move.Path = clamped
	}
	return move
}

func defenseAreaClearance(motionClearance float64, gi *info.GameInfo) float64 {
	if gi == nil || gi.Status == nil {
		return motionClearance
	}

	gameEvent := gi.Status.GetGameEvent()
	if gameEvent == nil || gameEvent.BallInPlay {
		return motionClearance
	}

	requiredCenterClearance := defenseAreaBallOutBodyClearanceMM + sslRobotRadiusMM
	return math.Max(motionClearance, requiredCenterClearance)
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
	if len(areas) == 2 {
		return areas
	}

	// Some SSL-Vision sources report the field dimensions without individual
	// line segments. The dimensions still fully describe the rectangular
	// defense areas, so do not disable this safety behavior in that case.
	geometry, ok := gi.FieldGeometry()
	if !ok || geometry.PenaltyAreaDepth <= 0 || geometry.PenaltyAreaWidth <= 0 {
		return areas
	}
	halfLength := geometry.Length / 2
	halfPenaltyWidth := geometry.PenaltyAreaWidth / 2
	return []goalAreaBounds{
		{
			frontX: -halfLength + geometry.PenaltyAreaDepth,
			backX:  -halfLength,
			minY:   -halfPenaltyWidth,
			maxY:   halfPenaltyWidth,
		},
		{
			frontX: halfLength - geometry.PenaltyAreaDepth,
			backX:  halfLength,
			minY:   -halfPenaltyWidth,
			maxY:   halfPenaltyWidth,
		},
	}
}
