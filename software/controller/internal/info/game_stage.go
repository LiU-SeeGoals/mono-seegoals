package info

import "fmt"

type GameStage int

const (
	// The first half is about to start.
	// A kickoff is called within this stage.
	// This stage ends with the NORMAL_START.
	NORMAL_FIRST_HALF_PRE GameStage = iota
	// The first half of the normal game, before half time.
	NORMAL_FIRST_HALF
	// Half time between first and second halves.
	NORMAL_HALF_TIME
	// The second half is about to start.
	// A kickoff is called within this stage.
	// This stage ends with the NORMAL_START.
	NORMAL_SECOND_HALF_PRE
	// The second half of the normal game, after half time.
	NORMAL_SECOND_HALF
	// The break before extra time.
	EXTRA_TIME_BREAK
	// The first half of extra time is about to start.
	// A kickoff is called within this stage.
	// This stage ends with the NORMAL_START.
	EXTRA_FIRST_HALF_PRE
	// The first half of extra time.
	EXTRA_FIRST_HALF
	// Half time between first and second extra halves.
	EXTRA_HALF_TIME
	// The second half of extra time is about to start.
	// A kickoff is called within this stage.
	// This stage ends with the NORMAL_START.
	EXTRA_SECOND_HALF_PRE
	// The second half of extra time.
	EXTRA_SECOND_HALF
	// The break before penalty shootout.
	PENALTY_SHOOTOUT_BREAK
	// The penalty shootout.
	PENALTY_SHOOTOUT
	// The game is over.
	POST_GAME
)

// String method for GameStage (this is a function on the enum type)
func (gs GameStage) String() string {
	switch gs {
	case NORMAL_FIRST_HALF_PRE:
		return "Normal First Half Pre"
	case NORMAL_FIRST_HALF:
		return "Normal First Half"
	case NORMAL_HALF_TIME:
		return "Normal Half Time"
	case NORMAL_SECOND_HALF_PRE:
		return "Normal Second Half Pre"
	case NORMAL_SECOND_HALF:
		return "Normal Second Half"
	case EXTRA_TIME_BREAK:
		return "Extra Time Break"
	case EXTRA_FIRST_HALF_PRE:
		return "Extra First Half Pre"
	case EXTRA_FIRST_HALF:
		return "Extra First Half"
	case EXTRA_HALF_TIME:
		return "Extra Half Time"
	case EXTRA_SECOND_HALF_PRE:
		return "Extra Second Half Pre"
	case EXTRA_SECOND_HALF:
		return "Extra Second Half"
	case PENALTY_SHOOTOUT_BREAK:
		return "Penalty Shootout Break"
	case PENALTY_SHOOTOUT:
		return "Penalty Shootout"
	case POST_GAME:
		return "Post Game"
	default:
		return fmt.Sprintf("Unknown GameStage (%d)", gs)
	}
}
