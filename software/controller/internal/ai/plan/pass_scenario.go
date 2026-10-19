package ai

import (
	"fmt"
	"math"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

type GameScenario struct {
	plannerCore
	start             time.Time
	ballOwner         ID
	previousBallOwner ID
	ballOwnerTeam     Team
}

func NewGameScenario(team Team) *GameScenario {
	return &GameScenario{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *GameScenario) Init(
	incoming <-chan GameInfo,
	activities *[TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true

	go m.run()
}

func (g *GameScenario) getRobotClosestToPosition(
	gi *info.GameInfo,
	activeRobots []info.ID,
	target info.Position,
) info.ID {
	if len(activeRobots) == 0 {
		return info.ID(0)
	}

	closestId := activeRobots[0]
	bestDist := math.Inf(1)

	for _, id := range activeRobots {
		robotPos, err := gi.State.GetRobotPosition(g.team, id)
		if err != nil {
			continue
		}

		dist := robotPos.Dist2d(target)
		if dist < bestDist {
			bestDist = dist
			closestId = id
		}
	}

	return closestId
}

func (g *GameScenario) getRobotForBall(gi *info.GameInfo, activeRobots []info.ID) info.ID {
	targetPos, _ := gi.State.BallPositionAfter(ballInterceptLookahead, ballInterceptMinSpeed)

	return g.getRobotClosestToPosition(gi, activeRobots, targetPos)
}

func (g *GameScenario) syncKickers(
	gi *info.GameInfo,
	activeRobots []info.ID,
	kickers map[info.ID]*roles.OffenseRole,
) {
	active := make(map[info.ID]bool, len(activeRobots))
	for _, id := range activeRobots {
		active[id] = true
		if _, exists := kickers[id]; exists {
			continue
		}
		kicker := roles.NewOffenseRole(id, g.ActivityHandler, gi, g.team)
		kicker.Init()
		kickers[id] = kicker
	}

	for id := range kickers {
		if active[id] {
			continue
		}
		g.ActivityHandler.ClearActivity(id)
		delete(kickers, id)
	}
}

func (g *GameScenario) run() {
	kickers := make(map[info.ID]*roles.OffenseRole)

	gi := <-g.incomingGameInfo
	activeRobots := gi.State.ActiveRobotIDs(g.team, time.Now())
	g.syncKickers(&gi, activeRobots, kickers)
	// refereeHandler := referee.NewRefereeHandler(&gi, activeRobots, g.team, &g.ActivityHandler)

	fmt.Println(gi.Status)

	var activeReceiver info.ID
	var activeReceiverStart time.Time
	hasActiveReceiver := false
	possessionTracker := &ballPossessionTracker{}
	actorTracker := &offenseBallActorTracker{}
	frameMonitor := helper.NewFrameSkipMonitor(g.team.String() + " pass_scenario")

	for {
		tickStart := time.Now()

		gi = <-g.incomingGameInfo
		frameMonitor.Observe(gi.VisionFrame())
		activeRobots = gi.State.ActiveRobotIDs(g.team, tickStart)
		g.syncKickers(&gi, activeRobots, kickers)
		possession := possessionTracker.update(&gi, tickStart)
		rawOwner := observedBallOwner(&gi)
		// handleRef := refereeHandler.HandleReferee()
		// Only coordinate robot roles, trigger ball events

		ballVel, ballVelOK := gi.State.GetTrackedBall().GetFreshTrackedVelocity(tickStart, info.TrackedBallFreshness)
		ballMoving := ballVelOK && ballVel.Norm2d() > ballInterceptMinSpeed
		ownerRetained := possession.owner.valid && ownerStillRetained(&gi, possession.owner)

		passInFlight := hasActiveReceiver &&
			containsRobot(activeRobots, activeReceiver) &&
			time.Since(activeReceiverStart) < 2*time.Second &&
			(!rawOwner.valid || rawOwner.team != g.team) &&
			(ballMoving || !ownerRetained)

		if !passInFlight && possession.owner.valid && possession.owner.team == g.team {
			ownerID := possession.owner.id
			owner, ok := kickers[ownerID]
			if ok {
				hasActiveReceiver = false

				actorTracker.switchTo(kickers, newOffenseBallActor(ownerID, offenseBallActorOwner))
				owner.TriggerEvent("BALL_OWNER")
				decision := owner.CurrentDecision()
				if decision.IsPass && decision.ReceiverID != ownerID {
					receiver, ok := kickers[decision.ReceiverID]
					if ok {
						activeReceiver = decision.ReceiverID
						activeReceiverStart = time.Now()
						hasActiveReceiver = true
						receiver.ReceivePass(decision.Target)
					}
				}
			}
		} else {
			if passInFlight {
				actorTracker.switchTo(kickers, newOffenseBallActor(activeReceiver, offenseBallActorReceiver))
			} else if len(activeRobots) > 0 {
				hasActiveReceiver = false
				interceptorID := g.getRobotForBall(&gi, activeRobots)

				actorTracker.switchTo(kickers, newOffenseBallActor(interceptorID, offenseBallActorChaser))
				kickers[interceptorID].TriggerEvent("BALL_APPROACHING")
			} else {
				hasActiveReceiver = false
				actorTracker.switchTo(kickers, noOffenseBallActor())
			}
		}

		for _, kicker := range kickers {
			kicker.Run()
		}

		// Defense strategy

		// Etc...
	}
}

func (m *GameScenario) Kill() {
	fmt.Println("Killing planner goalie")
	m.Active = false
}
