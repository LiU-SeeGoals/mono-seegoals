package referee

import (
	"fmt"
	"math"
	"sort"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	BALL_IN_PLAY_DISTANCE_MM = 50.0

	// Restart kicks hand control to the normal planner as soon as the ball has
	// moved BALL_IN_PLAY_DISTANCE_MM. Keep the kicker on its kick heading long
	// enough for the ball and the robot's kicker/dribbler to separate before a
	// normal-play role may request a completely different orientation.
	restartPostKickMinHeadingHold  = 300 * time.Millisecond
	restartPostKickMaxHeadingHold  = 600 * time.Millisecond
	restartPostKickBallClearanceMM = 300.0

	KICKOFF_MAX_TIME_DIVISION_A  = 10 * time.Second
	KICKOFF_MAX_TIME_DIVISION_B  = 10 * time.Second
	FREEKICK_MAX_TIME_DIVISION_A = 5 * time.Second
	FREEKICK_MAX_TIME_DIVISION_B = 10 * time.Second
	PENALTY_MAX_TIME_DIVISION_A  = 10 * time.Second
	PENALTY_MAX_TIME_DIVISION_B  = 10 * time.Second
)

const (
	defaultKickoffKickerID   = info.ID(1)
	defaultKickoffReceiverID = info.ID(3)
	defaultGoalieID          = info.ID(6)

	kickoffCenterCircleClearanceMM = 700.0
	kickoffFieldMarginMM           = 300.0
	kickoffKickTargetDistanceMM    = 1000.0
	kickoffPrepareDistanceMM       = 500.0
	freeKickPrepareDistanceMM      = 550.0
	freeKickWallDistanceMM         = 700.0
	freeKickWallSpacingMM          = 300.0
	freeKickWallRobotCount         = 3
	freeKickFieldClearanceMM       = 200.0
	freeKickDefenseAreaClearanceMM = 390.0
	freeKickBallClearanceMM        = 510.0
	penaltyPrepareDistanceMM       = 700.0
	penaltyBehindBallDistanceMM    = 1200.0
)

const (
	HALT            = "Halt"
	STOP            = "Stop"
	NORMAL_START    = "NormalStart"
	FORCE_START     = "ForceStart"
	PREPARE_KICKOFF = "PrepareKickoff"
	FREE_KICK       = "FreeKick"
	BALL_PLACEMENT  = "BallPlacement"
	PREPARE_PENALTY = "PreparePenalty"
	TIMEOUT         = "Timeout"
	CONTINUE        = "Continue"
	UNKNOWN         = "Unknown"
)

const (
	GAME_RUNNING_DETECTED = "Running"
)

type RefereeInfo struct {
	gi                *info.GameInfo
	activeRobots      []info.ID
	team              info.Team
	activityHandler   *ai.ActivityHandler
	name              StateName
	teamWithPossesion info.Team
}

func (s *RefereeInfo) GetName() StateName {
	return s.name
}

type Halt struct {
	RefereeInfo
}

type FreeKick struct {
	RefereeInfo
	freeKick          *StateMachine
	kickerID          info.ID
	freeKickStart     time.Time
	originalBallPos   info.Position
	kickTaken         bool
	postKickStart     time.Time
	postKickHeading   float64
	postKickHeadingOK bool
}

type Stop struct {
	RefereeInfo
	positioningDelegated bool
}

type PrepareKickoff struct {
	RefereeInfo
	kickOffID  info.ID
	receiverID info.ID
}

type PreparePenalty struct {
	RefereeInfo
	kickerID  info.ID
	hasKicker bool
}

type Penalty struct {
	RefereeInfo
	penaltyKick       *StateMachine
	kickerID          info.ID
	ballInPlay        bool
	penaltyStart      time.Time
	originalBallPos   info.Position
	postKickHeading   float64
	postKickHeadingOK bool
}

type Kickoff struct {
	RefereeInfo
	kickOff           *StateMachine
	kickOffID         info.ID
	receiverID        info.ID
	originalBallPos   info.Position
	kickStart         time.Time
	kickTaken         bool
	postKickStart     time.Time
	postKickHeading   float64
	postKickHeadingOK bool
}

type Running struct {
}

func (s *Halt) Initialize() {
}

func (s *Halt) Update() EventName {

	for _, id := range s.activeRobots {
		activity := act.NewStop(id)
		s.activityHandler.AddActivity(activity)
	}

	return "NONE"
}

func (s *Stop) Initialize() {
}

func (s *Stop) Update() EventName {
	if s.positioningDelegated {
		return "NONE"
	}

	goalieID, hasGoalie := moveGoalieToPosition(s.activeRobots, s.team, s.gi, s.activityHandler)
	preparedKickerID, preparingFreeKick := PrepareKickerForUpcomingFreeKick(
		s.gi,
		s.team,
		s.activeRobots,
		s.activityHandler,
	)

	for _, id := range s.activeRobots {
		if hasGoalie && id == goalieID {
			continue
		}
		if preparingFreeKick && id == preparedKickerID {
			continue
		}
		activity := act.NewRefStop(s.team, id)
		s.activityHandler.AddActivity(activity)
	}

	return "NONE"
}

func (s *PreparePenalty) Initialize() {
	s.kickerID = 0
	s.hasKicker = false
	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		s.kickerID, s.hasKicker = selectPenaltyKicker(s.gi, s.team, s.activeRobots)
	}
}

func (s *PreparePenalty) Update() EventName {
	if s.gi.Status.GetGameEvent().TeamWithPossession != s.team {
		s.kickerID = 0
		s.hasKicker = false
	} else if !s.hasKicker || !robotIDInList(s.activeRobots, s.kickerID) {
		s.kickerID, s.hasKicker = selectPenaltyKicker(s.gi, s.team, s.activeRobots)
	}

	penaltyMark := kickoffBallPosition(s.gi)
	positionPenaltyNonParticipants(
		s.gi,
		s.activeRobots,
		s.team,
		s.activityHandler,
		s.kickerID,
		s.hasKicker,
		penaltyMark,
		false,
	)
	if s.hasKicker {
		move := act.NewMoveToPosition(s.team, s.kickerID, penaltyKickerPreparePosition(s.gi, s.team))
		move.AllowOutsideField(true)
		s.activityHandler.AddActivity(move)
	}
	return "NONE"
}

func (s *Penalty) Initialize() {
	s.penaltyKick = nil
	s.kickerID = 0
	s.ballInPlay = false
	s.postKickHeading = 0
	s.postKickHeadingOK = false
	s.penaltyStart = time.Now()
	s.originalBallPos = kickoffBallPosition(s.gi)

	if s.gi.Status.GetGameEvent().TeamWithPossession != s.team {
		return
	}
	s.initializeKick()
}

func (s *Penalty) initializeKick() {
	s.penaltyKick = nil
	s.kickerID = 0
	kickerID, ok := selectPenaltyKicker(s.gi, s.team, s.activeRobots)
	if !ok {
		return
	}
	s.kickerID = kickerID

	kickPrepareName := StateName(fmt.Sprintf("PenaltyKickPrepare ID %d", kickerID))
	kickName := StateName(fmt.Sprintf("PenaltyKick ID %d", kickerID))
	intent := &PenaltyIntent{
		gi:     s.gi,
		target: penaltyKickTargetPosition(s.gi, s.team),
	}
	prepareKick := &roles.AlignState{
		Ctx: intent, Gi: s.gi, Team: s.team, RobotId: kickerID,
		Name: kickPrepareName, ActivityHandler: s.activityHandler,
	}
	kick := &roles.KickState{
		Ctx: intent, Gi: s.gi, Team: s.team, RobotId: kickerID,
		Name: kickName, ActivityHandler: s.activityHandler,
	}

	// AlignBall can orbit around the ball when the attacker starts on the wrong
	// side. KickBall assumes that alignment has already happened and otherwise
	// drives directly at a point behind the ball, where it can stall.
	s.penaltyKick = NewStateMachine(prepareKick)
	s.penaltyKick.AddTransition(kickPrepareName, "ALIGNED", kick)
}

