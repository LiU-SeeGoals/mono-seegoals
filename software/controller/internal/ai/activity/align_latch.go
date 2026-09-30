package ai

import (
	"sync"
	"time"
)

const (
	alignRollingEnterSpeed  = 0.4
	alignRollingExitSpeed   = 0.2
	alignOrbitReleaseMargin = 100.0
	alignModeMinHold        = 250 * time.Millisecond
	alignTrackedBallMaxAge  = 200 * time.Millisecond
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

type AlignLatch struct {
	mu      sync.Mutex
	rolling hysteresisLatch
	orbit   hysteresisLatch
	direct  hysteresisLatch
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
