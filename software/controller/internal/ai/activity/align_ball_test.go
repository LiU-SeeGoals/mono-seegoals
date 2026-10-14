package ai

import (
	"math"
	"sync"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

var alignKickTarget = info.Position{X: 3000}

func alignScene() *info.GameInfo {
	gi := info.NewGameInfo(10)
	gi.State.SetBall(0, 0, 0, 1000)
	return gi
}

func visionFrameAt(gi *info.GameInfo, now int64, robots map[info.Team]map[uint32]info.Position) {
	for team, byID := range robots {
		for id, pos := range byID {
			gi.State.SetRobotFromVision(team, id, pos, now, 0)
		}
	}
	gi.State.SetMessageReceivedTime(now)
	gi.State.Update()
}

func blueRobot(pos info.Position) map[info.Team]map[uint32]info.Position {
	return map[info.Team]map[uint32]info.Position{info.Blue: {1: pos}}
}

func TestAlignAchievedRequiresCurrentBall(t *testing.T) {
	lineup := info.Position{X: -130}

	gi := alignScene()
	visionFrameAt(gi, 1000, blueRobot(lineup))
	if !NewAlign(info.Blue, 1, alignKickTarget, info.Position{}).Achieved(gi) {
		t.Fatal("expected alignment against a ball vision sees")
	}

	visionFrameAt(gi, 1200, blueRobot(lineup))
	if NewAlign(info.Blue, 1, alignKickTarget, info.Position{}).Achieved(gi) {
		t.Fatal("expected no alignment against a ball vision lost 200 ms ago")
	}
}

func TestAlignAchievedTrustsBallHiddenInOwnDribbler(t *testing.T) {
	gi := alignScene()
	visionFrameAt(gi, 1200, blueRobot(info.Position{X: -110}))
	if kind, _ := gi.State.GetBall().GetEstimateKind(); kind != info.BallInDribbler {
		t.Fatalf("setup: estimate kind = %d, want BallInDribbler", kind)
	}
	if !NewAlign(info.Blue, 1, alignKickTarget, info.Position{}).Achieved(gi) {
		t.Fatal("expected alignment with the ball hidden in the robot's own dribbler")
	}
}

func TestAlignAchievedRejectsBallHiddenInOtherDribbler(t *testing.T) {
	gi := alignScene()
	visionFrameAt(gi, 1200, map[info.Team]map[uint32]info.Position{
		info.Blue:   {1: {X: -130}},
		info.Yellow: {2: {X: 105, Angle: math.Pi}},
	})
	if NewAlign(info.Blue, 1, alignKickTarget, info.Position{}).Achieved(gi) {
		t.Fatal("expected no alignment while an opponent's dribbler hides the ball")
	}
}

func TestAlignHeadingStableWhileBallSpeedHoversAtOldCutoff(t *testing.T) {
	gi := alignScene()
	visionFrameAt(gi, 1000, blueRobot(info.Position{X: 1500}))
	latch := &AlignLatch{}

	heading := func(speed float64) float64 {
		gi.State.SetTrackedBall(info.Position{}, info.Position{X: speed}, 0)
		align := NewDirectAlign(info.Blue, 1, alignKickTarget, info.Position{})
		align.SetLatch(latch)
		return align.GetAction(gi).(*action.MoveTo).Dest.Angle
	}

	lying := heading(0)
	for i, speed := range []float64{0.32, 0.28, 0.35, 0.27, 0.31, 0.29} {
		if got := heading(speed); math.Abs(info.NormalizeAngleDelta(got, lying)) > 1e-9 {
			t.Fatalf("frame %d: speed %.2f m/s moved the commanded heading from %.2f to %.2f rad", i, speed, lying, got)
		}
	}

	towardBall := math.Pi
	for _, speed := range []float64{0.6, 0.3} {
		if got := heading(speed); math.Abs(info.NormalizeAngleDelta(got, towardBall)) > 1e-9 {
			t.Fatalf("speed %.2f m/s: commanded heading %.2f rad, want %.2f toward the ball", speed, got, towardBall)
		}
	}
}

func TestAlignLatchSharedByConcurrentAlignBalls(t *testing.T) {
	gi := alignScene()
	visionFrameAt(gi, 1000, blueRobot(info.Position{X: -300}))
	gi.State.SetTrackedBall(info.Position{}, info.Position{X: 0.3}, 0)
	latch := &AlignLatch{}

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			align := NewDirectAlign(info.Blue, 1, alignKickTarget, info.Position{})
			align.SetLatch(latch)
			for j := 0; j < 50; j++ {
				align.Achieved(gi)
				align.GetAction(gi)
			}
		}()
	}
	wg.Wait()
}
