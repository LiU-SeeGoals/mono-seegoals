
package roles

import (

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

type KickerRole2 struct {
	id info.ID
	sm *StateMachine
	activityHandler *ai.ActivityHandler
	gi *GameInfo
	team Team
}

func NewKickerRole2(robotID ID, activityHandler ai.ActivityHandler, gi *GameInfo, team Team) *KickerRole2 {
	return &KickerRole2{
		id: robotID,
		sm: nil,
		activityHandler: &activityHandler,
		gi: gi,
		team: team,
	}
}

func (kr *KickerRole2) Init() {

	position := info.Position{1,1,1,1}

	align := &AlignState{gi: kr.gi, team: kr.team, robotId: kr.id, name: "Align", from: position, activityHandler: kr.activityHandler}
	prepareKick := &AlignState{gi: kr.gi, team: kr.team, robotId: kr.id, name: "PrepareKick", from: position, activityHandler: kr.activityHandler}
	kick := &KickState{name: "Kick", gi: kr.gi, team: kr.team, robotId: kr.id}

	sm := NewStateMachine(align)

	sm.SetGlobalTransition("BALL_LOST", align)
	sm.AddTransition("Align", "BALL_OWNER", prepareKick)
	sm.AddTransition("KickPrepare", "ALIGNED", kick)
	sm.AddTransition("Kick", "KICKED", align)

	kr.sm = sm

}

func (kr *KickerRole2) Run() {
	kr.sm.Update()
}

func (kr *KickerRole2) TriggerEvent(event EventName) {
	kr.sm.TriggerEvent(event)
}
