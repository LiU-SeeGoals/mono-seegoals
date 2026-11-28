package pathplanner

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestIsPathClearHonorsExtraMargin(t *testing.T) {
	circle := []Obstacle{{Position: info.Position{X: 500, Y: 120}, Size: 100}}
	start, end := info.Position{}, info.Position{X: 1000}
	if !IsPathClear(start, end, circle, 0) {
		t.Fatal("segment 120 mm from a 100 mm obstacle should be clear")
	}
	if IsPathClear(start, end, circle, 30) {
		t.Fatal("30 mm margin ignored for circles")
	}
	rect := []Obstacle{rectObstacle(400, 600, 20, 300, 0)}
	if !IsPathClear(start, end, rect, 0) || IsPathClear(start, end, rect, 30) {
		t.Fatal("extra margin ignored for rects")
	}
}
