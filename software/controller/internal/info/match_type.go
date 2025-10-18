package info

import "fmt"

type MatchType int

const (
	// not set
	UNKNOWN_MATCH = iota
	// match is part of the group phase
	GROUP_PHASE
	// match is part of the elimination phase
	ELIMINATION_PHASE
	// a friendly match, not part of a tournament
	FRIENDLY
)

// String method for MatchType
func (mt MatchType) String() string {
	switch mt {
	case UNKNOWN_MATCH:
		return "Unknown Match"
	case GROUP_PHASE:
		return "Group Phase"
	case ELIMINATION_PHASE:
		return "Elimination Phase"
	case FRIENDLY:
		return "Friendly"
	default:
		return fmt.Sprintf("Unknown MatchType (%d)", mt)
	}
}
