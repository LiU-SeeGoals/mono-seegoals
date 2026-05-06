package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const RRT = true

// Hysteresis for the look-ahead destination so it does not jump every frame when
// the path wiggles slightly (reduces in-front / behind oscillation on the polyline).
const (
	stickyAdvanceMM   = 95.0  // robot this close to current carrot -> advance to new look-ahead
	stickyHysteresis  = 170.0 // keep previous carrot if new candidate is within this of it
	stickyMinPursueMM = 40.0  // if closer than this to carrot, do not keep stale; take fresh cand
)

// pathLookID is a cheap fingerprint to reset sticky look-ahead when the plan changes.
type pathLookID struct {
	n              int
	ax, ay, bx, by int64 // deci-mm of first/last vertex
}

func makePathLookID(p []info.Position) pathLookID {
	if len(p) == 0 {
		return pathLookID{}
	}
	a, b := p[0], p[len(p)-1]
	return pathLookID{
		n:  len(p),
		ax: int64(a.X * 10), ay: int64(a.Y * 10),
		bx: int64(b.X * 10), by: int64(b.Y * 10),
	}
}

// MoveToPositionWithCollisionAvoidance handles collision avoidance using RRT
type MoveToPosition struct {
	team              info.Team
	id                info.ID
	final_destination info.Position
	path              []info.Position
	rrtConfig         pathplanner.RRTConfig
	gi                *info.GameInfo
	avoidBall         bool
	dribble           bool
	lastPosition      info.Position // Last position to detect lack of movement
	stuckThreshold    int           // Number of cycles to consider robot as stuck
	useRRT            bool          // Flag to enable/disable RRT-based collision avoidance
	// Sticky look-ahead (avoids flicker from per-tick reprojection)
	stickyDest   info.Position
	stickySet    bool
	stickyPathID pathLookID
}

// NewMoveToPosition creates a new instance.
func NewMoveToPosition(team info.Team, id info.ID, dest info.Position) *MoveToPosition {
	rrtConfig := pathplanner.RRTConfig{
		MaxIterations:      1000,
		StepSize:           50.0,
		GoalBias:           0.05,
		WaypointThreshold:  600.0,
		FieldWidth:         9000.0,
		FieldHeight:        6000.0,
		CompletionDistance: 50.0,
	}
	return &MoveToPosition{
		team:              team,
		id:                id,
		final_destination: dest,
		path:              []info.Position{},
		rrtConfig:         rrtConfig,
		useRRT:            true, // Enable RRT by default
		avoidBall:         true, // Enable ball avoidance by default
	}
}

func (m *MoveToPosition) SetUseRRT(use bool) {
	m.useRRT = use
}

func (m *MoveToPosition) AvoidBall(avoid bool) {
	m.avoidBall = avoid
}

func (m *MoveToPosition) SetDribble(dribble bool) {
	m.dribble = dribble
}

// GetAction returns an action for the robot with RRT-based collision avoidance
func (m *MoveToPosition) GetAction(gi *info.GameInfo) action.Action {
	moveToAction := m.GetMoveToAction(gi)
	m.gi = gi
	return moveToAction
}

func (m *MoveToPosition) GetMoveToAction(gi *info.GameInfo) *action.MoveTo {
	myRobot := gi.State.GetTeam(m.team)[m.id]
	myPos, _ := myRobot.GetPosition()

	if m.closeEnoughToGoal(gi) {
		act := &action.MoveTo{}
		act.Id = int(m.id)
		act.Team = m.team
		act.Pos = myPos
		act.Dest = m.final_destination
		act.Dest.Angle = m.final_destination.Angle
		act.Dribble = false
		return act
	}

	var targetPos info.Position

	if m.useRRT {
		m.rrtConfig.StepSize = min(max(myPos.Dist2d(m.final_destination)/100, 1), m.rrtConfig.StepSize)
		// fmt.Println(m.rrtConfig.StepSize)
		// fmt.Println(m.final_destination)
		// fmt.Println(myPos)

		targetPos = myPos

		ps := getPathService(m.team)
		if ps != nil {
			cfg := m.rrtConfig
			m.path = ps.PlanPath(m.team, m.id, m.final_destination, m.avoidBall, gi, cfg)
		} else {
			m.path = []info.Position{m.final_destination}
		}

		obstacles := pathplanner.ObstaclesForRobot(m.team, m.id, m.avoidBall, gi)
		if len(m.path) > 0 {
			cand := m.pickLookAheadTarget(myPos, obstacles)
			targetPos = m.applyStickyLookAhead(myPos, cand)
		}
	} else {
		targetPos = m.final_destination
	}

	act := &action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = myPos
	act.Dest = targetPos
	act.Dest.Angle = m.final_destination.Angle
	act.Dribble = m.dribble
	// Include the full planned path for visualization in the GameViewer.
	// Note: this is a copy so UI serialization can't race on m.path.
	if len(m.path) > 0 {
		act.Path = pathFromRobotProjection(myPos, m.path)
	}
	return act
}

func (m *MoveToPosition) closeEnoughToGoal(gi *info.GameInfo) bool {
	currPos, _ := gi.State.GetTeam(m.team)[m.id].GetPosition()
	return pathplanner.DistanceBetween(currPos, m.final_destination) <= m.rrtConfig.CompletionDistance
}

// Achieved returns true if the robot is sufficiently close to the final destination.
// When true, the path planner is cleared for this robot so no stale path state remains.
func (m *MoveToPosition) Achieved(gi *info.GameInfo) bool {
	if m.closeEnoughToGoal(gi) {
		if p := getPathService(m.team); p != nil {
			p.Clear(m.id)
		}
		return true
	}
	return false
}

