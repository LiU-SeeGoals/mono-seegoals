// Package pathplanner holds RRT path planning and obstacle helpers used by activities.
package pathplanner

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Field/planning constants (mm) — same semantics as the former move_to_position locals.
const (
	RobotSafetyRadius        = 240.0
	BallSafetyRadius         = 150.0
	RestartBallKeepoutRadius = 500.0
	GoalLineSafetyRadius     = 100.0
	PlanningRadius           = 400.0
	MotionRadius             = 100.0
)

const noGoZoneEscapeClearance = MotionRadius
const ballKeepoutDestinationEpsilon = 1.0

// Defaults for persistent path reuse (used when the corresponding RRTConfig field is zero).
const (
	defaultCloseRobotMaxPlanAge = 250 * time.Millisecond
	defaultFarRobotMaxPlanAge   = 2 * time.Second
	defaultNearRobotDistance    = 1000.0 // mm, other robot close enough to require young paths
	defaultGoalMatchEpsilon     = 10.0   // mm, goal “same” if closer than this

	defaultGoalProximityDistance           = 1000.0 // mm, near goal requires young paths
	defaultBallGoalProximityDistance       = 500.0  // mm, ball close enough to the final goal
	defaultBallApproachRobotIgnoreDistance = 500.0  // mm, robot close enough to accept robot contact
)

// RRTConfig holds parameters for the RRT algorithm and optional persistence gating.
type RRTConfig struct {
	MaxIterations      int
	StepSize           float64
	GoalBias           float64
	WaypointThreshold  float64
	FieldWidth         float64
	FieldHeight        float64
	CompletionDistance float64
	// Persistence / replan (zero = use package defaults for that field).
	CloseRobotMaxPlanAge time.Duration
	FarRobotMaxPlanAge   time.Duration
	NearRobotDistance    float64
	GoalMatchEpsilon     float64

	GoalProximityDistance           float64
	BallGoalProximityDistance       float64
	BallApproachRobotIgnoreDistance float64
}

type resolvedPersistence struct {
	closeRobotMaxPlanAge time.Duration
	farRobotMaxPlanAge   time.Duration
	nearRobotDistance    float64
	goalMatchEpsilon     float64

	goalProximityDistance           float64
	ballGoalProximityDistance       float64
	ballApproachRobotIgnoreDistance float64
}

func (c RRTConfig) persistence() resolvedPersistence {
	r := resolvedPersistence{
		closeRobotMaxPlanAge: c.CloseRobotMaxPlanAge,
		farRobotMaxPlanAge:   c.FarRobotMaxPlanAge,
		nearRobotDistance:    c.NearRobotDistance,
		goalMatchEpsilon:     c.GoalMatchEpsilon,

		goalProximityDistance:           c.GoalProximityDistance,
		ballGoalProximityDistance:       c.BallGoalProximityDistance,
		ballApproachRobotIgnoreDistance: c.BallApproachRobotIgnoreDistance,
	}
	if r.closeRobotMaxPlanAge == 0 {
		r.closeRobotMaxPlanAge = defaultCloseRobotMaxPlanAge
	}
	if r.farRobotMaxPlanAge == 0 {
		r.farRobotMaxPlanAge = defaultFarRobotMaxPlanAge
	}
	if r.nearRobotDistance == 0 {
		r.nearRobotDistance = defaultNearRobotDistance
	}
	if r.goalMatchEpsilon == 0 {
		r.goalMatchEpsilon = defaultGoalMatchEpsilon
	}
	if r.goalProximityDistance == 0 {
		r.goalProximityDistance = defaultGoalProximityDistance
	}
	if r.ballGoalProximityDistance == 0 {
		r.ballGoalProximityDistance = defaultBallGoalProximityDistance
	}
	if r.ballApproachRobotIgnoreDistance == 0 {
		r.ballApproachRobotIgnoreDistance = defaultBallApproachRobotIgnoreDistance
	}
	return r
}

// RRTNode represents a node in the RRT tree.
type RRTNode struct {
	position info.Position
	parent   *RRTNode
}

// Obstacle is either a disc obstacle (robots/ball) or a rectangular no-go zone.
type Obstacle struct {
	Position info.Position
	Size     float64
	rect     bool
	minX     float64
	maxX     float64
	minY     float64
	maxY     float64
}

