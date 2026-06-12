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

func TestIsPathClearRejectsRectObstacle(t *testing.T) {
	obstacles := []Obstacle{
		rectObstacle(100, 200, -50, 50, 0),
	}

	if IsPathClear(info.Position{X: 0, Y: 0}, info.Position{X: 300, Y: 0}, obstacles, 0) {
		t.Fatal("expected segment through rectangle to be blocked")
	}
}

func TestIsPathClearAllowsLeavingRectObstacle(t *testing.T) {
	obstacles := []Obstacle{
		rectObstacle(100, 200, -50, 50, 0),
	}

	if !IsPathClear(info.Position{X: 150, Y: 0}, info.Position{X: 250, Y: 0}, obstacles, 0) {
		t.Fatal("expected segment leaving rectangle to be allowed")
	}
}

func TestIsNodeValidRejectsRectObstacle(t *testing.T) {
	obstacles := []Obstacle{
		rectObstacle(100, 200, -50, 50, 10),
	}

	if isNodeValid(info.Position{X: 95, Y: 0}, obstacles, false) {
		t.Fatal("expected node inside inflated rectangle to be invalid")
	}
	if !isNodeValid(info.Position{X: 80, Y: 0}, obstacles, false) {
		t.Fatal("expected node outside inflated rectangle to be valid")
	}
}

func TestNoGoZoneEscapePathUsesNearestSide(t *testing.T) {
	obstacles := []Obstacle{
		rectObstacle(100, 200, -50, 50, 10),
	}
	pos := info.Position{X: 205, Y: 0, Angle: 1.5}

	got, ok := noGoZoneEscapePath(pos, obstacles)

	if !ok {
		t.Fatal("expected escape path for position inside no-go zone")
	}
	if len(got) != 1 {
		t.Fatalf("expected one escape waypoint, got %d", len(got))
	}
	wantX := 200.0 + 10.0 + noGoZoneEscapeClearance
	if got[0].X != wantX || got[0].Y != pos.Y {
		t.Fatalf("expected nearest right-side exit at (%v,%v), got %#v", wantX, pos.Y, got[0])
	}
	if got[0].Angle != pos.Angle {
		t.Fatalf("expected escape waypoint to preserve angle, got %v want %v", got[0].Angle, pos.Angle)
	}
	if pointInsideObstacle(got[0], obstacles[0]) {
		t.Fatal("expected escape waypoint outside inflated no-go zone")
	}
}

func TestNoGoZoneEscapePathIgnoresOutsidePosition(t *testing.T) {
	obstacles := []Obstacle{
		rectObstacle(100, 200, -50, 50, 10),
	}

	if got, ok := noGoZoneEscapePath(info.Position{X: 50, Y: 0}, obstacles); ok {
		t.Fatalf("expected no escape path outside no-go zone, got %#v", got)
	}
}

func TestStoredPathInvalidWhenOutsidePlanningBounds(t *testing.T) {
	goal := info.Position{X: 3900, Y: 0}
	st := &robotPathState{
		path: []info.Position{
			{X: 0, Y: 0},
			{X: 4500, Y: 0},
			goal,
		},
		goal: goal,
	}
	cfg := RRTConfig{FieldWidth: 8000, FieldHeight: 6000}

	if isStoredPathValid(st, info.Position{X: 0, Y: 0}, goal, nil, cfg, 10) {
		t.Fatal("expected cached path outside current field bounds to be invalid")
	}
}

func TestClampToPlanningBoundsPreservesAngle(t *testing.T) {
	cfg := RRTConfig{FieldWidth: 8000, FieldHeight: 6000}
	got := clampToPlanningBounds(info.Position{X: 4500, Y: -3500, Angle: 1.5}, cfg)

	if got.X != 4000 {
		t.Fatalf("expected X clamped to 4000, got %v", got.X)
	}
	if got.Y != -3000 {
		t.Fatalf("expected Y clamped to -3000, got %v", got.Y)
	}
	if got.Angle != 1.5 {
		t.Fatalf("expected angle to be preserved, got %v", got.Angle)
	}
}
