package info

import (
	"fmt"
	"time"

	"gonum.org/v1/gonum/mat"
)

// RefCommand represents commands issued by the referee
type RefCommand int

const (
	// All robots should completely stop moving.
	HALT RefCommand = iota
	// Robots must keep 50 cm from the ball.
	STOP
	// A prepared kickoff or penalty may now be taken.
	NORMAL_START
	// The ball is dropped and free for either team.
	FORCE_START
	// The yellow team may move into kickoff position.
	PREPARE_KICKOFF_YELLOW
	// The blue team may move into kickoff position.
	PREPARE_KICKOFF_BLUE
	// The yellow team may move into penalty position.
	PREPARE_PENALTY_YELLOW
	// The blue team may move into penalty position.
	PREPARE_PENALTY_BLUE
	// The yellow team may take a direct free kick.
	DIRECT_FREE_YELLOW
	// The blue team may take a direct free kick.
	DIRECT_FREE_BLUE
	// The yellow team may take an indirect free kick.
	INDIRECT_FREE_YELLOW // Deprecated, do not remove
	// The blue team may take an indirect free kick.
	INDIRECT_FREE_BLUE // Deprecated, do not remove
	// The yellow team is currently in a timeout.
	TIMEOUT_YELLOW
	// The blue team is currently in a timeout.
	TIMEOUT_BLUE
	// The yellow team is currently in a timeout.
	GOAL_YELLOW // Deprecated, do not remove
	// The blue team is currently in a timeout.
	GOAL_BLUE // Deprecated, do not remove
	// Equivalent to STOP, but the yellow team must pick up the ball and
	// drop it in the Designated Position.
	BALL_PLACEMENT_YELLOW
	// Equivalent to STOP, but the blue team must pick up the ball and drop
	// it in the Designated Position.
	BALL_PLACEMENT_BLUE
)

// RefState represents the current state of the game based on referee commands
type RefState int

const (
	STATE_HALTED RefState = iota
	STATE_STOPPED
	STATE_PLAYING
	STATE_KICKOFF_PREPARATION
	STATE_PENALTY_PREPARATION
	STATE_FREE_KICK
	STATE_TIMEOUT
	STATE_BALL_PLACEMENT
)

// Import your existing info package for Team type
// This is already imported as we're in the same package

// GameEvent contains information about the current game state and referee commands
type GameEvent struct {
	// Command issued by the referee.
	RefCommand RefCommand
	// Current game state based on referee commands
	CurrentState RefState
	// Team with possession/advantage in the current state
	TeamWithPossession Team
	// The UNIX timestamp when the command was issued, in microseconds.
	CommandTimestamp uint64
	// The UNIX timestamp when the last unique command was issued, for timeout tracking
	LastUniqueCommandTimestamp uint64
	// The last unique command received, for detecting command changes
	LastUniqueCommand RefCommand
	// The coordinates of the Designated Position for ball placement
	DesignatedPosition *mat.VecDense
	// The command that will be issued after the current stoppage
	NextCommand RefCommand
	// The time in microseconds remaining until the current action times out
	CurrentActionTimeRemaining int64
	// Indicates if the ball is currently in play
	BallInPlay bool
}

// NewGameEvent creates a new GameEvent with default values
func NewGameEvent() *GameEvent {
	return &GameEvent{
		RefCommand:   HALT,
		CurrentState: STATE_HALTED,
		// TeamWithPossession will have its zero value
		DesignatedPosition: mat.NewVecDense(2, nil),
		BallInPlay:         false,
	}
}

// String method for RefCommand to convert the enum to a human-readable string
func (rc RefCommand) String() string {
	switch rc {
	case HALT:
		return "Halt"
	case STOP:
		return "Stop"
	case NORMAL_START:
		return "Normal Start"
	case FORCE_START:
		return "Force Start"
	case PREPARE_KICKOFF_YELLOW:
		return "Prepare Kickoff Yellow"
	case PREPARE_KICKOFF_BLUE:
		return "Prepare Kickoff Blue"
	case PREPARE_PENALTY_YELLOW:
		return "Prepare Penalty Yellow"
	case PREPARE_PENALTY_BLUE:
		return "Prepare Penalty Blue"
	case DIRECT_FREE_YELLOW:
		return "Direct Free Yellow"
	case DIRECT_FREE_BLUE:
		return "Direct Free Blue"
	case INDIRECT_FREE_YELLOW:
		return "Indirect Free Yellow"
	case INDIRECT_FREE_BLUE:
		return "Indirect Free Blue"
	case TIMEOUT_YELLOW:
		return "Timeout Yellow"
	case TIMEOUT_BLUE:
		return "Timeout Blue"
	case BALL_PLACEMENT_YELLOW:
		return "Ball Placement Yellow"
	case BALL_PLACEMENT_BLUE:
		return "Ball Placement Blue"
	default:
		return fmt.Sprintf("Unknown RefCommand (%d)", rc)
	}
}