// robotPathState is per-robot cached plan + metadata for persistence.
type robotPathState struct {
	path      []info.Position
	plannedAt time.Time
	goal      info.Position
}

// Planner stores per-robot path state. One instance per team is expected.
type Planner struct {
	mu      sync.Mutex
	session map[info.ID]*robotPathState
}

// New returns a planner with empty per-robot state.
func New() *Planner {
	return &Planner{
		session: make(map[info.ID]*robotPathState),
	}
}

// Clear removes stored path state for a robot. Safe to call repeatedly.
func (p *Planner) Clear(id info.ID) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.session, id)
}

// PlanPath runs RRT or returns a cached path while it is valid and young enough.
func (p *Planner) PlanPath(
	team info.Team,
	id info.ID,
	finalDestination info.Position,
	avoidBall bool,
	avoidGoallines bool,
	gi *info.GameInfo,
	cfg RRTConfig,
) []info.Position {
	if p == nil {
		return nil
	}
	perm := cfg.persistence()
	myPos, _ := gi.State.GetTeam(team)[id].GetPosition()
	finalDestination = clampToPlanningBounds(finalDestination, cfg)
	finalDestination = ClampBallKeepoutDestination(team, myPos, finalDestination, gi)
	finalDestination = clampToPlanningBounds(finalDestination, cfg)
	obstacles := planningObstacles(team, id, myPos, finalDestination, avoidBall, avoidGoallines, gi, perm)

	if escapePath, ok := noGoZoneEscapePath(myPos, obstacles); ok {
		return clampPathToPlanningBounds(escapePath, cfg)
	}

	// Inflated “collision”: escape this tick only. Do not commit: overwriting the
	// session with a 1-point path is common near the goal in traffic and made the
	// full plan appear to “vanish” until age-based replanning created a new path.
	robotsNearby, nearest := collisionTarget(myPos, obstacles)
	if robotsNearby {
		return clampPathToPlanningBounds(makeEscapePath(myPos, nearest), cfg)
	}

	p.mu.Lock()
	st := p.session[id]
	p.mu.Unlock()

	if st != nil && len(st.path) > 0 {
		if !shouldCreateNewPath(st, myPos, finalDestination, obstacles, gi, team, id, cfg, perm) {
			return p.copyPath(id)
		}
	}

	// Full RRT
	startNode := &RRTNode{position: myPos, parent: nil}
	nodes := []*RRTNode{startNode}
	goalNode := runRRT(nodes, obstacles, finalDestination, cfg)
	if goalNode == nil {
		p.mu.Lock()
		delete(p.session, id)
		p.mu.Unlock()
		return nil
	}
	path := []info.Position{}
	for cur := goalNode; cur != nil; cur = cur.parent {
		path = append([]info.Position{cur.position}, path...)
	}
	path = shortcutPath(path, obstacles, PlanningRadius)
	path = clampPathToPlanningBounds(path, cfg)
	p.commit(id, path, finalDestination)
	return p.copyPath(id)
}

func (p *Planner) copyPath(id info.ID) []info.Position {
	p.mu.Lock()
	defer p.mu.Unlock()
	st := p.session[id]
	if st == nil || len(st.path) == 0 {
		return nil
	}
	return append([]info.Position(nil), st.path...)
}

func newCommittedState(
	path []info.Position,
	goal info.Position,
) *robotPathState {
	s := &robotPathState{
		path:      append([]info.Position(nil), path...),
		plannedAt: time.Now(),
		goal:      goal,
	}
	return s
}

func (p *Planner) commit(
	id info.ID,
	path []info.Position,
	goal info.Position,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.session[id] = newCommittedState(path, goal)
}

func shouldCreateNewPath(
	st *robotPathState,
	myPos, goal info.Position,
	obstacles []Obstacle,
	gi *info.GameInfo,
	team info.Team,
	selfID info.ID,
	cfg RRTConfig,
	perm resolvedPersistence,
) bool {
	if st == nil || len(st.path) == 0 {
		return true
	}
	if !isStoredPathValid(st, myPos, goal, obstacles, cfg, perm.goalMatchEpsilon) {
		return true
	}
	if time.Since(st.plannedAt) > maxPlanAgeForContext(myPos, goal, team, selfID, gi, perm) {
		return true
	}
	return false
}

