package referee

import (
	"sync"
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	activity "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestRestartPostKickHoldComplete(t *testing.T) {
	started := time.Unix(10, 0)
	original := info.Position{}
	clearBall := info.Position{X: restartPostKickBallClearanceMM}
	nearBall := info.Position{X: BALL_IN_PLAY_DISTANCE_MM}

	tests := []struct {
		name    string
		started time.Time
		ball    info.Position
		elapsed time.Duration
		want    bool
	}{
		{name: "not started", ball: clearBall, elapsed: restartPostKickMaxHeadingHold, want: false},
		{name: "minimum delay is mandatory", started: started, ball: clearBall, elapsed: restartPostKickMinHeadingHold - time.Millisecond, want: false},
		{name: "clear ball releases after minimum delay", started: started, ball: clearBall, elapsed: restartPostKickMinHeadingHold, want: true},
		{name: "near ball remains held", started: started, ball: nearBall, elapsed: restartPostKickMaxHeadingHold - time.Millisecond, want: false},
		{name: "maximum delay prevents a stuck hold", started: started, ball: nearBall, elapsed: restartPostKickMaxHeadingHold, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := restartPostKickHoldComplete(tc.started, original, tc.ball, started.Add(tc.elapsed)); got != tc.want {
				t.Fatalf("restartPostKickHoldComplete() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHoldRestartKickerHeadingUsesStoredKickHeading(t *testing.T) {
	gi := info.NewGameInfo(int(info.TEAM_SIZE))
	gi.State.SetBlueRobot(1, 100, 200, 1.2, 1)

	var activities [info.TEAM_SIZE]activity.Activity
	var lock sync.Mutex
	handler := &coreai.ActivityHandler{Activities: &activities, Activity_lock: &lock}

	holdRestartKickerHeading(gi, info.Blue, 1, 0.75, true, handler)
	held := handler.GetActivity(1)
	if held == nil {
		t.Fatal("restart kicker did not receive a heading-hold activity")
	}
	move, ok := held.GetAction(gi).(*action.MoveTo)
	if !ok {
		t.Fatalf("hold action = %T, want *action.MoveTo", held.GetAction(gi))
	}
	if move.Dest.X != 100 || move.Dest.Y != 200 || move.Dest.Angle != 0.75 {
		t.Fatalf("hold destination = %+v, want current position with angle 0.75", move.Dest)
	}
}
