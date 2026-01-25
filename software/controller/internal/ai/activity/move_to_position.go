package ai

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// RobotSafetyRadius defines the no-movement zone around each robot
const RobotSafetyRadius = 240.0 // mm - increased for better safety margin
const BallSafetyRadius = 150.0  // mm - increased for better safety margin
const PlanningRadius = 400.0    // mm - increased for better safety margin
const MotionRadius = 100.0      // mm - increased for better safety margin

// MoveToPositionWithCollisionAvoidance handles collision avoidance using RRT
type MoveToPosition struct {
	team              info.Team
	id                info.ID
	final_destination info.Position    // The ultimate goal position
	path              []info.Position  // Current path from RRT planning
	rrtConfig         rrtConfiguration // Configuration for the RRT algorithm
	lastPlanningTime  time.Time        // Time when path was last planned
	planningInterval  time.Duration    // How often to replan the path
	previousObstacles []info.Position  // Previous obstacle positions for change detection
	significantChange bool             // Flag to indicate if obstacles have moved significantly
	gi                *info.GameInfo
	avoidBall         bool
	lastPosition      info.Position // Last position to detect lack of movement
	stuckThreshold    int           // Number of cycles to consider robot as stuck
}

// rrtConfiguration holds parameters for the RRT algorithm
type rrtConfiguration struct {
	maxIterations      int     // Maximum iterations for the RRT algorithm
	stepSize           float64 // How far to extend the tree in each step
	goalBias           float64 // Probability of sampling the goal directly
	waypointThreshold  float64 // Distance to consider a waypoint reached
	fieldWidth         float64 // Width of the field in mm
	fieldHeight        float64 // Height of the field in mm
	completionDistance float64 // Distance to consider goal reached
}

// RRTNode represents a node in the RRT tree
type RRTNode struct {
	position info.Position
	parent   *RRTNode
	cost     float64
}

// RRTNode represents a node in the RRT tree
type Obstacle struct {
	position info.Position
	size     float64
}

// NewMoveToPositionWithCollisionAvoidance creates a new instance
func NewMoveToPosition(team info.Team, id info.ID, dest info.Position) *MoveToPosition {
	// Initialize with reasonable RRT parameters
	rrtConfig := rrtConfiguration{
		maxIterations:      1000,
		stepSize:           50.0,   // mm per step (increased for more aggressive exploration)
		goalBias:           0.05,   // 20% chance of sampling the goal directly (increased for more direct paths)
		waypointThreshold:  600.0,  // mm to consider waypoint reached
		fieldWidth:         9000.0, // Standard SSL field width in mm
		fieldHeight:        6000.0, // Standard SSL field height in mm
		completionDistance: 50.0,   // mm to consider the goal reached
	}
	return &MoveToPosition{
		team:              team,
		id:                id,
		final_destination: dest,
		path:              []info.Position{},
		rrtConfig:         rrtConfig,
		lastPlanningTime:  time.Now(),
		planningInterval:  50 * time.Millisecond, // Replan more frequently (50ms instead of 100ms)
		previousObstacles: []info.Position{},
		significantChange: true, // Force initial planning
	}
}

func (m *MoveToPosition) AvoidBall(avoid bool) {
	m.avoidBall = avoid
}

// GetAction returns an action for the robot with RRT-based collision avoidance
func (m *MoveToPosition) GetAction(gi *info.GameInfo) action.Action {
	moveToAction := m.GetMoveToAction(gi)
	m.gi = gi
	return &moveToAction
}

func (m *MoveToPosition) GetMoveToAction(gi *info.GameInfo) action.MoveTo {
	myRobot := gi.State.GetTeam(m.team)[m.id]
	myPos, _ := myRobot.GetPosition()

	m.rrtConfig.stepSize = min(max(myPos.Dist2d(m.final_destination)/100, 1), m.rrtConfig.stepSize)

	m.AvoidBall(true)
	targetPos := myPos

	m.lastPosition = myPos

	m.PlanPath(gi, myPos)

	// fmt.Println(len(m.path))
	if len(m.path) > 0 {
		bestPoint := m.final_destination
		bestPointDist := 100000000.0
		if distanceBetween(m.path[len(m.path)-1], myPos) < m.rrtConfig.waypointThreshold &&
			m.IsPathClear(myPos, m.path[len(m.path)-1], m.GetObstaclePositions(gi), MotionRadius) ||
			len(m.path) == 1 {

			bestPoint = m.path[len(m.path)-1]
		} else {
			for i := 0; i < len(m.path)-1; i++ {
				a := info.Vec2{X: m.path[i].X, Y: m.path[i].Y}
				b := info.Vec2{X: m.path[i+1].X, Y: m.path[i+1].Y}
				p := info.Vec2{X: myPos.X, Y: myPos.Y}
				pointToLine := info.PointToLineSegment(a, b, p)
				distanceToThreshold := math.Abs(info.NormV2(info.Sub(pointToLine, p)) - m.rrtConfig.waypointThreshold)
				if distanceToThreshold < float64(bestPointDist) && m.IsPathClear(myPos, info.Position{X: pointToLine.X, Y: pointToLine.Y, Z: 0, Angle: 0}, m.GetObstaclePositions(gi), MotionRadius) {
					bestPointDist = distanceToThreshold

					bestPoint = info.Position{X: pointToLine.X, Y: pointToLine.Y, Z: myPos.Z, Angle: myPos.Angle}
				}
			}
		}
		targetPos = bestPoint
	}

	// targetPos := m.final_destination
	// Create move action to the current target
	act := action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = myPos
	act.Dest = targetPos
	act.Dest.Angle = targetPos.Angle
	act.Dribble = false
	return act
}

