package referee

import (
	"fmt"
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
	"github.com/LiU-SeeGoals/controller/internal/rules"
	"time"
)

type PrepareKickoff struct {
	RefereeInfo
	kickOffID  info.ID
	receiverID info.ID
}

type Kickoff struct {
	RefereeInfo
	kickOff           *state_machine.StateMachine
	kickOffID         info.ID
	receiverID        info.ID
	kickTaken         bool
	postKickStart     time.Time
	postKickHeading   float64
	postKickHeadingOK bool
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

func (s *PrepareKickoff) Initialize() {

	goalieID, hasGoalie := selectGoalieID(s.gi, s.team, s.activeRobots)
	kickOffID, receiverID := selectKickoffRobots(fieldRobots(s.activeRobots, goalieID, hasGoalie))
	s.kickOffID = kickOffID
	s.receiverID = receiverID

}

func (s *PrepareKickoff) Update() state_machine.EventName {

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

	return state_machine.NoEvent
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

	kickPrepareName := state_machine.StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))
	kickName := state_machine.StateName(fmt.Sprintf("Kick ID %d", kickOffID))

	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickName, ActivityHandler: s.activityHandler}

	s.kickOff = state_machine.NewStateMachine(prepareKick)
	s.kickOff.AddTransition(kickPrepareName, roles.EventAligned, kick)
	s.kickOff.AddTransition(kickName, roles.EventMisaligned, prepareKick)

	// If we dont have possession we take good defense position
}

func (s *Kickoff) Update() state_machine.EventName {

	gameEvent := s.gi.Status.GetGameEvent()
	if !s.kickTaken && gameEvent.BallInPlay {
		if !gameEvent.RestartKicked {
			return GAME_RUNNING_DETECTED // timed out
		}
		s.kickTaken = true
		s.postKickStart = s.gi.Now()
		s.postKickHeading, s.postKickHeadingOK = restartKickerHeading(s.gi, s.team, s.kickOffID)
	}
	if s.kickTaken {
		if postKickHoldComplete(s.gi, s.postKickStart) {
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
		return state_machine.NoEvent
	}

	goalieID, hasGoalie := moveGoalieToPosition(s.activeRobots, s.team, s.gi, s.activityHandler)
	if gameEvent.TeamWithPossession == s.team {
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

	return state_machine.NoEvent
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

func moveKickerToKickoffPreparePosition(
	team info.Team,
	gi *info.GameInfo,
	activityHandler *ai.ActivityHandler,
	kickOffID info.ID,
) {
	activityHandler.AddActivity(activity.NewMoveToPosition(team, kickOffID, kickoffPreparePosition(gi, team)))
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
		activityHandler.AddActivity(activity.NewMoveToPosition(team, robotID, slots[i]))
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
		activityHandler.AddActivity(activity.NewMoveToPosition(team, robotID, slots[i]))
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
			rules.KickoffCenterCircleClearance+200,
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

func keepOutsideKickoffCenterCircle(pos info.Position, xSign float64) info.Position {
	if pos.Norm2d() >= rules.KickoffCenterCircleClearance {
		return pos
	}

	pos.X = xSign * rules.KickoffCenterCircleClearance
	return pos
}
