package roles

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

func ptr[T any](v T) *T {
	return &v
}

func testGameInfoWithRobots() *info.GameInfo {
	gi := info.NewGameInfo(4)
	gi.SetField(&ssl_vision.SSL_GeometryFieldSize{
		FieldLength: ptr[int32](9000),
		FieldWidth:  ptr[int32](6000),
		GoalWidth:   ptr[int32](1000),
		FieldLines: []*ssl_vision.SSL_FieldLineSegment{
			{
				Name: ptr("CenterLine"),
				P1:   &ssl_vision.Vector2F{X: ptr[float32](4500), Y: ptr[float32](0)},
				P2:   &ssl_vision.Vector2F{X: ptr[float32](-4500), Y: ptr[float32](0)},
			},
		},
	})
	gi.State.SetBall(0, 0, 0, 1)
	gi.State.Update()
	gi.State.SetYellowRobot(1, 0, 0, 0, 1)
	gi.State.SetYellowRobot(2, 3500, 0, 0, 1)
	gi.State.SetYellowRobot(3, 2500, 0, 0, 1)
	return gi
}

func TestBestReceiverIDUsesConfiguredPassCandidates(t *testing.T) {
	gi := testGameInfoWithRobots()
	intent := &AttemptGoalIntent{gi: gi, team: info.Yellow, id: 1}
	intent.SetPassCandidates([]info.ID{1, 3})

	got, ok := intent.bestReceiverID()
	if !ok {
		t.Fatal("bestReceiverID did not find a receiver")
	}
	if got != 3 {
		t.Fatalf("bestReceiverID = %d, want 3", got)
	}
}

func TestBestReceiverIDDoesNotUseRobotsOutsidePassCandidates(t *testing.T) {
	gi := testGameInfoWithRobots()
	intent := &AttemptGoalIntent{gi: gi, team: info.Yellow, id: 1}
	intent.SetPassCandidates([]info.ID{1})

	_, ok := intent.bestReceiverID()
	if ok {
		t.Fatal("bestReceiverID found a receiver even though no teammate candidate was configured")
	}
}
