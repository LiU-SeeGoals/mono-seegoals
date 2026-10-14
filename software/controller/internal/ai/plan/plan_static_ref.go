package ai

import (
	"sync"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/referee"
)

// StaticPlanWithRef mirrors CombinedPlanWithRef, but routes normal play to a
// StaticPlan instead of a dynamically assigned CombinedPlan.
type StaticPlanWithRef struct {
	plannerCore
	normalPlan      *StaticPlan
	refereeHandler  *referee.RefereeHandler
	currentGameInfo info.GameInfo
}

func NewStaticPlanWithRef(team info.Team) *StaticPlanWithRef {
	return &StaticPlanWithRef{
		plannerCore: plannerCore{team: team},
		normalPlan:  NewStaticPlan(team),
	}
}

func (m *StaticPlanWithRef) Init(
	activities *[info.TEAM_SIZE]act.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true
	if m.normalPlan == nil {
		m.normalPlan = NewStaticPlan(team)
	}

	m.normalPlan.Init(activities, lock, team)
}

func (m *StaticPlanWithRef) Tick(gi *info.GameInfo) {
	if !m.Active || gi == nil {
		return
	}
	m.routeFrame(*gi)
}

func (m *StaticPlanWithRef) routeFrame(gi info.GameInfo) bool {
	m.currentGameInfo = gi
	refereeHandled := m.refereeHandlesFrame()
	if refereeHandled && !isStoppedFrame(&m.currentGameInfo) {
		return false
	}

	m.normalPlan.Tick(&m.currentGameInfo)
	return true
}

func (m *StaticPlanWithRef) refereeHandlesFrame() bool {
	if m.refereeHandler == nil {
		if m.normalPlan == nil {
			m.normalPlan = NewStaticPlan(m.team)
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

func (m *StaticPlanWithRef) Kill() {
	m.Active = false
	if m.normalPlan != nil {
		m.normalPlan.Kill()
	}
}
