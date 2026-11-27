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
	UNINITIALIZED
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

// GameEvent contains information about the current game state and referee commands
type GameEvent struct {
	RefCommand                 RefCommand
	CurrentState               RefState
	PreviousState              RefState
	TeamWithPossession         Team
	CommandTimestamp           uint64
	LastUniqueCommandTimestamp uint64
	LastUniqueCommand          RefCommand
	// The coordinates of the Designated Position for ball placement
	DesignatedPosition              *mat.VecDense
	NextCommand                     RefCommand
	CurrentActionTimeRemaining      int64
	CurrentActionTimeRemainingValid bool
	BallInPlay                      bool
	// RestartKicked is set when a kickoff, free kick or penalty was taken
	// (the ball moved BallInPlayDistance), as opposed to timing out.
	RestartKicked bool

	// Restart bookkeeping for ObserveRestart.
	restartPending  bool // a restart was issued and waits for its first observation
	restartActive   bool
	restartStart    time.Time
	restartAnchor   Position // ball position when the restart was issued
	restartAnchored bool
}

// BallInPlayDistance is how far the ball must move after a kickoff, free kick
// or penalty is issued before it is in play (SSL rules: 0.05 m).
const BallInPlayDistance = 50.0

// RestartTimeLimit is how long a team may take to execute a restart in state
// when the game controller sends no action timer.
func RestartTimeLimit(state RefState, division Division) time.Duration {
	switch state {
	case STATE_FREE_KICK:
		if division == DivisionA {
			return 5 * time.Second
		}
		return 10 * time.Second
	default: // kickoff and penalty
		return 10 * time.Second
	}
}

func NewGameEvent() *GameEvent {
	return &GameEvent{
		RefCommand:        UNINITIALIZED,
		LastUniqueCommand: UNINITIALIZED,
		NextCommand:       UNINITIALIZED,
		CurrentState:      STATE_HALTED,
		PreviousState:     STATE_HALTED,
		// TeamWithPossession will have its zero value
		DesignatedPosition: nil,
		BallInPlay:         false,
	}
}

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

// FreeKickTeam returns the team awarded a direct or legacy indirect free kick.
func (rc RefCommand) FreeKickTeam() (Team, bool) {
	switch rc {
	case DIRECT_FREE_BLUE, INDIRECT_FREE_BLUE:
		return Blue, true
	case DIRECT_FREE_YELLOW, INDIRECT_FREE_YELLOW:
		return Yellow, true
	default:
		return Blue, false
	}
}

// KickoffTeam returns the team awarded a kickoff preparation.
func (rc RefCommand) KickoffTeam() (Team, bool) {
	switch rc {
	case PREPARE_KICKOFF_BLUE:
		return Blue, true
	case PREPARE_KICKOFF_YELLOW:
		return Yellow, true
	default:
		return Blue, false
	}
}

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
			"  Current Action Time Remaining: %s",
		ge.RefCommand.String(),
		ge.CurrentState.String(),
		teamPossession,
		ge.CommandTimestamp,
		ge.LastUniqueCommand.String(),
		elapsedTime,
		position,
		ge.NextCommand.String(),
		ge.BallInPlay,
		ge.currentActionTimeRemainingString(),
	)
}

func (ge *GameEvent) UpdateFromRefCommand(
	refCommand RefCommand,
	commandTimestamp uint64,
	desPosX float64,
	desPosY float64,
	nextCommand RefCommand,
	currentActionTimeRemaining int64,
	currentActionTimeRemainingValid bool) {
	ge.UpdateFromRefCommandWithDesignatedPosition(
		refCommand,
		commandTimestamp,
		desPosX,
		desPosY,
		true,
		nextCommand,
		currentActionTimeRemaining,
		currentActionTimeRemainingValid,
	)
}

func (ge *GameEvent) UpdateFromRefCommandWithDesignatedPosition(
	refCommand RefCommand,
	commandTimestamp uint64,
	desPosX float64,
	desPosY float64,
	hasDesignatedPosition bool,
	nextCommand RefCommand,
	currentActionTimeRemaining int64,
	currentActionTimeRemainingValid bool) {
	newCommand := refCommand != ge.RefCommand || commandTimestamp != ge.CommandTimestamp

	ge.RefCommand = refCommand
	ge.CommandTimestamp = commandTimestamp
	ge.CurrentActionTimeRemaining = currentActionTimeRemaining
	ge.CurrentActionTimeRemainingValid = currentActionTimeRemainingValid
	if nextCommand != UNINITIALIZED {
		ge.NextCommand = nextCommand
	} else if newCommand || (refCommand != STOP && !isBallPlacementRefCommand(refCommand)) {
		ge.NextCommand = UNINITIALIZED
	}

	if refCommand != ge.LastUniqueCommand {
		ge.LastUniqueCommand = refCommand
		ge.LastUniqueCommandTimestamp = commandTimestamp
	}

	if hasDesignatedPosition {
		if ge.DesignatedPosition == nil {
			ge.DesignatedPosition = mat.NewVecDense(2, nil)
		}
		ge.DesignatedPosition.SetVec(0, desPosX)
		ge.DesignatedPosition.SetVec(1, desPosY)
	} else {
		ge.DesignatedPosition = nil
	}

	if newCommand {
		ge.PreviousState = ge.CurrentState
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
			// Keep the prepared restart state. The ball only enters play after it
			// moved 50mm or the action timer expires.

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
	}

	_, freeKickCommand := refCommand.FreeKickTeam()
	if newCommand {
		ge.restartPending = !ge.BallInPlay && (freeKickCommand ||
			(refCommand == NORMAL_START && (ge.CurrentState == STATE_KICKOFF_PREPARATION ||
				ge.CurrentState == STATE_PENALTY_PREPARATION)))
		ge.restartActive = false
		ge.restartAnchored = false
		ge.RestartKicked = false
	}

	// Preparation timers are not permission to kick. Only an issued start or
	// free-kick command can time out.
	if ge.CurrentActionTimedOut() && (freeKickCommand || refCommand == NORMAL_START) {
		ge.timeOutRestart()
	}
}

