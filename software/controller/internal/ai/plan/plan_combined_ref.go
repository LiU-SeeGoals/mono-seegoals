package plan

import (
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/referee"
)

type CombinedPlanWithRef struct {
	plannerCore
	normalPlan      *CombinedPlan
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
	activities *[info.TEAM_SIZE]activity.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true
	if m.normalPlan == nil {
		m.normalPlan = NewCombinedPlan(team)
	}

	m.normalPlan.Init(activities, lock, team)
}

func (m *CombinedPlanWithRef) Tick(gi *info.GameInfo) {
	if !m.Active {
		return
	}
	m.routeFrame(*gi)
}

func (m *CombinedPlanWithRef) routeFrame(gi info.GameInfo) bool {
	m.currentGameInfo = gi
	if m.refereeHandlesFrame() {
		return false
	}
	// STOP and normal play: the planner owns the robots.
	m.normalPlan.Tick(&m.currentGameInfo)
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
			&ai.ActivityHandler{
				Activities:    m.ActivityHandler.Activities,
				Activity_lock: m.ActivityHandler.Activity_lock,
			},
		)
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

// RestartKicker implements ai.RestartKickerReporter.
func (m *CombinedPlanWithRef) RestartKicker() (info.ID, bool) {
	if m.refereeHandler == nil {
		return 0, false
	}
	return m.refereeHandler.RestartKicker()
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
