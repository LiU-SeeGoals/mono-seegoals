package ai

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const RRT = true

// RobotSafetyRadius defines the no-movement zone around each robot
const RobotSafetyRadius = 240.0 // mm - increased for better safety margin
const BallSafetyRadius = 150.0  // mm - increased for better safety margin
const PlanningRadius = 100.0    // mm - increased for better safety margin
const MotionRadius = 50.0      // mm - increased for better safety margin

// MoveToPositionWithCollisionAvoidance handles collision avoidance using RRT
type MoveToPosition struct {
	team              info.Team
	id                info.ID
	final_destination info.Position    // The ultimate goal position
	path              []info.Position  // Current path from RRT planning
	rrtConfig         rrtConfiguration // Configuration for the RRT algorithm
	gi                *info.GameInfo
	avoidBall         bool
}

// rrtConfiguration holds parameters for the RRT algorithm
type rrtConfiguration struct {
	maxIterations      int
	stepSize           float64 // How far to extend the tree in each step
	goalBias           float64 // Probability of sampling the goal directly
	lookAheadHorizon   float64 // How far to look when selection move point
	fieldWidth         float64 // Width of the field in mm
	fieldHeight        float64 // Height of the field in mm
	completionDistance float64 // Distance to consider goal reached
}

// RRTNode represents a node in the RRT tree
type RRTNode struct {
	position info.Position
	parent   *RRTNode
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
		stepSize:           2000.0,   // mm per step (increased for more aggressive exploration)
		goalBias:           0.05,   // 20% chance of sampling the goal directly (increased for more direct paths)
		lookAheadHorizon:   600.0,  // mm to consider waypoint reached
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

	m.gi = gi
	var targetPos info.Position

	if RRT {
		// m.rrtConfig.stepSize = min(max(myPos.Dist2d(m.final_destination)/100, 1), m.rrtConfig.stepSize)
		// fmt.Println(m.rrtConfig.stepSize)
		// fmt.Println(m.final_destination)
		// fmt.Println(myPos)

		m.AvoidBall(true)
		targetPos = myPos

		m.PlanPath(gi, myPos)

		// fmt.Println(len(m.path))
		if len(m.path) > 0 {
			bestPoint := m.final_destination
			bestPointDist := math.Inf(1)
			if info.Dist2d(m.path[len(m.path)-1], myPos) < m.rrtConfig.lookAheadHorizon &&
				m.IsPathClear(myPos, m.path[len(m.path)-1], m.GetObstaclePositions(gi), MotionRadius) ||
				len(m.path) == 1 {

				bestPoint = m.path[len(m.path)-1]
			} else {
				for i := 0; i < len(m.path)-1; i++ {
					a := info.Vec2{X: m.path[i].X, Y: m.path[i].Y}
					b := info.Vec2{X: m.path[i+1].X, Y: m.path[i+1].Y}
					p := info.Vec2{X: myPos.X, Y: myPos.Y}
					pointOnLine := info.PointToLineSegment(a, b, p)
					distToPathSegment := info.DistToLineSegment(a,b,p)
					distanceToThreshold := math.Abs(distToPathSegment - m.rrtConfig.lookAheadHorizon)
					if distanceToThreshold < float64(bestPointDist) && m.IsPathClear(myPos, info.Position{X: pointOnLine.X, Y: pointOnLine.Y, Z: 0, Angle: 0}, m.GetObstaclePositions(gi), MotionRadius) {
						bestPointDist = distanceToThreshold

						bestPoint = info.Position{X: pointOnLine.X, Y: pointOnLine.Y, Z: myPos.Z, Angle: myPos.Angle}
					}
				}
			}
			targetPos = bestPoint
		}
	} else {
		targetPos = m.final_destination
	}

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
		dist := info.Dist2d(startPos, obstacle.position)
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

	nodeClearFailed := 0
	pathClearFailed := 0

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
		if !m.IsNodeValid(newNode.position, obstacles, PlanningRadius) {
			nodeClearFailed += 1
			continue
		}

		// Check if the path to the new node is clear
		if !m.IsPathClear(nearestNode.position, newNode.position, obstacles, PlanningRadius) {
			pathClearFailed += 1
			continue
		}

		// Add the new node to the tree
		newNode.parent = nearestNode
		nodes = append(nodes, newNode)

		// Check if we're close enough to the goal
		if info.Dist2d(newNode.position, m.final_destination) < m.rrtConfig.completionDistance {
			// Create a final node at the exact goal position
			goalNode := &RRTNode{
				position: m.final_destination,
				parent:   newNode,
			}

			// Check if the path to the goal is clear
			if m.IsPathClear(newNode.position, goalNode.position, obstacles, PlanningRadius) {
				return goalNode
			}
		}
	}

	// If we reach max iterations without finding a path, connect to the node closest to the goal
	fmt.Println("node", nodeClearFailed)
	fmt.Println("path", pathClearFailed)
	closestNode := m.FindNearestNode(nodes, m.final_destination)

	// If the closest node is too far from the goal, return nil
	// if info.Dist2d(closestNode.position, m.final_destination) > 500.0 {
	// 	return nil
	// }

	return closestNode
}