// isStoredPathValid checks whether the cached polyline can still be used now.
func isStoredPathValid(
	st *robotPathState,
	myPos, goal info.Position,
	obstacles []Obstacle,
	cfg RRTConfig,
	goalMatchEpsilon float64,
) bool {
	if st == nil || len(st.path) == 0 {
		return false
	}
	if !pathWithinPlanningBounds(st.path, cfg) {
		return false
	}
	if distanceSquared(st.goal, goal) > goalMatchEpsilon*goalMatchEpsilon {
		return false
	}
	if !IsPathClear(myPos, st.path[0], obstacles, PlanningRadius) {
		return false
	}
	for i := 0; i < len(st.path)-1; i++ {
		if !IsPathClear(st.path[i], st.path[i+1], obstacles, PlanningRadius) {
			return false
		}
	}
	return true
}

func hasPlanningBounds(cfg RRTConfig) bool {
	return cfg.FieldWidth > 0 && cfg.FieldHeight > 0
}

func clampToPlanningBounds(pos info.Position, cfg RRTConfig) info.Position {
	if !hasPlanningBounds(cfg) {
		return pos
	}

	halfX := cfg.FieldWidth / 2
	halfY := cfg.FieldHeight / 2
	pos.X = math.Max(-halfX, math.Min(halfX, pos.X))
	pos.Y = math.Max(-halfY, math.Min(halfY, pos.Y))
	return pos
}

func pathWithinPlanningBounds(path []info.Position, cfg RRTConfig) bool {
	if !hasPlanningBounds(cfg) {
		return true
	}

	halfX := cfg.FieldWidth / 2
	halfY := cfg.FieldHeight / 2
	for _, pos := range path {
		if pos.X < -halfX || pos.X > halfX || pos.Y < -halfY || pos.Y > halfY {
			return false
		}
	}
	return true
}

func clampPathToPlanningBounds(path []info.Position, cfg RRTConfig) []info.Position {
	if len(path) == 0 || !hasPlanningBounds(cfg) {
		return path
	}

	clamped := make([]info.Position, len(path))
	for i, pos := range path {
		clamped[i] = clampToPlanningBounds(pos, cfg)
	}
	return clamped
}

func maxPlanAgeForContext(
	myPos, goal info.Position,
	team info.Team,
	selfID info.ID,
	gi *info.GameInfo,
	perm resolvedPersistence,
) time.Duration {
	if distanceSquared(myPos, goal) < perm.goalProximityDistance*perm.goalProximityDistance {
		return perm.closeRobotMaxPlanAge
	}
	for _, t := range []info.Team{info.Blue, info.Yellow} {
		for i := 0; i < int(info.TEAM_SIZE); i++ {
			oid := info.ID(i)
			if t == team && oid == selfID {
				continue
			}
			robot := gi.State.GetTeam(t)[oid]
			pos, rototTime, err := robot.GetPositionTime()
			if err != nil {
				continue
			}
			if time.Now().UnixMilli()-rototTime > 200 {
				continue
			}
			if distanceSquared(myPos, pos) < perm.nearRobotDistance*perm.nearRobotDistance {
				return perm.closeRobotMaxPlanAge
			}
		}
	}
	return perm.farRobotMaxPlanAge
}

func planningObstacles(
	team info.Team,
	id info.ID,
	myPos, finalDestination info.Position,
	avoidBall bool,
	avoidGoallines bool,
	gi *info.GameInfo,
	perm resolvedPersistence,
) []Obstacle {
	if !RestartBallKeepoutActive(team, gi) &&
		ignoreRobotObstaclesForBallApproach(myPos, finalDestination, avoidBall, gi, perm) {
		ballPos, _ := gi.State.Ball.GetPosition()
		obstacles := []Obstacle{{Position: ballPos, Size: BallSafetyRadius}}
		if avoidGoallines {
			obstacles = append(obstacles, goalLineObstacles(gi)...)
		}
		return obstacles
	}
	return ObstaclesForRobot(team, id, avoidBall, avoidGoallines, gi)
}

func ignoreRobotObstaclesForBallApproach(
	myPos, finalDestination info.Position,
	avoidBall bool,
	gi *info.GameInfo,
	perm resolvedPersistence,
) bool {
	if !avoidBall {
		return false
	}
	if distanceSquared(myPos, finalDestination) > perm.ballApproachRobotIgnoreDistance*perm.ballApproachRobotIgnoreDistance {
		return false
	}
	ballPos, _ := gi.State.Ball.GetPosition()
	return distanceSquared(ballPos, finalDestination) <= perm.ballGoalProximityDistance*perm.ballGoalProximityDistance
}

