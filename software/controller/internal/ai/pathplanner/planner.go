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
	RobotSafetyRadius    = 240.0
	BallSafetyRadius     = 150.0
	GoalLineSafetyRadius = 100.0
	PlanningRadius       = 400.0
	MotionRadius         = 100.0
)

const goalLineObstacleSpacing = GoalLineSafetyRadius

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

// Obstacle is a disc obstacle in the plane (e.g. another robot or the ball).
type Obstacle struct {
	Position info.Position
	Size     float64
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
	obstacles := planningObstacles(team, id, myPos, finalDestination, avoidBall, avoidGoallines, gi, perm)

	// Inflated “collision”: escape this tick only. Do not commit: overwriting the
	// session with a 1-point path is common near the goal in traffic and made the
	// full plan appear to “vanish” until age-based replanning created a new path.
	robotsNearby, nearest := collisionTarget(myPos, obstacles)
	if robotsNearby {
		return makeEscapePath(myPos, nearest)
	}

	p.mu.Lock()
	st := p.session[id]
	p.mu.Unlock()

	if st != nil && len(st.path) > 0 {
		if !shouldCreateNewPath(st, myPos, finalDestination, obstacles, gi, team, id, perm) {
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
	perm resolvedPersistence,
) bool {
	if st == nil || len(st.path) == 0 {
		return true
	}
	if !isStoredPathValid(st, myPos, goal, obstacles, perm.goalMatchEpsilon) {
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
	goalMatchEpsilon float64,
) bool {
	if st == nil || len(st.path) == 0 {
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
	if ignoreRobotObstaclesForBallApproach(myPos, finalDestination, avoidBall, gi, perm) {
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

func collisionTarget(myPos info.Position, obstacles []Obstacle) (bool, Obstacle) {
	robotsNearby := false
	var nearestObstacle Obstacle
	shortestDistSq := math.MaxFloat64
	for _, obstacle := range obstacles {
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
	if isNodeValid(finalDestination, obstacles, false) &&
		IsPathClear(startNode.position, finalDestination, obstacles, PlanningRadius) {
		return &RRTNode{
			position: finalDestination,
			parent:   startNode,
		}
	}

	for i := 0; i < cfg.MaxIterations; i++ {
		var randomPoint info.Position
		if rng.Float64() < cfg.GoalBias {
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

		if distanceSquared(newNode.position, finalDestination) < cfg.CompletionDistance*cfg.CompletionDistance {
			goalNode := &RRTNode{
				position: finalDestination,
				parent:   newNode,
			}
			if IsPathClear(newNode.position, goalNode.position, obstacles, PlanningRadius) {
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
		if distanceSquared(position, obstacle.Position) <= obstacle.Size*obstacle.Size {
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

// ObstaclesForRobot lists disc obstacles (other robots, optional ball) for path checks.
func ObstaclesForRobot(team info.Team, id info.ID, avoidBall bool, avoidGoallines bool, gi *info.GameInfo) []Obstacle {
	obstacles := make([]Obstacle, 0)
	allRobots := append(gi.State.GetTeam(info.Blue)[:], gi.State.GetTeam(info.Yellow)[:]...)

	if avoidBall {
		ballPos, _ := gi.State.Ball.GetPosition()
		obstacles = append(obstacles, Obstacle{Position: ballPos, Size: BallSafetyRadius})
	}

	for _, robot := range allRobots {
		if robot.GetTeam() == team && robot.GetID() == id {
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
	if avoidGoallines {
		obstacles = append(obstacles, goalLineObstacles(gi)...)
	}
	return obstacles
}

func goalLineObstacles(gi *info.GameInfo) []Obstacle {
	if gi == nil || !gi.HasField() {
		return nil
	}

	obstacles := make([]Obstacle, 0)
	obstacles = append(obstacles, goalZoneObstacles(gi, "LeftPenaltyStretch", "LeftGoalLine")...)
	obstacles = append(obstacles, goalZoneObstacles(gi, "RightPenaltyStretch", "RightGoalLine")...)
	return obstacles
}

func goalZoneObstacles(gi *info.GameInfo, frontLineName, backLineName string) []Obstacle {
	front := gi.GetFieldLine(frontLineName)
	back := gi.GetFieldLine(backLineName)
	if front == nil || back == nil || front.GetP1() == nil || front.GetP2() == nil || back.GetP1() == nil {
		return nil
	}

	frontX := float64(front.GetP1().GetX())
	backX := float64(back.GetP1().GetX())
	y1 := float64(front.GetP1().GetY())
	y2 := float64(front.GetP2().GetY())

	return obstaclesInRectangle(
		math.Min(frontX, backX),
		math.Max(frontX, backX),
		math.Min(y1, y2),
		math.Max(y1, y2),
		GoalLineSafetyRadius,
	)
}

func obstaclesInRectangle(minX, maxX, minY, maxY, size float64) []Obstacle {
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minY > maxY {
		minY, maxY = maxY, minY
	}

	xSegments := int(math.Ceil((maxX - minX) / goalLineObstacleSpacing))
	ySegments := int(math.Ceil((maxY - minY) / goalLineObstacleSpacing))
	if xSegments < 1 {
		xSegments = 1
	}
	if ySegments < 1 {
		ySegments = 1
	}

	obstacles := make([]Obstacle, 0, (xSegments+1)*(ySegments+1))
	for ix := 0; ix <= xSegments; ix++ {
		x := minX + float64(ix)*(maxX-minX)/float64(xSegments)
		for iy := 0; iy <= ySegments; iy++ {
			y := minY + float64(iy)*(maxY-minY)/float64(ySegments)
			obstacles = append(obstacles, Obstacle{
				Position: info.Position{X: x, Y: y},
				Size:     size,
			})
		}
	}

	return obstacles
}