func (m *MoveToPosition) String() string {
	currPos, _ := m.gi.State.GetTeam(m.team)[m.id].GetPosition()
	return fmt.Sprintf("MoveToPosition: dist%f", pathplanner.DistanceBetween(currPos, m.final_destination))
}

func (m *MoveToPosition) GetID() info.ID {
	return m.id
}

// pickLookAheadTarget uses arc-length along the path (forward from the robot’s projection)
// instead of a ring offset from the polyline, which used to pick points that could sit
// “behind” the robot on another segment and oscillate. Near the goal, snap to the last vertex
// when already inside the waypoint band and the segment is free.
func (m *MoveToPosition) pickLookAheadTarget(myPos info.Position, obstacles []pathplanner.Obstacle) info.Position {
	path := m.path
	last := path[len(path)-1]
	if pathplanner.DistanceBetween(myPos, last) < m.rrtConfig.WaypointThreshold &&
		pathplanner.IsPathClear(myPos, last, obstacles, pathplanner.MotionRadius) {
		out := last
		out.Z = myPos.Z
		out.Angle = myPos.Angle
		return out
	}
	cand, ok := lookAheadAlongPathArclength(myPos, path, m.rrtConfig.WaypointThreshold)
	if !ok {
		cand = last
	}
	cand.Z = myPos.Z
	cand.Angle = myPos.Angle
	return cand
}

func (m *MoveToPosition) applyStickyLookAhead(myPos, cand info.Position) info.Position {
	id := makePathLookID(m.path)
	if !m.stickySet || m.stickyPathID != id {
		m.stickyDest = cand
		m.stickySet = true
		m.stickyPathID = id
		return cand
	}
	dRob := pathplanner.DistanceBetween(myPos, m.stickyDest)
	dCand := pathplanner.DistanceBetween(cand, m.stickyDest)
	if dRob < stickyAdvanceMM {
		m.stickyDest = cand
		return cand
	}
	if dCand < stickyHysteresis && dRob > stickyMinPursueMM {
		return m.stickyDest
	}
	m.stickyDest = cand
	return cand
}

func pathVertexCumulative(path []info.Position) []float64 {
	c := make([]float64, len(path))
	for i := 1; i < len(path); i++ {
		c[i] = c[i-1] + pathplanner.DistanceBetween(path[i-1], path[i])
	}
	return c
}

// projectRobotArcLength returns arc length from path[0] along the polyline to the foot of
// the perpendicular from the robot. Ties in distance prefer the foot farther along the path
// (avoids choosing a segment behind the robot when two are almost equidistant).
func projectRobotArcLength(robot info.Position, path []info.Position, cum []float64) (float64, bool) {
	if len(path) < 2 {
		return 0, len(path) == 1
	}
	r := robot.ToV2()
	bestD := math.MaxFloat64
	bestArc := -1.0
	for i := 0; i < len(path)-1; i++ {
		a := path[i].ToV2()
		b := path[i+1].ToV2()
		ab := info.Sub(b, a)
		ac := info.Sub(r, a)
		denom := info.DotV2(ab, ab)
		t := 0.0
		if denom > 1e-12 {
			t = info.DotV2(ac, ab) / denom
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
		}
		foot := info.Add(a, ab.Mult(t))
		d := info.DistV2(foot, r)
		segLen := math.Sqrt(denom)
		arc := cum[i] + t*segLen
		if d < bestD-1e-4 {
			bestD, bestArc = d, arc
		} else if math.Abs(d-bestD) < 1e-4 && arc > bestArc {
			bestArc = arc
		}
	}
	if bestArc < 0 {
		return 0, false
	}
	return bestArc, true
}

func pointAtArclength(path []info.Position, cum []float64, s float64) info.Position {
	n := len(path)
	if n == 0 {
		return info.Position{}
	}
	if n == 1 || s <= cum[0] {
		return path[0]
	}
	if s >= cum[n-1] {
		return path[n-1]
	}
	for i := 0; i < n-1; i++ {
		if s < cum[i]-1e-6 {
			continue
		}
		if s <= cum[i+1]+1e-6 {
			seg := cum[i+1] - cum[i]
			if seg < 1e-9 {
				return path[i+1]
			}
			t := (s - cum[i]) / seg
			return info.Position{
				X: path[i].X + t*(path[i+1].X-path[i].X),
				Y: path[i].Y + t*(path[i+1].Y-path[i].Y),
				Z: 0,
			}
		}
	}
	return path[n-1]
}

func pathFromRobotProjection(robot info.Position, path []info.Position) []info.Position {
	if len(path) == 0 {
		return nil
	}
	if len(path) == 1 {
		return append([]info.Position(nil), path...)
	}
	cum := pathVertexCumulative(path)
	arc, ok := projectRobotArcLength(robot, path, cum)
	if !ok {
		return append([]info.Position(nil), path...)
	}
	out := make([]info.Position, 0, len(path))
	for i := 1; i < len(path); i++ {
		if cum[i] > arc+1e-6 {
			out = append(out, path[i])
		}
	}
	if len(out) == 0 {
		out = append(out, path[len(path)-1])
	}
	return out
}

// lookAheadAlongPathArclength walks forward by lookMm from the robot’s projection on the path.
func lookAheadAlongPathArclength(robot info.Position, path []info.Position, lookMm float64) (info.Position, bool) {
	if len(path) == 0 {
		return info.Position{}, false
	}
	if len(path) == 1 {
		return path[0], true
	}
	cum := pathVertexCumulative(path)
	total := cum[len(cum)-1]
	arc, ok := projectRobotArcLength(robot, path, cum)
	if !ok {
		return info.Position{}, false
	}
	s := math.Min(arc+lookMm, total)
	p := pointAtArclength(path, cum, s)
	return p, true
}