func (s *Penalty) Update() EventName {
	trackedBall := s.gi.State.GetTrackedBall()
	pos, ok := trackedBall.GetTrackedPosition()
	ballMoved := ok && restartBallMovedIntoPlay(s.originalBallPos, pos)
	gameEvent := s.gi.Status.GetGameEvent()
	if restartActionTimedOut(gameEvent, s.penaltyStart, PenaltyMaxTime(s.gi.Status.GetDivision())) {
		stopPenaltyRobots(s.activeRobots, s.activityHandler)
		return "NONE"
	}

	if !s.ballInPlay && (gameEvent.BallInPlay || ballMoved) {
		s.ballInPlay = true
		s.postKickHeading, s.postKickHeadingOK = restartKickerHeading(s.gi, s.team, s.kickerID)
		gameEvent.SetBallMoved()
	}

	if !s.ballInPlay && gameEvent.TeamWithPossession == s.team &&
		(s.penaltyKick == nil || !robotIDInList(s.activeRobots, s.kickerID)) {
		s.initializeKick()
	}

	hasKicker := gameEvent.TeamWithPossession == s.team &&
		s.penaltyKick != nil && robotIDInList(s.activeRobots, s.kickerID)
	positionPenaltyNonParticipants(
		s.gi,
		s.activeRobots,
		s.team,
		s.activityHandler,
		s.kickerID,
		hasKicker,
		s.originalBallPos,
		s.ballInPlay,
	)
	if hasKicker && s.ballInPlay {
		holdRestartKickerHeading(
			s.gi,
			s.team,
			s.kickerID,
			s.postKickHeading,
			s.postKickHeadingOK,
			s.activityHandler,
		)
	} else if hasKicker {
		s.penaltyKick.Update()
	}
	return "NONE"
}

// PenaltyIntent keeps the selected shot target stable while the robot aligns.
// Re-evaluating the keeper on every frame can otherwise make the shooter
// oscillate between both corners of the goal.
type PenaltyIntent struct {
	gi     *info.GameInfo
	target info.Position
}

func (s *PenaltyIntent) GetTargetPosition() info.Position {
	return s.target
}

func (s *PenaltyIntent) GetFromPosition() info.Position {
	return kickoffBallPosition(s.gi)
}

type KickOffIntent struct {
	gi   *info.GameInfo
	team info.Team
	id   info.ID
}

func (s *KickOffIntent) GetTargetPosition() info.Position {
	ballPos := kickoffBallPosition(s.gi)
	ballPos.X += -ownHalfXSign(s.gi, s.team) * kickoffKickTargetDistanceMM
	ballPos.Z = 0
	ballPos.Angle = 0
	return ballPos
}

func (s *KickOffIntent) GetFromPosition() info.Position {
	return kickoffBallPosition(s.gi)
}

type FreeKickIntent struct {
	gi   *info.GameInfo
	team info.Team
	id   info.ID
}

func (s *FreeKickIntent) GetTargetPosition() info.Position {
	return freeKickTargetPosition(s.gi, s.team, kickoffBallPosition(s.gi))
}

func (s *FreeKickIntent) GetFromPosition() info.Position {
	return kickoffBallPosition(s.gi)
}

func (s *FreeKick) Initialize() {
	s.freeKickStart = time.Now()
	s.freeKick = nil
	s.kickTaken = false
	s.postKickStart = time.Time{}
	s.postKickHeading = 0
	s.postKickHeadingOK = false

	ballPos := kickoffBallPosition(s.gi)
	kickerID, ok := selectFreeKickKicker(s.gi, s.team, s.activeRobots, ballPos)
	if ok {
		s.kickerID = kickerID
		kickPrepareName := StateName(fmt.Sprintf("KickPrepare ID %d", kickerID))
		kickName := StateName(fmt.Sprintf("Kick ID %d", kickerID))
		freeKick := FreeKickIntent{gi: s.gi, team: s.team, id: kickerID}
		prepareKick := &roles.AlignState{Ctx: &freeKick, Gi: s.gi, Team: s.team, RobotId: kickerID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
		kick := &roles.KickState{Ctx: &freeKick, Gi: s.gi, Team: s.team, RobotId: kickerID, Name: kickName, ActivityHandler: s.activityHandler}

		s.freeKick = NewStateMachine(prepareKick)
		s.freeKick.AddTransition(kickPrepareName, "ALIGNED", kick)
	}

	originalBallPos, ok := s.gi.State.GetTrackedBall().GetTrackedPosition()
	if !ok {
		originalBallPos = ballPos
	}
	s.originalBallPos = originalBallPos
}

func (s *FreeKick) Update() EventName {

	trackedBall := s.gi.State.GetTrackedBall()
	pos, ok := trackedBall.GetTrackedPosition()
	ballMoved := ok && restartBallMovedIntoPlay(s.originalBallPos, pos)
	if !ok {
		pos = s.originalBallPos
	}

	if !s.kickTaken && ballMoved {
		s.kickTaken = true
		s.postKickStart = time.Now()
		s.postKickHeading, s.postKickHeadingOK = restartKickerHeading(s.gi, s.team, s.kickerID)
		s.gi.Status.GetGameEvent().SetBallMoved()
	}
	if s.kickTaken {
		if restartPostKickHoldComplete(s.postKickStart, s.originalBallPos, pos, time.Now()) {
			return GAME_RUNNING_DETECTED
		}
		holdRestartKickerHeading(
			s.gi,
			s.team,
			s.kickerID,
			s.postKickHeading,
			s.postKickHeadingOK,
			s.activityHandler,
		)
		return "NONE"
	}

	if restartActionTimedOut(s.gi.Status.GetGameEvent(), s.freeKickStart, FreeKickMaxTime(s.gi.Status.GetDivision())) {
		s.gi.Status.GetGameEvent().SetBallMoved()
		return GAME_RUNNING_DETECTED
	}

	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		if s.freeKick != nil {
			s.freeKick.Update()
		}
	} else {
		moveRobotsToDefensePosition(s.gi, s.activeRobots, s.team, s.activityHandler)
	}

	return "NONE"
}

func (s *PrepareKickoff) Initialize() {

	goalieID, hasGoalie := selectGoalieID(s.gi, s.team, s.activeRobots)
	kickOffID, receiverID := selectKickoffRobots(fieldRobots(s.activeRobots, goalieID, hasGoalie))
	s.kickOffID = kickOffID
	s.receiverID = receiverID

}

func (s *PrepareKickoff) Update() EventName {

	goalieID, hasGoalie := moveGoalieToPosition(s.activeRobots, s.team, s.gi, s.activityHandler)
	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		moveKickerToKickoffPreparePosition(s.team, s.gi, s.activityHandler, s.kickOffID)
		moveRobotsToKickoffSupportPositions(
			s.activeRobots,
			s.team,
			s.gi,
			s.activityHandler,
			s.kickOffID,
			s.receiverID,
			goalieID,
			hasGoalie,
		)
	} else {
		moveRobotsToKickoffDefensePositions(
			s.activeRobots,
			s.team,
			s.gi,
			s.activityHandler,
			goalieID,
			hasGoalie,
		)
	}

	return "NONE"
}

func (s *Kickoff) Initialize() {
	s.kickTaken = false
	s.postKickStart = time.Time{}
	s.postKickHeading = 0
	s.postKickHeadingOK = false

	goalieID, hasGoalie := selectGoalieID(s.gi, s.team, s.activeRobots)
	kickOffID, receiverID := selectKickoffRobots(fieldRobots(s.activeRobots, goalieID, hasGoalie))
	s.kickOffID = kickOffID
	s.receiverID = receiverID

	kickPrepareName := StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))
	kickName := StateName(fmt.Sprintf("Kick ID %d", kickOffID))

	originalBallPos, ok := s.gi.State.GetTrackedBall().GetTrackedPosition()
	s.kickStart = time.Now()

	if !ok {
		fmt.Println("Failed retriving original ball pos during KickOff")
	}
	s.originalBallPos = originalBallPos

	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickName, ActivityHandler: s.activityHandler}

	s.kickOff = NewStateMachine(prepareKick)
	s.kickOff.AddTransition(kickPrepareName, "ALIGNED", kick)

	// If we dont have possession we take good defense position
}

