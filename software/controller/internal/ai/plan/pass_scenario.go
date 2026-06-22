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
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()

	targetPos := ballPos

	if ok && ballVel.Norm2d() > 0.3 {
		lookaheadSeconds := 0.5

		targetPos.X = ballPos.X + ballVel.X*1000*lookaheadSeconds
		targetPos.Y = ballPos.Y + ballVel.Y*1000*lookaheadSeconds
	}

	return g.getRobotClosestToPosition(gi, activeRobots, targetPos)
}

func (g *GameScenario) run() {

	activeRobots := []info.ID{1, 2, 3, 4, 5, 6}
	kickers := make(map[info.ID]*roles.OffenseRole)

	gi := <-g.incomingGameInfo
	// refereeHandler := referee.NewRefereeHandler(&gi, activeRobots, g.team, &g.ActivityHandler)
	for _, id := range activeRobots {
		kicker := roles.NewOffenseRole(id, g.ActivityHandler, &gi, g.team)
		kicker.Init()
		kickers[id] = kicker
	}

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
		possession := possessionTracker.update(&gi, tickStart)
		rawOwner := observedBallOwner(&gi)
		// handleRef := refereeHandler.HandleReferee()
		// Only coordinate robot roles, trigger ball events

		ballVel, ballVelOK := gi.State.GetTrackedBall().GetTrackedVelocity()
		ballMoving := ballVelOK && ballVel.Norm2d() > 0.3
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
			} else {
				hasActiveReceiver = false
				interceptorID := g.getRobotForBall(&gi, activeRobots)

				actorTracker.switchTo(kickers, newOffenseBallActor(interceptorID, offenseBallActorChaser))
				kickers[interceptorID].TriggerEvent("BALL_APPROACHING")
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