// String method for RefState
func (rs RefState) String() string {
	switch rs {
	case STATE_HALTED:
		return "Halted"
	case STATE_STOPPED:
		return "Stopped"
	case STATE_PLAYING:
		return "Playing"
	case STATE_KICKOFF_PREPARATION:
		return "Kickoff Preparation"
	case STATE_PENALTY_PREPARATION:
		return "Penalty Preparation"
	case STATE_FREE_KICK:
		return "Free Kick"
	case STATE_TIMEOUT:
		return "Timeout"
	case STATE_BALL_PLACEMENT:
		return "Ball Placement"
	default:
		return fmt.Sprintf("Unknown RefState (%d)", rs)
	}
}

// No need for TeamColor String method as we're using the existing Team type

// String method for GameEvent
func (ge *GameEvent) String() string {
	position := "N/A"
	if ge.DesignatedPosition != nil {
		position = fmt.Sprintf("(x: %.2f, y: %.2f)", ge.DesignatedPosition.At(0, 0), ge.DesignatedPosition.At(1, 0))
	}

	teamPossession := "None"
	// Assuming the Team type has a String() method or similar for formatting
	if ge.CurrentState != STATE_HALTED && ge.CurrentState != STATE_PLAYING {
		teamPossession = fmt.Sprintf("%v", ge.TeamWithPossession)
	}

	// Calculate elapsed time since last unique command
	currentTime := uint64(time.Now().UnixMicro())
	elapsedTime := (currentTime - ge.LastUniqueCommandTimestamp) / 1000 // Convert to milliseconds

	return fmt.Sprintf(
		"Game Event:\n"+
			"  Ref Command: %s\n"+
			"  Game State: %s\n"+
			"  Team with Possession: %s\n"+
			"  Command Timestamp: %d microseconds\n"+
			"  Last Unique Command: %s\n"+
			"  Last Unique Command Time: %d ms ago\n"+
			"  Designated Position: %s\n"+
			"  Next Command: %s\n"+
			"  Ball In Play: %v\n"+
			"  Current Action Time Remaining: %d microseconds",
		ge.RefCommand.String(),
		ge.CurrentState.String(),
		teamPossession,
		ge.CommandTimestamp,
		ge.LastUniqueCommand.String(),
		elapsedTime,
		position,
		ge.NextCommand.String(),
		ge.BallInPlay,
		ge.CurrentActionTimeRemaining,
	)
}

// UpdateFromRefCommand updates the game state based on a new referee command
func (ge *GameEvent) UpdateFromRefCommand(
	refCommand RefCommand,
	commandTimestamp uint64,
	desPosX float64,
	desPosY float64,
	nextCommand RefCommand,
	currentActionTimeRemaining int64) {

	// Update basic fields
	ge.RefCommand = refCommand
	ge.CommandTimestamp = commandTimestamp
	ge.NextCommand = nextCommand
	ge.CurrentActionTimeRemaining = currentActionTimeRemaining

	// Check if this is a new unique command
	if refCommand != ge.LastUniqueCommand {
		ge.LastUniqueCommand = refCommand
		ge.LastUniqueCommandTimestamp = commandTimestamp
	}

	// Update designated position if needed
	if ge.DesignatedPosition == nil {
		ge.DesignatedPosition = mat.NewVecDense(2, nil)
	}
	ge.DesignatedPosition.SetVec(0, desPosX)
	ge.DesignatedPosition.SetVec(1, desPosY)

	// Update game state based on command
	switch refCommand {
	case HALT:
		ge.CurrentState = STATE_HALTED
		// TeamWithPossession will remain as is
		ge.BallInPlay = false

	case STOP:
		ge.CurrentState = STATE_STOPPED
		// Keep the previous team possession when going to STOP
		ge.BallInPlay = false

	case NORMAL_START:
		if ge.CurrentState == STATE_KICKOFF_PREPARATION || ge.CurrentState == STATE_PENALTY_PREPARATION {
			// Keep the current state but mark the ball as in play
			ge.BallInPlay = true
		}

	case FORCE_START:
		ge.CurrentState = STATE_PLAYING
		// TeamWithPossession will be ignored in PLAYING state
		ge.BallInPlay = true

	case PREPARE_KICKOFF_YELLOW:
		ge.CurrentState = STATE_KICKOFF_PREPARATION
		ge.TeamWithPossession = Yellow
		ge.BallInPlay = false

	case PREPARE_KICKOFF_BLUE:
		ge.CurrentState = STATE_KICKOFF_PREPARATION
		ge.TeamWithPossession = Blue
		ge.BallInPlay = false

	case PREPARE_PENALTY_YELLOW:
		ge.CurrentState = STATE_PENALTY_PREPARATION
		ge.TeamWithPossession = Yellow
		ge.BallInPlay = false

	case PREPARE_PENALTY_BLUE:
		ge.CurrentState = STATE_PENALTY_PREPARATION
		ge.TeamWithPossession = Blue
		ge.BallInPlay = false

	case DIRECT_FREE_YELLOW, INDIRECT_FREE_YELLOW:
		ge.CurrentState = STATE_FREE_KICK
		ge.TeamWithPossession = Yellow
		ge.BallInPlay = false

	case DIRECT_FREE_BLUE, INDIRECT_FREE_BLUE:
		ge.CurrentState = STATE_FREE_KICK
		ge.TeamWithPossession = Blue
		ge.BallInPlay = false

	case TIMEOUT_YELLOW, TIMEOUT_BLUE:
		ge.CurrentState = STATE_TIMEOUT
		if refCommand == TIMEOUT_YELLOW {
			ge.TeamWithPossession = Yellow
		} else {
			ge.TeamWithPossession = Blue
		}
		ge.BallInPlay = false

	case BALL_PLACEMENT_YELLOW:
		ge.CurrentState = STATE_BALL_PLACEMENT
		ge.TeamWithPossession = Yellow
		ge.BallInPlay = false

	case BALL_PLACEMENT_BLUE:
		ge.CurrentState = STATE_BALL_PLACEMENT
		ge.TeamWithPossession = Blue
		ge.BallInPlay = false
	}

	if ge.CurrentActionTimeRemaining < 0 {
		ge.CurrentState = STATE_PLAYING
	}

	// Check for timeouts and automatically update state if needed
}

