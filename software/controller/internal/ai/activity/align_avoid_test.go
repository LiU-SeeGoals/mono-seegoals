package ai

import (
	"math"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func segmentClearance(a, b, p info.Position) float64 {
	abX, abY := b.X-a.X, b.Y-a.Y
	lenSq := abX*abX + abY*abY
	t := 0.0
	if lenSq > 0 {
		t = math.Max(0, math.Min(1, ((p.X-a.X)*abX+(p.Y-a.Y)*abY)/lenSq))
	}
	return math.Hypot(p.X-(a.X+t*abX), p.Y-(a.Y+t*abY))
}

func TestDeflectKeepsClearPathUnchanged(t *testing.T) {
	myPos := info.Position{X: -400}
	dest := info.Position{X: -140, Angle: 0.3}
	robots := []info.Position{{X: -300, Y: 400}, {X: -900}}
	if got := deflectAroundRobots(myPos, dest, robots); got != dest {
		t.Fatalf("clear path changed to %+v, want %+v", got, dest)
	}
}

func TestDeflectRoutesAroundRobotInTheWay(t *testing.T) {
	myPos := info.Position{X: -600}
	dest := info.Position{X: -140, Y: 20, Angle: 0.3}
	blocker := info.Position{X: -370}
	got := deflectAroundRobots(myPos, dest, []info.Position{blocker})
	if d := segmentClearance(myPos, got, blocker); d < aroundBallRobotClearance {
		t.Fatalf("detour passes %.0f mm from the blocker, want at least %.0f", d, aroundBallRobotClearance)
	}
	if got.Y <= 0 {
		t.Fatalf("detour went to y=%.0f, want the side the destination is on (y>0)", got.Y)
	}
	if got.Angle != dest.Angle {
		t.Fatalf("detour changed heading to %.2f, want %.2f", got.Angle, dest.Angle)
	}
}

func TestDeflectMovesDestinationOffOccupiedSpot(t *testing.T) {
	myPos := info.Position{X: -600}
	dest := info.Position{X: -140}
	occupant := info.Position{X: -100, Y: 30}
	got := deflectAroundRobots(myPos, dest, []info.Position{occupant})
	if d := got.Dist2d(occupant); d < aroundBallRobotClearance-1e-6 {
		t.Fatalf("destination %.0f mm from an occupying robot, want at least %.0f", d, aroundBallRobotClearance)
	}
}

func TestDeflectIgnoresRobotBehind(t *testing.T) {
	myPos := info.Position{X: -300}
	dest := info.Position{X: -140}
	if got := deflectAroundRobots(myPos, dest, []info.Position{{X: -420}}); got != dest {
		t.Fatalf("robot behind changed the destination to %+v", got)
	}
}

func TestDeflectAllowsBallDuel(t *testing.T) {
	myPos := info.Position{X: -110}
	dest := info.Position{X: -kickerStandoffDist(ballPushMargin)}
	opponent := info.Position{X: kickerStandoffDist(0)}
	if got := deflectAroundRobots(myPos, dest, []info.Position{opponent}); got != dest {
		t.Fatalf("opponent touching the far side of the ball changed the destination to %+v", got)
	}
}

func TestAlignOrbitAvoidsTeammateOnTheArc(t *testing.T) {
	gi := alignScene()
	robots := map[info.Team]map[uint32]info.Position{
		info.Blue: {1: {X: 0, Y: -200, Angle: math.Pi / 2}},
	}
	visionFrameAt(gi, 1000, robots)
	free := NewDirectAlign(info.Blue, 1, alignKickTarget, info.Position{}).GetAction(gi).(*action.MoveTo).Dest

	robots[info.Blue][2] = info.Position{
		X: 0.75 * free.X,
		Y: -200 + 0.75*(free.Y+200),
	}
	visionFrameAt(gi, 1016, robots)
	got := NewDirectAlign(info.Blue, 1, alignKickTarget, info.Position{}).GetAction(gi).(*action.MoveTo).Dest

	myPos := info.Position{X: 0, Y: -200}
	if d := segmentClearance(myPos, got, robots[info.Blue][2]); d < aroundBallRobotClearance {
		t.Fatalf("orbit carrot path passes %.0f mm from a teammate, want at least %.0f", d, aroundBallRobotClearance)
	}
}

func TestAlignApproachIgnoresStaleRobots(t *testing.T) {
	gi := alignScene()
	visionFrameAt(gi, 1000, map[info.Team]map[uint32]info.Position{
		info.Yellow: {3: {X: -370}},
	})
	visionFrameAt(gi, 1000+aroundBallRobotMaxAgeMs+50, blueRobot(info.Position{X: -600}))
	if got := nearbyRobotPositions(info.Blue, 1, gi); len(got) != 0 {
		t.Fatalf("got %d nearby robots, want the stale opponent dropped", len(got))
	}
}
