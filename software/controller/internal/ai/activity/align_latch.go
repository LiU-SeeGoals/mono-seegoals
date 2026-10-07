package ai

import (
	"math"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	alignRollingEnterSpeed  = 0.4
	alignRollingExitSpeed   = 0.2
	alignOrbitReleaseMargin = 100.0
	alignModeMinHold        = 250 * time.Millisecond
	alignTrackedBallMaxAge  = 200 * time.Millisecond

	alignBallHoldRadius     = 30.0
	alignBallHoldTau        = 100 * time.Millisecond
	alignBallMoveConfirm    = 60 * time.Millisecond
	alignBallHoldStale      = 250 * time.Millisecond
	alignLineEnterTolerance = 20.0
	alignLineExitTolerance  = 35.0
	alignLineEnterHeading   = 2 * roughAngleTolerance
	alignLineExitHeading    = 4 * roughAngleTolerance
	alignLineMaxSideSpeed   = 0.1
)

type hysteresisLatch struct {
	on        bool
	changedAt time.Time
}

func (l *hysteresisLatch) update(turnOn, turnOff bool, now time.Time, minHold time.Duration) bool {
	if !l.changedAt.IsZero() && now.Sub(l.changedAt) < minHold {
		return l.on
	}
	if (!l.on && turnOn) || (l.on && turnOff) {
		l.on = !l.on
		l.changedAt = now
	}
	return l.on
}

type ballHold struct {
	pos       info.Position
	held      bool
	updatedAt time.Time
	awaySince time.Time
}

func (h *ballHold) update(ball info.Position, now time.Time) info.Position {
	if !h.held || now.Sub(h.updatedAt) > alignBallHoldStale {
		h.pos, h.held, h.updatedAt, h.awaySince = ball, true, now, time.Time{}
		return h.pos
	}
	if h.pos.Dist2d(ball) > alignBallHoldRadius {
		if h.awaySince.IsZero() {
			h.awaySince = now
		}
		if now.Sub(h.awaySince) >= alignBallMoveConfirm {
			h.pos, h.awaySince = ball, time.Time{}
		}
		h.updatedAt = now
		return h.pos
	}
	h.awaySince = time.Time{}
	alpha := 1 - math.Exp(-now.Sub(h.updatedAt).Seconds()/alignBallHoldTau.Seconds())
	held := ball
	held.X = h.pos.X + alpha*(ball.X-h.pos.X)
	held.Y = h.pos.Y + alpha*(ball.Y-h.pos.Y)
	h.pos, h.updatedAt = held, now
	return h.pos
}

type AlignLatch struct {
	mu      sync.Mutex
	rolling hysteresisLatch
	orbit   hysteresisLatch
	direct  hysteresisLatch
	onLine  hysteresisLatch
	ball    ballHold
}

func (l *AlignLatch) holdBall(ball info.Position, now time.Time) info.Position {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ball.update(ball, now)
}

func (l *AlignLatch) releaseBall() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ball.held = false
}

func (l *AlignLatch) onKickLine(along, sideErr, sideSpeed, headingErr float64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	behind := along < -info.Center2DribblerDist
	return l.onLine.update(
		behind && sideErr < alignLineEnterTolerance &&
			headingErr < alignLineEnterHeading &&
			math.Abs(sideSpeed) < alignLineMaxSideSpeed,
		!behind || sideErr > alignLineExitTolerance || headingErr > alignLineExitHeading,
		now, 0,
	)
}

func (l *AlignLatch) ballRolling(speed float64, known bool, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rolling.update(
		known && speed > alignRollingEnterSpeed,
		!known || speed < alignRollingExitSpeed,
		now, alignModeMinHold,
	)
}

func (l *AlignLatch) orbitBall(dist, enterDist float64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.orbit.update(
		dist <= enterDist,
		dist > enterDist+alignOrbitReleaseMargin,
		now, alignModeMinHold,
	)
}

func (l *AlignLatch) DirectApproach(enemyInEnterRadius, enemyInReleaseRadius bool, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.direct.update(enemyInEnterRadius, !enemyInReleaseRadius, now, alignModeMinHold)
}
