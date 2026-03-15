package referee

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	HALT = "Halt"
	STOP = "Stop"
	NORMAL_START = "NormalStart"
	FORCE_START = "ForceStart"
	PREPARE_KICKOFF = "PrepareKickoff"
	FREE_KICK = "FreeKick"
	BALL_PLACEMENT = "BallPlacement"
	TIMEOUT = "Timeout"
	CONTINUE = "Continue"
	UNKNOWN = "Unknown"
)

const (
	GAME_RUNNING_DETECTED = "Running"
)

type RefereeInfo struct {
	gi                *info.GameInfo
	activeRobots      []info.ID
	team              info.Team
	activityHandler   *ai.ActivityHandler
	name              roles.StateName
	teamWithPossesion info.Team
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

	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		// If we dont have possession we should chill or smth
	} else {
		kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}
		recieve := RecieveIntent{gi: s.gi, team: s.team, id: kickOffID}

		prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
		kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickName, ActivityHandler: s.activityHandler}
		reciever := &roles.AlignState{Ctx: &recieve, Gi: s.gi, Team: s.team, RobotId: recieveID, Name: recieveName, ActivityHandler: s.activityHandler}

		s.kickOff = roles.NewStateMachine(prepareKick)
		s.kickOff.AddTransition(kickPrepareName, "ALIGNED", kick)

		s.recieve = roles.NewStateMachine(reciever)
	}
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

type UninitializedRef struct {
}

func (s *UninitializedRef) Initialize() {
}

func (s *UninitializedRef) GetName() roles.StateName {
	return "UNINITIALIZED"
}

func (s *UninitializedRef) Update() roles.EventName {
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
	// Not needed but might be good in the future
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
	uninitialized := &UninitializedRef{}

	// Note that Timeout, ballplacement, PreparePenalty and Penalty are not implemented
	// Since it is not required to play a match

	refereeSM := roles.NewStateMachine(uninitialized)

	refereeSM.AddTransition("HALT", STOP, stop)
	// refereeSM.AddTransition("STOP", "Timeout", timeout)

	refereeSM.AddTransition("STOP", PREPARE_KICKOFF, prepareKickoff)
	refereeSM.AddTransition("STOP", FREE_KICK, freeKick)
	refereeSM.AddTransition("STOP", BALL_PLACEMENT, ballPlacement)
	refereeSM.AddTransition("STOP", FORCE_START, running)

	refereeSM.AddTransition("BALLPLACEMENT", STOP, stop)
	refereeSM.AddTransition("BALLPLACEMENT", CONTINUE, freeKick)

	refereeSM.AddTransition("PREPAREKICKOFF", NORMAL_START, kickOff)

	refereeSM.AddTransition("KICKOFF", GAME_RUNNING_DETECTED, running)
	refereeSM.AddTransition("FREEKICK", GAME_RUNNING_DETECTED, running)

	refereeSM.AddTransition("RUNNING", STOP, stop)

	// All states go to halt
	refereeSM.AddTransition("STOP", HALT, halt)
	refereeSM.AddTransition("BALLPLACEMENT", HALT, halt)
	refereeSM.AddTransition("PREPAREKICKOFF", HALT, halt)
	refereeSM.AddTransition("KICKOFF", HALT, halt)
	refereeSM.AddTransition("FREEKICK", HALT, halt)
	refereeSM.AddTransition("RUNNING", HALT, halt)

	// When starting the state is unknown, try parse the latest ref command
	refereeSM.AddTransition(uninitialized.GetName(), PREPARE_KICKOFF, prepareKickoff)
	refereeSM.AddTransition(uninitialized.GetName(), STOP, stop)
	refereeSM.AddTransition(uninitialized.GetName(), HALT, halt)
	refereeSM.AddTransition(uninitialized.GetName(), NORMAL_START, running)
	refereeSM.AddTransition(uninitialized.GetName(), TIMEOUT, stop)
	refereeSM.AddTransition(uninitialized.GetName(), GAME_RUNNING_DETECTED, running)
	refereeSM.AddTransition(uninitialized.GetName(), FORCE_START, running)

	return &RefereeHandler{gi, refereeSM, activeRobots}
}

func refCommandToEventName(rc info.RefCommand) string {
	switch rc {
	case info.HALT:
		return HALT
	case info.STOP:
		return STOP
	case info.NORMAL_START:
		return NORMAL_START
	case info.FORCE_START:
		return FORCE_START
	case info.PREPARE_KICKOFF_BLUE:
		return PREPARE_KICKOFF
	case info.PREPARE_KICKOFF_YELLOW:
		return PREPARE_KICKOFF
	case info.DIRECT_FREE_YELLOW:
		return FREE_KICK
	case info.DIRECT_FREE_BLUE:
		return FREE_KICK
	case info.TIMEOUT_YELLOW:
		return TIMEOUT
	case info.TIMEOUT_BLUE:
		return TIMEOUT
	case info.BALL_PLACEMENT_BLUE:
		return BALL_PLACEMENT
	case info.BALL_PLACEMENT_YELLOW:
		return BALL_PLACEMENT
	default:
		return UNKNOWN
	}
}

/*
Returns true if referee is being handeled
Returns false if game is in running state
*/
func (s *RefereeHandler) HandleReferee() bool {

	// Rules to follow are defined in the ssl Rules
	// Appendix B: Game States https://robocup-ssl.github.io/ssl-rules/sslrules.html

	refereeCommand := s.gi.Status.GetGameEvent().RefCommand
	// refereeCommandString := refereeCommand.String()

	// fmt.Println(refereeCommandString)

	refEvent := refCommandToEventName(refereeCommand)
	s.refereeSM.TriggerEvent(roles.EventName(refEvent))

	// fmt.Println(s.refereeSM.CurrentStateName())
	if s.refereeSM.CurrentStateName() == "RUNNING" {
		return false
	}

	return true
}