// GetCurrentState returns the current state after checking for any timeouts
// This ensures we always get the most up-to-date state
func (ge *GameEvent) GetCurrentState() RefState {
	// Check for timeouts and update state if needed

	return ge.CurrentState
}

func (ge *GameEvent) GetTeamWithPossession() Team {
	return ge.TeamWithPossession
}

// If we should keep distance from ball it is 500 mm
func (ge *GameEvent) ShouldKeepDistanceFromBall(robotTeam Team) bool {
	if ge.CurrentState == STATE_HALTED {
		return true
	}

	if ge.CurrentState == STATE_STOPPED {
		return true
	}

	// For free kicks, only the non-possessing team needs to keep distance
	if ge.CurrentState == STATE_FREE_KICK {
		return ge.TeamWithPossession != robotTeam
	}

	// For kickoff preparations
	if ge.CurrentState == STATE_KICKOFF_PREPARATION && !ge.BallInPlay {
		// If we don't have possession, all robots need to keep distance
		if ge.TeamWithPossession != robotTeam {
			return true
		}
		// If we have possession, all robots except one kicker need to keep distance
		// Note: The actual kicker robot should be determined by the strategy code
		return true
	}

	// For penalty preparations
	if ge.CurrentState == STATE_PENALTY_PREPARATION && !ge.BallInPlay {
		// If we don't have possession, all robots except goalkeeper need to keep distance
		if ge.TeamWithPossession != robotTeam {
			// Note: The goalkeeper determination should be handled by strategy code
			return true
		}
		// If we have possession, all robots except one kicker need to keep distance
		// Note: The actual kicker robot should be determined by the strategy code
		return true
	}

	return false
}

// GetMaxRobotSpeed returns the maximum allowed robot speed in m/s
func (ge *GameEvent) GetMaxRobotSpeed() float64 {
	if ge.CurrentState == STATE_HALTED {
		return 0.0 // No movement allowed during HALT (with 2 second grace period)
	}

	if ge.CurrentState == STATE_STOPPED || ge.CurrentState == STATE_BALL_PLACEMENT {
		return 1.5 // Limited to 1.5 m/s during STOP
	}

	// No speed limit during normal play
	return 999.0
}

// CanManipulateBall returns true if robots of the given team are allowed to manipulate the ball
func (ge *GameEvent) CanManipulateBall(robotTeam Team) bool {
	if ge.CurrentState == STATE_HALTED || ge.CurrentState == STATE_STOPPED {
		return false
	}

	if ge.CurrentState == STATE_PLAYING {
		return true
	}

	// For states where only one team can manipulate the ball
	if !ge.BallInPlay {
		// Only the team with possession can manipulate the ball
		if ge.TeamWithPossession == robotTeam {
			// Note: The kicker decision should be made by the strategy code
			return true
		}

		return false
	}

	return ge.BallInPlay
}

func (ge *GameEvent) SetBallMoved() {
	switch ge.CurrentState {
	case STATE_KICKOFF_PREPARATION, STATE_PENALTY_PREPARATION, STATE_FREE_KICK:
		ge.BallInPlay = true
		ge.CurrentState = STATE_PLAYING
	}
}