// PlanPath uses RRT to plan a collision-free path
func (m *MoveToPosition) PlanPath(gi *info.GameInfo, startPos info.Position) {
	// Create a list of obstacle positions (other robots)
	obstacles := m.GetObstaclePositions(gi)

	// Check if we're already in collision
	robotsNearby := false
	var nearestObstacle Obstacle
	shortestDist := math.MaxFloat64

	for _, obstacle := range obstacles {
		dist := distanceBetween(startPos, obstacle.position)
		if dist <= obstacle.size {
			robotsNearby = true
			if dist < shortestDist {
				shortestDist = dist
				nearestObstacle = obstacle
			}
		}
	}

	// If we're stuck in collision, generate a temporary escape path
	if robotsNearby {
		// Calculate direction away from nearest obstacle
		dx := startPos.X - nearestObstacle.position.X
		dy := startPos.Y - nearestObstacle.position.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		// Normalize and scale to get a point outside the safety radius
		safeDistance := nearestObstacle.size + MotionRadius // Add extra margin
		if dist > 0 {
			dx = dx / dist * safeDistance
			dy = dy / dist * safeDistance
		} else {
			// If exactly overlapping, move in random direction
			angle := rand.Float64() * 2 * math.Pi
			dx = math.Cos(angle) * safeDistance
			dy = math.Sin(angle) * safeDistance
		}

		// Create escape position
		escapePos := info.Position{
			X:     nearestObstacle.position.X + dx,
			Y:     nearestObstacle.position.Y + dy,
			Angle: startPos.Angle,
		}

		// Set this as our immediate path
		m.path = []info.Position{escapePos}
		return
	}

	// Initialize RRT
	startNode := &RRTNode{
		position: startPos,
		parent:   nil,
		cost:     0,
	}

	// Create tree with root at start position
	nodes := []*RRTNode{startNode}

	// Run RRT algorithm
	goalNode := m.RunRRT(nodes, obstacles)
	if goalNode == nil {
		// If no path found, keep the existing path or try to move directly to goal
		if len(m.path) == 0 {
			m.path = []info.Position{m.final_destination}
		}
		return
	}

	// Extract path from goal node back to start by following parents
	path := []info.Position{}
	current := goalNode
	for current != nil {
		path = append([]info.Position{current.position}, path...)
		current = current.parent
	}

	m.path = path
}

// RunRRT executes the RRT algorithm and returns the goal node if a path is found
func (m *MoveToPosition) RunRRT(nodes []*RRTNode, obstacles []Obstacle) *RRTNode {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < m.rrtConfig.maxIterations; i++ {
		// Sample a random point with goal bias
		var randomPoint info.Position
		if rand.Float64() < m.rrtConfig.goalBias {
			randomPoint = m.final_destination
		} else {
			randomPoint = info.Position{
				X:     rand.Float64()*m.rrtConfig.fieldWidth - m.rrtConfig.fieldWidth/2,
				Y:     rand.Float64()*m.rrtConfig.fieldHeight - m.rrtConfig.fieldHeight/2,
				Angle: 0, // Angle doesn't matter for path planning
			}
		}

		// Find nearest node in the tree
		nearestNode := m.FindNearestNode(nodes, randomPoint)

		// Extend tree toward random point
		newNode := m.ExtendTree(nearestNode, randomPoint, m.rrtConfig.stepSize)

		// Check if the new node would collide with any obstacle
		if !m.IsNodeValid(newNode.position, obstacles, false) {
			continue
		}

		// Check if the path to the new node is clear
		if !m.IsPathClear(nearestNode.position, newNode.position, obstacles, PlanningRadius) {
			continue
		}

		// Add the new node to the tree
		newNode.parent = nearestNode
		newNode.cost = nearestNode.cost + distanceBetween(nearestNode.position, newNode.position)
		nodes = append(nodes, newNode)

		// Check if we're close enough to the goal
		if distanceBetween(newNode.position, m.final_destination) < m.rrtConfig.completionDistance {
			// Create a final node at the exact goal position
			goalNode := &RRTNode{
				position: m.final_destination,
				parent:   newNode,
				cost:     newNode.cost + distanceBetween(newNode.position, m.final_destination),
			}

			// Check if the path to the goal is clear
			if m.IsPathClear(newNode.position, goalNode.position, obstacles, PlanningRadius) {
				return goalNode
			}
		}
	}

	// If we reach max iterations without finding a path, connect to the node closest to the goal
	closestNode := m.FindNearestNode(nodes, m.final_destination)

	// If the closest node is too far from the goal, return nil
	if distanceBetween(closestNode.position, m.final_destination) > 500.0 {
		return nil
	}

	return closestNode
}

