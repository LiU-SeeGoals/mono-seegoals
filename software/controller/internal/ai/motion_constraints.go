package ai

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/rules"
)

// motionConstraints is the last layer between the activities and the robots:
// it makes every action legal under the current rules.MatchRules, whatever
// the planner asked for. Distances come from the rules clearance table.
//
// The passes, in order:
//  1. ball rules: no kick/dribble while the team may not touch the ball, and
//     destinations projected out of the ball keep-out;
//  2. stationary robots inside the opponent's defense area during STOP are
//     walked out (holding their heading);
//  3. optional opponent-half restriction (config);
//  4. field, goal-line and defense-area clamping;
//  5. during STOP, a final check that the straight move is legal, otherwise a
//     legal waypoint on the keep-out circle (or a stop);
//  6. the rules' speed limit (STOP, ball placement) on every MoveTo.
type motionConstraints struct {
	team                   info.Team
	escape                 defenseAreaEscapeState
	restrictToOpponentHalf bool
	// restartKicker is our robot taking the pending restart, as reported by
	// the planner (RestartKickerReporter); only it may touch the ball.
	restartKicker    info.ID
	hasRestartKicker bool
}

// beginFrame resets per-frame state and, while halted, returns the stop
// commands to send instead of any activity output.
func (c *motionConstraints) beginFrame(gi *info.GameInfo) ([]action.Action, bool) {
	if !rules.For(gi, c.team).Stopped {
		c.escape.reset()
	}
	return haltSafetyActions(gi, c.team)
}

// apply makes one action legal.
func (c *motionConstraints) apply(act action.Action, gi *info.GameInfo) action.Action {
	act = ballRulesAction(act, c.rulesFor(act, gi), gi)
	act = c.escape.apply(act, c.team, gi)
	if c.restrictToOpponentHalf {
		act = clampActionToOpponentHalf(act, gi, c.team)
	}
	act = clampMoveActionToField(act, gi)
	act = finalStoppedPlayAction(act, gi)
	if move, ok := act.(*action.MoveTo); ok {
		applySpeedLimit(move, rules.For(gi, c.team))
	}
	return act
}

type goalAreaBounds struct {
	frontX float64
	backX  float64
	minY   float64
	maxY   float64
}

type defenseAreaEscapeState struct {
	heading    [info.TEAM_SIZE]float64
	headingSet [info.TEAM_SIZE]bool
}

// rulesFor is the team's rules narrowed to the robot act drives.
func (c *motionConstraints) rulesFor(act action.Action, gi *info.GameInfo) rules.MatchRules {
	r := rules.For(gi, c.team)
	if id, ok := ballActionRobotID(act); ok {
		r = r.ForRobot(info.ID(id), c.restartKicker, c.hasRestartKicker)
	}
	return r
}

func ballActionRobotID(act action.Action) (int, bool) {
	switch typed := act.(type) {
	case *action.MoveTo:
		return typed.Id, true
	case *action.Kick:
		return typed.Id, true
	default:
		return 0, false
	}
}

// stoppedPlaySafetyAction enforces the ball rules: while the team may not
// touch the ball a stale attack activity must not kick or dribble, and every
// movement target is projected out of the ball keep-out (STOP for both teams,
// kickoffs and free kicks for the defending team).
func stoppedPlaySafetyAction(act action.Action, team info.Team, gi *info.GameInfo) action.Action {
	return ballRulesAction(act, rules.For(gi, team), gi)
}

func ballRulesAction(act action.Action, r rules.MatchRules, gi *info.GameInfo) action.Action {
	if r.MayTouchBall && r.BallKeepout == 0 {
		return act
	}

	switch typed := act.(type) {
	case *action.MoveTo:
		if !r.MayTouchBall {
			typed.KickSpeed = 0
			typed.SimKickSpeed = 0
			typed.KickAngle = 0
			typed.Dribble = false
		}
		if r.Stopped {
			typed.MinLinearSpeed = 0
		}
		typed.Dest = pathplanner.ClampDestinationOutsideBall(typed.Pos, typed.Dest, gi, r.BallKeepout)
		return typed
	case *action.Kick:
		if r.MayTouchBall {
			return act
		}
		return &action.Stop{Id: typed.Id}
	default:
		return act
	}
}

// applySpeedLimit caps move at the rules' speed limit, less SpeedLimitMargin.
func applySpeedLimit(move *action.MoveTo, r rules.MatchRules) {
	if r.MaxSpeed <= 0 || math.IsInf(r.MaxSpeed, 1) {
		return
	}
	limit := math.Max(0.1, r.MaxSpeed-rules.SpeedLimitMargin)
	if move.MaxLinearSpeed == 0 || move.MaxLinearSpeed > limit {
		move.MaxLinearSpeed = limit
	}
}

