// Package pathplanner holds RRT path planning and obstacle helpers used by activities.
package pathplanner

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Field/planning constants (mm) — same semantics as the former move_to_position locals.
const (
	RobotSafetyRadius = 240.0
	BallSafetyRadius  = 150.0
	PlanningRadius    = 400.0
	MotionRadius      = 100.0
)

// Defaults for persistent replan gating (used when the corresponding RRTConfig field is zero).
const (
	defaultMaxPlanAge        = 2 * time.Second
	defaultPathDeviationMax  = 100.0 // mm, perpendicular to stored polyline (replan, do not just trim)
	defaultLocalCheckRadius  = 2000.0
	defaultObstacleMoveDelta = 50.0 // mm, any key in snapshot
	defaultGoalChangeEpsilon = 10.0 // mm, goal “same” if closer than this
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
	// Persistence / replan (zero = use package defaults for that field)
	MaxPlanAge        time.Duration
	PathDeviationMax  float64
	LocalCheckRadius  float64
	ObstacleMoveDelta float64
	GoalChangeEpsilon float64
}

type resolvedPersistence struct {
	maxPlanAge        time.Duration
	pathDeviationMax  float64
	localR            float64
	obstacleMoveDelta float64
	goalChangeEpsilon float64
}

func (c RRTConfig) persistence() resolvedPersistence {
	r := resolvedPersistence{
		maxPlanAge:        c.MaxPlanAge,
		pathDeviationMax:  c.PathDeviationMax,
		localR:            c.LocalCheckRadius,
		obstacleMoveDelta: c.ObstacleMoveDelta,
		goalChangeEpsilon: c.GoalChangeEpsilon,
	}
	if r.maxPlanAge == 0 {
		r.maxPlanAge = defaultMaxPlanAge
	}
	if r.pathDeviationMax == 0 {
		r.pathDeviationMax = defaultPathDeviationMax
	}
	if r.localR == 0 {
		r.localR = defaultLocalCheckRadius
	}
	if r.obstacleMoveDelta == 0 {
		r.obstacleMoveDelta = defaultObstacleMoveDelta
	}
	if r.goalChangeEpsilon == 0 {
		r.goalChangeEpsilon = defaultGoalChangeEpsilon
	}
	return r
}

// RRTNode represents a node in the RRT tree.
type RRTNode struct {
	position info.Position
	parent   *RRTNode
	cost     float64
}

// Obstacle is a disc obstacle in the plane (e.g. another robot or the ball).
type Obstacle struct {
	Position info.Position
	Size     float64
}

// robotPathState is per-robot cached plan + metadata for persistence.
type robotPathState struct {
	path          []info.Position
	plannedAt     time.Time
	goal          info.Position
	planStart     info.Position
	localSnapshot map[string]info.Position
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

// GetPath returns a copy of the last planned path for id, or nil if none.
func (p *Planner) GetPath(id info.ID) []info.Position {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	st := p.session[id]
	if st == nil || len(st.path) == 0 {
		return nil
	}
	return append([]info.Position(nil), st.path...)
}

// PlanPath runs RRT or returns a cached path when replanning is not warranted.
func (p *Planner) PlanPath(
	team info.Team,
	id info.ID,
	finalDestination info.Position,
	avoidBall bool,
	gi *info.GameInfo,
	cfg RRTConfig,
) []info.Position {
	if p == nil {
		return nil
	}
	perm := cfg.persistence()
	myPos, _ := gi.State.GetTeam(team)[id].GetPosition()
	obstacles := ObstaclesForRobot(team, id, avoidBall, gi)

	// Inflated “collision”: escape this tick only. Do not commit: overwriting the
	// session with a 1-point path is common near the goal in traffic and made the
	// full plan appear to “vanish” until MaxPlanAge forced a replan.
	robotsNearby, nearest := collisionTarget(myPos, obstacles)
	if robotsNearby {
		return makeEscapePath(myPos, nearest)
	}

	p.mu.Lock()
	st := p.session[id]
	p.mu.Unlock()

	if st != nil && len(st.path) > 0 {
		// No prefix trimming: keep the full polyline. move_to / look-ahead uses the path
		// from the current pose; dropping vertices behind the robot caused flicker and
		// shortcutting against replan gating.
		if !shouldReplan(st, myPos, finalDestination, gi, team, id, avoidBall, perm) {
			return p.copyPath(id)
		}
	}

	// Full RRT
	startNode := &RRTNode{position: myPos, parent: nil, cost: 0}
	nodes := []*RRTNode{startNode}
	goalNode := runRRT(nodes, obstacles, finalDestination, cfg)
	if goalNode == nil {
		p.mu.Lock()
		prev := p.session[id]
		if prev == nil || len(prev.path) == 0 {
			p.session[id] = newCommittedState(
				[]info.Position{finalDestination},
				myPos, finalDestination, team, id, avoidBall, gi, perm,
			)
		}
		p.mu.Unlock()
		return p.copyPath(id)
	}
	path := []info.Position{}
	for cur := goalNode; cur != nil; cur = cur.parent {
		path = append([]info.Position{cur.position}, path...)
	}
	p.commit(id, path, myPos, finalDestination, team, id, avoidBall, gi, perm)
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
	robotPos, goal info.Position,
	team info.Team,
	selfID info.ID,
	avoidBall bool,
	gi *info.GameInfo,
	perm resolvedPersistence,
) *robotPathState {
	s := &robotPathState{
		path:          append([]info.Position(nil), path...),
		plannedAt:     time.Now(),
		goal:          goal,
		planStart:     robotPos,
		localSnapshot: makeLocalSnapshot(robotPos, team, selfID, avoidBall, gi, perm.localR),
	}
	return s
}

func (p *Planner) commit(
	id info.ID,
	path []info.Position,
	robotPos, goal info.Position,
	team info.Team,
	selfID info.ID,
	avoidBall bool,
	gi *info.GameInfo,
	perm resolvedPersistence,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.session[id] = newCommittedState(path, robotPos, goal, team, selfID, avoidBall, gi, perm)
}

func makeLocalSnapshot(
	robotPos info.Position,
	team info.Team,
	selfID info.ID,
	avoidBall bool,
	gi *info.GameInfo,
	radius float64,
) map[string]info.Position {
	m := make(map[string]info.Position)
	if avoidBall {
		ballPos, _ := gi.State.Ball.GetPosition()
		if distanceBetween(robotPos, ballPos) < radius {
			m[ballKey] = ballPos
		}
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
			if distanceBetween(robotPos, pos) < radius {
				m[robotKey(t, oid)] = pos
			}
		}
	}
	return m
}

