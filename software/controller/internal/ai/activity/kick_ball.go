package ai

import (
	"fmt"
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type KickBall struct {
	team           info.Team
	id             info.ID
	orignalBallPos info.Position
	inited         bool
	to             info.Position
	dribbleSince   time.Time
}

type KickConfig struct {
	driveThrough    float64
	doneDist        float64
	ballAbortRadius float64
	kickContactDist float64
}

const (
	kickMouthLateralTolerance = info.DribblerHalfWidth + 1.5*info.BallRadius
	kickDribbleSettleTime     = 250 * time.Millisecond
	kickHeldHeadingTolerance  = 0.4
	kickImpactPreContactDist  = 35.0
)

func GetKickConfig() KickConfig {
	return KickConfig{
		driveThrough:    100,
		doneDist:        20,
		ballAbortRadius: 200,
		kickContactDist: 130,
	}
}

func (m *KickBall) String() string {
	return fmt.Sprintf("KickBall(%d)", m.id)
}

func NewKickBall(team info.Team, id info.ID, to, ballPos info.Position) *KickBall {
	// fmt.Println("New kick ball")
	return &KickBall{
		team,
		id,
		ballPos,
		false,
		to,
		time.Time{},
	}
}

func (m *KickBall) GetTargetPos(gi *info.GameInfo) info.Position {

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	// fmt.Println("new pos", ballPos, m.orignalBallPos, ballPos.Norm2d(m.orignalBallPos))
	ballV2 := info.Vec2{X: ballPos.X, Y: ballPos.Y}

	robot := gi.State.GetTeam(m.team)[m.id]
	robotPos, err := robot.GetPosition()
	if err != nil {
		fmt.Println(err)
	}

	// Assume the robot was aligned prior to kicking
	// Meaning we should keep the same angle when driving into the ball
	targetV2 := info.Vec2{X: m.to.X, Y: m.to.Y}
	ballToTarget := info.Sub(targetV2, ballV2)
	if ballToTarget.Norm() < 1 {
		return robotPos
	}
	kickAngle := ballToTarget.Angle()
	ballToTarget.DivNorm()

	headingErr := math.Abs(info.NormalizeAngleDelta(kickAngle, robotPos.Angle))
	dribblerPos := robot.DribblerPos()
	forward, lateral, ok := robot.BallLocalOffset(ballPos)
	closeEnoughToFinish := ok &&
		forward > info.Center2DribblerDist &&
		math.Abs(lateral) < captureLineTolerance &&
		dribblerPos.Dist2d(ballPos) < GetKickConfig().kickContactDist &&
		headingErr < 2*roughAngleTolerance
	if !captureApproachReady(robotPos, ballPos, m.to, headingErr) && !closeEnoughToFinish {
		return behindBallDest(ballPos, m.to, captureMarginToBall)
	}

	robotXY := info.Add(ballV2, ballToTarget.Mult(GetKickConfig().driveThrough))
	robotTargetPos := info.Position{X: robotXY.X, Y: robotXY.Y, Z: 0, Angle: kickAngle}
	return robotTargetPos
}

func (m *KickBall) GetAction(gi *info.GameInfo) action.Action {
	robotTargetPos := m.GetTargetPos(gi)
	robot := gi.State.GetTeam(m.team)[m.id]

	robotPos, err := robot.GetPosition()
	if err != nil {
		fmt.Println(err)
	}

	ballUntracked, err := gi.State.GetBall().GetPosition()

	if err != nil {
		fmt.Println(err)
	}
	ball := gi.State.GetTrackedBall()
	//ballPos, err := ball.GetTrackedPosition()
	ballVel, _ := ball.GetTrackedVelocity()

	speed := ballVel.Norm2d()
	if speed > 0.05 {
		robotTargetPos = robotPos
	}

	act := action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = robotPos
	act.Dest = robotTargetPos

	dribblerPos := robot.DribblerPos()
	dribblerDist := dribblerPos.Dist2d(ballUntracked)
	forward, lateral, ok := robot.BallLocalOffset(ballUntracked)
	ballCentered := ok && math.Abs(lateral) < info.KickCenterTolerance
	headingErr := math.Abs(info.NormalizeAngleDelta(robotTargetPos.Angle, robotPos.Angle))
	captureReady := captureApproachReady(robotPos, ballUntracked, m.to, headingErr)

	ballHeldInMouth := ok &&
		kickBallHeldInMouth(dribblerDist, forward, lateral, headingErr)
	kickAfterSettle := updateKickDribbleSettle(&m.dribbleSince, ballHeldInMouth)
	impactReady := captureReady && ok && kickBallImpactReady(forward, lateral, headingErr)
	// Keep control of the ball while the kick is armed. In particular, the
	// dribbler must already be running before the real robot receives its kick
	// command because that command preserves, rather than changes, dribbler
	// state.
	act.Dribble = true
	if kickBallShouldFire(dribblerDist, kickAfterSettle, impactReady) {
		act.KickSpeed = 2
	}
	printCaptureDebug(
		"kick-ball",
		m.team,
		m.id,
		robot,
		robotPos,
		ballUntracked,
		m.to,
		robotTargetPos,
		headingErr,
		captureReady,
		ballCentered,
		act.Dribble,
		act.KickSpeed,
		math.NaN(),
	)

	return &act
}

func kickBallHeldInMouth(dribblerDist, forward, lateral, headingErr float64) bool {
	return forward > info.Center2DribblerDist &&
		dribblerDist <= GetKickConfig().kickContactDist &&
		math.Abs(lateral) < kickMouthLateralTolerance &&
		headingErr < kickHeldHeadingTolerance
}

func kickBallReachableByMouth(forward, lateral, headingErr float64) bool {
	return forward >= info.Center2DribblerDist-info.BallRadius &&
		math.Abs(lateral) < captureLineTolerance &&
		headingErr < kickHeldHeadingTolerance
}

func kickBallImpactReady(forward, lateral, headingErr float64) bool {
	return kickBallReachableByMouth(forward, lateral, headingErr) &&
		forward <= info.Center2DribblerDist+info.BallRadius+kickImpactPreContactDist
}

func updateKickDribbleSettle(dribbleSince *time.Time, ballHeldInMouth bool) bool {
	if !ballHeldInMouth {
		*dribbleSince = time.Time{}
		return false
	}
	if dribbleSince.IsZero() {
		*dribbleSince = time.Now()
	}
	return time.Since(*dribbleSince) >= kickDribbleSettleTime
}

func kickBallShouldFire(dribblerDist float64, kickAfterSettle bool, impactReady bool) bool {
	return impactReady || (dribblerDist <= GetKickConfig().kickContactDist && kickAfterSettle)
}

func (m *KickBall) Achieved(gi *info.GameInfo) bool {
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	// myRobotPos, err := gi.State.GetTeam(m.team)[m.id].GetPosition()
	// if err != nil{
	// 	fmt.Println(err)
	// }
	// robotTargetPos := m.GetTargetPos(gi)

	// robotDist := robotTargetPos.Norm2d(myRobotPos)

	ballDist := ballPos.Dist2d(m.orignalBallPos)

	// return false
	if ballDist > GetKickConfig().ballAbortRadius {
		// fmt.Println("Done")
		// fmt.Println(ballDist)
		// fmt.Println(ballPos, m.orignalBallPos)
	}

	return ballDist > GetKickConfig().ballAbortRadius
}

func (m *KickBall) GetID() info.ID {

	return m.id
}
