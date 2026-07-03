package action

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestTranslateSimAppliesRequestedMinimumLinearSpeed(t *testing.T) {
	move := MoveTo{
		Pos:            info.Position{},
		Dest:           info.Position{X: 50},
		MinLinearSpeed: 0.5,
	}

	velocity := move.TranslateSim().GetMoveCommand().GetLocalVelocity()
	speed := math.Hypot(float64(velocity.GetForward()), float64(velocity.GetLeft()))
	if math.Abs(speed-0.5) > 1e-6 {
		t.Fatalf("expected 0.5 m/s minimum linear speed, got %.6f", speed)
	}
}

func TestTranslateSimCapsMinimumLinearSpeedAtMaximum(t *testing.T) {
	move := MoveTo{
		Pos:            info.Position{},
		Dest:           info.Position{X: 50},
		MinLinearSpeed: 2,
	}

	velocity := move.TranslateSim().GetMoveCommand().GetLocalVelocity()
	speed := math.Hypot(float64(velocity.GetForward()), float64(velocity.GetLeft()))
	if math.Abs(speed-1) > 1e-6 {
		t.Fatalf("expected minimum speed to be capped at 1 m/s, got %.6f", speed)
	}
}
