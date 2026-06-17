package ai

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	captureDebugEnabled     = true
	captureDebugPrintPeriod = 250 * time.Millisecond
)

var (
	captureDebugMu   sync.Mutex
	captureDebugLast = map[string]time.Time{}
)

func printCaptureDebug(
	label string,
	team info.Team,
	id info.ID,
	robot *info.Robot,
	robotPos info.Position,
	ballPos info.Position,
	target info.Position,
	dest info.Position,
	headingErr float64,
	captureReady bool,
	ballCentered bool,
	dribble bool,
	kickSpeed int,
	margin float64,
) {
	if !captureDebugEnabled {
		return
	}

	key := fmt.Sprintf("%s:%s:%d", label, team, id)
	now := time.Now()
	captureDebugMu.Lock()
	if last, ok := captureDebugLast[key]; ok && now.Sub(last) < captureDebugPrintPeriod {
		captureDebugMu.Unlock()
		return
	}
	captureDebugLast[key] = now
	captureDebugMu.Unlock()

	along, side, lineOK := lineErrorToTarget(robotPos, ballPos, target)
	forward := math.NaN()
	lateral := math.NaN()
	if robot != nil {
		if f, l, ok := robot.BallLocalOffset(ballPos); ok {
			forward = f
			lateral = l
		}
	}
	dribblerDist := math.NaN()
	if robot != nil {
		driblerpos := robot.DribblerPos()
		dribblerDist = driblerpos.Dist2d(ballPos)
	}

	fmt.Printf(
		"[capture-debug] %s robot=%s/%d lineOK=%t along=%.1f side=%.1f localF=%.1f localL=%.1f dribDist=%.1f heading=%.1fdeg capReady=%t centered=%t dribble=%t kick=%d margin=%.1f pos=(%.0f,%.0f,%.1fdeg) ball=(%.0f,%.0f) dest=(%.0f,%.0f,%.1fdeg)\n",
		label,
		team,
		id,
		lineOK,
		along,
		side,
		forward,
		lateral,
		dribblerDist,
		headingErr*180/math.Pi,
		captureReady,
		ballCentered,
		dribble,
		kickSpeed,
		margin,
		robotPos.X,
		robotPos.Y,
		robotPos.Angle*180/math.Pi,
		ballPos.X,
		ballPos.Y,
		dest.X,
		dest.Y,
		dest.Angle*180/math.Pi,
	)
}