// ObserveRestart tracks a pending kickoff, free kick or penalty. Call it once
// per frame, after the frame's referee packets and ball observations; ballOK
// is false when the ball is not seen. It puts the ball in play once it moved
// BallInPlayDistance from where it lay when the restart was issued, and
// applies the restart time limit when the game controller sends no timer.
//
// This is the only place, besides referee packets, that changes the restart
// state, so every team's planner sees the same answer.
func (ge *GameEvent) ObserveRestart(ball Position, ballOK bool, now time.Time, division Division) {
	if ge.restartPending {
		ge.restartPending = false
		ge.restartActive = true
		ge.restartStart = now
	}
	if !ge.restartActive || ge.BallInPlay || !isRestartState(ge.CurrentState) {
		ge.restartActive = false
		return
	}
	if ballOK {
		if !ge.restartAnchored {
			ge.restartAnchor = ball
			ge.restartAnchored = true
		} else if ball.Dist2d(ge.restartAnchor) >= BallInPlayDistance {
			ge.RestartKicked = true
			ge.SetBallMoved()
			ge.restartActive = false
			return
		}
	}
	if !ge.CurrentActionTimeRemainingValid &&
		now.Sub(ge.restartStart) >= RestartTimeLimit(ge.CurrentState, division) {
		ge.timeOutRestart()
	}
}

// RestartBallAnchor is where the ball lay when the current restart was issued.
func (ge *GameEvent) RestartBallAnchor() (Position, bool) {
	return ge.restartAnchor, ge.restartAnchored
}

// timeOutRestart ends a restart that was not taken in time: an expired penalty
// returns to STOP, an expired kickoff or free kick continues as open play.
func (ge *GameEvent) timeOutRestart() {
	ge.restartActive = false
	switch ge.CurrentState {
	case STATE_PENALTY_PREPARATION:
		if !ge.BallInPlay {
			ge.PreviousState = ge.CurrentState
			ge.CurrentState = STATE_STOPPED
		}
	case STATE_KICKOFF_PREPARATION, STATE_FREE_KICK:
		ge.SetBallMoved()
	}
}

func isRestartState(state RefState) bool {
	return state == STATE_KICKOFF_PREPARATION || state == STATE_PENALTY_PREPARATION || state == STATE_FREE_KICK
}

func isBallPlacementRefCommand(command RefCommand) bool {
	return command == BALL_PLACEMENT_BLUE || command == BALL_PLACEMENT_YELLOW
}

func (ge *GameEvent) GetCurrentState() RefState {
	// Check for timeouts and update state if needed
	return ge.CurrentState
}

// IsPlacementStop applies to both teams while a restart is being prepared.
func (ge *GameEvent) IsPlacementStop() bool {
	return ge != nil && (ge.CurrentState == STATE_STOPPED || ge.CurrentState == STATE_BALL_PLACEMENT)
}

func (ge *GameEvent) GetPreviousState() RefState {
	// Check for timeouts and update state if needed
	return ge.PreviousState
}

func (ge *GameEvent) GetTeamWithPossession() Team {
	return ge.TeamWithPossession
}

func (ge *GameEvent) GetDesignatedPosition() *mat.VecDense {
	return ge.DesignatedPosition
}

func (ge *GameEvent) HasCurrentActionTimeRemaining() bool {
	return ge.CurrentActionTimeRemainingValid
}

func (ge *GameEvent) CurrentActionTimedOut() bool {
	return ge.CurrentActionTimeRemainingValid && ge.CurrentActionTimeRemaining <= 0
}

func (ge *GameEvent) currentActionTimeRemainingString() string {
	if !ge.CurrentActionTimeRemainingValid {
		return "N/A"
	}
	return fmt.Sprintf("%d microseconds", ge.CurrentActionTimeRemaining)
}

// SetBallMoved puts a pending restart in play. Normally ObserveRestart calls it.
func (ge *GameEvent) SetBallMoved() {
	if isRestartState(ge.CurrentState) {
		ge.BallInPlay = true
		ge.PreviousState = ge.CurrentState
		ge.CurrentState = STATE_PLAYING
	}
}