// FindNearestNode finds the node in the tree closest to the target position
func (m *MoveToPosition) FindNearestNode(nodes []*RRTNode, target info.Position) *RRTNode {
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

// ExtendTree extends the tree from the nearest node toward the random point
func (m *MoveToPosition) ExtendTree(nearest *RRTNode, random info.Position, stepSize float64) *RRTNode {
	// Calculate direction from nearest to random
	dx := random.X - nearest.position.X
	dy := random.Y - nearest.position.Y

	// Calculate distance
	dist := math.Sqrt(dx*dx + dy*dy)

	// If distance is less than step size, just return the random point
	if dist <= stepSize {
		return &RRTNode{
			position: random,
			parent:   nil,
			cost:     0,
		}
	}

	// Otherwise, scale the direction to step size
	dx = dx / dist * stepSize
	dy = dy / dist * stepSize

	// Create new position
	newPos := info.Position{
		X:     nearest.position.X + dx,
		Y:     nearest.position.Y + dy,
		Angle: nearest.position.Angle, // Maintain the same angle
	}

	return &RRTNode{
		position: newPos,
		parent:   nil,
		cost:     0,
	}
}

// IsNodeValid checks if a node is valid (not too close to obstacles)
// Added isStartPosition parameter to allow the starting position even if it's near obstacles
func (m *MoveToPosition) IsNodeValid(position info.Position, obstacles []Obstacle, isStartPosition bool) bool {
	// Skip obstacle check for the starting position if specified
	if isStartPosition {
		return true
	}

	for _, obstacle := range obstacles {
		if distanceBetween(position, obstacle.position) <= obstacle.size {
			return false
		}
	}
	return true
}

// IsPathClear checks if the path between two positions is clear of obstacles
func (m *MoveToPosition) IsPathClear(start, end info.Position, obstacles []Obstacle, extraMargin float64) bool {
	// Check several points along the path
	const numChecks = 10

	for i := 0; i <= numChecks; i++ {
		t := float64(i) / float64(numChecks)
		checkPos := info.Position{
			X:     start.X + t*(end.X-start.X),
			Y:     start.Y + t*(end.Y-start.Y),
			Angle: start.Angle, // Angle doesn't matter here
		}

		// Skip the first point (which is the start position)
		if i == 0 {
			continue
		}

		if !m.IsNodeValid(checkPos, obstacles, false) {
			return false
		}
	}

	return true
}

// GetObstaclePositions gets positions of all other robots on the field
func (m *MoveToPosition) GetObstaclePositions(gi *info.GameInfo) []Obstacle {
	obstacles := []Obstacle{}

	// Get all robots
	allRobots := append(gi.State.GetTeam(info.Blue)[:], gi.State.GetTeam(info.Yellow)[:]...)

	if m.avoidBall {
		ballPos, _ := gi.State.Ball.GetPosition()
		ballObstacle := Obstacle{ballPos, BallSafetyRadius}
		obstacles = append(obstacles, ballObstacle)
	}

	for _, robot := range allRobots {
		// Skip self
		if robot.GetTeam() == m.team && robot.GetID() == m.id {
			continue
		}

		pos, rototTime, err := robot.GetPositionTime()
		if err != nil {
			continue
		}
		if time.Now().UnixMilli()-rototTime > 200 {
			continue
		}

		// Skip robots at exactly (0,0) as they are likely removed from the field
		// if pos.X == 0 && pos.Y == 0 {
		// 	continue
		// }
		robotObstacle := Obstacle{pos, RobotSafetyRadius}
		obstacles = append(obstacles, robotObstacle)
	}

	return obstacles
}

// Achieved returns true if the robot is sufficiently close to the final destination
func (m *MoveToPosition) Achieved(gi *info.GameInfo) bool {
	currPos, _ := gi.State.GetTeam(m.team)[m.id].GetPosition()
	distanceLeft := distanceBetween(currPos, m.final_destination)

	return distanceLeft <= m.rrtConfig.completionDistance
}

// Helper function to calculate distance between two positions
func distanceBetween(pos1, pos2 info.Position) float64 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (m *MoveToPosition) String() string {
	currPos, _ := m.gi.State.GetTeam(m.team)[m.id].GetPosition()
	return fmt.Sprintf("MoveToPosition: dist%f", distanceBetween(currPos, m.final_destination))
}

func (m *MoveToPosition) GetID() info.ID {
	return m.id
}