// Validate after all field and defense-area projections. A projection can undo
// ball clearance, and checking only the endpoint also permits crossing the ball.
func finalStoppedPlayAction(act action.Action, gi *info.GameInfo) action.Action {
	move, ok := act.(*action.MoveTo)
	if !ok || gi == nil || gi.State == nil {
		return act
	}
	r := rules.For(gi, move.Team)
	if !r.Stopped {
		return act
	}
	ball, err := gi.State.GetBall().GetPosition()
	if err != nil {
		return &action.Stop{Id: move.Id}
	}
	radius := r.BallKeepout
	areas := getGoalAreaBounds(gi)
	clearance := defenseAreaClearance(goalLineClearance(move, gi), r)
	obstacles := pathplanner.ObstaclesForRobot(move.Team, info.ID(move.Id), false, false, gi)
	legal := func(dest info.Position) bool {
		if dest.Dist2d(ball) < radius ||
			(!move.AllowGoalArea && positionInGoalArea(dest, areas, clearance)) {
			return false
		}
		if move.Pos.Dist2d(ball) < radius {
			// A robot already too close may escape, but must not first approach
			// the ball on its way to a nominally legal destination.
			if (dest.X-move.Pos.X)*(move.Pos.X-ball.X)+
				(dest.Y-move.Pos.Y)*(move.Pos.Y-ball.Y) < 0 {
				return false
			}
		}
		return pathplanner.IsPathClear(move.Pos, dest, obstacles, 0)
	}
	if legal(move.Dest) {
		return move
	}

	// Search the keepout perimeter for a reachable waypoint. Run each candidate
	// through the same field/defense constraints, then validate their intersection.
	bestDistance := math.Inf(1)
	var best *action.MoveTo
	for i := 0; i < 72; i++ {
		angle := float64(i) * 2 * math.Pi / 72
		// Build a fresh action; MoveTo's cached protobuf contains a mutex and
		// must not be shallow-copied after serialization.
		candidate := action.MoveTo{
			Id: move.Id, Team: move.Team, Pos: move.Pos, Dest: move.Dest,
			AllowOutsideField:   move.AllowOutsideField,
			AllowBehindGoalLine: move.AllowBehindGoalLine,
			AllowGoalArea:       move.AllowGoalArea,
			MaxLinearSpeed:      move.MaxLinearSpeed,
		}
		candidate.Dest.X = ball.X + (radius+1)*math.Cos(angle)
		candidate.Dest.Y = ball.Y + (radius+1)*math.Sin(angle)
		clampMoveActionToField(&candidate, gi)
		if !legal(candidate.Dest) {
			continue
		}
		if distance := candidate.Dest.Dist2d(move.Dest); distance < bestDistance {
			bestDistance = distance
			best = &candidate
		}
	}
	if best == nil {
		return &action.Stop{Id: move.Id}
	}
	return best
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

	r := rules.For(gi, team)
	if !r.Stopped {
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

	clearance := defenseAreaClearance(rules.GoalLineBaseClearance, r)
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
func haltSafetyActions(gi *info.GameInfo, team info.Team) ([]action.Action, bool) {
	if !rules.For(gi, team).Halted {
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
	defenseAreaMargin := defenseAreaClearance(goalLineMargin, rules.For(gi, move.Team))

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

// defenseAreaClearance is the larger of the motion clearance and the rules'
// minimum (rules.DefenseAreaStoppedClearance while the ball is out of play).
func defenseAreaClearance(motionClearance float64, r rules.MatchRules) float64 {
	return math.Max(motionClearance, r.DefenseAreaClearance)
}

func goalLineClearance(move *action.MoveTo, gi *info.GameInfo) float64 {
	if move == nil || gi == nil || gi.State == nil || move.Id < 0 || move.Id >= int(info.TEAM_SIZE) {
		return rules.GoalLineBaseClearance
	}

	robot := gi.State.GetTeam(move.Team)[info.ID(move.Id)]
	if robot == nil {
		return rules.GoalLineBaseClearance
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
		return rules.GoalLineBaseClearance
	}

	reactionDistance := outwardSpeed * rules.GoalLineReactionTime * 1000
	brakingDistance := outwardSpeed * outwardSpeed / (2 * rules.GoalLineBrakeDeceleration) * 1000
	return math.Min(rules.GoalLineMaxClearance, rules.GoalLineBaseClearance+reactionDistance+brakingDistance)
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
	if gi == nil {
		return nil
	}
	areas := make([]goalAreaBounds, 0, 2)
	for _, area := range gi.DefenseAreas() {
		areas = append(areas, goalAreaBounds{frontX: area.FrontX, backX: area.BackX, minY: area.MinY, maxY: area.MaxY})
	}
	return areas
}
