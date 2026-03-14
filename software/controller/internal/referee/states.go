package referee

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

type RefereeInfo struct {
	gi              *info.GameInfo
	activeRobots    []info.ID
	team            info.Team
	activityHandler *ai.ActivityHandler
	name            roles.StateName
}

func (s *RefereeInfo) GetName() roles.StateName {
	return s.name
}

type Halt struct {
	RefereeInfo
}

type FreeKick struct {
	RefereeInfo
	freeKick *roles.StateMachine
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

type Running struct {
}

func (s *Halt) Initialize() {
}

func (s *Halt) Update() roles.EventName {

	for _, id := range s.activeRobots {
		activity := act.NewStop(id)
		s.activityHandler.AddActivity(activity)
	}

	return "NONE"
}

func (s *Stop) Initialize() {
}

func (s *Stop) Update() roles.EventName {

	for _, id := range s.activeRobots {
		activity := act.NewRefStop(s.team, id)
		s.activityHandler.AddActivity(activity)
	}

	return "NONE"
}

type RecieveIntent struct {
	gi   *info.GameInfo
	team info.Team
	id   info.ID
}

func (s *RecieveIntent) GetTargetPosition() info.Position {
	return info.Position{}
}

func (s *RecieveIntent) GetFromPosition() info.Position {
	return info.Position{X: 1000, Y: 0, Z: 0, Angle: 0}
}

type KickOffIntent struct {
	gi   *info.GameInfo
	team info.Team
	id   info.ID
}

func (s *KickOffIntent) GetTargetPosition() info.Position {
	return info.Position{X: 1000, Y: 0, Z: 0, Angle: 0}
}

func (s *KickOffIntent) GetFromPosition() info.Position {
	return info.Position{}
}

func (s *FreeKick) Initialize() {

	kickOffID := info.ID(1)

	kickPrepareName := roles.StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))

	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	// kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	s.freeKick = roles.NewStateMachine(prepareKick)
}

func (s *FreeKick) Update() roles.EventName {

	s.freeKick.Update()

	return "NONE"
}

func (s *PrepareKickoff) Initialize() {

	kickOffID := info.ID(1)
	recieveID := info.ID(3)

	kickPrepareName := roles.StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))

	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}
	recieve := RecieveIntent{gi: s.gi, team: s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	reciever := &roles.AlignState{Ctx: &recieve, Gi: s.gi, Team: s.team, RobotId: recieveID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	// kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	s.kickOff = roles.NewStateMachine(prepareKick)
	s.recieve = roles.NewStateMachine(reciever)
}

func (s *PrepareKickoff) Update() roles.EventName {

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

	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}
	recieve := RecieveIntent{gi: s.gi, team: s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickName, ActivityHandler: s.activityHandler}
	reciever := &roles.AlignState{Ctx: &recieve, Gi: s.gi, Team: s.team, RobotId: recieveID, Name: recieveName, ActivityHandler: s.activityHandler}

	s.kickOff = roles.NewStateMachine(prepareKick)
	s.kickOff.AddTransition(kickPrepareName, "ALIGNED", kick)

	s.recieve = roles.NewStateMachine(reciever)
}

func (s *Kickoff) Update() roles.EventName {

	s.kickOff.Update()
	s.recieve.Update()

	return "NONE"
}

func (s *Running) Initialize() {
}

func (s *Running) GetName() roles.StateName {
	return "RUNNING"
}

func (s *Running) Update() roles.EventName {
	return "NONE"
}

type RefereeHandler struct {
	gi           *info.GameInfo
	refereeSM    *roles.StateMachine
	activeRobots []info.ID
}

func NewRefereeHandler(gi *info.GameInfo, activeRobots []info.ID, team info.Team, activityHandler *ai.ActivityHandler) *RefereeHandler {

	freeKickName := roles.StateName("FREEKICK")
	stopName := roles.StateName("STOP")
	haltName := roles.StateName("HALT")
	prepareKickoffName := roles.StateName("PREPAREKICKOFF")
	kickOffName := roles.StateName("KICKOFF")
	// timeoutName := roles.StateName("TIMEOUT")
	ballPlacementName := roles.StateName("BALLPLACEMENT")

	freeKick := &FreeKick{
		RefereeInfo: RefereeInfo{
			gi:           gi,
			activeRobots: activeRobots,
			team:         team,
			name:         freeKickName,
		},
	}

	prepareKickoff := &PrepareKickoff{
		RefereeInfo: RefereeInfo{
			gi:           gi,
			activeRobots: activeRobots,
			team:         team,
			name:         prepareKickoffName,
		},
	}
	stop := &Stop{
		RefereeInfo: RefereeInfo{
			gi:           gi,
			activeRobots: activeRobots,
			team:         team,
			name:         stopName,
		},
	}
	ballPlacement := &Stop{
		RefereeInfo: RefereeInfo{
			gi:           gi,
			activeRobots: activeRobots,
			team:         team,
			name:         ballPlacementName,
		},
	}
	// timeout := &Stop{
	// 	RefereeInfo: RefereeInfo{
	// 		gi: gi,
	// 		activeRobots: activeRobots,
	// 		team: team,
	// 		name: timeoutName,
	// 		},
	// }
	halt := &Halt{
		RefereeInfo: RefereeInfo{
			gi:           gi,
			activeRobots: activeRobots,
			team:         team,
			name:         haltName,
		},
	}
	kickOff := &Kickoff{
		RefereeInfo: RefereeInfo{
			gi:           gi,
			activeRobots: activeRobots,
			team:         team,
			name:         kickOffName,
		},
	}
	running := &Running{}

	// Note that Timeout, ballplacement, PreparePenalty and Penalty are not implemented
	// Since it is not required
	refereeSM := roles.NewStateMachine(halt)

	refereeSM.AddTransition("HALT", "Stop", stop)
	// refereeSM.AddTransition("STOP", "Timeout", timeout)

	refereeSM.AddTransition("STOP", "PrepareKickoff", prepareKickoff)
	refereeSM.AddTransition("STOP", "FreeKick", freeKick)
	refereeSM.AddTransition("STOP", "BallPlacement", ballPlacement)

	refereeSM.AddTransition("BALLPLACEMENT", "Stop", stop)
	refereeSM.AddTransition("BALLPLACEMENT", "Continue", freeKick)

	refereeSM.AddTransition("PREPAREKICKOFF", "NormalStart", kickOff)

	refereeSM.AddTransition("KICKOFF", "Running", running)
	refereeSM.AddTransition("FREEKICK", "Running", running)

	return &RefereeHandler{gi, refereeSM, activeRobots}
}

/*
Returns true if referee is being handeled
Returns false if game is in running state
*/
func (s *RefereeHandler) HandleReferee() bool {

	// Rules to follow are defined in the ssl Rules
	// Appendix B: Game States https://robocup-ssl.github.io/ssl-rules/sslrules.html

	refereeCommand := s.gi.Status.GetGameEvent().RefCommand.String()
	fmt.Println(refereeCommand)
	if s.refereeSM.CurrentStateName() == "RUNNING" {
		return false
	}

	return true
}
