package referee

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/roles"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type StateName string
type EventName string

type RefereeInfo struct {
	gi *info.GameInfo
	activeRobots []info.ID
	team info.Team
	activityHandler *ai.ActivityHandler
	name StateName
}

type Halt struct {
	RefereeInfo
}

type Stop struct {
	RefereeInfo
}

type PrepareKickoff struct {
	RefereeInfo
	kickOff *roles.StateMachine
	recieve *roles.StateMachine
}

type Kickoff struct {
	RefereeInfo
	kickOff *roles.StateMachine
	recieve *roles.StateMachine
}

func (s *Halt) Initialize() {
}

func (s *Halt) GetName() StateName {
	return s.name
}

func (s *Halt) Update() EventName{

	for _, id := range s.activeRobots{
		activity := act.NewStop(id)
		s.activityHandler.AddActivity(activity)
	}

	return "NONE"
}

func (s *Stop) Initialize() {
}

func (s *Stop) GetName() StateName {
	return s.name
}

func (s *Stop) Update() EventName{

	for _, id := range s.activeRobots{
		activity := act.NewRefStop(s.team, id)
		s.activityHandler.AddActivity(activity)
	}

	return "NONE"
}

type RecieveIntent struct{
	gi *info.GameInfo
	team info.Team
	id info.ID
}

func (s *RecieveIntent) GetTargetPosition() info.Position{
	return info.Position{}
}

func (s *RecieveIntent) GetFromPosition() info.Position{
	return info.Position{X:1000,Y:0,Z:0,Angle: 0}
}

type KickOffIntent struct{
	gi *info.GameInfo
	team info.Team
	id info.ID
}

func (s *KickOffIntent) GetTargetPosition() info.Position{
	return info.Position{X:1000,Y:0,Z:0,Angle: 0}
}

func (s *KickOffIntent) GetFromPosition() info.Position{
	return info.Position{}
}

func (s *PrepareKickoff) Initialize() {

	kickOffID := info.ID(1)
	recieveID := info.ID(3)

	kickPrepareName := roles.StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))

	kickoff := KickOffIntent{gi: s.gi, team:s.team, id: kickOffID}
	recieve := RecieveIntent{gi: s.gi, team:s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	reciever := &roles.AlignState{Ctx: &recieve, Gi: s.gi, Team: s.team, RobotId: recieveID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	// kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	s.kickOff = roles.NewStateMachine(prepareKick)
	s.recieve = roles.NewStateMachine(reciever)
}

func (s *PrepareKickoff) GetName() StateName {
	return s.name
}

func (s *PrepareKickoff) Update() EventName {

	s.kickOff.Update()
	s.recieve.Update()

	return "NONE"
}

func (s *Kickoff) Initialize() {

	kickOffID := info.ID(1)
	recieveID := info.ID(3)

	kickPrepareName := roles.StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))
	kickName := roles.StateName(fmt.Sprintf("Kick ID %d", kickOffID))

	recieveName := roles.StateName(fmt.Sprintf("Kickoff recieve ID %d", recieveID))

	kickoff := KickOffIntent{gi: s.gi, team:s.team, id: kickOffID}
	recieve := RecieveIntent{gi: s.gi, team:s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickName, ActivityHandler: s.activityHandler}
	reciever := &roles.AlignState{Ctx: &recieve, Gi: s.gi, Team: s.team, RobotId: recieveID, Name: recieveName, ActivityHandler: s.activityHandler}

	s.kickOff = roles.NewStateMachine(prepareKick)
	s.kickOff.AddTransition(kickPrepareName,"ALIGNED", kick)

	s.recieve = roles.NewStateMachine(reciever)
}

func (s *Kickoff) GetName() StateName {
	return s.name
}

func (s *Kickoff) Update() EventName {

	s.kickOff.Update()
	s.recieve.Update()

	return "NONE"
}