// RestartBallKeepoutActive reports whether this team must keep the restart
// distance from the ball. This is intentionally narrower than
// GameEvent.ShouldKeepDistanceFromBall, because the team taking a restart still
// needs to approach and kick the ball.
func RestartBallKeepoutActive(team info.Team, gi *info.GameInfo) bool {
	if gi == nil || gi.Status == nil {
		return false
	}
	gameEvent := gi.Status.GetGameEvent()
	if gameEvent == nil || gameEvent.BallInPlay {
		return false
	}

	switch gameEvent.CurrentState {
	case info.STATE_FREE_KICK, info.STATE_KICKOFF_PREPARATION:
		return gameEvent.TeamWithPossession != team
	default:
		return false
	}
}

// ClampBallKeepoutDestination projects illegal restart destinations to the
// nearest point just outside the required ball keep-out circle.
func ClampBallKeepoutDestination(
	team info.Team,
	myPos info.Position,
	destination info.Position,
	gi *info.GameInfo,
) info.Position {
	if !RestartBallKeepoutActive(team, gi) || gi == nil || gi.State == nil || gi.State.Ball == nil {
		return destination
	}
	ballPos, err := gi.State.Ball.GetPosition()
	if err != nil {
		return destination
	}
	return projectOutsideBallKeepout(destination, ballPos, myPos, RestartBallKeepoutRadius+ballKeepoutDestinationEpsilon)
}

func projectOutsideBallKeepout(destination, ballPos, fallback info.Position, radius float64) info.Position {
	dx := destination.X - ballPos.X
	dy := destination.Y - ballPos.Y
	distSq := dx*dx + dy*dy
	if distSq > radius*radius {
		return destination
	}

	if distSq == 0 {
		dx = fallback.X - ballPos.X
		dy = fallback.Y - ballPos.Y
		distSq = dx*dx + dy*dy
	}
	if distSq == 0 {
		dx = 1
		dy = 0
		distSq = 1
	}

	scale := radius / math.Sqrt(distSq)
	destination.X = ballPos.X + dx*scale
	destination.Y = ballPos.Y + dy*scale
	return destination
}

func collisionTarget(myPos info.Position, obstacles []Obstacle) (bool, Obstacle) {
	robotsNearby := false
	var nearestObstacle Obstacle
	shortestDistSq := math.MaxFloat64
	for _, obstacle := range obstacles {
		if obstacle.rect {
			continue
		}
		distSq := distanceSquared(myPos, obstacle.Position)
		obstacleSizeSq := obstacle.Size * obstacle.Size
		if distSq <= obstacleSizeSq {
			robotsNearby = true
			if distSq < shortestDistSq {
				shortestDistSq = distSq
				nearestObstacle = obstacle
			}
		}
	}
	return robotsNearby, nearestObstacle
}

func noGoZoneEscapePath(myPos info.Position, obstacles []Obstacle) ([]info.Position, bool) {
	found := false
	bestDistSq := math.MaxFloat64
	var best info.Position

	for _, obstacle := range obstacles {
		if !obstacle.rect {
			continue
		}
		target, distSq, inside := nearestRectExit(myPos, obstacle)
		if !inside {
			continue
		}
		if distSq < bestDistSq {
			found = true
			bestDistSq = distSq
			best = target
		}
	}

	if !found {
		return nil, false
	}
	return []info.Position{best}, true
}

func nearestRectExit(myPos info.Position, obstacle Obstacle) (info.Position, float64, bool) {
	minX, maxX, minY, maxY := expandedRect(obstacle)
	if !pointInRect(myPos, minX, maxX, minY, maxY) {
		return info.Position{}, 0, false
	}

	target := myPos
	bestDist := myPos.X - minX
	target.X = minX - noGoZoneEscapeClearance

	if dist := maxX - myPos.X; dist < bestDist {
		bestDist = dist
		target = myPos
		target.X = maxX + noGoZoneEscapeClearance
	}
	if dist := myPos.Y - minY; dist < bestDist {
		bestDist = dist
		target = myPos
		target.Y = minY - noGoZoneEscapeClearance
	}
	if dist := maxY - myPos.Y; dist < bestDist {
		bestDist = dist
		target = myPos
		target.Y = maxY + noGoZoneEscapeClearance
	}

	return target, bestDist * bestDist, true
}

