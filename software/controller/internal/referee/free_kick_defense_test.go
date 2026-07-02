package referee

import (
	"sync"
	"testing"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestFreeKickDefenseMirrorsAssignedHalfAndKeepsGoalie(t *testing.T) {
	tests := []struct {
		name             string
		bluePositiveHalf bool
		wantDefenseX     float64
	}{
		{name: "negative half", bluePositiveHalf: false, wantDefenseX: -2000},
		{name: "positive half", bluePositiveHalf: true, wantDefenseX: 2000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gi := info.NewGameInfo(int(info.TEAM_SIZE))
			gi.Status.SetGameStatus(0, 0, 0, 0, 0, tc.bluePositiveHalf, "")
			activities := [info.TEAM_SIZE]act.Activity{}
			handler := &coreai.ActivityHandler{
				Activities:    &activities,
				Activity_lock: &sync.Mutex{},
			}

			moveRobotsToDefensePosition(gi, []info.ID{1, 2, 6}, info.Blue, handler)

			if _, ok := activities[6].(*act.Goalie); !ok {
				t.Fatalf("goalie activity = %T, want *activity.Goalie", activities[6])
			}
			for _, robotID := range []info.ID{1, 2} {
				if _, ok := activities[robotID].(*act.MoveToPosition); !ok {
					t.Errorf("defender %d activity = %T, want *activity.MoveToPosition", robotID, activities[robotID])
				}
			}

			slots := freeKickDefenseSlots(gi, info.Blue, 2)
			if slots[0].X != tc.wantDefenseX || slots[1].X != tc.wantDefenseX {
				t.Fatalf("defense X positions = %.0f, %.0f, want %.0f", slots[0].X, slots[1].X, tc.wantDefenseX)
			}
			if slots[0].Y != -150 || slots[1].Y != 150 {
				t.Fatalf("defense Y positions = %.0f, %.0f, want -150, 150", slots[0].Y, slots[1].Y)
			}
		})
	}
}