func (s *Kickoff) Update() EventName {

	trackedBall := s.gi.State.GetTrackedBall()
	pos, ok := trackedBall.GetTrackedPosition()
	ballMoved := ok && restartBallMovedIntoPlay(s.originalBallPos, pos)
	if !ok {
		pos = s.originalBallPos
	}

	if !s.kickTaken && ballMoved {
		s.kickTaken = true
		s.postKickStart = time.Now()
		s.postKickHeading, s.postKickHeadingOK = restartKickerHeading(s.gi, s.team, s.kickOffID)
		s.gi.Status.GetGameEvent().SetBallMoved()
	}
	if s.kickTaken {
		if restartPostKickHoldComplete(s.postKickStart, s.originalBallPos, pos, time.Now()) {
			return GAME_RUNNING_DETECTED
		}
		holdRestartKickerHeading(
			s.gi,
			s.team,
			s.kickOffID,
			s.postKickHeading,
			s.postKickHeadingOK,
			s.activityHandler,
		)
		return "NONE"
	}

	if restartActionTimedOut(s.gi.Status.GetGameEvent(), s.kickStart, KickoffMaxTime(s.gi.Status.GetDivision())) {
		s.gi.Status.GetGameEvent().SetBallMoved()
		return GAME_RUNNING_DETECTED
	}

	goalieID, hasGoalie := moveGoalieToPosition(s.activeRobots, s.team, s.gi, s.activityHandler)
	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		s.kickOff.Update()
		moveRobotsToKickoffSupportPositions(
			s.activeRobots,
			s.team,
			s.gi,
			s.activityHandler,
			s.kickOffID,
			s.receiverID,
			goalieID,
			hasGoalie,
		)
	} else {
		moveRobotsToKickoffDefensePositions(
			s.activeRobots,
			s.team,
			s.gi,
			s.activityHandler,
			goalieID,
			hasGoalie,
		)
	}

	return "NONE"
}

func (s *Running) Initialize() {
}

func (s *Running) GetName() StateName {
	return "RUNNING"
}

func (s *Running) Update() EventName {
	return "NONE"
}

type UninitializedRef struct {
}

func (s *UninitializedRef) Initialize() {
}

func (s *UninitializedRef) GetName() StateName {
	return "UNINITIALIZED"
}

func (s *UninitializedRef) Update() EventName {
	return "NONE"
}

type RefereeHandler struct {
	gi                            *info.GameInfo
	refereeSM                     *StateMachine
	activeRobots                  []info.ID
	nextCommandAfterBallPlacement info.RefCommand
	kickOff                       *Kickoff
	freeKick                      *FreeKick
	stop                          *Stop
	stateInfos                    []*RefereeInfo
	activityHandler               *ai.ActivityHandler
	kickoffTouchRestriction       kickoffTouchRestriction
}

func KickoffMaxTime(division info.Division) time.Duration {
	switch division {
	case info.DivisionA:
		return KICKOFF_MAX_TIME_DIVISION_A
	default:
		return KICKOFF_MAX_TIME_DIVISION_B
	}
}

func FreeKickMaxTime(division info.Division) time.Duration {
	switch division {
	case info.DivisionA:
		return FREEKICK_MAX_TIME_DIVISION_A
	default:
		return FREEKICK_MAX_TIME_DIVISION_B
	}
}

func PenaltyMaxTime(division info.Division) time.Duration {
	switch division {
	case info.DivisionA:
		return PENALTY_MAX_TIME_DIVISION_A
	default:
		return PENALTY_MAX_TIME_DIVISION_B
	}
}

func restartBallMovedIntoPlay(originalBallPos, currentBallPos info.Position) bool {
	diff := currentBallPos.Sub(&originalBallPos)
	return diff.Norm2d() >= BALL_IN_PLAY_DISTANCE_MM
}

func restartPostKickHoldComplete(
	started time.Time,
	originalBallPos info.Position,
	currentBallPos info.Position,
	now time.Time,
) bool {
	if started.IsZero() {
		return false
	}

	elapsed := now.Sub(started)
	if elapsed >= restartPostKickMaxHeadingHold {
		return true
	}
	if elapsed < restartPostKickMinHeadingHold {
		return false
	}
	return originalBallPos.Dist2d(currentBallPos) >= restartPostKickBallClearanceMM
}

func restartKickerHeading(gi *info.GameInfo, team info.Team, id info.ID) (float64, bool) {
	if gi == nil || gi.State == nil || id >= info.TEAM_SIZE {
		return 0, false
	}
	robot := gi.State.GetTeam(team)[id]
	if robot == nil {
		return 0, false
	}
	pos, err := robot.GetPosition()
	if err != nil {
		return 0, false
	}
	return pos.Angle, true
}

func holdRestartKickerHeading(
	gi *info.GameInfo,
	team info.Team,
	id info.ID,
	heading float64,
	headingOK bool,
	activityHandler *ai.ActivityHandler,
) {
	if !headingOK || gi == nil || gi.State == nil || activityHandler == nil || id >= info.TEAM_SIZE {
		return
	}
	robot := gi.State.GetTeam(team)[id]
	if robot == nil {
		return
	}
	pos, err := robot.GetPosition()
	if err != nil {
		return
	}

	// Hold translation at the measured position while retaining the heading at
	// which the restart was taken. Disabling RRT prevents a zero-distance hold
	// from acquiring a path waypoint with an unrelated orientation.
	pos.Angle = heading
	move := act.NewMoveToPosition(team, id, pos)
	move.SetUseRRT(false)
	move.AvoidBall(false)
	activityHandler.AddActivity(move)
}

func restartActionTimedOut(gameEvent *info.GameEvent, fallbackStart time.Time, fallbackTimeout time.Duration) bool {
	if gameEvent != nil && gameEvent.HasCurrentActionTimeRemaining() {
		return gameEvent.CurrentActionTimedOut()
	}
	return time.Since(fallbackStart) >= fallbackTimeout
}

func isBallPlacementCommand(rc info.RefCommand) bool {
	return rc == info.BALL_PLACEMENT_BLUE || rc == info.BALL_PLACEMENT_YELLOW
}

func refCommandToEventName(rc info.RefCommand) string {
	switch rc {
	case info.HALT:
		return HALT
	case info.STOP:
		return STOP
	case info.NORMAL_START:
		return NORMAL_START
	case info.FORCE_START:
		return FORCE_START
	case info.PREPARE_KICKOFF_BLUE:
		return PREPARE_KICKOFF
	case info.PREPARE_KICKOFF_YELLOW:
		return PREPARE_KICKOFF
	case info.PREPARE_PENALTY_YELLOW, info.PREPARE_PENALTY_BLUE:
		return PREPARE_PENALTY
	case info.DIRECT_FREE_YELLOW:
		return FREE_KICK
	case info.DIRECT_FREE_BLUE:
		return FREE_KICK
	case info.INDIRECT_FREE_YELLOW:
		return FREE_KICK
	case info.INDIRECT_FREE_BLUE:
		return FREE_KICK
	case info.TIMEOUT_YELLOW:
		return TIMEOUT
	case info.TIMEOUT_BLUE:
		return TIMEOUT
	case info.BALL_PLACEMENT_BLUE:
		return BALL_PLACEMENT
	case info.BALL_PLACEMENT_YELLOW:
		return BALL_PLACEMENT
	default:
		return UNKNOWN
	}
}

func moveRobotsToDefensePosition(
	gi *info.GameInfo,
	activeRobots []info.ID,
	team info.Team,
	activityHandler *ai.ActivityHandler,
) {
	goalieID, hasGoalie := selectGoalieID(gi, team, activeRobots)
	if hasGoalie {
		activityHandler.AddActivity(act.NewGoalie(team, goalieID))
	}

	defenders := fieldRobots(activeRobots, goalieID, hasGoalie)
	ballPos := kickoffBallPosition(gi)
	wallSlots := freeKickDefenseSlots(gi, team, ballPos, len(defenders))
	wallRobots, reserveRobots := assignFreeKickWallRobots(gi, team, defenders, wallSlots)
	for i, robotID := range wallRobots {
		activityHandler.AddActivity(act.NewMoveToPosition(team, robotID, wallSlots[i]))
	}

	reserveSlots := freeKickReserveSlots(gi, team, ballPos, len(reserveRobots))
	for i, robotID := range reserveRobots {
		activityHandler.AddActivity(act.NewMoveToPosition(team, robotID, reserveSlots[i]))
	}
}