func makeEscapePath(myPos info.Position, nearestObstacle Obstacle) []info.Position {
	dx := myPos.X - nearestObstacle.Position.X
	dy := myPos.Y - nearestObstacle.Position.Y
	distSq := dx*dx + dy*dy
	safeDistance := nearestObstacle.Size + MotionRadius
	if distSq > 0 {
		dist := math.Sqrt(distSq)
		dx = dx / dist * safeDistance
		dy = dy / dist * safeDistance
	} else {
		angle := rand.Float64() * 2 * math.Pi
		dx = math.Cos(angle) * safeDistance
		dy = math.Sin(angle) * safeDistance
	}
	escapePos := info.Position{
		X:     nearestObstacle.Position.X + dx,
		Y:     nearestObstacle.Position.Y + dy,
		Angle: myPos.Angle,
	}
	return []info.Position{escapePos}
}

// DistanceBetween is the Euclidean distance in the XY plane.
func DistanceBetween(pos1, pos2 info.Position) float64 {
	return distanceBetween(pos1, pos2)
}

func distanceBetween(pos1, pos2 info.Position) float64 {
	return math.Sqrt(distanceSquared(pos1, pos2))
}

func distanceSquared(pos1, pos2 info.Position) float64 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	return dx*dx + dy*dy
}

func runRRT(
	nodes []*RRTNode,
	obstacles []Obstacle,
	finalDestination info.Position,
	cfg RRTConfig,
) *RRTNode {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	startNode := nodes[0]
	if goalNode := connectToGoalIfClear(startNode, finalDestination, obstacles); goalNode != nil {
		return goalNode
	}

	for i := 0; i < cfg.MaxIterations; i++ {
		var randomPoint info.Position
		sampledGoal := rng.Float64() < cfg.GoalBias
		if sampledGoal {
			randomPoint = finalDestination
		} else {
			randomPoint = info.Position{
				X:     rng.Float64()*cfg.FieldWidth - cfg.FieldWidth/2,
				Y:     rng.Float64()*cfg.FieldHeight - cfg.FieldHeight/2,
				Angle: 0,
			}
		}

		nearestNode := findNearestNode(nodes, randomPoint)
		newNode := extendTree(nearestNode, randomPoint, cfg.StepSize)

		if !isNodeValid(newNode.position, obstacles, false) {
			continue
		}
		if !IsPathClear(nearestNode.position, newNode.position, obstacles, PlanningRadius) {
			continue
		}

		newNode.parent = nearestNode
		nodes = append(nodes, newNode)

		if sampledGoal {
			if goalNode := connectToGoalIfClear(newNode, finalDestination, obstacles); goalNode != nil {
				return goalNode
			}
		}

		if distanceSquared(newNode.position, finalDestination) < cfg.CompletionDistance*cfg.CompletionDistance {
			if goalNode := connectToGoalIfClear(newNode, finalDestination, obstacles); goalNode != nil {
				return goalNode
			}
		}
	}

	closestNode := findNearestNode(nodes, finalDestination)
	if distanceSquared(closestNode.position, finalDestination) > 500.0*500.0 {
		return nil
	}
	return closestNode
}

func connectToGoalIfClear(from *RRTNode, finalDestination info.Position, obstacles []Obstacle) *RRTNode {
	if from == nil {
		return nil
	}
	if !isNodeValid(finalDestination, obstacles, false) {
		return nil
	}
	if !IsPathClear(from.position, finalDestination, obstacles, PlanningRadius) {
		return nil
	}
	return &RRTNode{
		position: finalDestination,
		parent:   from,
	}
}

func shortcutPath(path []info.Position, obstacles []Obstacle, extraMargin float64) []info.Position {
	if len(path) <= 2 {
		return path
	}

	out := make([]info.Position, 0, len(path))
	out = append(out, path[0])

	for i := 0; i < len(path)-1; {
		next := i + 1
		for j := len(path) - 1; j > i+1; j-- {
			if IsPathClear(path[i], path[j], obstacles, extraMargin) {
				next = j
				break
			}
		}
		out = append(out, path[next])
		i = next
	}

	return out
}

