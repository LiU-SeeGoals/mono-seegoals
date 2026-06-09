package pathplanner

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestIsPathClearDetectsObstacleBetweenSamplePoints(t *testing.T) {
	start := info.Position{X: 0, Y: 0}
	end := info.Position{X: 1000, Y: 0}
	obstacles := []Obstacle{
		{Position: info.Position{X: 50, Y: 0}, Size: 10},
	}

	if IsPathClear(start, end, obstacles, 0) {
		t.Fatal("expected long segment crossing obstacle to be blocked")
	}
}

func TestIsPathClearAllowsOffsetObstacle(t *testing.T) {
	start := info.Position{X: 0, Y: 0}
	end := info.Position{X: 1000, Y: 0}
	obstacles := []Obstacle{
		{Position: info.Position{X: 500, Y: 25}, Size: 10},
	}

	if !IsPathClear(start, end, obstacles, 0) {
		t.Fatal("expected segment away from obstacle to be clear")
	}
}

func TestIsPathClearAllowsMovingAwayFromStartOverlap(t *testing.T) {
	start := info.Position{X: 0, Y: 0}
	end := info.Position{X: -1000, Y: 0}
	obstacles := []Obstacle{
		{Position: info.Position{X: 50, Y: 0}, Size: 100},
	}

	if !IsPathClear(start, end, obstacles, 0) {
		t.Fatal("expected segment moving away from an initial overlap to be clear")
	}
}

func TestShortcutPathRemovesClearMiddlePoints(t *testing.T) {
	path := []info.Position{
		{X: 0, Y: 0},
		{X: 100, Y: 100},
		{X: 200, Y: -100},
		{X: 300, Y: 0},
	}

	got := shortcutPath(path, nil, 0)

	if len(got) != 2 {
		t.Fatalf("expected shortcut path to keep only endpoints, got %d points", len(got))
	}
	if got[0] != path[0] || got[1] != path[len(path)-1] {
		t.Fatalf("expected shortcut path endpoints to be preserved, got %#v", got)
	}
}

func TestShortcutPathKeepsWaypointAroundObstacle(t *testing.T) {
	path := []info.Position{
		{X: 0, Y: 0},
		{X: 100, Y: 100},
		{X: 200, Y: 0},
	}
	obstacles := []Obstacle{
		{Position: info.Position{X: 100, Y: 0}, Size: 20},
	}

	got := shortcutPath(path, obstacles, 0)

	if len(got) != len(path) {
		t.Fatalf("expected obstacle to preserve waypoint, got %d points", len(got))
	}
	for i := range path {
		if got[i] != path[i] {
			t.Fatalf("expected waypoint %d to be preserved: got %#v want %#v", i, got[i], path[i])
		}
	}
}

func TestConnectToGoalIfClearReturnsGoalNode(t *testing.T) {
	from := &RRTNode{position: info.Position{X: 0, Y: 100}}
	goal := info.Position{X: 200, Y: 100}
	obstacles := []Obstacle{
		{Position: info.Position{X: 100, Y: 0}, Size: 20},
	}

	got := connectToGoalIfClear(from, goal, obstacles)

	if got == nil {
		t.Fatal("expected clear goal connection")
	}
	if got.parent != from {
		t.Fatal("expected goal node to use source node as parent")
	}
	if got.position != goal {
		t.Fatalf("expected goal node at final destination, got %#v", got.position)
	}
}

func TestConnectToGoalIfClearRejectsBlockedSegment(t *testing.T) {
	from := &RRTNode{position: info.Position{X: 0, Y: 100}}
	goal := info.Position{X: 200, Y: 100}
	obstacles := []Obstacle{
		{Position: info.Position{X: 100, Y: 100}, Size: 20},
	}

	if got := connectToGoalIfClear(from, goal, obstacles); got != nil {
		t.Fatalf("expected blocked goal connection to be rejected, got %#v", got)
	}
}
