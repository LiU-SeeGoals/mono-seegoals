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
const RobotSafetyRadius = 90.0 // mm - increased for better safety margin

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
	stuckCounter      int           // Counter to detect when robot is stuck
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

// NewMoveToPositionWithCollisionAvoidance creates a new instance
func NewMoveToPosition(team info.Team, id info.ID, dest info.Position) *MoveToPosition {
	// Initialize with reasonable RRT parameters
	rrtConfig := rrtConfiguration{
		maxIterations:      1000,
		stepSize:           150.0,   // mm per step (increased for more aggressive exploration)
		goalBias:           0.01,    // 20% chance of sampling the goal directly (increased for more direct paths)
		waypointThreshold:  50.0,    // mm to consider waypoint reached
		fieldWidth:         13400.0, // Standard SSL field width in mm
		fieldHeight:        10400.0, // Standard SSL field height in mm
		completionDistance: 50.0,    // mm to consider the goal reached
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
		stuckCounter:      0,
		stuckThreshold:    10, // Consider robot stuck after 10 cycles without movement
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

	// Check if robot is stuck by comparing current position with last position
	if m.lastPosition.X != 0 || m.lastPosition.Y != 0 { // Skip first cycle
		moveDistance := distanceBetween(myPos, m.lastPosition)
		if moveDistance < 5.0 { // If robot has moved less than 5mm, increment stuck counter
			m.stuckCounter++
			if m.stuckCounter > m.stuckThreshold {
				// Robot is stuck, force immediate replanning with shorter interval
				m.planningInterval = 20 * time.Millisecond
				m.significantChange = true
			}
		} else {
			// Robot is moving, reset stuck counter
			m.stuckCounter = 0
			m.planningInterval = 50 * time.Millisecond
		}
	}
	m.lastPosition = myPos

	// Check for immediate collisions - Emergency avoidance
	obstacles := m.GetObstaclePositions(gi)
	inCollision := false

	// Calculate repulsive vector if we're too close to obstacles
	repulsiveX, repulsiveY := 0.0, 0.0

	for _, obstacle := range obstacles {
		dist := distanceBetween(myPos, obstacle)
		if dist <= RobotSafetyRadius {
			inCollision = true

			// Calculate unit vector away from obstacle
			dx := myPos.X - obstacle.X
			dy := myPos.Y - obstacle.Y

			// Normalize (avoid division by zero)
			if dist > 0.001 {
				dx /= dist
				dy /= dist
			} else {
				// If almost exactly overlapping, move in random direction
				angle := rand.Float64() * 2 * math.Pi
				dx = math.Cos(angle)
				dy = math.Sin(angle)
			}

			// Stronger repulsion for closer obstacles (inverse square law)
			force := 1.0 / math.Max(0.001, dist*dist) * 10000.0

			repulsiveX += dx * force
			repulsiveY += dy * force
		}
	}

	// If in collision, use emergency evasion movement
	if inCollision {
		// Normalize the repulsive vector
		magnitude := math.Sqrt(repulsiveX*repulsiveX + repulsiveY*repulsiveY)
		if magnitude > 0 {
			repulsiveX /= magnitude
			repulsiveY /= magnitude
		}

		// Create an emergency target position in the direction of the repulsive force
		emergencyTarget := info.Position{
			X:     myPos.X + repulsiveX*300.0, // Move 300mm in the repulsive direction
			Y:     myPos.Y + repulsiveY*300.0,
			Angle: myPos.Angle,
		}

		// Create move action to the emergency target
		act := action.MoveTo{}
		act.Id = int(m.id)
		act.Team = m.team
		act.Pos = myPos
		act.Dest = emergencyTarget
		act.Dribble = false
		return act
	}

	// Check for significant obstacle changes
	currentObstacles := obstacles
	m.CheckForSignificantChanges(currentObstacles)

	// Check if we need to replan due to time, path emptiness, or significant obstacle changes
	if time.Since(m.lastPlanningTime) >= m.planningInterval ||
		len(m.path) == 0 ||
		m.significantChange {
		m.PlanPath(gi, myPos)
		m.lastPlanningTime = time.Now()
		m.previousObstacles = currentObstacles
		m.significantChange = false
	}

	// If we have a path, navigate to the next waypoint
	targetPos := m.final_destination
	if len(m.path) > 0 {
		// Check if we've reached the current waypoint
		distance := distanceBetween(myPos, m.path[0])
		if distance <= m.rrtConfig.waypointThreshold && len(m.path) > 1 {
			// Remove the first waypoint
			m.path = m.path[1:]
		}

		// Get target waypoint
		targetPos = m.path[0]

		// Extend the waypoint further in the same direction to avoid slow PID behavior
		// Only extend if we're not at the final waypoint
		if len(m.path) > 1 || targetPos != m.final_destination {
			// Calculate direction vector from current position to target
			dx := targetPos.X - myPos.X
			dy := targetPos.Y - myPos.Y
			dist := math.Sqrt(dx*dx + dy*dy)

			// Only apply extension if we're close enough to be affected by PID slowdown
			// but not so close that we've essentially reached the waypoint
			const minExtensionDist = 100.0 // Don't extend if we're closer than this
			const extensionAmount = 150.0  // Extend by this amount

			if dist > minExtensionDist {
				// Normalize direction
				if dist > 0 {
					dx /= dist
					dy /= dist
				}

				// Extend the waypoint
				targetPos.X += dx * extensionAmount
				targetPos.Y += dy * extensionAmount

				// If this is the last waypoint and it's our final destination, don't modify
				if len(m.path) == 1 && distanceBetween(targetPos, m.final_destination) < 100.0 {
					targetPos = m.final_destination
				}
			}
		}
	}

	// Create move action to the current target
	act := action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = myPos
	act.Dest = targetPos
	act.Dribble = false
	return act
}

// CheckForSignificantChanges detects if obstacles have moved enough to require replanning
func (m *MoveToPosition) CheckForSignificantChanges(currentObstacles []info.Position) {
	if len(m.previousObstacles) != len(currentObstacles) {
		m.significantChange = true
		return
	}

	// Check if any obstacle has moved more than a threshold
	const significantMovementThreshold = 100.0 // mm

	for i, current := range currentObstacles {
		if i >= len(m.previousObstacles) {
			m.significantChange = true
			return
		}

		previous := m.previousObstacles[i]
		dist := distanceBetween(current, previous)

		if dist > significantMovementThreshold {
			m.significantChange = true
			return
		}
	}
}

// PlanPath uses RRT to plan a collision-free path
func (m *MoveToPosition) PlanPath(gi *info.GameInfo, startPos info.Position) {
	// Create a list of obstacle positions (other robots)
	obstacles := m.GetObstaclePositions(gi)

	// Check if we're already in collision
	robotsNearby := false
	var nearestObstacle info.Position
	shortestDist := math.MaxFloat64

	for _, obstacle := range obstacles {
		dist := distanceBetween(startPos, obstacle)
		if dist <= RobotSafetyRadius {
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
		dx := startPos.X - nearestObstacle.X
		dy := startPos.Y - nearestObstacle.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		// Normalize and scale to get a point outside the safety radius
		safeDistance := RobotSafetyRadius + 100.0 // Add extra margin
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
			X:     nearestObstacle.X + dx,
			Y:     nearestObstacle.Y + dy,
			Angle: startPos.Angle,
		}

		// Set this as our immediate path
		m.path = []info.Position{escapePos}
		return
	}

	// Check if direct path to goal is clear
	if m.IsPathClear(startPos, m.final_destination, obstacles) {
		m.path = []info.Position{m.final_destination}
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

	// Skip the start position if it's in the path
	if len(path) > 1 {
		path = path[1:]
	}

	// Simplify the path by removing redundant waypoints
	m.path = m.SimplifyPath(path, obstacles)
}

// RunRRT executes the RRT algorithm and returns the goal node if a path is found
func (m *MoveToPosition) RunRRT(nodes []*RRTNode, obstacles []info.Position) *RRTNode {
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
		if !m.IsPathClear(nearestNode.position, newNode.position, obstacles) {
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
			if m.IsPathClear(newNode.position, goalNode.position, obstacles) {
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
func (m *MoveToPosition) IsNodeValid(position info.Position, obstacles []info.Position, isStartPosition bool) bool {
	// Skip obstacle check for the starting position if specified
	if isStartPosition {
		return true
	}

	for _, obstacle := range obstacles {
		if distanceBetween(position, obstacle) <= RobotSafetyRadius {
			return false
		}
	}
	return true
}

// IsPathClear checks if the path between two positions is clear of obstacles
func (m *MoveToPosition) IsPathClear(start, end info.Position, obstacles []info.Position) bool {
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
func (m *MoveToPosition) GetObstaclePositions(gi *info.GameInfo) []info.Position {
	obstacles := []info.Position{}

	// Get all robots
	allRobots := append(gi.State.GetTeam(info.Blue)[:], gi.State.GetTeam(info.Yellow)[:]...)

	if m.avoidBall {
		ballPos, _ := gi.State.Ball.GetPosition()
		obstacles = append(obstacles, ballPos)
	}

	for _, robot := range allRobots {
		// Skip self
		if robot.GetTeam() == m.team && robot.GetID() == m.id {
			continue
		}

		pos, err := robot.GetPosition()
		if err != nil {
			continue
		}

		// Skip robots at exactly (0,0) as they are likely removed from the field
		// if pos.X == 0 && pos.Y == 0 {
		// 	continue
		// }

		obstacles = append(obstacles, pos)
	}

	return obstacles
}

// SimplifyPath removes redundant waypoints from the path with more aggressive pruning
func (m *MoveToPosition) SimplifyPath(path []info.Position, obstacles []info.Position) []info.Position {
	if len(path) <= 2 {
		return path
	}

	// Start with first node
	simplified := []info.Position{path[0]}

	// Initially try to connect directly to the goal
	if m.IsPathClear(path[0], path[len(path)-1], obstacles) {
		return []info.Position{path[len(path)-1]}
	}

	// Use a sliding window approach with increasing window size for more aggressive pruning
	startIdx := 0

	// Keep extending the window until we've processed the whole path
	for startIdx < len(path)-1 {
		// Try largest possible skip first
		foundSkip := false
		for endIdx := len(path) - 1; endIdx > startIdx+1; endIdx-- {
			if m.IsPathClear(path[startIdx], path[endIdx], obstacles) {
				// We can skip directly to this node
				simplified = append(simplified, path[endIdx])
				startIdx = endIdx
				foundSkip = true
				break
			}
		}

		// If we couldn't skip ahead, just add the next node
		if !foundSkip {
			simplified = append(simplified, path[startIdx+1])
			startIdx++
		}
	}

	// Make sure we have at most MAX_WAYPOINTS
	const MAX_WAYPOINTS = 5
	if len(simplified) > MAX_WAYPOINTS {
		// Keep first and last waypoints, and evenly sample the rest
		result := []info.Position{simplified[0]}
		step := float64(len(simplified)-2) / float64(MAX_WAYPOINTS-2)

		for i := 1; i < MAX_WAYPOINTS-1; i++ {
			idx := int(float64(i) * step)
			if idx >= len(simplified)-1 {
				break
			}
			result = append(result, simplified[idx+1])
		}

		result = append(result, simplified[len(simplified)-1])
		return result
	}

	return simplified
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