func findNearestNode(nodes []*RRTNode, target info.Position) *RRTNode {
	minDistSq := math.MaxFloat64
	var nearest *RRTNode
	for _, node := range nodes {
		distSq := distanceSquared(node.position, target)
		if distSq < minDistSq {
			minDistSq = distSq
			nearest = node
		}
	}
	return nearest
}

func extendTree(nearest *RRTNode, random info.Position, stepSize float64) *RRTNode {
	dx := random.X - nearest.position.X
	dy := random.Y - nearest.position.Y
	distSq := dx*dx + dy*dy

	if distSq <= stepSize*stepSize {
		return &RRTNode{position: random, parent: nil}
	}
	dist := math.Sqrt(distSq)
	dx = dx / dist * stepSize
	dy = dy / dist * stepSize
	newPos := info.Position{
		X:     nearest.position.X + dx,
		Y:     nearest.position.Y + dy,
		Angle: nearest.position.Angle,
	}
	return &RRTNode{position: newPos, parent: nil}
}

func isNodeValid(position info.Position, obstacles []Obstacle, isStartPosition bool) bool {
	if isStartPosition {
		return true
	}
	for _, obstacle := range obstacles {
		if pointInsideObstacle(position, obstacle) {
			return false
		}
	}
	return true
}

// IsPathClear checks if the path between two positions is clear of obstacles.
func IsPathClear(start, end info.Position, obstacles []Obstacle, extraMargin float64) bool {
	for _, obstacle := range obstacles {
		if segmentIntersectsObstacle(start, end, obstacle, extraMargin) {
			return false
		}
	}
	return true
}

func segmentIntersectsObstacle(start, end info.Position, obstacle Obstacle, extraMargin float64) bool {
	if obstacle.rect {
		return segmentIntersectsRectObstacle(start, end, obstacle, extraMargin)
	}
	return segmentIntersectsCircleObstacle(start, end, obstacle, extraMargin)
}

func segmentIntersectsCircleObstacle(start, end info.Position, obstacle Obstacle, extraMargin float64) bool {
	_ = extraMargin
	radius := obstacle.Size

	abX := end.X - start.X
	abY := end.Y - start.Y
	acX := obstacle.Position.X - start.X
	acY := obstacle.Position.Y - start.Y
	abLenSq := abX*abX + abY*abY
	radiusSq := radius * radius

	if abLenSq == 0 {
		return distanceSquared(start, obstacle.Position) <= radiusSq
	}

	t := (acX*abX + acY*abY) / abLenSq
	if t <= 0 {
		return false
	}
	if t > 1 {
		t = 1
	}

	closestX := start.X + t*abX
	closestY := start.Y + t*abY
	dx := obstacle.Position.X - closestX
	dy := obstacle.Position.Y - closestY
	return dx*dx+dy*dy <= radiusSq
}

func segmentIntersectsRectObstacle(start, end info.Position, obstacle Obstacle, extraMargin float64) bool {
	_ = extraMargin
	minX, maxX, minY, maxY := expandedRect(obstacle)
	startInside := pointInRect(start, minX, maxX, minY, maxY)
	endInside := pointInRect(end, minX, maxX, minY, maxY)
	if startInside {
		return endInside
	}
	if endInside {
		return true
	}
	return segmentIntersectsRect(start, end, minX, maxX, minY, maxY)
}

func pointInsideObstacle(position info.Position, obstacle Obstacle) bool {
	if obstacle.rect {
		minX, maxX, minY, maxY := expandedRect(obstacle)
		return pointInRect(position, minX, maxX, minY, maxY)
	}
	return distanceSquared(position, obstacle.Position) <= obstacle.Size*obstacle.Size
}

func expandedRect(obstacle Obstacle) (float64, float64, float64, float64) {
	return obstacle.minX - obstacle.Size,
		obstacle.maxX + obstacle.Size,
		obstacle.minY - obstacle.Size,
		obstacle.maxY + obstacle.Size
}

func pointInRect(position info.Position, minX, maxX, minY, maxY float64) bool {
	return position.X >= minX && position.X <= maxX &&
		position.Y >= minY && position.Y <= maxY
}