func positionPenaltyNonParticipants(
	gi *info.GameInfo,
	activeRobots []info.ID,
	team info.Team,
	activityHandler *ai.ActivityHandler,
	exemptID info.ID,
	hasExempt bool,
	penaltyMark info.Position,
	goalieFree bool,
) {
	attackingTeam := gi.Status.GetGameEvent().TeamWithPossession
	waitingRobots := append([]info.ID(nil), activeRobots...)
	if team != attackingTeam {
		goalieID, hasGoalie := selectGoalieID(gi, team, activeRobots)
		if hasGoalie {
			if goalieFree {
				activityHandler.AddActivity(act.NewGoalie(team, goalieID))
			} else {
				movePenaltyGoalieToGoalLine(gi, team, goalieID, activityHandler)
			}
			waitingRobots = sortedRobotIDsExcluding(waitingRobots, goalieID)
		}
	}

	if hasExempt {
		waitingRobots = sortedRobotIDsExcluding(waitingRobots, exemptID)
	}
	movePenaltyWaitingRobots(gi, waitingRobots, team, attackingTeam, penaltyMark, activityHandler)
}

func stopPenaltyRobots(
	activeRobots []info.ID,
	activityHandler *ai.ActivityHandler,
) {
	for _, robotID := range activeRobots {
		activityHandler.AddActivity(act.NewStop(robotID))
	}
}

func movePenaltyGoalieToGoalLine(
	gi *info.GameInfo,
	team info.Team,
	goalieID info.ID,
	activityHandler *ai.ActivityHandler,
) {
	move := act.NewMoveToPosition(team, goalieID, penaltyGoalieHomePosition(gi, team))
	move.AvoidGoallines(false)
	activityHandler.AddActivity(move)
}

func movePenaltyWaitingRobots(
	gi *info.GameInfo,
	robotIDs []info.ID,
	team info.Team,
	attackingTeam info.Team,
	penaltyMark info.Position,
	activityHandler *ai.ActivityHandler,
) {
	slots := penaltyWaitingSlots(gi, team, attackingTeam, penaltyMark, len(robotIDs))
	for i, robotID := range robotIDs {
		move := act.NewMoveToPosition(team, robotID, slots[i])
		move.AllowOutsideField(true)
		move.AllowBehindGoalLine(true)
		activityHandler.AddActivity(move)
	}
}

func penaltyGoalieHomePosition(gi *info.GameInfo, team info.Team) info.Position {
	// Match the keeper's legal penalty position with its center just inside the
	// goal line. This follows Sumatra's dedicated prepare-penalty keeper state.
	const goalieCenterInsetMM = 70.0

	halfLength := 4500.0
	if gi != nil {
		if geometry, ok := gi.FieldGeometry(); ok && geometry.Length > 0 {
			halfLength = geometry.Length / 2
		}
	}

	pos := info.Position{X: ownHalfXSign(gi, team) * (halfLength - goalieCenterInsetMM)}
	pos.Angle = pos.AngleToPosition(kickoffBallPosition(gi))
	return pos
}

func penaltyKickerPreparePosition(gi *info.GameInfo, team info.Team) info.Position {
	ballPos := kickoffBallPosition(gi)
	targetPos := penaltyKickTargetPosition(gi, team)
	direction := targetPos.Sub(&ballPos)
	direction.Z = 0
	direction.Angle = 0
	if direction.Norm2d() < 1 {
		direction.X = -ownHalfXSign(gi, team)
	}
	direction.Div2d(direction.Norm2d())

	preparePos := ballPos
	preparePos.X -= direction.X * penaltyPrepareDistanceMM
	preparePos.Y -= direction.Y * penaltyPrepareDistanceMM
	preparePos.Z = 0
	preparePos.Angle = preparePos.AngleToPosition(targetPos)
	return preparePos
}

func penaltyWaitingSlots(
	gi *info.GameInfo,
	team info.Team,
	attackingTeam info.Team,
	penaltyMark info.Position,
	count int,
) []info.Position {
	if count <= 0 {
		return nil
	}

	const (
		defaultHalfWidth = 3000.0
		fieldMargin      = 300.0
		minimumLaneY     = 1500.0
	)
	halfWidth := defaultHalfWidth
	if gi != nil {
		if geometry, ok := gi.FieldGeometry(); ok {
			halfWidth = geometry.Width / 2
		}
	}

	// SSL defines forward movement for a penalty by the field X coordinate.
	// Keep the waiting line behind the mark on that same axis, independent of
	// which point inside the goal the attacker aims at.
	direction := info.Position{X: -ownHalfXSign(gi, attackingTeam)}
	base := info.Position{
		X: penaltyMark.X - direction.X*penaltyBehindBallDistanceMM,
		Y: penaltyMark.Y - direction.Y*penaltyBehindBallDistanceMM,
	}
	maxLaneY := math.Max(0, halfWidth-fieldMargin-math.Abs(penaltyMark.Y))
	minLaneY := math.Min(minimumLaneY, maxLaneY)
	laneStep := 0.0
	if count > 1 {
		laneStep = (maxLaneY - minLaneY) / float64(count-1)
	}
	laneSign := 1.0
	if team != attackingTeam {
		laneSign = -1
	}
	slots := make([]info.Position, count)
	for i := range slots {
		laneOffset := laneSign * (minLaneY + laneStep*float64(i))
		pos := info.Position{
			X: base.X,
			Y: clampFloat(base.Y+laneOffset, -halfWidth+fieldMargin, halfWidth-fieldMargin),
		}
		pos.Angle = pos.AngleToPosition(penaltyMark)
		slots[i] = pos
	}
	return slots
}

func selectPenaltyKicker(gi *info.GameInfo, team info.Team, activeRobots []info.ID) (info.ID, bool) {
	return selectFreeKickKicker(gi, team, activeRobots, kickoffBallPosition(gi))
}

func penaltyKickTargetPosition(gi *info.GameInfo, team info.Team) info.Position {
	ballPos := kickoffBallPosition(gi)
	if gi == nil || !gi.HasField() {
		ballPos.X += -ownHalfXSign(gi, team) * kickoffKickTargetDistanceMM
		return ballPos
	}

	target := gi.EnemyGoalCenter(team)
	goalLine := gi.EnemyGoalLine(team)
	if len(goalLine) < 2 || gi.State == nil {
		return target
	}

	// Aim away from the opponent closest to the goal. This is the small,
	// deterministic equivalent of Sumatra's open-goal target rating.
	closestDistance := math.Inf(1)
	closestY := 0.0
	foundKeeper := false
	for _, robot := range gi.State.GetOtherTeam(team) {
		if robot == nil || !robot.IsActive() {
			continue
		}
		pos, err := robot.GetPosition()
		if err != nil {
			continue
		}
		distance := pos.Dist2d(target)
		if distance < closestDistance {
			closestDistance = distance
			closestY = pos.Y
			foundKeeper = true
		}
	}
	if !foundKeeper {
		return target
	}

	goalWidth := goalLine[0].Dist2d(goalLine[1])
	offset := goalWidth * 0.30
	if closestY >= target.Y {
		target.Y -= offset
	} else {
		target.Y += offset
	}
	return target
}

func freeKickDefenseSlots(
	gi *info.GameInfo,
	team info.Team,
	ballPos info.Position,
	availableRobots int,
) []info.Position {
	count := min(availableRobots, freeKickWallRobotCount)
	if count <= 0 {
		return nil
	}

	goal := ownGoalCenter(gi, team)
	directionX := goal.X - ballPos.X
	directionY := goal.Y - ballPos.Y
	directionLength := math.Hypot(directionX, directionY)
	if directionLength == 0 {
		directionX = ownHalfXSign(gi, team)
		directionY = 0
	} else {
		directionX /= directionLength
		directionY /= directionLength
	}

	// Prefer Sumatra's stop-radius + 200 mm free-kick protection distance.
	// If that would put any wall robot too close to a defense area or field
	// line, search along the shot line for the nearest legal straight wall.
	for delta := 0.0; delta <= 1500; delta += 10 {
		for _, distance := range []float64{freeKickWallDistanceMM - delta, freeKickWallDistanceMM + delta} {
			if distance < freeKickBallClearanceMM {
				continue
			}
			slots := wallSlotsAtDistance(ballPos, directionX, directionY, distance, count)
			if freeKickSlotsLegal(gi, ballPos, slots) {
				return slots
			}
		}
	}

	// This fallback is only expected for invalid/non-regulation ball positions.
	// Runtime motion safety still applies the restart ball and defense-area
	// obstacles while the robots travel to these clamped destinations.
	slots := wallSlotsAtDistance(ballPos, directionX, directionY, freeKickWallDistanceMM, count)
	for i := range slots {
		slots[i] = clampFreeKickSlot(gi, slots[i])
		slots[i].Angle = slots[i].AngleToPosition(ballPos)
	}
	return slots
}

