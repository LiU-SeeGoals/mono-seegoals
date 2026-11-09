package ai

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func gameInfoWithSides(blueOnPositiveHalf bool) *info.GameInfo {
	gi := info.NewGameInfo(1)
	gi.Status.SetGameStatus(0, 0, 0, 0, 0, blueOnPositiveHalf, "")
	return gi
}

func TestRestrictsTeamToOpponentHalf(t *testing.T) {
	cases := []struct {
		setting string
		team    info.Team
		want    bool
	}{
		{"", info.Yellow, false},
		{"", info.Blue, false},
		{"yellow", info.Yellow, true},
		{"yellow", info.Blue, false},
		{" Blue ", info.Blue, true},
		{"true", info.Blue, false},
	}
	for _, c := range cases {
		if got := restrictsTeamToOpponentHalf(c.setting, c.team); got != c.want {
			t.Errorf("restrictsTeamToOpponentHalf(%q, %v) = %v, want %v", c.setting, c.team, got, c.want)
		}
	}
}

func TestClampActionToOpponentHalfDestination(t *testing.T) {
	cases := []struct {
		name               string
		team               info.Team
		blueOnPositiveHalf bool
		destX              float64
		wantX              float64
	}{
		{"yellow defends positive, dest on own half", info.Yellow, false, 2000, -opponentHalfRobotRadiusMM},
		{"yellow defends positive, dest on opponent half", info.Yellow, false, -2000, -2000},
		{"blue defends positive, dest on own half", info.Blue, true, 50, -opponentHalfRobotRadiusMM},
		{"blue defends negative, dest on own half", info.Blue, false, -1000, opponentHalfRobotRadiusMM},
		{"blue defends negative, dest on opponent half", info.Blue, false, 1000, 1000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			move := &action.MoveTo{Dest: info.Position{X: c.destX, Y: 300}, KickSpeed: 3}
			clampActionToOpponentHalf(move, gameInfoWithSides(c.blueOnPositiveHalf), c.team)
			if move.Dest.X != c.wantX || move.Dest.Y != 300 {
				t.Fatalf("dest = (%v, %v), want (%v, 300)", move.Dest.X, move.Dest.Y, c.wantX)
			}
			wantKick := 3
			if c.wantX != c.destX {
				wantKick = 0
			}
			if move.KickSpeed != wantKick {
				t.Fatalf("kick speed = %v, want %v", move.KickSpeed, wantKick)
			}
		})
	}
}

func TestClampActionToOpponentHalfSkipsPlacementStates(t *testing.T) {
	for _, state := range []info.RefState{info.STATE_KICKOFF_PREPARATION, info.STATE_PENALTY_PREPARATION, info.STATE_BALL_PLACEMENT} {
		gi := gameInfoWithSides(false)
		gi.Status.GetGameEvent().CurrentState = state
		move := &action.MoveTo{Dest: info.Position{X: 2000}}
		clampActionToOpponentHalf(move, gi, info.Yellow)
		if move.Dest.X != 2000 {
			t.Errorf("%v: dest clamped to %v during placement", state, move.Dest.X)
		}
	}
}

func TestClampActionToOpponentHalfPath(t *testing.T) {
	gi := gameInfoWithSides(false)

	inside := &action.MoveTo{
		Pos:  info.Position{X: -1000},
		Dest: info.Position{X: 1500, Y: 200},
		Path: []info.Position{{X: -800, Y: 100}, {X: -200, Y: 150}, {X: 600, Y: 180}, {X: 1500, Y: 200}},
	}
	clampActionToOpponentHalf(inside, gi, info.Yellow)
	want := []info.Position{{X: -800, Y: 100}, {X: -200, Y: 150}, {X: -opponentHalfRobotRadiusMM, Y: 200}}
	assertPath(t, "inside", inside.Path, want)

	leaving := &action.MoveTo{
		Pos:  info.Position{X: 1000},
		Dest: info.Position{X: -1500},
		Path: []info.Position{{X: 800}, {X: 200}, {X: -500}, {X: -1500}},
	}
	clampActionToOpponentHalf(leaving, gi, info.Yellow)
	assertPath(t, "leaving", leaving.Path, []info.Position{{X: 800}, {X: 200}, {X: -500}, {X: -1500}})
}

func assertPath(t *testing.T, name string, got, want []info.Position) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: path = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i].X != want[i].X || got[i].Y != want[i].Y {
			t.Fatalf("%s: path = %v, want %v", name, got, want)
		}
	}
}
