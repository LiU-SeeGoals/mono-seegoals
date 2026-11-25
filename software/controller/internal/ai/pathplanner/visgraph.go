package pathplanner

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Visibility-graph geometry for the A* planner (see astar.go).

// pointSegmentDist returns the shortest distance from p to the segment a→b.
func pointSegmentDist(p, a, b info.Position) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	lenSq := dx*dx + dy*dy
	if lenSq == 0 {
		return distanceBetween(p, a)
	}
	// t is how far along a→b the closest point lies (0 = a, 1 = b).
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / lenSq
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(p.X-(a.X+t*dx), p.Y-(a.Y+t*dy))
}