const ballKey = "ball"

func robotKey(oteam info.Team, oid info.ID) string {
	return fmt.Sprintf("%d:%d", oteam, oid)
}

func shouldReplan(
	st *robotPathState,
	myPos, goal info.Position,
	gi *info.GameInfo,
	team info.Team,
	selfID info.ID,
	avoidBall bool,
	perm resolvedPersistence,
) bool {
	if st == nil || len(st.path) == 0 {
		return true
	}
	if distanceBetween(st.goal, goal) > perm.goalChangeEpsilon {
		return true
	}
	if time.Since(st.plannedAt) > perm.maxPlanAge {
		return true
	}
	if pathDeviation(myPos, st.path) > perm.pathDeviationMax {
		return true
	}
	if localNeighborhoodChanged(myPos, st.localSnapshot, team, selfID, avoidBall, perm, gi) {
		return true
	}
	return false
}

// pathDeviation is min distance to the polyline in XY.
func pathDeviation(robot info.Position, path []info.Position) float64 {
	if len(path) == 0 {
		return 0
	}
	if len(path) == 1 {
		return distanceBetween(robot, path[0])
	}
	best := math.MaxFloat64
	for i := 0; i < len(path)-1; i++ {
		a := info.Vec2{X: path[i].X, Y: path[i].Y}
		b := info.Vec2{X: path[i+1].X, Y: path[i+1].Y}
		p := info.Vec2{X: robot.X, Y: robot.Y}
		d := info.DistToLineSegment(a, b, p)
		if d < best {
			best = d
		}
	}
	return best
}

// localNeighborhoodChanged returns true if a new disc entered R around the robot, an in–R
// agent moved more than the threshold, or any agent that was in the last snapshot has moved
// by more than the threshold in world space (prefer to replan a bit too often).
func localNeighborhoodChanged(
	myPos info.Position,
	oldSnap map[string]info.Position,
	team info.Team,
	selfID info.ID,
	avoidBall bool,
	perm resolvedPersistence,
	gi *info.GameInfo,
) bool {
	R := perm.localR
	delta := perm.obstacleMoveDelta
	// (a) In R of me now: new key, or moved vs snapshot for that key
	if avoidBall {
		ballPos, _ := gi.State.Ball.GetPosition()
		if distanceBetween(myPos, ballPos) < R {
			old, ok := oldSnap[ballKey]
			if !ok {
				return true
			}
			if distanceBetween(old, ballPos) > delta {
				return true
			}
		}
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
			k := robotKey(t, oid)
			if distanceBetween(myPos, pos) < R {
				old, ok := oldSnap[k]
				if !ok {
					return true
				}
				if distanceBetween(old, pos) > delta {
					return true
				}
			}
		}
	}
	// (b) Anything we tracked in the last snapshot: large displacement
	if oldSnap == nil {
		return false
	}
	for k, oldP := range oldSnap {
		newP, ok := currentPositionForKey(k, team, selfID, avoidBall, gi)
		if !ok {
			continue
		}
		if distanceBetween(oldP, newP) > delta {
			return true
		}
	}
	return false
}

