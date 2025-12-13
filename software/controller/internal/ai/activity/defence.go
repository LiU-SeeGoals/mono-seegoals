package ai

import (
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// DefenceActivity emulates the behaviour from plan_defence: if an opponent is close to the ball,
// robot #4 moves to the midpoint between own goal and ball.
type DefenceActivity struct {
	team  info.Team
	id    info.ID
	gi    info.GameInfo
	ready bool
}

func NewDefenceActivity(team info.Team) *DefenceActivity {
	return &DefenceActivity{team: team, id: 4}
}

func (d *DefenceActivity) GetAction(gi *info.GameInfo) action.Action {
	d.gi = *gi
	// Keep it simple: if no opponent near ball, idle
	if !opponentCloseToBall(d.gi, d.team, 300) {
		return &action.Stop{Id: int(d.id)}
	}

	ballPos, err := d.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		ballPos = info.Position{}
	}
	goalX := -4300.0
	if d.team == info.Yellow {
		goalX = 4300.0
	}
	target := info.Position{
		X:     (goalX + ballPos.X) / 2,
		Y:     ballPos.Y / 2,
		Z:     0,
		Angle: 0,
	}
	robot := d.gi.State.GetTeam(d.team)[d.id]
	robotPos, err := robot.GetPosition()
	if err != nil {
		robotPos = info.Position{}
	}
	return &action.MoveTo{
		Id:   int(d.id),
		Pos:  robotPos,
		Dest: target,
		Team: d.team,
	}
}

func (d *DefenceActivity) Achieved(*info.GameInfo) bool { return false }
func (d *DefenceActivity) GetID() info.ID               { return d.id }
func (d *DefenceActivity) String() string               { return "DefenceActivity" }

func opponentCloseToBall(gi info.GameInfo, myTeam info.Team, threshold float64) bool {
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return false
	}
	opponentTeam := myTeam.Opponent()
	for _, r := range gi.State.GetTeam(opponentTeam) {
		if r == nil {
			continue
		}
		pos, err := r.GetPosition()
		if err != nil {
			continue
		}
		if pos.Distance(ballPos) < threshold {
			return true
		}
	}
	return false
}

// Optional: throttle behaviour if needed
func (d *DefenceActivity) wait(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

// Helper: midpoint if needed elsewhere
func midpoint(a, b info.Position) info.Position {
	return info.Position{
		X:     (a.X + b.X) / 2,
		Y:     (a.Y + b.Y) / 2,
		Z:     0,
		Angle: 0,
	}
}

func clamp(val, min, max float64) float64 {
	return math.Max(min, math.Min(max, val))
}
