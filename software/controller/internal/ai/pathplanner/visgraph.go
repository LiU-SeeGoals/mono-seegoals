package pathplanner

import "github.com/LiU-SeeGoals/controller/internal/info"

// Visibility-graph construction for the A* planner (see astar.go).
//
// Graph nodes are placed just outside every obstacle: a ring of points around
// each disc and one point outside each corner of each rectangle. Edges are
// checked lazily by A* with IsPathClear.

const (
	// visRingNodes is the number of graph nodes placed around each disc.
	visRingNodes = 8
	// visMargin is extra clearance (mm) between graph nodes and obstacles.
	visMargin = 20.0
)

// graphNodes returns all obstacle-hugging nodes (without start and goal),
// keeping only those inside the planning bounds and outside every obstacle.
func graphNodes(obstacles []Obstacle, cfg RRTConfig) []info.Position {
	// TODO: ringNodes for discs, rectCornerNodes for rectangles, then filter.
	return nil
}

// ringNodes returns visRingNodes points on a circle around a disc obstacle,
// far enough out that edges between neighbouring points don't cut the disc.
func ringNodes(obstacle Obstacle) []info.Position {
	// TODO
	return nil
}

// rectCornerNodes returns one point just outside each corner of a
// rectangular obstacle (using its expandedRect).
func rectCornerNodes(obstacle Obstacle) []info.Position {
	// TODO
	return nil
}
