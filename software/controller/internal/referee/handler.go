// Package referee turns game-controller commands into robot activities for
// the restarts a team must execute: HALT, timeout, ball placement, kickoff,
// free kick and penalty. STOP and normal play are left to the team's planner.
//
// What the rules allow (ball distance, who may touch the ball, whether a
// restart is in play) is not decided here: it is read from info.GameEvent via
// rules.MatchRules, which every team shares. The handler only keeps per-team
// execution state (selected kicker, post-kick heading hold).
//
// Files: handler.go (state machine and dispatch), halt_stop.go, kickoff.go,
// free_kick.go, penalty.go, restart_common.go (shared formation helpers).
package referee

import (
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Events of the referee state machine: one per game-controller command, plus
// GAME_RUNNING_DETECTED when a taken restart has handed over to normal play.
const (
	HALT                  state_machine.EventName = "Halt"
	STOP                  state_machine.EventName = "Stop"
	NORMAL_START          state_machine.EventName = "NormalStart"
	FORCE_START           state_machine.EventName = "ForceStart"
	PREPARE_KICKOFF       state_machine.EventName = "PrepareKickoff"
	FREE_KICK             state_machine.EventName = "FreeKick"
	BALL_PLACEMENT        state_machine.EventName = "BallPlacement"
	PREPARE_PENALTY       state_machine.EventName = "PreparePenalty"
	TIMEOUT               state_machine.EventName = "Timeout"
	GAME_RUNNING_DETECTED state_machine.EventName = "Running"
)

// States of the referee state machine.
const (
	StateUninitialized  state_machine.StateName = "UNINITIALIZED"
	StateHalt           state_machine.StateName = "HALT"
	StateTimeout        state_machine.StateName = "TIMEOUT"
	StateStop           state_machine.StateName = "STOP"
	StateBallPlacement  state_machine.StateName = "BALLPLACEMENT"
	StatePrepareKickoff state_machine.StateName = "PREPAREKICKOFF"
	StateKickoff        state_machine.StateName = "KICKOFF"
	StateFreeKick       state_machine.StateName = "FREEKICK"
	StatePreparePenalty state_machine.StateName = "PREPAREPENALTY"
	StatePenalty        state_machine.StateName = "PENALTY"
	StateRunning        state_machine.StateName = "RUNNING"
)

// refereeContext is shared by every state of one handler.
type refereeContext struct {
	gi              *info.GameInfo
	activeRobots    []info.ID
	team            info.Team
	activityHandler *ai.ActivityHandler
}

// RefereeInfo is embedded by every referee state.
type RefereeInfo struct {
	*refereeContext
	name state_machine.StateName
}

func (s *RefereeInfo) GetName() state_machine.StateName {
	return s.name
}

// plannerOwned is a state in which the handler emits no activities. In STOP
// and RUNNING the team's planner controls the robots; before the first
// command nobody does.
type plannerOwned struct {
	name state_machine.StateName
}

func (s *plannerOwned) Initialize()                      {}
func (s *plannerOwned) GetName() state_machine.StateName { return s.name }
func (s *plannerOwned) Update() state_machine.EventName  { return state_machine.NoEvent }
func (s *RefereeHandler) plannerOwnsFrame() bool {
	return s.isPlannerOwned(s.refereeSM.CurrentStateName())
}
func (s *RefereeHandler) isPlannerOwned(n state_machine.StateName) bool {
	return n == StateStop || n == StateRunning
}

type RefereeHandler struct {
	ctx                           *refereeContext
	refereeSM                     *state_machine.StateMachine
	nextCommandAfterBallPlacement info.RefCommand
	kickOff                       *Kickoff
	freeKick                      *FreeKick
	prepareKickoff                *PrepareKickoff
	preparePenalty                *PreparePenalty
	penalty                       *Penalty
	kickoffTouchRestriction       kickoffTouchRestriction
	lastCommand                   info.RefCommand
	lastCommandTimestamp          uint64
	commandSeen                   bool
}

func isBallPlacementCommand(rc info.RefCommand) bool {
	return rc == info.BALL_PLACEMENT_BLUE || rc == info.BALL_PLACEMENT_YELLOW
}

func refCommandToEventName(rc info.RefCommand) state_machine.EventName {
	switch rc {
	case info.HALT:
		return HALT
	case info.STOP:
		return STOP
	case info.NORMAL_START:
		return NORMAL_START
	case info.FORCE_START:
		return FORCE_START
	case info.PREPARE_KICKOFF_BLUE, info.PREPARE_KICKOFF_YELLOW:
		return PREPARE_KICKOFF
	case info.PREPARE_PENALTY_YELLOW, info.PREPARE_PENALTY_BLUE:
		return PREPARE_PENALTY
	case info.DIRECT_FREE_YELLOW, info.DIRECT_FREE_BLUE, info.INDIRECT_FREE_YELLOW, info.INDIRECT_FREE_BLUE:
		return FREE_KICK
	case info.TIMEOUT_YELLOW, info.TIMEOUT_BLUE:
		return TIMEOUT
	case info.BALL_PLACEMENT_BLUE, info.BALL_PLACEMENT_YELLOW:
		return BALL_PLACEMENT
	default:
		return state_machine.NoEvent
	}
}

func NewRefereeHandler(gi *info.GameInfo, activeRobots []info.ID, team info.Team, activityHandler *ai.ActivityHandler) *RefereeHandler {
	ctx := &refereeContext{
		gi:              gi,
		activeRobots:    append([]info.ID(nil), activeRobots...),
		team:            team,
		activityHandler: activityHandler,
	}
	base := func(name state_machine.StateName) RefereeInfo {
		return RefereeInfo{refereeContext: ctx, name: name}
	}

	uninitialized := &plannerOwned{name: StateUninitialized}
	running := &plannerOwned{name: StateRunning}
	stop := &plannerOwned{name: StateStop}
	halt := &Halt{RefereeInfo: base(StateHalt)}
	timeout := &Timeout{Halt{RefereeInfo: base(StateTimeout)}}
	ballPlacement := &BallPlacement{RefereeInfo: base(StateBallPlacement)}
	prepareKickoff := &PrepareKickoff{RefereeInfo: base(StatePrepareKickoff)}
	kickOff := &Kickoff{RefereeInfo: base(StateKickoff)}
	freeKick := &FreeKick{RefereeInfo: base(StateFreeKick)}
	preparePenalty := &PreparePenalty{RefereeInfo: base(StatePreparePenalty)}
	penalty := &Penalty{RefereeInfo: base(StatePenalty)}

	sm := state_machine.NewStateMachine(uninitialized)

	// A multicast receiver can miss an intermediate STOP, so every command is
	// honoured from every state. HandleReferee deduplicates repeated packets.
	commandTargets := map[state_machine.EventName]state_machine.State{
		STOP:            stop,
		HALT:            halt,
		TIMEOUT:         timeout,
		FORCE_START:     running,
		FREE_KICK:       freeKick,
		BALL_PLACEMENT:  ballPlacement,
		PREPARE_KICKOFF: prepareKickoff,
		PREPARE_PENALTY: preparePenalty,
	}
	allStates := []state_machine.StateName{StateUninitialized, StateHalt, StateStop, StateTimeout,
		StateBallPlacement, StatePrepareKickoff, StatePreparePenalty, StateKickoff, StatePenalty,
		StateFreeKick, StateRunning}
	for _, from := range allStates {
		for event, to := range commandTargets {
			sm.AddTransition(from, event, to)
		}
	}

	// NORMAL_START executes the prepared restart.
	sm.AddTransition(StatePrepareKickoff, NORMAL_START, kickOff)
	sm.AddTransition(StatePreparePenalty, NORMAL_START, penalty)
	// Started without having seen the preparation: assume normal play.
	sm.AddTransition(StateUninitialized, NORMAL_START, running)
	// A taken (or timed-out) restart hands over to normal play.
	sm.AddTransition(StateKickoff, GAME_RUNNING_DETECTED, running)
	sm.AddTransition(StateFreeKick, GAME_RUNNING_DETECTED, running)

	return &RefereeHandler{
		ctx:            ctx,
		refereeSM:      sm,
		kickOff:        kickOff,
		freeKick:       freeKick,
		prepareKickoff: prepareKickoff,
		preparePenalty: preparePenalty,
		penalty:        penalty,
	}
}

// UpdateActiveRobots keeps every referee state synchronized with current
// vision membership. A substituted-out robot must not retain its old activity,
// while its replacement must be eligible for goalkeeper selection immediately.
func (s *RefereeHandler) UpdateActiveRobots(activeRobots []info.ID) {
	for _, oldID := range s.ctx.activeRobots {
		if robotIDInList(activeRobots, oldID) {
			continue
		}
		if s.ctx.activityHandler != nil {
			s.ctx.activityHandler.ClearActivity(oldID)
		}
	}
	s.ctx.activeRobots = append(s.ctx.activeRobots[:0], activeRobots...)
}

// HandleReferee runs one frame. It returns true when the referee handler
// produced this frame's activities and false when the planner should run
// (STOP and normal play). Rules to follow are defined in the SSL rules,
// Appendix B: Game States https://robocup-ssl.github.io/ssl-rules/sslrules.html
func (s *RefereeHandler) HandleReferee() bool {
	gameEvent := s.ctx.gi.Status.GetGameEvent()
	if !s.commandSeen || s.lastCommand != gameEvent.RefCommand ||
		s.lastCommandTimestamp != gameEvent.CommandTimestamp {
		// A new command of the same kind (a second free kick) restarts the state.
		s.refereeSM.TriggerEventReentering(s.refEventForGameEvent(gameEvent))
		s.lastCommand = gameEvent.RefCommand
		s.lastCommandTimestamp = gameEvent.CommandTimestamp
		s.commandSeen = true
	}

	stateBeforeUpdate := s.refereeSM.CurrentStateName()
	if stateBeforeUpdate != StateRunning && stateBeforeUpdate != StateKickoff {
		s.kickoffTouchRestriction.clear()
	}

	s.refereeSM.Update()
	stateAfterUpdate := s.refereeSM.CurrentStateName()
	if stateBeforeUpdate == StateKickoff && stateAfterUpdate == StateRunning &&
		s.kickOff.kickTaken && gameEvent.TeamWithPossession == s.ctx.team {
		s.kickoffTouchRestriction.arm(s.ctx.team, s.kickOff.kickOffID)
	}
	if stateBeforeUpdate == StateFreeKick && stateAfterUpdate == StateRunning &&
		s.freeKick.freeKick != nil && s.freeKick.kickTaken &&
		gameEvent.TeamWithPossession == s.ctx.team {
		s.kickoffTouchRestriction.arm(s.ctx.team, s.freeKick.kickerID)
	}
	if stateAfterUpdate == StateRunning {
		s.kickoffTouchRestriction.update(s.ctx.gi)
	}
	return !s.plannerOwnsFrame()
}

// KickoffRestrictedRobot returns the restart kicker that may not touch the ball
// again until another robot has touched it. It applies to kickoffs and free kicks.
func (s *RefereeHandler) KickoffRestrictedRobot() (info.ID, bool) {
	return s.kickoffTouchRestriction.robot()
}

// RestartKicker returns the robot taking our pending kickoff, free kick or
// penalty. Until the ball is in play no other robot of ours may touch it.
func (s *RefereeHandler) RestartKicker() (info.ID, bool) {
	gameEvent := s.ctx.gi.Status.GetGameEvent()
	if gameEvent == nil || gameEvent.BallInPlay || gameEvent.TeamWithPossession != s.ctx.team {
		return 0, false
	}
	switch s.refereeSM.CurrentStateName() {
	case StatePrepareKickoff:
		return s.prepareKickoff.kickOffID, true
	case StateKickoff:
		return s.kickOff.kickOffID, s.kickOff.kickOff != nil
	case StateFreeKick:
		return s.freeKick.kickerID, s.freeKick.freeKick != nil
	case StatePreparePenalty:
		return s.preparePenalty.kickerID, s.preparePenalty.hasKicker
	case StatePenalty:
		return s.penalty.kickerID, s.penalty.penaltyKick != nil
	}
	return 0, false
}

func (s *RefereeHandler) refEventForGameEvent(gameEvent *info.GameEvent) state_machine.EventName {
	refereeCommand := gameEvent.RefCommand
	if isBallPlacementCommand(refereeCommand) && gameEvent.NextCommand != info.UNINITIALIZED {
		s.nextCommandAfterBallPlacement = gameEvent.NextCommand
	}

	if s.refereeSM.CurrentStateName() == StateBallPlacement &&
		s.nextCommandAfterBallPlacement != info.UNINITIALIZED &&
		refereeCommand == s.nextCommandAfterBallPlacement {
		return refCommandToEventName(s.nextCommandAfterBallPlacement)
	}

	return refCommandToEventName(refereeCommand)
}
