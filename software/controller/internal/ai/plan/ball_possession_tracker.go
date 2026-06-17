package ai

import (
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	ballPossessionAcquireDelay    = 150 * time.Millisecond
	ballPossessionLossDelay       = 1500 * time.Millisecond
	ballPossessionRetainDistance  = 200.0
	ballPossessionFacingTolerance = 0.9
)

type trackedBallOwner struct {
	team  info.Team
	id    info.ID
	valid bool
}

func noTrackedBallOwner() trackedBallOwner {
	return trackedBallOwner{}
}

func newTrackedBallOwner(team info.Team, id info.ID) trackedBallOwner {
	return trackedBallOwner{team: team, id: id, valid: true}
}

func (o trackedBallOwner) same(other trackedBallOwner) bool {
	return o.valid == other.valid && (!o.valid || (o.team == other.team && o.id == other.id))
}

type possessionUpdate struct {
	owner trackedBallOwner
}

type ballPossessionTracker struct {
	owner         trackedBallOwner
	nearCandidate trackedBallOwner
	nearSince     time.Time
	farSince      time.Time
}

func observedBallOwner(gi *info.GameInfo) trackedBallOwner {
	possessor := gi.State.GetBall().GetPossessor()
	if possessor == nil {
		return noTrackedBallOwner()
	}
	return newTrackedBallOwner(possessor.GetTeam(), possessor.GetID())
}

func (t *ballPossessionTracker) update(gi *info.GameInfo, now time.Time) possessionUpdate {
	observed := observedBallOwner(gi)

	if observed.valid {
		t.farSince = time.Time{}
		if !t.nearCandidate.same(observed) {
			t.nearCandidate = observed
			t.nearSince = now
		}

		if t.owner.same(observed) || now.Sub(t.nearSince) >= ballPossessionAcquireDelay {
			t.owner = observed
			return possessionUpdate{owner: t.owner}
		}

		if t.owner.valid && ownerStillRetained(gi, t.owner) {
			return possessionUpdate{owner: t.owner}
		}

		return possessionUpdate{owner: t.owner}
	}

	t.nearCandidate = noTrackedBallOwner()
	t.nearSince = time.Time{}

	if t.owner.valid && ownerStillRetained(gi, t.owner) {
		t.farSince = time.Time{}
		return possessionUpdate{owner: t.owner}
	}

	if t.owner.valid {
		if t.farSince.IsZero() {
			t.farSince = now
		}
		if now.Sub(t.farSince) >= ballPossessionLossDelay {
			t.owner = noTrackedBallOwner()
			t.farSince = time.Time{}
		}
	}

	return possessionUpdate{owner: t.owner}
}

func ownerStillRetained(gi *info.GameInfo, owner trackedBallOwner) bool {
	robot := gi.State.GetRobot(owner.id, owner.team)
	if robot == nil {
		return false
	}
	if _, err := robot.GetPosition(); err != nil {
		return false
	}

	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return false
	}

	dribblerPos := robot.DribblerPos()
	return dribblerPos.Dist2d(ballPos) <= ballPossessionRetainDistance &&
		robot.Facing(ballPos, ballPossessionFacingTolerance)
}

type offenseBallActorKind string

const (
	offenseBallActorOwner    offenseBallActorKind = "owner"
	offenseBallActorReceiver offenseBallActorKind = "receiver"
	offenseBallActorChaser   offenseBallActorKind = "chaser"
)

type offenseBallActor struct {
	id    info.ID
	kind  offenseBallActorKind
	valid bool
}

func noOffenseBallActor() offenseBallActor {
	return offenseBallActor{}
}

func newOffenseBallActor(id info.ID, kind offenseBallActorKind) offenseBallActor {
	return offenseBallActor{id: id, kind: kind, valid: true}
}

func (a offenseBallActor) same(other offenseBallActor) bool {
	return a.valid == other.valid && (!a.valid || (a.id == other.id && a.kind == other.kind))
}

type offenseBallActorTracker struct {
	current offenseBallActor
}

func (t *offenseBallActorTracker) switchTo(attackers map[info.ID]*roles.OffenseRole, next offenseBallActor) {
	if t.current.same(next) {
		return
	}
	if t.current.valid {
		if attacker, ok := attackers[t.current.id]; ok {
			attacker.TriggerEvent("BALL_LOST")
		}
	}
	t.current = next
}
