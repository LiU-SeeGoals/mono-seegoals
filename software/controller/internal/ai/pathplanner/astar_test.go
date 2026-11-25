package pathplanner

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestPointSegmentDist(t *testing.T) {
	a := info.Position{X: 0, Y: 0}
	b := info.Position{X: 1000, Y: 0}

	cases := []struct {
		name string
		p    info.Position
		want float64
	}{
		{"above middle", info.Position{X: 500, Y: 300}, 300},
		{"on segment", info.Position{X: 200, Y: 0}, 0},
		{"before start", info.Position{X: -300, Y: 400}, 500},
		{"after end", info.Position{X: 1300, Y: -400}, 500},
	}
	for _, c := range cases {
		if got := pointSegmentDist(c.p, a, b); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}

	// Zero-length segment: distance to the single point.
	if got := pointSegmentDist(info.Position{X: 300, Y: 400}, a, a); math.Abs(got-500) > 1e-9 {
		t.Errorf("zero-length: got %v, want 500", got)
	}
}
