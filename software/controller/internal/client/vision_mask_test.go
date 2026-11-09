package client

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

func withVisionMask(t *testing.T, team info.Team) {
	t.Helper()
	previous := visionMaskTeam
	visionMaskTeam = team
	t.Cleanup(func() { visionMaskTeam = previous })
}

func TestVisionMaskedDisabled(t *testing.T) {
	withVisionMask(t, info.UNKNOWN)
	gi := info.NewGameInfo(1)
	if visionMasked(gi, 3000) || visionMasked(gi, -3000) {
		t.Fatal("nothing should be masked when the restriction is off")
	}
}

func TestVisionMaskedHidesRestrictedTeamsOwnHalf(t *testing.T) {
	cases := []struct {
		team               info.Team
		blueOnPositiveHalf bool
		x                  float64
		want               bool
	}{
		{info.Yellow, false, 3000, true},
		{info.Yellow, false, visionMaskMarginMM - 1, false},
		{info.Yellow, false, -3000, false},
		{info.Blue, false, -3000, true},
		{info.Blue, false, 3000, false},
		{info.Blue, true, 3000, true},
	}
	for _, c := range cases {
		withVisionMask(t, c.team)
		gi := info.NewGameInfo(1)
		gi.Status.SetGameStatus(0, 0, 0, 0, 0, c.blueOnPositiveHalf, "")
		if got := visionMasked(gi, c.x); got != c.want {
			t.Errorf("%v, blueOnPositiveHalf=%v, x=%v: masked=%v, want %v", c.team, c.blueOnPositiveHalf, c.x, got, c.want)
		}
	}
}

func TestUnmaskedBallsDropsBallsOnMaskedHalf(t *testing.T) {
	withVisionMask(t, info.Yellow)
	gi := info.NewGameInfo(1)
	near, far := float32(-500), float32(2000)
	balls := []*ssl_vision.SSL_DetectionBall{{X: &far}, {X: &near}}
	kept := unmaskedBalls(gi, balls)
	if len(kept) != 1 || kept[0].GetX() != near {
		t.Fatalf("kept %v, want only the ball at x=%v", kept, near)
	}
}
