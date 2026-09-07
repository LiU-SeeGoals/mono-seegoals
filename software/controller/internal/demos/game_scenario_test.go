package demos

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestBallPlacementTargetUsesAutoRefPositionAfterRepeatedPacket(t *testing.T) {
	gameEvent := info.NewGameEvent()
	gameEvent.UpdateFromRefCommand(info.BALL_PLACEMENT_BLUE, 42, 1250, -2750, info.DIRECT_FREE_BLUE, 0, false)
	// Repeated referee packets used to erase the state transition that triggered
	// teleportation while retaining the same designated position.
	gameEvent.UpdateFromRefCommand(info.BALL_PLACEMENT_BLUE, 42, 1250, -2750, info.DIRECT_FREE_BLUE, 0, false)

	target, ok := refereePlacementTarget(gameEvent, info.Position{X: 4700, Y: -100})
	if !ok {
		t.Fatal("expected the ball to be teleported during repeated ball-placement packets")
	}
	if target.X != 1250 || target.Y != -2750 {
		t.Fatalf("expected AutoRef target (1250, -2750), got (%v, %v)", target.X, target.Y)
	}
}

func TestBallPlacementTargetDoesNotRetriggerAtTarget(t *testing.T) {
	gameEvent := info.NewGameEvent()
	gameEvent.UpdateFromRefCommand(info.BALL_PLACEMENT_YELLOW, 42, 100, 200, info.DIRECT_FREE_YELLOW, 0, false)

	if _, ok := refereePlacementTarget(gameEvent, info.Position{X: 105, Y: 205}); ok {
		t.Fatal("did not expect another teleport when the ball is already at the AutoRef target")
	}
}

func TestRefereePlacementTargetUsesPositionDuringStop(t *testing.T) {
	gameEvent := info.NewGameEvent()
	gameEvent.UpdateFromRefCommand(info.STOP, 42, -2097, 2800, info.DIRECT_FREE_BLUE, 0, false)

	target, ok := refereePlacementTarget(gameEvent, info.Position{X: -3000, Y: 3100})
	if !ok {
		t.Fatal("expected the STOP packet's referee placement position to trigger teleportation")
	}
	if target.X != -2097 || target.Y != 2800 {
		t.Fatalf("expected AutoRef target (-2097, 2800), got (%v, %v)", target.X, target.Y)
	}
}

func TestRefereePlacementTargetIgnoresAbsentPosition(t *testing.T) {
	gameEvent := info.NewGameEvent()
	gameEvent.UpdateFromRefCommandWithDesignatedPosition(
		info.STOP, 42, 0, 0, false, info.DIRECT_FREE_BLUE, 0, false,
	)

	if _, ok := refereePlacementTarget(gameEvent, info.Position{X: 1000}); ok {
		t.Fatal("did not expect an absent designated position to teleport the ball to zero")
	}
}

func TestRefereePlacementTargetIgnoresOtherStates(t *testing.T) {
	gameEvent := info.NewGameEvent()
	gameEvent.UpdateFromRefCommand(info.FORCE_START, 42, 100, 200, info.UNINITIALIZED, 0, false)

	if _, ok := refereePlacementTarget(gameEvent, info.Position{}); ok {
		t.Fatal("did not expect a teleport outside ball placement")
	}
}
