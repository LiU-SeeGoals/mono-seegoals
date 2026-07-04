package ai

import (
	"sync"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/referee"
)

type CombinedPlanWithRef struct {
	plannerCore
	normalPlan      *CombinedPlan
	normalIncoming  chan info.GameInfo
	refereeHandler  *referee.RefereeHandler
	currentGameInfo info.GameInfo
}

func NewCombinedPlanWithRef(team info.Team) *CombinedPlanWithRef {
	return &CombinedPlanWithRef{
		plannerCore: plannerCore{team: team},
		normalPlan:  NewCombinedPlan(team),
	}
}

func (m *CombinedPlanWithRef) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]act.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true
	m.normalIncoming = make(chan info.GameInfo)
	if m.normalPlan == nil {
		m.normalPlan = NewCombinedPlan(team)
	}

	m.normalPlan.Init(m.normalIncoming, activities, lock, team)
	go m.run()
}

func (m *CombinedPlanWithRef) run() {
	for m.Active {
		gi := <-m.incomingGameInfo
		m.routeFrame(gi)
	}
}

func (m *CombinedPlanWithRef) routeFrame(gi info.GameInfo) bool {
	m.currentGameInfo = gi
	refereeHandled := m.refereeHandlesFrame()
	if refereeHandled && !isStoppedFrame(&m.currentGameInfo) {
		return false
	}

	m.normalIncoming <- m.currentGameInfo
	return true
}

func (m *CombinedPlanWithRef) refereeHandlesFrame() bool {
	if m.refereeHandler == nil {
		if m.normalPlan == nil {
			m.normalPlan = NewCombinedPlan(m.team)
		}
		activeRobots := m.normalPlan.activeRobots(&m.currentGameInfo)
		m.refereeHandler = referee.NewRefereeHandler(
			&m.currentGameInfo,
			activeRobots,
			m.team,
			&coreai.ActivityHandler{
				Activities:    m.ActivityHandler.Activities,
				Activity_lock: m.ActivityHandler.Activity_lock,
			},
		)
		m.refereeHandler.DelegateStopPositioning()
	}
	activeRobots := m.normalPlan.activeRobots(&m.currentGameInfo)
	m.refereeHandler.UpdateActiveRobots(activeRobots)

	handled := m.refereeHandler.HandleReferee()
	if restrictedID, restricted := m.refereeHandler.KickoffRestrictedRobot(); restricted {
		m.normalPlan.setBallTouchRestriction(restrictedID)
	} else {
		m.normalPlan.clearBallTouchRestriction()
	}
	return handled
}

func isStoppedFrame(gi *info.GameInfo) bool {
	return gi != nil && gi.Status != nil && gi.Status.GetGameEvent() != nil &&
		gi.Status.GetGameEvent().GetCurrentState() == info.STATE_STOPPED
}

func (m *CombinedPlanWithRef) Kill() {
	m.Active = false
	if m.normalPlan != nil {
		m.normalPlan.Kill()
	}
}
