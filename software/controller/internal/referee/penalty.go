package referee

import (
	"fmt"
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
	"math"
)

type PreparePenalty struct {
	RefereeInfo
	kickerID  info.ID
	hasKicker bool
}

type Penalty struct {
	RefereeInfo
	penaltyKick       *state_machine.StateMachine
	kickerID          info.ID
	ballInPlay        bool
	penaltyMark       info.Position
	postKickHeading   float64
	postKickHeadingOK bool
}

func (s *PreparePenalty) Initialize() {
	s.kickerID = 0
	s.hasKicker = false
	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		s.kickerID, s.hasKicker = selectPenaltyKicker(s.gi, s.team, s.activeRobots)
	}
}

func (s *PreparePenalty) Update() state_machine.EventName {
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
		move := activity.NewMoveToPosition(s.team, s.kickerID, penaltyKickerPreparePosition(s.gi, s.team))
		move.AllowOutsideField(true)
		s.activityHandler.AddActivity(move)
	}
	return state_machine.NoEvent
}

func (s *Penalty) Initialize() {
	s.penaltyKick = nil
	s.kickerID = 0
	s.ballInPlay = false
	s.postKickHeading = 0
	s.postKickHeadingOK = false
	s.penaltyMark = kickoffBallPosition(s.gi)

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

	kickPrepareName := state_machine.StateName(fmt.Sprintf("PenaltyKickPrepare ID %d", kickerID))
	kickName := state_machine.StateName(fmt.Sprintf("PenaltyKick ID %d", kickerID))
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
	s.penaltyKick = state_machine.NewStateMachine(prepareKick)
	s.penaltyKick.AddTransition(kickPrepareName, roles.EventAligned, kick)
	s.penaltyKick.AddTransition(kickName, roles.EventMisaligned, prepareKick)
}

func (s *Penalty) Update() state_machine.EventName {
	gameEvent := s.gi.Status.GetGameEvent()
	if gameEvent.CurrentState == info.STATE_STOPPED {
		// The penalty was not taken in time (see info.GameEvent.ObserveRestart).
		stopPenaltyRobots(s.activeRobots, s.activityHandler)
		return STOP
	}

	if !s.ballInPlay && gameEvent.BallInPlay {
		s.ballInPlay = true
		s.postKickHeading, s.postKickHeadingOK = restartKickerHeading(s.gi, s.team, s.kickerID)
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
		s.penaltyMark,
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
	return state_machine.NoEvent
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
		goalieID, hasGoalie := SelectGoalieID(gi, team, activeRobots)
		if hasGoalie {
			if goalieFree {
				activityHandler.AddActivity(activity.NewGoalie(team, goalieID))
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
		activityHandler.AddActivity(activity.NewStop(robotID))
	}
}

func movePenaltyGoalieToGoalLine(
	gi *info.GameInfo,
	team info.Team,
	goalieID info.ID,
	activityHandler *ai.ActivityHandler,
) {
	move := activity.NewMoveToPosition(team, goalieID, penaltyGoalieHomePosition(gi, team))
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
		move := activity.NewMoveToPosition(team, robotID, slots[i])
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
