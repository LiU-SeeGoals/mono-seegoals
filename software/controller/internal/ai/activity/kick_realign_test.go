package ai

import (
	"math"
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func kickRealignScene(robot info.Position) (*info.GameInfo, *KickBall) {
	gi := alignScene()
	visionFrameAt(gi, 1000, blueRobot(robot))
	return gi, NewKickBall(info.Blue, 1, alignKickTarget, info.Position{})
}

func TestKickRealignStaysWhileLinedUp(t *testing.T) {
	gi, kick := kickRealignScene(info.Position{X: -130, Y: 10, Angle: 0.05})
	now := time.Now()
	for i := 0; i < 20; i++ {
		if kick.needsRealignAt(gi, now.Add(time.Duration(i)*frameTime)) {
			t.Fatalf("frame %d: lined-up robot asked to realign", i)
		}
	}
}

func TestKickRealignAfterConfirmWhenOffLine(t *testing.T) {
	gi, kick := kickRealignScene(info.Position{X: -130, Y: 80})
	now := time.Now()
	if kick.needsRealignAt(gi, now) {
		t.Fatal("realign fired on the first off-line frame")
	}
	if kick.needsRealignAt(gi, now.Add(kickRealignConfirm-time.Millisecond)) {
		t.Fatal("realign fired before the confirm time")
	}
	if !kick.needsRealignAt(gi, now.Add(kickRealignConfirm)) {
		t.Fatal("robot 80 mm off the kick line never asked to realign")
	}
}

func TestKickRealignWhenRobotPassedBall(t *testing.T) {
	gi, kick := kickRealignScene(info.Position{X: 200})
	now := time.Now()
	kick.needsRealignAt(gi, now)
	if !kick.needsRealignAt(gi, now.Add(kickRealignConfirm)) {
		t.Fatal("robot in front of the ball never asked to realign")
	}
}

func TestKickRealignNotWithBallAtMouth(t *testing.T) {
	heading := 0.3
	reach := kickerStandoffDist(0)
	robot := info.Position{X: -reach * math.Cos(heading), Y: -reach * math.Sin(heading), Angle: heading}
	gi, kick := kickRealignScene(robot)
	now := time.Now()
	kick.needsRealignAt(gi, now)
	if kick.needsRealignAt(gi, now.Add(kickRealignConfirm)) {
		t.Fatal("robot with the ball at its mouth asked to realign")
	}
}

func TestKickRealignTimerResetsOnRecovery(t *testing.T) {
	gi, kick := kickRealignScene(info.Position{X: -130, Y: 80})
	now := time.Now()
	kick.needsRealignAt(gi, now)

	visionFrameAt(gi, 1050, blueRobot(info.Position{X: -130}))
	kick.needsRealignAt(gi, now.Add(kickRealignConfirm/2))

	visionFrameAt(gi, 1100, blueRobot(info.Position{X: -130, Y: 80}))
	if kick.needsRealignAt(gi, now.Add(kickRealignConfirm)) {
		t.Fatal("a brief off-line spike after recovery triggered realign")
	}
}

func TestOrbitFloorKeepsSideSpeedUnderKickLineGate(t *testing.T) {
	ball := info.Position{}
	for y := -alignLineExitTolerance + 1; y < alignLineExitTolerance; y += 2.5 {
		bot := info.Position{X: -kickerStandoffDist(captureMarginToBall), Y: y}
		for _, dest := range []info.Position{
			behindBallDest(ball, alignKickTarget, captureMarginToBall),
			behindBallDest(ball, alignKickTarget, ballPushMargin),
			{X: bot.X + 3, Y: 0},
		} {
			floor := aroundBallLinearSpeed(bot, dest, ball, alignKickTarget)
			if floor == 0 {
				continue
			}
			dist := bot.Dist2d(dest)
			sideSpeed := math.Abs(sideSpeedToLine(info.Position{X: dest.X - bot.X, Y: dest.Y - bot.Y}, ball, alignKickTarget)) / dist * floor
			if sideSpeed >= alignLineMaxSideSpeed {
				t.Fatalf("bot y=%.1f dest %+v: floor drives %.2f m/s sideways, gate needs < %.2f", y, dest, sideSpeed, alignLineMaxSideSpeed)
			}
		}
	}
}

func TestOrbitFloorStaysWhileCircling(t *testing.T) {
	ball := info.Position{}
	bot := info.Position{X: 0, Y: -200}
	dest := aroundBallDest(ball, bot, behindBallDest(ball, alignKickTarget, captureMarginToBall), captureMarginToBall)
	if got := aroundBallLinearSpeed(bot, dest, ball, alignKickTarget); got != aroundBallMinLinearSpeed {
		t.Fatalf("circling floor = %.2f, want %.2f", got, aroundBallMinLinearSpeed)
	}
}

func TestOrbitFloorStaysWhenClosingAlongLine(t *testing.T) {
	ball := info.Position{}
	bot := info.Position{X: -kickerStandoffDist(captureMarginToBall), Y: 3}
	dest := behindBallDest(ball, alignKickTarget, ballPushMargin)
	if got := aroundBallLinearSpeed(bot, dest, ball, alignKickTarget); got != aroundBallMinLinearSpeed {
		t.Fatalf("closing-in floor = %.2f, want %.2f", got, aroundBallMinLinearSpeed)
	}
}

func TestChordClearanceMatchesSampling(t *testing.T) {
	for _, r1 := range []float64{120, 180, 260} {
		for _, r2 := range []float64{110, 150, 230} {
			for angle := 0.05; angle < math.Pi; angle += 0.15 {
				p := info.Position{X: r1}
				q := info.Position{X: r2 * math.Cos(angle), Y: r2 * math.Sin(angle)}
				sampled := math.Inf(1)
				for i := 0; i <= 2000; i++ {
					f := float64(i) / 2000
					sampled = math.Min(sampled, math.Hypot(p.X+f*(q.X-p.X), p.Y+f*(q.Y-p.Y)))
				}
				if got := chordClearance(r1, r2, angle); math.Abs(got-sampled) > 0.5 {
					t.Fatalf("r1=%.0f r2=%.0f angle=%.2f: clearance %.1f, sampled %.1f", r1, r2, angle, got, sampled)
				}
			}
		}
	}
}

func TestOrbitCarrotPathKeepsClearOfBall(t *testing.T) {
	ball := info.Position{}
	lineup := behindBallDest(ball, alignKickTarget, captureMarginToBall)
	for radius := 120.0; radius <= 300; radius += 10 {
		for bearing := -math.Pi; bearing < math.Pi; bearing += 0.1 {
			bot := info.Position{X: radius * math.Cos(bearing), Y: radius * math.Sin(bearing)}
			carrot := aroundBallDest(ball, bot, lineup, captureMarginToBall)
			shift := math.Abs(info.NormalizeAngleDelta(ball.AngleToPosition(carrot), bearing))
			if math.Min(radius, ball.Dist2d(carrot)) <= aroundBallChordClearance {
				if shift > aroundBallInsideShiftAngle+1e-9 {
					t.Fatalf("bot inside clearance at r=%.0f bearing=%.2f shifts %.2f rad", radius, bearing, shift)
				}
				continue
			}
			if d := segmentClearance(bot, carrot, ball); d < aroundBallChordClearance-1e-6 {
				t.Fatalf("bot r=%.0f bearing=%.2f: path to carrot passes %.1f mm from ball, want %.1f", radius, bearing, d, aroundBallChordClearance)
			}
		}
	}
}