// FindNearestNode finds the node in the tree closest to the target position
func (m *MoveToPosition) FindNearestNode(nodes []*RRTNode, target info.Position) *RRTNode {
	minDist := math.MaxFloat64
	var nearest *RRTNode

	for _, node := range nodes {
		dist := info.Dist2d(node.position, target)
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
	}
}

// IsNodeValid checks if a node is valid (not too close to obstacles)
// Added isStartPosition parameter to allow the starting position even if it's near obstacles
func (m *MoveToPosition) IsNodeValid(position info.Position, obstacles []Obstacle, extraMargin float64) bool {
	// Skip obstacle check for the starting position if specified
	for _, obstacle := range obstacles {
		// fmt.Println(info.Dist2d(position, obstacle.position), obstacle.size+extraMargin)
		
		if info.Dist2d(position, obstacle.position) < obstacle.size + extraMargin {
			asdf,err := m.gi.State.GetRobotPosition(m.team,m.id)
			if err != nil{
			return false
			}
			fmt.Println(obstacle.position, asdf)
			return false
		}
	}
	// fmt.Println("true")
	return true
}

// IsPathClear checks if the path between two positions is clear of obstacles
func (m *MoveToPosition) IsPathClear(start, end info.Position, obstacles []Obstacle, extraMargin float64) bool {

	for _, obstacle := range obstacles {
		dist := info.DistToLineSegment(start.ToV2(), end.ToV2(), obstacle.position.ToV2())
			if dist < obstacle.size + extraMargin {
				return false
			}
	}

	// for i := 0; i <= numChecks; i++ {
	// 	t := float64(i) / float64(numChecks)
	// 	checkPos := info.Position{
	// 		X:     start.X + t*(end.X-start.X),
	// 		Y:     start.Y + t*(end.Y-start.Y),
	// 		Angle: start.Angle, // Angle doesn't matter here
	// 	}
	//
	// 	// Skip the first point (which is the start position)
	// 	if i == 0 {
	// 		continue
	// 	}
	//
	// 	if !m.IsNodeValid(checkPos, obstacles, false) {
	// 		return false
	// 	}
	// }

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

		pos, robotTime, err := robot.GetPositionTime()
		if err != nil {
			continue
		}

		if time.Now().UnixMilli() - robotTime > 500 {
			continue
		}

		robotObstacle := Obstacle{pos, RobotSafetyRadius}
		obstacles = append(obstacles, robotObstacle)
	}

	return obstacles
}

// Achieved returns true if the robot is sufficiently close to the final destination
func (m *MoveToPosition) Achieved(gi *info.GameInfo) bool {
	currPos, _ := gi.State.GetTeam(m.team)[m.id].GetPosition()
	distanceLeft := info.Dist2d(currPos, m.final_destination)

	return distanceLeft <= m.rrtConfig.completionDistance
}

func (m *MoveToPosition) String() string {
	currPos, _ := m.gi.State.GetTeam(m.team)[m.id].GetPosition()
	return fmt.Sprintf("MoveToPosition: dist%f", info.Dist2d(currPos, m.final_destination))
}

func (m *MoveToPosition) GetID() info.ID {
	return m.id
}