func wallSlotsAtDistance(
	ballPos info.Position,
	directionX float64,
	directionY float64,
	distance float64,
	count int,
) []info.Position {
	centerX := ballPos.X + directionX*distance
	centerY := ballPos.Y + directionY*distance
	// A normal to the ball-to-goal line makes the robots form a wall across
	// the most direct shot path.
	normalX := -directionY
	normalY := directionX
	// Keep slot ordering stable from low to high field Y on both field halves.
	if normalY < 0 || (normalY == 0 && normalX < 0) {
		normalX = -normalX
		normalY = -normalY
	}
	startOffset := -freeKickWallSpacingMM * float64(count-1) / 2

	slots := make([]info.Position, count)
	for i := range slots {
		offset := startOffset + freeKickWallSpacingMM*float64(i)
		slots[i] = info.Position{
			X: centerX + normalX*offset,
			Y: centerY + normalY*offset,
		}
		slots[i].Angle = slots[i].AngleToPosition(ballPos)
	}
	return slots
}

func ownGoalCenter(gi *info.GameInfo, team info.Team) info.Position {
	halfLength := 4500.0
	if gi != nil && gi.HasField() {
		if field := gi.FieldSize(); field.X > 0 {
			halfLength = field.X / 2
		}
	}
	return info.Position{X: ownHalfXSign(gi, team) * halfLength}
}

func freeKickSlotsLegal(gi *info.GameInfo, ballPos info.Position, slots []info.Position) bool {
	for _, slot := range slots {
		if slot.Dist2d(ballPos) < freeKickBallClearanceMM || !freeKickSlotLegal(gi, slot) {
			return false
		}
	}
	return true
}

func freeKickSlotLegal(gi *info.GameInfo, slot info.Position) bool {
	if gi == nil || !gi.HasField() {
		return true
	}
	geometry, ok := gi.FieldGeometry()
	if !ok {
		return true
	}

	halfLength := geometry.Length / 2
	halfWidth := geometry.Width / 2
	if math.Abs(slot.X) > halfLength-freeKickFieldClearanceMM ||
		math.Abs(slot.Y) > halfWidth-freeKickFieldClearanceMM {
		return false
	}

	return !insideInflatedDefenseArea(slot, geometry, -1) &&
		!insideInflatedDefenseArea(slot, geometry, 1)
}

func insideInflatedDefenseArea(slot info.Position, geometry info.FieldGeometry, goalSign float64) bool {
	halfLength := geometry.Length / 2
	halfPenaltyWidth := geometry.PenaltyAreaWidth / 2
	goalLineX := goalSign * halfLength
	frontX := goalSign * (halfLength - geometry.PenaltyAreaDepth)
	minX := math.Min(goalLineX, frontX) - freeKickDefenseAreaClearanceMM
	maxX := math.Max(goalLineX, frontX) + freeKickDefenseAreaClearanceMM
	minY := -halfPenaltyWidth - freeKickDefenseAreaClearanceMM
	maxY := halfPenaltyWidth + freeKickDefenseAreaClearanceMM
	return slot.X >= minX && slot.X <= maxX && slot.Y >= minY && slot.Y <= maxY
}

func clampFreeKickSlot(gi *info.GameInfo, slot info.Position) info.Position {
	if gi == nil || !gi.HasField() {
		return slot
	}
	slot = gi.ClampToField(slot, freeKickFieldClearanceMM)
	geometry, ok := gi.FieldGeometry()
	if !ok {
		return slot
	}

	for _, goalSign := range []float64{-1, 1} {
		if !insideInflatedDefenseArea(slot, geometry, goalSign) {
			continue
		}
		frontX := goalSign * (geometry.Length/2 - geometry.PenaltyAreaDepth)
		slot.X = frontX - goalSign*freeKickDefenseAreaClearanceMM
	}
	return slot
}

func assignFreeKickWallRobots(
	gi *info.GameInfo,
	team info.Team,
	defenders []info.ID,
	slots []info.Position,
) ([]info.ID, []info.ID) {
	ordered := append([]info.ID(nil), defenders...)
	sort.SliceStable(ordered, func(i, j int) bool {
		distanceI := robotDistanceToClosestSlot(gi, team, ordered[i], slots)
		distanceJ := robotDistanceToClosestSlot(gi, team, ordered[j], slots)
		if distanceI == distanceJ {
			return ordered[i] < ordered[j]
		}
		return distanceI < distanceJ
	})

	wallCount := min(len(ordered), len(slots))
	wallRobots := ordered[:wallCount]
	reserveRobots := ordered[wallCount:]

	// Match the selected robots to the wall from one end to the other. This
	// reduces crossing paths as the wall rotates for an angled free kick.
	if len(slots) > 1 {
		normalX := slots[len(slots)-1].X - slots[0].X
		normalY := slots[len(slots)-1].Y - slots[0].Y
		sort.SliceStable(wallRobots, func(i, j int) bool {
			projectionI, okI := robotProjection(gi, team, wallRobots[i], normalX, normalY)
			projectionJ, okJ := robotProjection(gi, team, wallRobots[j], normalX, normalY)
			if !okI || !okJ || projectionI == projectionJ {
				return wallRobots[i] < wallRobots[j]
			}
			return projectionI < projectionJ
		})
	}
	return wallRobots, reserveRobots
}

func robotDistanceToClosestSlot(
	gi *info.GameInfo,
	team info.Team,
	robotID info.ID,
	slots []info.Position,
) float64 {
	if gi == nil || gi.State == nil || len(slots) == 0 {
		return math.Inf(1)
	}
	robot := gi.State.GetTeam(team)[robotID]
	if robot == nil {
		return math.Inf(1)
	}
	pos, err := robot.GetPosition()
	if err != nil {
		return math.Inf(1)
	}
	best := math.Inf(1)
	for _, slot := range slots {
		best = math.Min(best, pos.Dist2d(slot))
	}
	return best
}

func robotProjection(
	gi *info.GameInfo,
	team info.Team,
	robotID info.ID,
	directionX float64,
	directionY float64,
) (float64, bool) {
	if gi == nil || gi.State == nil {
		return 0, false
	}
	robot := gi.State.GetTeam(team)[robotID]
	if robot == nil {
		return 0, false
	}
	pos, err := robot.GetPosition()
	if err != nil {
		return 0, false
	}
	return pos.X*directionX + pos.Y*directionY, true
}

func freeKickReserveSlots(
	gi *info.GameInfo,
	team info.Team,
	ballPos info.Position,
	count int,
) []info.Position {
	if count <= 0 {
		return nil
	}

	halfLength := 4500.0
	halfWidth := 3000.0
	if gi != nil && gi.HasField() {
		field := gi.FieldSize()
		halfLength = field.X / 2
		halfWidth = field.Y / 2
	}
	templates := []struct{ depth, lane float64 }{
		{depth: 0.55, lane: -0.60},
		{depth: 0.55, lane: 0.60},
		{depth: 0.30, lane: -0.75},
		{depth: 0.30, lane: 0.75},
	}

	slots := make([]info.Position, count)
	for i := range slots {
		template := templates[i%len(templates)]
		slots[i] = info.Position{
			X: ownHalfXSign(gi, team) * halfLength * template.depth,
			Y: halfWidth * template.lane,
		}
		slots[i] = clampFreeKickSlot(gi, slots[i])
		slots[i].Angle = slots[i].AngleToPosition(ballPos)
	}
	return slots
}

