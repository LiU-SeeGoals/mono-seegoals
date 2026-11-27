package referee

import (
	"fmt"
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
	"github.com/LiU-SeeGoals/controller/internal/rules"
	"math"
	"sort"
	"time"
)

type FreeKick struct {
	RefereeInfo
	freeKick          *state_machine.StateMachine
	kickerID          info.ID
	freeKickStart     time.Time
	kickTaken         bool
	postKickStart     time.Time
	postKickHeading   float64
	postKickHeadingOK bool
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
	s.freeKickStart = s.gi.Now()
	s.freeKick = nil
	s.kickTaken = false
	s.postKickStart = time.Time{}
	s.postKickHeading = 0
	s.postKickHeadingOK = false

	ballPos := kickoffBallPosition(s.gi)
	kickerID, ok := selectFreeKickKicker(s.gi, s.team, s.activeRobots, ballPos)
	if ok {
		s.kickerID = kickerID
		kickPrepareName := state_machine.StateName(fmt.Sprintf("KickPrepare ID %d", kickerID))
		kickName := state_machine.StateName(fmt.Sprintf("Kick ID %d", kickerID))
		freeKick := FreeKickIntent{gi: s.gi, team: s.team, id: kickerID}
		prepareKick := &roles.AlignState{Ctx: &freeKick, Gi: s.gi, Team: s.team, RobotId: kickerID, Name: kickPrepareName, ActivityHandler: s.activityHandler, PlanApproach: true}
		kick := &roles.KickState{Ctx: &freeKick, Gi: s.gi, Team: s.team, RobotId: kickerID, Name: kickName, ActivityHandler: s.activityHandler}

		s.freeKick = state_machine.NewStateMachine(prepareKick)
		s.freeKick.AddTransition(kickPrepareName, roles.EventAligned, kick)
		s.freeKick.AddTransition(kickName, roles.EventMisaligned, prepareKick)
	}
}

func (s *FreeKick) Update() state_machine.EventName {

	gameEvent := s.gi.Status.GetGameEvent()
	if !s.kickTaken && gameEvent.BallInPlay {
		if !gameEvent.RestartKicked {
			return GAME_RUNNING_DETECTED // timed out
		}
		s.kickTaken = true
		s.postKickStart = s.gi.Now()
		s.postKickHeading, s.postKickHeadingOK = restartKickerHeading(s.gi, s.team, s.kickerID)
	}
	if s.kickTaken {
		if postKickHoldComplete(s.gi, s.postKickStart) {
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
		return state_machine.NoEvent
	}

	if gameEvent.TeamWithPossession == s.team {
		if s.freeKick != nil {
			s.freeKick.Update()
		}
	} else {
		moveRobotsToDefensePosition(s.gi, s.activeRobots, s.team, s.activityHandler)
	}

	return state_machine.NoEvent
}

func moveRobotsToDefensePosition(
	gi *info.GameInfo,
	activeRobots []info.ID,
	team info.Team,
	activityHandler *ai.ActivityHandler,
) {
	goalieID, hasGoalie := selectGoalieID(gi, team, activeRobots)
	if hasGoalie {
		activityHandler.AddActivity(activity.NewGoalie(team, goalieID))
	}

	defenders := fieldRobots(activeRobots, goalieID, hasGoalie)
	ballPos := kickoffBallPosition(gi)
	wallSlots := freeKickDefenseSlots(gi, team, ballPos, len(defenders))
	wallRobots, reserveRobots := assignFreeKickWallRobots(gi, team, defenders, wallSlots)
	for i, robotID := range wallRobots {
		activityHandler.AddActivity(activity.NewMoveToPosition(team, robotID, wallSlots[i]))
	}

	reserveSlots := freeKickReserveSlots(gi, team, ballPos, len(reserveRobots))
	for i, robotID := range reserveRobots {
		activityHandler.AddActivity(activity.NewMoveToPosition(team, robotID, reserveSlots[i]))
	}
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
			if distance < rules.FreeKickWallBallClearance {
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

func freeKickSlotsLegal(gi *info.GameInfo, ballPos info.Position, slots []info.Position) bool {
	for _, slot := range slots {
		if slot.Dist2d(ballPos) < rules.FreeKickWallBallClearance || !freeKickSlotLegal(gi, slot) {
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
	if math.Abs(slot.X) > halfLength-rules.FreeKickFieldClearance ||
		math.Abs(slot.Y) > halfWidth-rules.FreeKickFieldClearance {
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
	minX := math.Min(goalLineX, frontX) - rules.DefenseAreaStoppedClearance
	maxX := math.Max(goalLineX, frontX) + rules.DefenseAreaStoppedClearance
	minY := -halfPenaltyWidth - rules.DefenseAreaStoppedClearance
	maxY := halfPenaltyWidth + rules.DefenseAreaStoppedClearance
	return slot.X >= minX && slot.X <= maxX && slot.Y >= minY && slot.Y <= maxY
}

func clampFreeKickSlot(gi *info.GameInfo, slot info.Position) info.Position {
	if gi == nil || !gi.HasField() {
		return slot
	}
	slot = gi.ClampToField(slot, rules.FreeKickFieldClearance)
	geometry, ok := gi.FieldGeometry()
	if !ok {
		return slot
	}

	for _, goalSign := range []float64{-1, 1} {
		if !insideInflatedDefenseArea(slot, geometry, goalSign) {
			continue
		}
		frontX := goalSign * (geometry.Length/2 - geometry.PenaltyAreaDepth)
		slot.X = frontX - goalSign*rules.DefenseAreaStoppedClearance
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

	move := activity.NewMoveToPosition(team, kickerID, freeKickPreparePosition(gi, team, ballPos))
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
	if gameEvent.IsPlacementStop() {
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
	preparePos.X -= direction.X * rules.FreeKickPrepareDistance
	preparePos.Y -= direction.Y * rules.FreeKickPrepareDistance
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
