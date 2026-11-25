package pathplanner

import "testing"

func TestAStarDirectPathWhenClear(t *testing.T) {
	t.Skip("TODO: no obstacles → path is exactly [start, goal]")
}

func TestAStarGoesAroundRobot(t *testing.T) {
	t.Skip("TODO: robot between start and goal → path bends around it")
}

func TestAStarGoesAroundGoalArea(t *testing.T) {
	t.Skip("TODO: rectangle obstacle between start and goal → path uses its corners")
}

func TestAStarPathSegmentsAreClear(t *testing.T) {
	t.Skip("TODO: every segment of the result passes IsPathClear")
}

func TestAStarIsDeterministic(t *testing.T) {
	t.Skip("TODO: same input twice → identical path")
}

func TestAStarBlockedGoalReturnsNil(t *testing.T) {
	t.Skip("TODO: goal unreachable → nil")
}

func TestGraphNodesAreOutsideObstacles(t *testing.T) {
	t.Skip("TODO: every node from graphNodes passes isNodeValid")
}
