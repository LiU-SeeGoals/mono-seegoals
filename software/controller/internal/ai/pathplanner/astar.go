package pathplanner

import "github.com/LiU-SeeGoals/controller/internal/info"

// A* search over the visibility graph built in visgraph.go.

// runAStar returns the shortest collision-free path from start to goal through
// the visibility graph, starting with start and ending with goal. It returns
// nil if no path exists.
func runAStar(start, goal info.Position, obstacles []Obstacle, cfg RRTConfig) []info.Position {
	// TODO: direct shot, build nodes, search with astarQueue, reconstructPath.
	return nil
}

// reconstructPath follows parent indices back from the goal node and returns
// the path in start→goal order.
func reconstructPath(nodes []info.Position, parent []int, goalIdx int) []info.Position {
	// TODO
	return nil
}

// astarItem is one entry in the open set: a node index and its f = g + h score.
type astarItem struct {
	node int
	f    float64
}

// astarQueue is a min-priority queue of astarItems for container/heap.
type astarQueue []astarItem

func (q astarQueue) Len() int { return len(q) }

// Less orders by f; ties are broken by node index so results are deterministic.
func (q astarQueue) Less(i, j int) bool {
	if q[i].f != q[j].f {
		return q[i].f < q[j].f
	}
	return q[i].node < q[j].node
}

func (q astarQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *astarQueue) Push(x any) { *q = append(*q, x.(astarItem)) }

func (q *astarQueue) Pop() any {
	old := *q
	item := old[len(old)-1]
	*q = old[:len(old)-1]
	return item
}
