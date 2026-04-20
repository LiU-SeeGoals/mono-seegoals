package referee

import (
	"fmt"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	FREEKICK_MAX_TIME     = 10
	BALL_IN_PLAY_DISTANCE = 1000
	BALL_IN_PLAY_VELOCITY = 1
)

const (
	HALT            = "Halt"
	STOP            = "Stop"
	NORMAL_START    = "NormalStart"
	FORCE_START     = "ForceStart"
	PREPARE_KICKOFF = "PrepareKickoff"
	FREE_KICK       = "FreeKick"
	BALL_PLACEMENT  = "BallPlacement"
	TIMEOUT         = "Timeout"
	CONTINUE        = "Continue"
	UNKNOWN         = "Unknown"
)

const (
	GAME_RUNNING_DETECTED = "Running"
)

type RefereeInfo struct {
	gi                *info.GameInfo
	activeRobots      []info.ID
	team              info.Team
	activityHandler   *ai.ActivityHandler
	name              StateName
	teamWithPossesion info.Team
}

func (s *RefereeInfo) GetName() StateName {
	return s.name
}

type Halt struct {
	RefereeInfo
}

type FreeKick struct {
	RefereeInfo
	freeKick        *StateMachine
	freeKickStart   time.Time
	originalBallPos info.Position
}

type Stop struct {
	RefereeInfo
}

type PrepareKickoff struct {
	RefereeInfo
	kickOff *StateMachine
	recieve *StateMachine
}

type Kickoff struct {
	RefereeInfo
	kickOff         *StateMachine
	recieve         *StateMachine
	originalBallPos info.Position
	kickStart       time.Time
}

type Running struct {
}

func (s *Halt) Initialize() {
}

func (s *Halt) Update() EventName {

	for _, id := range s.activeRobots {
		activity := act.NewStop(id)
		s.activityHandler.AddActivity(activity)
	}

	return "NONE"
}

func (s *Stop) Initialize() {
}

func (s *Stop) Update() EventName {

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
	return info.Position{X: 0, Y: 1000, Z: 0, Angle: 0}
}

type KickOffIntent struct {
	gi   *info.GameInfo
	team info.Team
	id   info.ID
}

func (s *KickOffIntent) GetTargetPosition() info.Position {
	receiverID := info.ID(2)
	pos, err := s.gi.State.GetRobotPosition(s.team, receiverID)
	if err != nil {
		return info.Position{X: 0, Y: 1000, Z: 0, Angle: 0}
	}
	return pos
}

func (s *KickOffIntent) GetFromPosition() info.Position {
	return info.Position{}
}

func (s *FreeKick) Initialize() {

	kickOffID := info.ID(1)
	kickPrepareName := StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))
	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}
	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	// kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	// After 10 seconds we are allowed to kick the ball
	s.freeKickStart = time.Now()
	originalBallPos, ok := s.gi.State.GetTrackedBall().GetTrackedPosition()
	if !ok {
		fmt.Println("Failed getting original ball pos during FreeKick")
	}

	s.freeKick = NewStateMachine(prepareKick)

	s.originalBallPos = originalBallPos

}

func (s *FreeKick) Update() EventName {

	trackedBall := s.gi.State.GetTrackedBall()
	vel, ok := trackedBall.GetTrackedVelocity()
	pos, ok := trackedBall.GetTrackedPosition()

	if ok {
		posDiff := pos.Sub(&s.originalBallPos)
		// If ball has velocity or has moved far enough from the start
		// We can assume that the ball touched rule is fullfilled
		if vel.Norm2d() > BALL_IN_PLAY_VELOCITY || posDiff.Norm2d() > BALL_IN_PLAY_DISTANCE || time.Now().Sub(s.freeKickStart) > time.Second*10 {
			return GAME_RUNNING_DETECTED
		}
	}

	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		s.freeKick.Update()
	} else {
		moveRobotsToDefensePosition(s.activeRobots, s.team, s.activityHandler)
	}

	return "NONE"
}

func (s *PrepareKickoff) Initialize() {

	kickOffID := info.ID(1)
	recieveID := info.ID(3)

	kickPrepareName := StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))

	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}
	recieve := RecieveIntent{gi: s.gi, team: s.team, id: kickOffID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	reciever := &roles.AlignState{Ctx: &recieve, Gi: s.gi, Team: s.team, RobotId: recieveID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	// kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}

	s.kickOff = NewStateMachine(prepareKick)
	s.recieve = NewStateMachine(reciever)

}

func (s *PrepareKickoff) Update() EventName {

	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		s.kickOff.Update()
		s.recieve.Update()
	} else {
		moveRobotsToDefensePosition(s.activeRobots, s.team, s.activityHandler)
	}

	return "NONE"
}

