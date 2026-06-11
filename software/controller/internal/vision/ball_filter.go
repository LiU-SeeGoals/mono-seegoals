// Package vision filters raw SSL-Vision detections before they enter the
// game state.
package vision

import (
	"math"

	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

const (
	// Detections below this SSL-Vision confidence are ignored entirely.
	minBallConfidence = 0.1
	// Fastest plausible ball movement in mm/ms (equal to m/s). The league
	// kick limit is 6.5 m/s; the margin absorbs measurement noise.
	maxBallSpeed = 8.0
	// Gate slack in mm on top of the physically possible travel distance,
	// absorbing camera handoff offsets and detection jitter.
	ballGateBase = 300.0
	// The gate never grows beyond this radius (mm) no matter how long the
	// ball has been missing; larger jumps go through reacquisition instead.
	ballGateMax = 2000.0
	// Consecutive candidate sightings must stay within this radius (mm) of
	// each other to count as the same spot.
	ballCandidateRadius = 500.0
	// Frames a far-away spot must persist before the track jumps to it.
	ballReacquireFrames = 3
	// Candidate sightings further apart than this (ms) are not consecutive.
	ballCandidateTimeout = 200
	// The track must have gone unseen this long (ms) before a jump beyond
	// the gate is accepted.
	ballTrackTimeout = 200
)

// BallFilter reduces the ball detections of a vision frame to at most one
// plausible ball. Vision regularly reports light reflections as extra balls;
// without filtering, whichever detection comes last in the packet wins and
// the ball appears to teleport between the real ball and the glare.
//
// The filter keeps a track of the last accepted position. Per frame it
// accepts the detection closest to the track if that is within a gate of
// physically possible movement. Detections outside the gate (a genuinely
// moved ball, e.g. placed by the referee) take over only after appearing at
// the same spot for ballReacquireFrames frames while the track stayed empty.
//
// Limitation: a permanent static false ball (steady glare that vision reports
// every frame) can hold the track while the real ball is occluded. Such spots
// should be masked out in SSL-Vision itself.
//
// The zero value is ready to use. Not safe for concurrent use.
type BallFilter struct {
	hasTrack  bool
	trackX    float64
	trackY    float64
	trackTime int64

	candX        float64
	candY        float64
	candCount    int
	candLastSeen int64
}

// Filter returns the detection to treat as the real ball, or nil when no
// detection in this frame should be trusted. now is in milliseconds.
func (f *BallFilter) Filter(balls []*ssl_vision.SSL_DetectionBall, now int64) *ssl_vision.SSL_DetectionBall {
	confident := make([]*ssl_vision.SSL_DetectionBall, 0, len(balls))
	for _, b := range balls {
		if b.GetConfidence() >= minBallConfidence {
			confident = append(confident, b)
		}
	}
	if len(confident) == 0 {
		return nil
	}

	if !f.hasTrack {
		best := mostConfidentBall(confident)
		f.setTrack(best, now)
		return best
	}

	best, dist := closestBall(confident, f.trackX, f.trackY)
	dt := float64(now - f.trackTime)
	if dt < 1 {
		dt = 1
	}
	gate := math.Min(ballGateBase+maxBallSpeed*dt, ballGateMax)
	if dist <= gate {
		f.setTrack(best, now)
		f.updateCandidate(ballsWithout(confident, best), now)
		return best
	}

	cand := f.updateCandidate(confident, now)
	if cand != nil && f.candCount >= ballReacquireFrames && now-f.trackTime >= ballTrackTimeout {
		f.setTrack(cand, now)
		f.candCount = 0
		return cand
	}
	return nil
}

func (f *BallFilter) setTrack(det *ssl_vision.SSL_DetectionBall, now int64) {
	f.hasTrack = true
	f.trackX = float64(det.GetX())
	f.trackY = float64(det.GetY())
	f.trackTime = now
}

// updateCandidate follows the most believable detection that is not part of
// the current track and counts how many consecutive frames it has been seen
// at the same spot. It returns the detection that extended the candidate.
func (f *BallFilter) updateCandidate(dets []*ssl_vision.SSL_DetectionBall, now int64) *ssl_vision.SSL_DetectionBall {
	if len(dets) == 0 {
		return nil
	}
	if f.candCount > 0 && now-f.candLastSeen > ballCandidateTimeout {
		f.candCount = 0
	}

	var matched *ssl_vision.SSL_DetectionBall
	if f.candCount > 0 {
		closest, dist := closestBall(dets, f.candX, f.candY)
		if dist <= ballCandidateRadius {
			matched = closest
		}
	}
	if matched == nil {
		matched = mostConfidentBall(dets)
		f.candCount = 0
	}

	f.candX = float64(matched.GetX())
	f.candY = float64(matched.GetY())
	f.candCount++
	f.candLastSeen = now
	return matched
}

func closestBall(dets []*ssl_vision.SSL_DetectionBall, x, y float64) (*ssl_vision.SSL_DetectionBall, float64) {
	var best *ssl_vision.SSL_DetectionBall
	bestDist := math.Inf(1)
	for _, d := range dets {
		dist := math.Hypot(float64(d.GetX())-x, float64(d.GetY())-y)
		if dist < bestDist {
			bestDist = dist
			best = d
		}
	}
	return best, bestDist
}

func mostConfidentBall(dets []*ssl_vision.SSL_DetectionBall) *ssl_vision.SSL_DetectionBall {
	best := dets[0]
	for _, d := range dets[1:] {
		if d.GetConfidence() > best.GetConfidence() {
			best = d
		}
	}
	return best
}

func ballsWithout(dets []*ssl_vision.SSL_DetectionBall, skip *ssl_vision.SSL_DetectionBall) []*ssl_vision.SSL_DetectionBall {
	rest := make([]*ssl_vision.SSL_DetectionBall, 0, len(dets)-1)
	for _, d := range dets {
		if d != skip {
			rest = append(rest, d)
		}
	}
	return rest
}
