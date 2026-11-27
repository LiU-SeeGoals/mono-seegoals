package info

import (
	"math"
	"testing"
)

func TestNormalize2dIsUnbiasedUnitVector(t *testing.T) {
	got := Position{X: 3, Y: 4, Z: 100, Angle: 1.5}.Normalize2d()
	if math.Abs(got.X-0.6) > 1e-12 || math.Abs(got.Y-0.8) > 1e-12 {
		t.Fatalf("Normalize2d = (%v, %v), want (0.6, 0.8)", got.X, got.Y)
	}
	if got.Z != 100 || got.Angle != 1.5 {
		t.Fatalf("Normalize2d changed Z/Angle: %v", got)
	}
}

func TestNormalizeZeroVectorIsNotNaN(t *testing.T) {
	for _, got := range []Position{
		Position{Angle: 2}.Normalize2d(),
		Position{Angle: 2}.Normalize(),
	} {
		if got.X != 0 || got.Y != 0 || got.Z != 0 || got.Angle != 2 {
			t.Fatalf("zero vector normalized to %v", got)
		}
	}
}

func TestNormalizeKeepsAngle(t *testing.T) {
	got := Position{X: 0, Y: 0, Z: 2, Angle: 3}.Normalize()
	if got.Z != 1 || got.Angle != 3 {
		t.Fatalf("Normalize = %v, want Z=1 Angle=3", got)
	}
}