func (s *Kickoff) Initialize() {

	kickOffID := info.ID(1)
	recieveID := info.ID(2)

	kickPrepareName := StateName(fmt.Sprintf("KickPrepare ID %d", kickOffID))
	kickName := StateName(fmt.Sprintf("Kick ID %d", kickOffID))

	recieveName := StateName(fmt.Sprintf("Kickoff recieve ID %d", recieveID))
	originalBallPos, ok := s.gi.State.GetTrackedBall().GetTrackedPosition()
	s.kickStart = time.Now()

	if !ok {
		fmt.Println("Failed retriving original ball pos during KickOff")
	}
	s.originalBallPos = originalBallPos

	kickoff := KickOffIntent{gi: s.gi, team: s.team, id: kickOffID}
	recieve := RecieveIntent{gi: s.gi, team: s.team, id: recieveID}

	prepareKick := &roles.AlignState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickPrepareName, ActivityHandler: s.activityHandler}
	kick := &roles.KickState{Ctx: &kickoff, Gi: s.gi, Team: s.team, RobotId: kickOffID, Name: kickName, ActivityHandler: s.activityHandler}
	reciever := &roles.AlignState{Ctx: &recieve, Gi: s.gi, Team: s.team, RobotId: recieveID, Name: recieveName, ActivityHandler: s.activityHandler}

	s.kickOff = NewStateMachine(prepareKick)
	s.kickOff.AddTransition(kickPrepareName, "ALIGNED", kick)

	s.recieve = NewStateMachine(reciever)
	// If we dont have possession we take good defense position
}

func (s *Kickoff) Update() EventName {

	trackedBall := s.gi.State.GetTrackedBall()
	vel, ok := trackedBall.GetTrackedVelocity()
	pos, ok := trackedBall.GetTrackedPosition()

	if ok {
		posDiff := pos.Sub(&s.originalBallPos)
		// If ball has velocity or has moved far enough from the start
		// We can assume that the ball touched rule is fullfilled
		if vel.Norm2d() > BALL_IN_PLAY_VELOCITY || posDiff.Norm2d() > BALL_IN_PLAY_DISTANCE || time.Now().Sub(s.kickStart) > time.Second*10 {
			return GAME_RUNNING_DETECTED
		}
	}

	if s.gi.Status.GetGameEvent().TeamWithPossession == s.team {
		s.kickOff.Update()
		s.recieve.Update()
	} else {
		moveRobotsToDefensePosition(s.activeRobots, s.team, s.activityHandler)
	}

	return "NONE"
}

func (s *Running) Initialize() {
}

func (s *Running) GetName() StateName {
	return "RUNNING"
}

func (s *Running) Update() EventName {
	return "NONE"
}

type UninitializedRef struct {
}

func (s *UninitializedRef) Initialize() {
}

func (s *UninitializedRef) GetName() StateName {
	return "UNINITIALIZED"
}

func (s *UninitializedRef) Update() EventName {
	return "NONE"
}

type RefereeHandler struct {
	gi           *info.GameInfo
	refereeSM    *StateMachine
	activeRobots []info.ID
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

func moveRobotsToDefensePosition(activeRobots []info.ID, team info.Team, activityHandler *ai.ActivityHandler) {

	numRobots := float64(len(activeRobots))
	increment := 120.0
	minPos := -increment * numRobots

	for i, robotID := range activeRobots {
		pos := info.Position{X: -2000, Y: minPos + increment*float64(i), Z: 0, Angle: 0}
		activityHandler.AddActivity(act.NewMoveToPosition(team, info.ID(robotID), pos))
	}
}

func NewRefereeHandler(gi *info.GameInfo, activeRobots []info.ID, team info.Team, activityHandler *ai.ActivityHandler) *RefereeHandler {

	freeKickName := StateName("FREEKICK")
	stopName := StateName("STOP")
	haltName := StateName("HALT")
	prepareKickoffName := StateName("PREPAREKICKOFF")
	kickOffName := StateName("KICKOFF")
	ballPlacementName := StateName("BALLPLACEMENT")

	freeKick := &FreeKick{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            freeKickName,
			activityHandler: activityHandler,
		},
	}

	prepareKickoff := &PrepareKickoff{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            prepareKickoffName,
			activityHandler: activityHandler,
		},
	}
	stop := &Stop{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            stopName,
			activityHandler: activityHandler,
		},
	}
	ballPlacement := &Stop{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            ballPlacementName,
			activityHandler: activityHandler,
		},
	}
	halt := &Halt{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            haltName,
			activityHandler: activityHandler,
		},
	}
	kickOff := &Kickoff{
		RefereeInfo: RefereeInfo{
			gi:              gi,
			activeRobots:    activeRobots,
			team:            team,
			name:            kickOffName,
			activityHandler: activityHandler,
		},
	}
	running := &Running{}
	uninitialized := &UninitializedRef{}

	// Note that some states are not implemented
	// e.g. Timeout, ballplacement, PreparePenalty and Penalty
	// Since it is not required to play a game

	refereeSM := NewStateMachine(uninitialized)

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
	refereeSM.AddTransition("KICKOFF", STOP, stop)

	refereeSM.AddTransition("FREEKICK", GAME_RUNNING_DETECTED, running)
	refereeSM.AddTransition("FREEKICK", STOP, stop)

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
	refereeSM.AddTransition(uninitialized.GetName(), FREE_KICK, freeKick)

	return &RefereeHandler{gi, refereeSM, activeRobots}
}

/*
Returns true if referee is being handeled
Returns false if game is in running state
*/
func (s *RefereeHandler) HandleReferee() bool {

	// Rules to follow are defined in the ssl Rules
	// Appendix B: Game States https://robocup-ssl.github.io/ssl-rules/sslrules.html

	refereeCommand := s.gi.Status.GetGameEvent().RefCommand
	refEvent := refCommandToEventName(refereeCommand)
	s.refereeSM.TriggerEvent(EventName(refEvent))

	s.refereeSM.Update()
	if s.refereeSM.CurrentStateName() == "RUNNING" {
		return false
	}

	return true
}