// PrepareKickerForUpcomingFreeKick moves the selected kicker behind the ball
// when STOP announces our free kick as the next command.
func PrepareKickerForUpcomingFreeKick(
	gi *info.GameInfo,
	team info.Team,
	activeRobots []info.ID,
	activityHandler *ai.ActivityHandler,
) (info.ID, bool) {
	if gi == nil || gi.Status == nil || activityHandler == nil {
		return 0, false
	}

	gameEvent := gi.Status.GetGameEvent()
	if gameEvent == nil {
		return 0, false
	}
	freeKickTeam, hasFreeKick := gameEvent.NextCommand.FreeKickTeam()
	if !hasFreeKick || freeKickTeam != team {
		return 0, false
	}

	ballPos := freeKickPreparationBallPosition(gi, gameEvent)
	kickerID, ok := selectFreeKickKicker(gi, team, activeRobots, ballPos)
	if !ok {
		return 0, false
	}

	move := act.NewMoveToPosition(team, kickerID, freeKickPreparePosition(gi, team, ballPos))
	move.AllowOutsideField(true)
	activityHandler.AddActivity(move)
	return kickerID, true
}

// PrepareForUpcomingKickoff applies the same legal kickoff formation while the
// referee is still in STOP. It returns false when no kickoff is announced.
func PrepareForUpcomingKickoff(
	gi *info.GameInfo,
	team info.Team,
	activeRobots []info.ID,
	activityHandler *ai.ActivityHandler,
) bool {
	if gi == nil || gi.Status == nil || activityHandler == nil {
		return false
	}
	gameEvent := gi.Status.GetGameEvent()
	if gameEvent == nil {
		return false
	}
	kickoffTeam, announced := gameEvent.NextCommand.KickoffTeam()
	if !announced {
		return false
	}

	goalieID, hasGoalie := moveGoalieToPosition(activeRobots, team, gi, activityHandler)
	fieldRobotIDs := fieldRobots(activeRobots, goalieID, hasGoalie)
	kickOffID, receiverID := selectKickoffRobots(fieldRobotIDs)
	if kickoffTeam == team {
		if len(fieldRobotIDs) > 0 {
			moveKickerToKickoffPreparePosition(team, gi, activityHandler, kickOffID)
		}
		moveRobotsToKickoffSupportPositions(
			activeRobots,
			team,
			gi,
			activityHandler,
			kickOffID,
			receiverID,
			goalieID,
			hasGoalie,
		)
	} else {
		moveRobotsToKickoffDefensePositions(
			activeRobots,
			team,
			gi,
			activityHandler,
			goalieID,
			hasGoalie,
		)
	}
	return true
}

func selectFreeKickKicker(
	gi *info.GameInfo,
	team info.Team,
	activeRobots []info.ID,
	ballPos info.Position,
) (info.ID, bool) {
	if gi == nil || gi.State == nil || len(activeRobots) == 0 {
		return 0, false
	}

	goalieID, hasGoalie := selectGoalieID(gi, team, activeRobots)
	candidates := fieldRobots(activeRobots, goalieID, hasGoalie)
	if len(candidates) == 0 {
		candidates = activeRobots
	}

	closestID := info.ID(0)
	closestDistance := math.Inf(1)
	found := false
	for _, robotID := range candidates {
		robotPos, err := gi.State.GetRobotPosition(team, robotID)
		if err != nil {
			continue
		}

		distance := robotPos.Dist2d(ballPos)
		if !found || distance < closestDistance || (distance == closestDistance && robotID < closestID) {
			closestID = robotID
			closestDistance = distance
			found = true
		}
	}
	return closestID, found
}

func freeKickPreparationBallPosition(gi *info.GameInfo, gameEvent *info.GameEvent) info.Position {
	if gameEvent != nil && gameEvent.CurrentState == info.STATE_BALL_PLACEMENT {
		if designatedPosition := gameEvent.GetDesignatedPosition(); designatedPosition != nil && designatedPosition.Len() >= 2 {
			return info.Position{X: designatedPosition.AtVec(0), Y: designatedPosition.AtVec(1)}
		}
	}
	return kickoffBallPosition(gi)
}

func freeKickPreparePosition(gi *info.GameInfo, team info.Team, ballPos info.Position) info.Position {
	targetPos := freeKickTargetPosition(gi, team, ballPos)

	direction := targetPos.Sub(&ballPos)
	direction.Z = 0
	direction.Angle = 0
	if direction.Norm2d() < 1 {
		direction.X = -ownHalfXSign(gi, team)
	}
	direction.Div2d(direction.Norm2d())

	preparePos := ballPos
	preparePos.X -= direction.X * freeKickPrepareDistanceMM
	preparePos.Y -= direction.Y * freeKickPrepareDistanceMM
	preparePos.Z = 0
	preparePos.Angle = preparePos.AngleToPosition(targetPos)
	return preparePos
}

func freeKickTargetPosition(gi *info.GameInfo, team info.Team, ballPos info.Position) info.Position {
	if gi != nil && gi.HasField() {
		target := gi.EnemyGoalCenter(team)
		target.Z = 0
		target.Angle = 0
		return target
	}

	// Preserve a safe fallback before SSL-Vision geometry is available.
	target := ballPos
	target.X += -ownHalfXSign(gi, team) * kickoffKickTargetDistanceMM
	target.Z = 0
	target.Angle = 0
	return target
}

func selectKickoffRobots(activeRobots []info.ID) (info.ID, info.ID) {
	kickerID := defaultKickoffKickerID
	if !robotIDInList(activeRobots, kickerID) && len(activeRobots) > 0 {
		kickerID = activeRobots[0]
	}

	receiverID := defaultKickoffReceiverID
	if receiverID == kickerID || !robotIDInList(activeRobots, receiverID) {
		receiverID = kickerID
		for _, robotID := range activeRobots {
			if robotID != kickerID {
				receiverID = robotID
				break
			}
		}
	}

	return kickerID, receiverID
}

func robotIDInList(robotIDs []info.ID, target info.ID) bool {
	for _, robotID := range robotIDs {
		if robotID == target {
			return true
		}
	}
	return false
}

func selectGoalieID(gi *info.GameInfo, team info.Team, activeRobots []info.ID) (info.ID, bool) {
	if len(activeRobots) == 0 {
		return 0, false
	}

	if configuredID, ok := configuredGoalieID(gi, team); ok && robotIDInList(activeRobots, configuredID) {
		return configuredID, true
	}

	if robotIDInList(activeRobots, defaultGoalieID) {
		return defaultGoalieID, true
	}

	return robotClosestToHomeGoal(gi, team, activeRobots), true
}

func configuredGoalieID(gi *info.GameInfo, team info.Team) (info.ID, bool) {
	if gi == nil || gi.Status == nil {
		return 0, false
	}

	teamInfo := gi.Status.GetTeamInfo(team == info.Yellow)
	if teamInfo == nil {
		return 0, false
	}
	return info.ID(teamInfo.Goalkeeper), true
}

func robotClosestToHomeGoal(gi *info.GameInfo, team info.Team, activeRobots []info.ID) info.ID {
	if gi == nil || gi.State == nil || len(activeRobots) == 0 {
		return 0
	}

	target := kickoffGoalieHomePosition(gi, team)
	bestID := activeRobots[0]
	bestDist := math.Inf(1)
	for _, robotID := range activeRobots {
		robot := gi.State.GetTeam(team)[robotID]
		if robot == nil {
			continue
		}
		pos, err := robot.GetPosition()
		if err != nil {
			continue
		}

		dist := pos.Dist2d(target)
		if dist < bestDist {
			bestID = robotID
			bestDist = dist
		}
	}
	return bestID
}

func kickoffGoalieHomePosition(gi *info.GameInfo, team info.Team) info.Position {
	if gi != nil && gi.HasField() {
		return gi.HomeGoalDefPos(team)
	}
	return info.Position{X: ownHalfXSign(gi, team) * 4000, Y: 0, Z: 0, Angle: 0}
}

func moveGoalieToPosition(
	activeRobots []info.ID,
	team info.Team,
	gi *info.GameInfo,
	activityHandler *ai.ActivityHandler,
) (info.ID, bool) {
	goalieID, hasGoalie := selectGoalieID(gi, team, activeRobots)
	if hasGoalie {
		activityHandler.AddActivity(act.NewGoalie(team, goalieID))
	}
	return goalieID, hasGoalie
}

