package info

import (
	"fmt"
)

// import "gonum.org/v1/gonum/mat"

type GameStatus struct {
	gameEvent *GameEvent
	// These are the "coarse" stages of the game.
	gameStage GameStage
	// The match type is a meta information about the current match that helps to process the logs after a competition
	matchType MatchType
	// The UNIX timestamp when the packet was sent, in microseconds.
	// Divide by 1,000,000 to get a time_t.
	packet_timestamp uint64
	// The number of microseconds left in the stage.
	// The following stages have this value; the rest do not:
	// NORMAL_FIRST_HALF
	// NORMAL_HALF_TIME
	// NORMAL_SECOND_HALF
	// EXTRA_TIME_BREAK
	// EXTRA_FIRST_HALF
	// EXTRA_HALF_TIME
	// EXTRA_SECOND_HALF
	// PENALTY_SHOOTOUT_BREAK
	//
	// If the stage runs over its specified time, this value
	// becomes negative.
	stage_time_left int64
	// The number of commands issued since startup (mod 2^32).
	command_counter uint32
	// Information about the direction of play.
	// True, if the blue team will have it's goal on the positive x-axis of the ssl-vision coordinate system.
	// Obviously, the yellow team will play on the opposite half.
	blue_team_on_positive_half bool
	// A message that can be displayed to the spectators, like a reason for a stoppage.
	status_message string

	// Information about a team.
	yellowInfo *TeamInfo
	blueInfo   *TeamInfo
}

// Constructor for creating a new GameStatus instance
func NewGameStatus() *GameStatus {
	gStatus := &GameStatus{
		gameEvent: NewGameEvent(),
		// gameStage:                  0,
		// matchType:                  0,
		// packet_timestamp:           0,
		// stage_time_left:            0,
		// command_counter:            0,
		// blue_team_on_positive_half: false,
		// status_message:             "",
		yellowInfo: NewTeamInfo(),
		blueInfo:   NewTeamInfo(),
	}
	return gStatus
}

// String method for GameStatus
func (gs *GameStatus) String() string {
	return fmt.Sprintf(
		"Game Status:\n"+
			"  Packet Timestamp: %d\n"+
			"  Match Type: %s\n"+
			"  Game Stage: %s\n"+
			"  Stage Time Left: %d microseconds\n"+
			"  Command Counter: %d\n"+
			"  Blue Team on Positive Half: %t\n"+
			"  Status Message: %s\n",
		gs.packet_timestamp,
		gs.matchType.String(),
		gs.gameStage.String(),
		gs.stage_time_left,
		gs.command_counter,
		gs.blue_team_on_positive_half,
		gs.status_message,
	)
}

// Setters

// SetTeamInfo sets the information for either the yellow or blue team.
func (gs *GameStatus) SetTeamInfo(yellow bool,
	name string,
	score,
	redCards,
	yellowCards,
	timeouts,
	timeoutTime,
	goalkeeper,
	foulCounter,
	ballPlacementFailures,
	maxAllowedBots,
	botSubstitutionsLeft,
	botSubstitutionTimeLeft uint32,
	yellowCardTimes []uint32,
	canPlaceBall,
	botSubstitutionIntent,
	ballPlacementFailuresReached,
	botSubstitutionAllowed bool) {

	teamInfo := gs.blueInfo // Default to blue team

	if yellow {
		teamInfo = gs.yellowInfo // If yellow is true, set to yellow team
	}

	teamInfo.SetName(name)
	teamInfo.SetScore(score)
	teamInfo.SetRedCards(redCards)
	teamInfo.SetYellowCardTimes(yellowCardTimes)
	teamInfo.SetYellowCards(yellowCards)
	teamInfo.SetTimeouts(timeouts)
	teamInfo.SetTimeoutTime(timeoutTime)
	teamInfo.SetGoalkeeper(goalkeeper)
	teamInfo.SetFoulCounter(foulCounter)
	teamInfo.SetBallPlacementFailures(ballPlacementFailures)
	teamInfo.SetCanPlaceBall(canPlaceBall)
	teamInfo.SetMaxAllowedBots(maxAllowedBots)
	teamInfo.SetBotSubstitutionIntent(botSubstitutionIntent)
	teamInfo.SetBallPlacementFailuresReached(ballPlacementFailuresReached)
	teamInfo.SetBotSubstitutionAllowed(botSubstitutionAllowed)
	teamInfo.SetBotSubstitutionsLeft(botSubstitutionsLeft)
	teamInfo.SetBotSubstitutionTimeLeft(botSubstitutionTimeLeft)
}

func (gs *GameStatus) SetGameEvent(refCommand RefCommand,
	commandTimestamp uint64,
	desPosX float64,
	desPosY float64,
	nextCommand RefCommand,
	currentActionTimeRemaining int64) {

	gs.gameEvent.UpdateFromRefCommand(
		refCommand,
		commandTimestamp,
		desPosX,
		desPosY,
		nextCommand,
		currentActionTimeRemaining)
}

func (gs *GameStatus) SetGameStatus(
	stage GameStage,
	matchType MatchType,
	timestamp uint64,
	timeLeft int64,
	counter uint32,
	positiveHalf bool,
	message string) {

	gs.gameStage = stage
	gs.matchType = matchType
	gs.packet_timestamp = timestamp
	gs.stage_time_left = timeLeft
	gs.command_counter = counter
	gs.blue_team_on_positive_half = positiveHalf
	gs.status_message = message
}

func (gs *GameStatus) GetGameStage() GameStage {
	return gs.gameStage
}

func (gs *GameStatus) GetMatchType() MatchType {
	return gs.matchType
}

func (gs *GameStatus) GetPacketTimestamp() uint64 {
	return gs.packet_timestamp
}

func (gs *GameStatus) GetStageTimeLeft() int64 {
	return gs.stage_time_left
}

func (gs *GameStatus) GetCommandCounter() uint32 {
	return gs.command_counter
}

func (gs *GameStatus) GetBlueTeamOnPositiveHalf() bool {
	return gs.blue_team_on_positive_half
}

func (gs *GameStatus) GetStatusMessage() string {
	return gs.status_message
}

func (gs *GameStatus) GetGameEvent() *GameEvent {
	return gs.gameEvent
}

func (gs *GameStatus) GetTeamInfo(yellow bool) *TeamInfo {
	teamInfo := gs.blueInfo

	if yellow {
		teamInfo = gs.yellowInfo
	}

	return teamInfo
}
