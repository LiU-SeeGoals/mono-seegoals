package roles

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	kickBallSpeedMMPerS    = 4000.0
	opponentReachMMPerS    = 1500.0
	kickLaneBaseRadius     = 90.0 + info.BallRadius + 30.0
	maxVelocityPredictionS = 0.5
	robotVelocityToMMPerS  = 1000.0

	supportGoalSightClearance = 400.0
	supportGridStep           = 400.0
	supportFieldMargin        = pathplanner.RobotSafetyRadius
	supportDefenseAreaMargin  = 200.0
	supportMinBallDistance    = 1000.0
	supportMaxBehindBall      = 1000.0

	supportClearanceCap        = 1000.0
	supportBlockedLaneFactor   = 3.0
	supportOpponentDistanceCap = 1500.0
	supportTeammateSpacing     = 1000.0
	supportIdealPassLength     = 3000.0
	supportIdealShotDistance   = 2500.0

	supportPassClearanceWeight = 1.0
	supportShotClearanceWeight = 1.0
	supportFreedomWeight       = 0.5
	supportGoalDistanceWeight  = 0.3
	supportPassLengthWeight    = 0.2
	supportTravelWeight        = 0.1
	supportCrowdingWeight      = 1.0

	supportStickRadius = 250.0
	supportStickBonus  = 300.0
)

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

type laneRobot struct {
	id  info.ID
	pos info.Vec2
	vel info.Vec2
}

func snapshotRobots(robots *info.RobotTeam) []laneRobot {
	if robots == nil {
		return nil
	}
	snapshot := make([]laneRobot, 0, len(robots))
	for id, robot := range robots {
		if robot == nil || !robot.IsActive() {
			continue
		}
		pos, err := robot.GetPosition()
		if err != nil {
			continue
		}
		vel := robot.GetVelocity()
		snapshot = append(snapshot, laneRobot{
			id:  info.ID(id),
			pos: pos.ToV2(),
			vel: info.Vec2{X: vel.X * robotVelocityToMMPerS, Y: vel.Y * robotVelocityToMMPerS},
		})
	}
	return snapshot
}

func kickLaneClearance(robots []laneRobot, from, target info.Position, reachMMPerS float64) float64 {
	minMargin := math.Inf(1)
	a, b := from.ToV2(), target.ToV2()
	if info.DistV2(a, b) < 1 {
		return minMargin
	}
	for _, robot := range robots {
		along := info.DistV2(a, info.PointToLineSegment(a, b, robot.pos))
		tBall := along / kickBallSpeedMMPerS
		tPredict := math.Min(tBall, maxVelocityPredictionS)
		predicted := info.Vec2{X: robot.pos.X + robot.vel.X*tPredict, Y: robot.pos.Y + robot.vel.Y*tPredict}
		margin := info.DistToLineSegment(a, b, predicted) - (kickLaneBaseRadius + reachMMPerS*tBall)
		if margin < minMargin {
			minMargin = margin
		}
	}
	return minMargin
}

func nearestRobotDistance(robots []laneRobot, spot info.Position) float64 {
	nearest := math.Inf(1)
	for _, robot := range robots {
		nearest = math.Min(nearest, info.DistV2(spot.ToV2(), robot.pos))
	}
	return nearest
}

func supportLaneScore(margin float64) float64 {
	if margin < 0 {
		return margin * supportBlockedLaneFactor
	}
	return math.Min(margin, supportClearanceCap)
}

func supportPositionBlocksGoalSight(ballPos, goalPos, supportPos info.Position) bool {
	distance := info.DistToLineSegment(goalPos.ToV2(), ballPos.ToV2(), supportPos.ToV2())
	return distance < supportGoalSightClearance
}

type SupportAttackIntent struct {
	gi            *info.GameInfo
	team          info.Team
	id            info.ID
	slot          OffenseSlot
	hasLastTarget bool
	lastTarget    info.Position
}

func (kr *SupportAttackIntent) SetSlot(slot OffenseSlot) {
	if slot.Kind != kr.slot.Kind {
		kr.hasLastTarget = false
	}
	kr.slot = slot
}

func (kr *SupportAttackIntent) attackSign(goalPos info.Position) float64 {
	if goalPos.X > 0 {
		return 1
	}
	if goalPos.X < 0 {
		return -1
	}
	return -kr.gi.OwnHalfXSign(kr.team)
}

func (kr *SupportAttackIntent) supportCandidates(currentPos info.Position) []info.Position {
	minX, maxX, minY, maxY, ok := kr.gi.FieldBounds(supportFieldMargin)
	if !ok {
		return nil
	}
	candidates := make([]info.Position, 0, 256)
	for x := minX; x <= maxX; x += supportGridStep {
		for y := minY; y <= maxY; y += supportGridStep {
			candidates = append(candidates, info.Position{X: x, Y: y})
		}
	}
	candidates = append(candidates, kr.gi.ClampToField(currentPos, supportFieldMargin))
	if kr.hasLastTarget {
		candidates = append(candidates, kr.gi.ClampToField(kr.lastTarget, supportFieldMargin))
	}
	return candidates
}