func fieldRobots(activeRobots []info.ID, goalieID info.ID, hasGoalie bool) []info.ID {
	if !hasGoalie {
		return activeRobots
	}
	return sortedRobotIDsExcluding(activeRobots, goalieID)
}

func moveKickerToKickoffPreparePosition(
	team info.Team,
	gi *info.GameInfo,
	activityHandler *ai.ActivityHandler,
	kickOffID info.ID,
) {
	activityHandler.AddActivity(act.NewMoveToPosition(team, kickOffID, kickoffPreparePosition(gi, team)))
}

func kickoffPreparePosition(gi *info.GameInfo, team info.Team) info.Position {
	ballPos := kickoffBallPosition(gi)
	targetPos := (&KickOffIntent{gi: gi, team: team}).GetTargetPosition()
	pos := ballPos
	pos.X += ownHalfXSign(gi, team) * kickoffPrepareDistanceMM
	pos.Z = 0
	pos.Angle = pos.AngleToPosition(targetPos)
	if gi != nil {
		pos = gi.ClampToField(pos, kickoffFieldMarginMM)
	}
	fmt.Println(pos)
	return pos
}

func moveRobotsToKickoffSupportPositions(
	activeRobots []info.ID,
	team info.Team,
	gi *info.GameInfo,
	activityHandler *ai.ActivityHandler,
	kickOffID info.ID,
	receiverID info.ID,
	goalieID info.ID,
	hasGoalie bool,
) {
	supportRobots := kickoffSupportRobots(fieldRobots(activeRobots, goalieID, hasGoalie), kickOffID, receiverID)
	if len(supportRobots) == 0 {
		return
	}

	slots := kickoffSupportSlots(gi, team, len(supportRobots))
	for i, robotID := range supportRobots {
		activityHandler.AddActivity(act.NewMoveToPosition(team, robotID, slots[i]))
	}
}

func moveRobotsToKickoffDefensePositions(
	activeRobots []info.ID,
	team info.Team,
	gi *info.GameInfo,
	activityHandler *ai.ActivityHandler,
	goalieID info.ID,
	hasGoalie bool,
) {
	defenders := fieldRobots(activeRobots, goalieID, hasGoalie)
	if len(defenders) == 0 {
		return
	}

	slots := kickoffDefenseSlots(gi, team, len(defenders))
	for i, robotID := range defenders {
		activityHandler.AddActivity(act.NewMoveToPosition(team, robotID, slots[i]))
	}
}

func kickoffSupportRobots(robotIDs []info.ID, kickOffID info.ID, receiverID info.ID) []info.ID {
	supportRobots := sortedRobotIDsExcluding(robotIDs, kickOffID)
	if receiverID == kickOffID || !robotIDInList(supportRobots, receiverID) {
		return supportRobots
	}

	ordered := make([]info.ID, 0, len(supportRobots))
	ordered = append(ordered, receiverID)
	for _, robotID := range supportRobots {
		if robotID != receiverID {
			ordered = append(ordered, robotID)
		}
	}
	return ordered
}

func sortedRobotIDsExcluding(robotIDs []info.ID, excludedID info.ID) []info.ID {
	filtered := make([]info.ID, 0, len(robotIDs))
	for _, robotID := range robotIDs {
		if robotID != excludedID {
			filtered = append(filtered, robotID)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i] < filtered[j]
	})
	return filtered
}

type kickoffSlotTemplate struct {
	depthFraction float64
	lane          float64
}

func kickoffSupportSlots(gi *info.GameInfo, team info.Team, count int) []info.Position {
	return kickoffFormationSlots(
		gi,
		team,
		count,
		[]kickoffSlotTemplate{
			{depthFraction: 0.20, lane: 0.00},
			{depthFraction: 0.42, lane: 0.35},
			{depthFraction: 0.64, lane: -0.35},
			{depthFraction: 0.64, lane: 0.60},
			{depthFraction: 0.78, lane: -0.60},
			{depthFraction: 0.78, lane: 0.20},
			{depthFraction: 0.90, lane: -0.20},
		},
	)
}

func kickoffDefenseSlots(gi *info.GameInfo, team info.Team, count int) []info.Position {
	return kickoffFormationSlots(
		gi,
		team,
		count,
		[]kickoffSlotTemplate{
			{depthFraction: 0.78, lane: 0.20},
			{depthFraction: 0.78, lane: -0.20},
			{depthFraction: 0.88, lane: 0.50},
			{depthFraction: 0.88, lane: -0.50},
			{depthFraction: 0.64, lane: 0.00},
			{depthFraction: 0.94, lane: 0.75},
			{depthFraction: 0.94, lane: -0.75},
		},
	)
}

func kickoffFormationSlots(
	gi *info.GameInfo,
	team info.Team,
	count int,
	templates []kickoffSlotTemplate,
) []info.Position {
	if count <= 0 || len(templates) == 0 {
		return nil
	}

	halfLength := 4500.0
	halfWidth := 3000.0
	if gi != nil && gi.HasField() {
		field := gi.FieldSize()
		if field.X > 0 {
			halfLength = field.X / 2
		}
		if field.Y > 0 {
			halfWidth = field.Y / 2
		}
	}

	xSign := ownHalfXSign(gi, team)
	ballPos := kickoffBallPosition(gi)

	slots := make([]info.Position, 0, count)
	for slotIndex := 0; len(slots) < count; slotIndex++ {
		template := templates[slotIndex%len(templates)]
		extraDepth := 0.0
		if slotIndex >= len(templates) {
			extraDepth = 0.05 * float64(slotIndex/len(templates))
		}

		depth := clampFloat(
			halfLength*(template.depthFraction+extraDepth),
			kickoffCenterCircleClearanceMM+200,
			halfLength-kickoffFieldMarginMM,
		)
		pos := info.Position{
			X: xSign * depth,
			Y: clampFloat(template.lane*halfWidth, -halfWidth+kickoffFieldMarginMM, halfWidth-kickoffFieldMarginMM),
			Z: 0,
		}
		pos = keepOutsideKickoffCenterCircle(pos, xSign)
		pos.Angle = pos.AngleToPosition(ballPos)
		if gi != nil {
			pos = gi.ClampToField(pos, kickoffFieldMarginMM)
		}
		slots = append(slots, pos)
	}

	return slots
}

func ownHalfXSign(gi *info.GameInfo, team info.Team) float64 {
	if gi != nil {
		return gi.OwnHalfXSign(team)
	}
	fmt.Println("Incorrect gi, giving incorrect field side")
	if team == info.Blue {
		return -1
	}
	return 1
}

func kickoffBallPosition(gi *info.GameInfo) info.Position {
	if gi == nil || gi.State == nil {
		return info.Position{}
	}

	if pos, ok := gi.State.GetTrackedBall().GetTrackedPosition(); ok {
		return pos
	}
	if pos, err := gi.State.GetBall().GetPosition(); err == nil {
		return pos
	}
	return info.Position{}
}

func keepOutsideKickoffCenterCircle(pos info.Position, xSign float64) info.Position {
	if pos.Norm2d() >= kickoffCenterCircleClearanceMM {
		return pos
	}

	pos.X = xSign * kickoffCenterCircleClearanceMM
	return pos
}