func currentPositionForKey(
	k string,
	team info.Team,
	selfID info.ID,
	avoidBall bool,
	gi *info.GameInfo,
) (info.Position, bool) {
	if k == ballKey {
		if !avoidBall {
			return info.Position{}, false
		}
		ballPos, _ := gi.State.Ball.GetPosition()
		return ballPos, true
	}
	parts := strings.Split(k, ":")
	if len(parts) != 2 {
		return info.Position{}, false
	}
	tt, err1 := strconv.ParseInt(parts[0], 10, 8)
	oid, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return info.Position{}, false
	}
	ot := info.Team(tt)
	oid2 := info.ID(oid)
	if ot == team && oid2 == selfID {
		return info.Position{}, false
	}
	robot := gi.State.GetTeam(ot)[oid2]
	pos, rototTime, err := robot.GetPositionTime()
	if err != nil {
		return info.Position{}, false
	}
	if time.Now().UnixMilli()-rototTime > 200 {
		return info.Position{}, false
	}
	return pos, true
}

func collisionTarget(myPos info.Position, obstacles []Obstacle) (bool, Obstacle) {
	robotsNearby := false
	var nearestObstacle Obstacle
	shortestDist := math.MaxFloat64
	for _, obstacle := range obstacles {
		dist := distanceBetween(myPos, obstacle.Position)
		if dist <= obstacle.Size {
			robotsNearby = true
			if dist < shortestDist {
				shortestDist = dist
				nearestObstacle = obstacle
			}
		}
	}
	return robotsNearby, nearestObstacle
}

func makeEscapePath(myPos info.Position, nearestObstacle Obstacle) []info.Position {
	dx := myPos.X - nearestObstacle.Position.X
	dy := myPos.Y - nearestObstacle.Position.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	safeDistance := nearestObstacle.Size + MotionRadius
	if dist > 0 {
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
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func runRRT(
	nodes []*RRTNode,
	obstacles []Obstacle,
	finalDestination info.Position,
	cfg RRTConfig,
) *RRTNode {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < cfg.MaxIterations; i++ {
		var randomPoint info.Position
		if rand.Float64() < cfg.GoalBias {
			randomPoint = finalDestination
		} else {
			randomPoint = info.Position{
				X:     rand.Float64()*cfg.FieldWidth - cfg.FieldWidth/2,
				Y:     rand.Float64()*cfg.FieldHeight - cfg.FieldHeight/2,
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
		newNode.cost = nearestNode.cost + distanceBetween(nearestNode.position, newNode.position)
		nodes = append(nodes, newNode)

		if distanceBetween(newNode.position, finalDestination) < cfg.CompletionDistance {
			goalNode := &RRTNode{
				position: finalDestination,
				parent:   newNode,
				cost:     newNode.cost + distanceBetween(newNode.position, finalDestination),
			}
			if IsPathClear(newNode.position, goalNode.position, obstacles, PlanningRadius) {
				return goalNode
			}
		}
	}

	closestNode := findNearestNode(nodes, finalDestination)
	if distanceBetween(closestNode.position, finalDestination) > 500.0 {
		return nil
	}
	return closestNode
}

func findNearestNode(nodes []*RRTNode, target info.Position) *RRTNode {
	minDist := math.MaxFloat64
	var nearest *RRTNode
	for _, node := range nodes {
		dist := distanceBetween(node.position, target)
		if dist < minDist {
			minDist = dist
			nearest = node
		}
	}
	return nearest
}

func extendTree(nearest *RRTNode, random info.Position, stepSize float64) *RRTNode {
	dx := random.X - nearest.position.X
	dy := random.Y - nearest.position.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist <= stepSize {
		return &RRTNode{position: random, parent: nil, cost: 0}
	}
	dx = dx / dist * stepSize
	dy = dy / dist * stepSize
	newPos := info.Position{
		X:     nearest.position.X + dx,
		Y:     nearest.position.Y + dy,
		Angle: nearest.position.Angle,
	}
	return &RRTNode{position: newPos, parent: nil, cost: 0}
}

func isNodeValid(position info.Position, obstacles []Obstacle, isStartPosition bool) bool {
	if isStartPosition {
		return true
	}
	for _, obstacle := range obstacles {
		if distanceBetween(position, obstacle.Position) <= obstacle.Size {
			return false
		}
	}
	return true
}

// IsPathClear checks if the path between two positions is clear of obstacles.
func IsPathClear(start, end info.Position, obstacles []Obstacle, extraMargin float64) bool {
	_ = extraMargin
	const numChecks = 10
	for i := 0; i <= numChecks; i++ {
		t := float64(i) / float64(numChecks)
		checkPos := info.Position{
			X:     start.X + t*(end.X-start.X),
			Y:     start.Y + t*(end.Y-start.Y),
			Angle: start.Angle,
		}
		if i == 0 {
			continue
		}
		if !isNodeValid(checkPos, obstacles, false) {
			return false
		}
	}
	return true
}

// ObstaclesForRobot lists disc obstacles (other robots, optional ball) for path checks.
func ObstaclesForRobot(team info.Team, id info.ID, avoidBall bool, gi *info.GameInfo) []Obstacle {
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
	return obstacles
}
