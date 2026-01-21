package ai

import (
	"fmt"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Constants for goalie positioning
const (
	// Goalie position constraints - these will be adjusted based on team half
	GOALIE_LINE_WIDTH = 1000 // Width of the goalie's movement range (500 to each side)
	// GOALIE_DIST_FROM_CENTER = 5500 // Distance from center to goalie line
	GOALIE_DIST_FROM_CENTER = 2300                           // Distance from center to goalie line
	GOAL_BEHIND_DIST        = GOALIE_DIST_FROM_CENTER + 1200 // Distance from center to position behind the goal
)

type Goalie struct {
	GenericComposition
	team info.Team
	id   info.ID
	Activity
}

func (g *Goalie) String() string {
	return fmt.Sprintf("Goalie(%d, %d)", g.team, g.id)
}

// NewGoalie creates a new Goalie struct.
func NewGoalie(team info.Team, id info.ID) *Goalie {
	return &Goalie{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		team: team,
		id:   id,
	}
}

func (g *Goalie) GetAction(gi *info.GameInfo) action.Action {
	ball := gi.State.GetBall()

	// Current ball position
	ballPos, err := ball.GetEstimatedPosition()
	if err != nil {
		fmt.Println("Error getting ball position:", err)
		return NewMoveToPosition(g.team, g.id, info.Position{X: 0, Y: 0}).GetAction(gi)
	}

	// Determine which half we're defending
	isBlueTeam := g.team == info.Blue
	isBlueOnPositiveHalf := gi.Status.GetBlueTeamOnPositiveHalf()
	isDefendingPositiveHalf := (isBlueTeam && isBlueOnPositiveHalf) || (!isBlueTeam && !isBlueOnPositiveHalf)

	xMultiplier := -1.0
	if !isDefendingPositiveHalf {
		xMultiplier = 1.0
	}
	goalieX := xMultiplier * GOALIE_DIST_FROM_CENTER
	goalSize := 1110.0
	goalieY := ballPos.Y

	if goalieY > goalSize {
		goalieY = goalSize
	} else if goalieY < -goalSize {
		goalieY = -goalSize
	}

	goaliePos := info.Position{X: goalieX, Y: goalieY, Z: 0.0, Angle: 0.0}

	move := NewMoveToPosition(g.team, g.id, goaliePos)
	return move.GetAction(gi)
}

// Achieved returns whether this action is "complete".
// The goalie never really finishes, so we return false unless higher-level AI changes it.
func (g *Goalie) Achieved(*info.GameInfo) bool {
	return false
}

func (m *Goalie) GetID() info.ID {
	return m.id
}
