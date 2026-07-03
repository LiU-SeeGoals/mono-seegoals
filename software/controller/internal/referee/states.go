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
	freeKick        *StateMachine
	kickerID        info.ID
	freeKickStart   time.Time
	originalBallPos info.Position
	kickTaken       bool
}

type Stop struct {
	RefereeInfo
}

type PrepareKickoff struct {
	RefereeInfo
	kickOffID  info.ID
	receiverID info.ID
}

type PreparePenalty struct {
	RefereeInfo
}

type Penalty struct {
	RefereeInfo
	penaltyStart    time.Time
	originalBallPos info.Position
}

type Kickoff struct {
	RefereeInfo
	kickOff         *StateMachine
	kickOffID       info.ID
	receiverID      info.ID
	originalBallPos info.Position
	kickStart       time.Time
	kickTaken       bool
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
	goalieID, hasGoalie := moveGoalieToPosition(s.activeRobots, s.team, s.gi, s.activityHandler)
	preparedKickerID, preparingFreeKick := prepareKickerForUpcomingFreeKick(
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

func (s *PreparePenalty) Initialize() {}

func (s *PreparePenalty) Update() EventName {
	moveRobotsToPenaltyPositions(s.gi, s.activeRobots, s.team, s.activityHandler)
	return "NONE"
}

func (s *Penalty) Initialize() {
	s.penaltyStart = time.Now()
	s.originalBallPos = kickoffBallPosition(s.gi)
}

func (s *Penalty) Update() EventName {
	trackedBall := s.gi.State.GetTrackedBall()
	pos, ok := trackedBall.GetTrackedPosition()
	ballMoved := ok && restartBallMovedIntoPlay(s.originalBallPos, pos)
	gameEvent := s.gi.Status.GetGameEvent()
	if gameEvent.BallInPlay || ballMoved ||
		restartActionTimedOut(gameEvent, s.penaltyStart, PenaltyMaxTime(s.gi.Status.GetDivision())) {
		gameEvent.SetBallMoved()
		return GAME_RUNNING_DETECTED
	}

	moveRobotsToPenaltyPositions(s.gi, s.activeRobots, s.team, s.activityHandler)
	return "NONE"
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

	if ballMoved ||
		restartActionTimedOut(s.gi.Status.GetGameEvent(), s.freeKickStart, FreeKickMaxTime(s.gi.Status.GetDivision())) {
		s.kickTaken = ballMoved
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

	if ballMoved ||
		restartActionTimedOut(s.gi.Status.GetGameEvent(), s.kickStart, KickoffMaxTime(s.gi.Status.GetDivision())) {
		s.kickTaken = ballMoved
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
	slots := freeKickDefenseSlots(gi, team, len(defenders))
	for i, robotID := range defenders {
		activityHandler.AddActivity(act.NewMoveToPosition(team, robotID, slots[i]))
	}
}

func moveRobotsToPenaltyPositions(
	gi *info.GameInfo,
	activeRobots []info.ID,
	team info.Team,
	activityHandler *ai.ActivityHandler,
) {
	goalieID, hasGoalie := selectGoalieID(gi, team, activeRobots)
	if hasGoalie {
		move := act.NewMoveToPosition(team, goalieID, penaltyGoalieHomePosition(gi, team))
		move.AvoidGoallines(false)
		activityHandler.AddActivity(move)
	}

	for _, robotID := range fieldRobots(activeRobots, goalieID, hasGoalie) {
		activityHandler.AddActivity(act.NewRefStop(team, robotID))
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

func freeKickDefenseSlots(gi *info.GameInfo, team info.Team, count int) []info.Position {
	const (
		defenseDepth = 2000.0
		spacing      = 300.0
	)

	slots := make([]info.Position, count)
	startY := -spacing * float64(count-1) / 2
	for i := range slots {
		slots[i] = info.Position{
			X: ownHalfXSign(gi, team) * defenseDepth,
			Y: startY + spacing*float64(i),
		}
	}
	return slots
}

func prepareKickerForUpcomingFreeKick(
	gi *info.GameInfo,
	team info.Team,
	activeRobots []info.ID,
	activityHandler *ai.ActivityHandler,
) (info.ID, bool) {
	if gi == nil || gi.Status == nil || activityHandler == nil {
		return 0, false
	}

	gameEvent := gi.Status.GetGameEvent()
	if gameEvent == nil || !isFreeKickForTeam(gameEvent.NextCommand, team) {
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

func isFreeKickForTeam(command info.RefCommand, team info.Team) bool {
	switch command {
	case info.DIRECT_FREE_BLUE, info.INDIRECT_FREE_BLUE:
		return team == info.Blue
	case info.DIRECT_FREE_YELLOW, info.INDIRECT_FREE_YELLOW:
		return team == info.Yellow
	default:
		return false
	}
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

	refereeSM.AddTransition("PENALTY", GAME_RUNNING_DETECTED, running)
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