func supportSpotAllowed(spot, ballPos, goalPos info.Position, sign float64, defenseAreas []info.DefenseArea) bool {
	if sign*(spot.X-ballPos.X) < -supportMaxBehindBall {
		return false
	}
	if spot.Dist2d(ballPos) < supportMinBallDistance {
		return false
	}
	if supportPositionBlocksGoalSight(ballPos, goalPos, spot) {
		return false
	}
	for _, area := range defenseAreas {
		minX := math.Min(area.FrontX, area.BackX) - supportDefenseAreaMargin
		maxX := math.Max(area.FrontX, area.BackX) + supportDefenseAreaMargin
		if spot.X >= minX && spot.X <= maxX &&
			spot.Y >= area.MinY-supportDefenseAreaMargin && spot.Y <= area.MaxY+supportDefenseAreaMargin {
			return false
		}
	}
	return true
}

type supportScene struct {
	currentPos   info.Position
	ballPos      info.Position
	goalPos      info.Position
	goalTargets  []info.Position
	opponents    []laneRobot
	teammates    []laneRobot
	defenseAreas []info.DefenseArea
	sign         float64
}

func (kr *SupportAttackIntent) newSupportScene(currentPos, ballPos, goalPos info.Position) supportScene {
	teammates := snapshotRobots(kr.gi.State.GetTeam(kr.team))
	others := teammates[:0]
	for _, robot := range teammates {
		if robot.id != kr.id {
			others = append(others, robot)
		}
	}
	return supportScene{
		currentPos:   currentPos,
		ballPos:      ballPos,
		goalPos:      goalPos,
		goalTargets:  goalMouthTargets(kr.team, kr.gi),
		opponents:    snapshotRobots(kr.gi.State.GetOtherTeam(kr.team)),
		teammates:    others,
		defenseAreas: kr.gi.DefenseAreas(),
		sign:         kr.attackSign(goalPos),
	}
}

func (kr *SupportAttackIntent) scoreSupportSpot(spot info.Position, scene *supportScene) float64 {
	pass := supportLaneScore(kickLaneClearance(scene.opponents, scene.ballPos, spot, opponentReachMMPerS))

	shot := 0.0
	if len(scene.goalTargets) > 0 {
		shot = math.Inf(-1)
		for _, target := range scene.goalTargets {
			shot = math.Max(shot, supportLaneScore(kickLaneClearance(scene.opponents, spot, target, opponentReachMMPerS)))
		}
	}

	freedom := math.Min(nearestRobotDistance(scene.opponents, spot), supportOpponentDistanceCap)

	crowding := 0.0
	for _, mate := range scene.teammates {
		crowding += math.Max(0, supportTeammateSpacing-info.DistV2(spot.ToV2(), mate.pos))
	}

	goalPenalty := math.Max(0, spot.Dist2d(scene.goalPos)-supportIdealShotDistance)
	passLengthPenalty := math.Max(0, spot.Dist2d(scene.ballPos)-supportIdealPassLength)
	travel := spot.Dist2d(scene.currentPos)

	score := supportPassClearanceWeight*pass +
		supportShotClearanceWeight*shot +
		supportFreedomWeight*freedom -
		supportGoalDistanceWeight*goalPenalty -
		supportPassLengthWeight*passLengthPenalty -
		supportTravelWeight*travel -
		supportCrowdingWeight*crowding

	if kr.hasLastTarget && spot.Dist2d(kr.lastTarget) <= supportStickRadius {
		score += supportStickBonus
	}
	return score
}

func (kr *SupportAttackIntent) bestSupportSpot(currentPos, ballPos, goalPos info.Position) (info.Position, bool) {
	scene := kr.newSupportScene(currentPos, ballPos, goalPos)

	var best info.Position
	bestScore := math.Inf(-1)
	found := false
	for _, spot := range kr.supportCandidates(currentPos) {
		if !supportSpotAllowed(spot, ballPos, goalPos, scene.sign, scene.defenseAreas) {
			continue
		}
		score := kr.scoreSupportSpot(spot, &scene)
		if !found || score > bestScore {
			best = spot
			bestScore = score
			found = true
		}
	}
	return best, found
}

func (kr *SupportAttackIntent) GetFromPosition() info.Position {
	currentPos, err := kr.gi.State.GetRobotPosition(kr.team, kr.id)
	if err != nil {
		return info.Position{}
	}

	ballPos, err := kr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil || !kr.gi.HasField() {
		return currentPos
	}

	goalPos := kr.gi.EnemyGoalCenter(kr.team)
	target, ok := kr.bestSupportSpot(currentPos, ballPos, goalPos)
	if !ok {
		kr.hasLastTarget = false
		currentPos.Angle = currentPos.AngleToPosition(ballPos)
		return currentPos
	}

	target.Z = 0
	target.Angle = target.AngleToPosition(ballPos)
	kr.lastTarget = target
	kr.hasLastTarget = true
	return target
}

func (kr *SupportAttackIntent) GetTargetPosition() info.Position {
	pos, _ := kr.gi.State.GetBall().GetEstimatedPosition()
	return pos
}
