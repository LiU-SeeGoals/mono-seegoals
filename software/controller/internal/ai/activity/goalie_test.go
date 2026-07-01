package ai

import (
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func newGoalieTestGameInfo(ballInPlay bool) *info.GameInfo {
	gi := info.NewGameInfo(2)
	now := time.Now().UnixMilli()
	gi.State.SetBlueRobot(0, 0, 0, 0, now)
	gi.State.SetBall(1000, 0, 0, now)
	gi.State.GetBall().SetEstimatedPosition(info.Position{X: 1000})
	gi.Status.GetGameEvent().BallInPlay = ballInPlay
	return gi
}

func TestGoalieUsesRRTWhenBallIsNotInPlay(t *testing.T) {
	previousPlanner := getPathService(info.Blue)
	SetPathService(info.Blue, pathplanner.New())
	defer SetPathService(info.Blue, previousPlanner)

	move, ok := NewGoalie(info.Blue, 0).GetAction(newGoalieTestGameInfo(false)).(*action.MoveTo)
	if !ok {
		t.Fatal("expected goalie move-to action")
	}
	if len(move.Path) == 0 {
		t.Fatal("expected stopped-play goalie action to include an RRT path")
	}
	if !move.AllowGoalArea {
		t.Fatal("expected goalie RRT movement to retain defense-area access")
	}
}

func TestGoalieKeepsDirectReactionWhenBallIsInPlay(t *testing.T) {
	move, ok := NewGoalie(info.Blue, 0).GetAction(newGoalieTestGameInfo(true)).(*action.MoveTo)
	if !ok {
		t.Fatal("expected goalie move-to action")
	}
	if len(move.Path) != 0 {
		t.Fatalf("expected in-play goalie movement to stay direct, got %d path points", len(move.Path))
	}
	if !move.AllowGoalArea {
		t.Fatal("expected goalie movement to retain defense-area access")
	}
}