func clampFloat(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func NewRefereeHandler(gi *info.GameInfo, activeRobots []info.ID, team info.Team, activityHandler *ai.ActivityHandler) *RefereeHandler {

	freeKickName := StateName("FREEKICK")
	stopName := StateName("STOP")
	haltName := StateName("HALT")
	prepareKickoffName := StateName("PREPAREKICKOFF")
	kickOffName := StateName("KICKOFF")
	ballPlacementName := StateName("BALLPLACEMENT")
	preparePenaltyName := StateName("PREPAREPENALTY")
	penaltyName := StateName("PENALTY")

	freeKick := &FreeKick{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            freeKickName,
			activityHandler: activityHandler,
		},
	}

	prepareKickoff := &PrepareKickoff{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            prepareKickoffName,
			activityHandler: activityHandler,
		},
	}
	stop := &Stop{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            stopName,
			activityHandler: activityHandler,
		},
	}
	ballPlacement := &Stop{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            ballPlacementName,
			activityHandler: activityHandler,
		},
	}
	halt := &Halt{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            haltName,
			activityHandler: activityHandler,
		},
	}
	kickOff := &Kickoff{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            kickOffName,
			activityHandler: activityHandler,
		},
	}
	preparePenalty := &PreparePenalty{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            preparePenaltyName,
			activityHandler: activityHandler,
		},
	}
	penalty := &Penalty{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            penaltyName,
			activityHandler: activityHandler,
		},
	}
	running := &Running{}
	uninitialized := &UninitializedRef{}

	// Timeout still uses stopped-play behavior.

	refereeSM := NewStateMachine(uninitialized)

	refereeSM.AddTransition("HALT", STOP, stop)
	// refereeSM.AddTransition("STOP", "Timeout", timeout)

	refereeSM.AddTransition("STOP", PREPARE_KICKOFF, prepareKickoff)
	refereeSM.AddTransition("STOP", PREPARE_PENALTY, preparePenalty)
	refereeSM.AddTransition("STOP", FREE_KICK, freeKick)
	refereeSM.AddTransition("STOP", BALL_PLACEMENT, ballPlacement)
	refereeSM.AddTransition("STOP", FORCE_START, running)

	refereeSM.AddTransition("BALLPLACEMENT", STOP, stop)
	refereeSM.AddTransition("BALLPLACEMENT", CONTINUE, freeKick)
	refereeSM.AddTransition("BALLPLACEMENT", FREE_KICK, freeKick)
	refereeSM.AddTransition("BALLPLACEMENT", FORCE_START, running)
	refereeSM.AddTransition("BALLPLACEMENT", PREPARE_KICKOFF, prepareKickoff)
	refereeSM.AddTransition("BALLPLACEMENT", PREPARE_PENALTY, preparePenalty)

	refereeSM.AddTransition("PREPAREKICKOFF", NORMAL_START, kickOff)
	refereeSM.AddTransition("PREPAREPENALTY", NORMAL_START, penalty)
	refereeSM.AddTransition("PREPAREPENALTY", STOP, stop)

	refereeSM.AddTransition("PENALTY", STOP, stop)

	refereeSM.AddTransition("KICKOFF", GAME_RUNNING_DETECTED, running)
	refereeSM.AddTransition("KICKOFF", STOP, stop)

	refereeSM.AddTransition("FREEKICK", GAME_RUNNING_DETECTED, running)
	refereeSM.AddTransition("FREEKICK", STOP, stop)

	refereeSM.AddTransition("RUNNING", STOP, stop)

	// All states go to halt
	refereeSM.AddTransition("STOP", HALT, halt)
	refereeSM.AddTransition("BALLPLACEMENT", HALT, halt)
	refereeSM.AddTransition("PREPAREKICKOFF", HALT, halt)
	refereeSM.AddTransition("PREPAREPENALTY", HALT, halt)
	refereeSM.AddTransition("PENALTY", HALT, halt)
	refereeSM.AddTransition("KICKOFF", HALT, halt)
	refereeSM.AddTransition("FREEKICK", HALT, halt)
	refereeSM.AddTransition("RUNNING", HALT, halt)

	// When starting the state is unknown, try parse the latest ref command
	refereeSM.AddTransition(uninitialized.GetName(), PREPARE_KICKOFF, prepareKickoff)
	refereeSM.AddTransition(uninitialized.GetName(), PREPARE_PENALTY, preparePenalty)
	refereeSM.AddTransition(uninitialized.GetName(), STOP, stop)
	refereeSM.AddTransition(uninitialized.GetName(), HALT, halt)
	refereeSM.AddTransition(uninitialized.GetName(), NORMAL_START, running)
	refereeSM.AddTransition(uninitialized.GetName(), TIMEOUT, stop)
	refereeSM.AddTransition(uninitialized.GetName(), GAME_RUNNING_DETECTED, running)
	refereeSM.AddTransition(uninitialized.GetName(), FORCE_START, running)
	refereeSM.AddTransition(uninitialized.GetName(), FREE_KICK, freeKick)
	refereeSM.AddTransition(uninitialized.GetName(), BALL_PLACEMENT, ballPlacement)

	return &RefereeHandler{
		gi:              gi,
		refereeSM:       refereeSM,
		activeRobots:    activeRobots,
		kickOff:         kickOff,
		freeKick:        freeKick,
		stop:            stop,
		activityHandler: activityHandler,
		stateInfos: []*RefereeInfo{
			&freeKick.RefereeInfo,
			&prepareKickoff.RefereeInfo,
			&stop.RefereeInfo,
			&ballPlacement.RefereeInfo,
			&halt.RefereeInfo,
			&kickOff.RefereeInfo,
			&preparePenalty.RefereeInfo,
			&penalty.RefereeInfo,
		},
	}
}

// DelegateStopPositioning leaves STOP formation activities to the caller while
// the referee state machine continues to process transitions normally. Other
// stopped-style states, such as ball placement, keep their dedicated handling.
func (s *RefereeHandler) DelegateStopPositioning() {
	if s != nil && s.stop != nil {
		s.stop.positioningDelegated = true
	}
}

// UpdateActiveRobots keeps every referee state synchronized with current
// vision membership. A substituted-out robot must not retain its old activity,
// while its replacement must be eligible for goalkeeper selection immediately.
func (s *RefereeHandler) UpdateActiveRobots(activeRobots []info.ID) {
	for _, oldID := range s.activeRobots {
		if robotIDInList(activeRobots, oldID) {
			continue
		}
		if s.activityHandler != nil {
			s.activityHandler.ClearActivity(oldID)
		}
	}

	s.activeRobots = append(s.activeRobots[:0], activeRobots...)
	for _, stateInfo := range s.stateInfos {
		stateInfo.activeRobots = s.activeRobots
	}
}

/*
Returns true if referee is being handeled
Returns false if game is in running state
*/
func (s *RefereeHandler) HandleReferee() bool {

	// Rules to follow are defined in the ssl Rules
	// Appendix B: Game States https://robocup-ssl.github.io/ssl-rules/sslrules.html

	gameEvent := s.gi.Status.GetGameEvent()
	fmt.Println(gameEvent)
	refEvent := s.refEventForGameEvent(gameEvent)
	s.refereeSM.TriggerEvent(EventName(refEvent))

	stateBeforeUpdate := s.refereeSM.CurrentStateName()
	if stateBeforeUpdate != "RUNNING" && stateBeforeUpdate != "KICKOFF" {
		s.kickoffTouchRestriction.clear()
	}

	s.refereeSM.Update()
	stateAfterUpdate := s.refereeSM.CurrentStateName()
	if stateBeforeUpdate == "KICKOFF" && stateAfterUpdate == "RUNNING" &&
		s.kickOff != nil && s.kickOff.kickTaken && gameEvent.TeamWithPossession == s.kickOff.team {
		s.kickoffTouchRestriction.arm(s.kickOff.team, s.kickOff.kickOffID)
	}
	if stateBeforeUpdate == "FREEKICK" && stateAfterUpdate == "RUNNING" &&
		s.freeKick != nil && s.freeKick.freeKick != nil && s.freeKick.kickTaken &&
		gameEvent.TeamWithPossession == s.freeKick.team {
		s.kickoffTouchRestriction.arm(s.freeKick.team, s.freeKick.kickerID)
	}
	if stateAfterUpdate == "RUNNING" {
		s.kickoffTouchRestriction.update(s.gi)
		return false
	}

	return true
}

// KickoffRestrictedRobot returns the restart kicker that may not touch the ball
// again until another robot has touched it. It applies to kickoffs and free kicks.
func (s *RefereeHandler) KickoffRestrictedRobot() (info.ID, bool) {
	return s.kickoffTouchRestriction.robot()
}

func (s *RefereeHandler) refEventForGameEvent(gameEvent *info.GameEvent) string {
	refereeCommand := gameEvent.RefCommand
	if isBallPlacementCommand(refereeCommand) && gameEvent.NextCommand != info.UNINITIALIZED {
		s.nextCommandAfterBallPlacement = gameEvent.NextCommand
	}

	if s.refereeSM.CurrentStateName() == "BALLPLACEMENT" &&
		s.nextCommandAfterBallPlacement != info.UNINITIALIZED &&
		refereeCommand == s.nextCommandAfterBallPlacement {
		return refCommandToEventName(s.nextCommandAfterBallPlacement)
	}

	return refCommandToEventName(refereeCommand)
}