func segmentIntersectsRect(start, end info.Position, minX, maxX, minY, maxY float64) bool {
	dx := end.X - start.X
	dy := end.Y - start.Y
	t0 := 0.0
	t1 := 1.0

	if !clipSegment(-dx, start.X-minX, &t0, &t1) {
		return false
	}
	if !clipSegment(dx, maxX-start.X, &t0, &t1) {
		return false
	}
	if !clipSegment(-dy, start.Y-minY, &t0, &t1) {
		return false
	}
	if !clipSegment(dy, maxY-start.Y, &t0, &t1) {
		return false
	}
	return t1 >= 0 && t0 <= 1
}

func clipSegment(p, q float64, t0, t1 *float64) bool {
	if p == 0 {
		return q >= 0
	}

	r := q / p
	if p < 0 {
		if r > *t1 {
			return false
		}
		if r > *t0 {
			*t0 = r
		}
		return true
	}

	if r < *t0 {
		return false
	}
	if r < *t1 {
		*t1 = r
	}
	return true
}

// ObstaclesForRobot lists disc obstacles (other robots, optional ball) for path checks.
func ObstaclesForRobot(team info.Team, id info.ID, avoidBall bool, avoidGoallines bool, gi *info.GameInfo) []Obstacle {
	obstacles := make([]Obstacle, 0, 1+int(info.TEAM_SIZE)*2+2)

	if ballRadius, ok := ballObstacleRadius(team, avoidBall, gi); ok {
		ballPos, _ := gi.State.Ball.GetPosition()
		obstacles = append(obstacles, Obstacle{Position: ballPos, Size: ballRadius})
	}

	obstacles = appendRobotObstacles(obstacles, gi.State.GetTeam(info.Blue), team, id)
	obstacles = appendRobotObstacles(obstacles, gi.State.GetTeam(info.Yellow), team, id)
	if avoidGoallines {
		obstacles = append(obstacles, goalLineObstacles(gi)...)
	}
	return obstacles
}

func ballObstacleRadius(team info.Team, avoidBall bool, gi *info.GameInfo) (float64, bool) {
	if RestartBallKeepoutActive(team, gi) {
		return RestartBallKeepoutRadius, true
	}
	if avoidBall {
		return BallSafetyRadius, true
	}
	return 0, false
}

func appendRobotObstacles(obstacles []Obstacle, robots *info.RobotTeam, ownTeam info.Team, ownID info.ID) []Obstacle {
	for _, robot := range robots {
		if robot == nil {
			continue
		}
		if robot.GetTeam() == ownTeam && robot.GetID() == ownID {
			continue
		}
		pos, rototTime, err := robot.GetPositionTime()
		if err != nil {
			continue
		}
		if time.Now().UnixMilli()-rototTime > 200 {
			continue
		}
		obstacles = append(obstacles, Obstacle{Position: pos, Size: RobotSafetyRadius})
	}
	return obstacles
}

func goalLineObstacles(gi *info.GameInfo) []Obstacle {
	if gi == nil || !gi.HasField() {
		return nil
	}

	zones := make([]Obstacle, 0, 2)
	if zone, ok := goalZoneObstacle(gi, "LeftPenaltyStretch", "LeftGoalLine"); ok {
		zones = append(zones, zone)
	}
	if zone, ok := goalZoneObstacle(gi, "RightPenaltyStretch", "RightGoalLine"); ok {
		zones = append(zones, zone)
	}
	return zones
}

func goalZoneObstacle(gi *info.GameInfo, frontLineName, backLineName string) (Obstacle, bool) {
	front := gi.GetFieldLine(frontLineName)
	back := gi.GetFieldLine(backLineName)
	if front == nil || back == nil || front.GetP1() == nil || front.GetP2() == nil || back.GetP1() == nil {
		return Obstacle{}, false
	}

	frontX := float64(front.GetP1().GetX())
	backX := float64(back.GetP1().GetX())
	y1 := float64(front.GetP1().GetY())
	y2 := float64(front.GetP2().GetY())

	return rectObstacle(
		math.Min(frontX, backX),
		math.Max(frontX, backX),
		math.Min(y1, y2),
		math.Max(y1, y2),
		GoalLineSafetyRadius,
	), true
}

func rectObstacle(minX, maxX, minY, maxY, size float64) Obstacle {
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minY > maxY {
		minY, maxY = maxY, minY
	}

	return Obstacle{
		Size: size,
		rect: true,
		minX: minX,
		maxX: maxX,
		minY: minY,
		maxY: maxY,
	}
}
