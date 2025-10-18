package info

import (
	"fmt"
	"strings"
)

type TeamInfo struct {
	// The team's name (empty string if operator has not typed anything).
	Name string
	// The number of goals scored by the team during normal play and overtime.
	Score uint32
	// The number of red cards issued to the team since the beginning of the game.
	RedCards uint32
	// The amount of time (in microseconds) left on each yellow card issued to the team.
	// If no yellow cards are issued, this array has no elements.
	// Otherwise, times are ordered from smallest to largest.
	YellowCardTimes []uint32
	// The total number of yellow cards ever issued to the team.
	YellowCards uint32
	// The number of timeouts this team can still call.
	// If in a timeout right now, that timeout is excluded.
	Timeouts uint32
	// The number of microseconds of timeout this team can use.
	TimeoutTime uint32
	// The pattern number of this team's goalkeeper.
	Goalkeeper uint32
	// The total number of countable fouls that act towards yellow cards.
	FoulCounter uint32
	// The number of consecutive ball placement failures of this team.
	BallPlacementFailures uint32
	// Indicate if the team is able and allowed to place the ball.
	CanPlaceBall bool
	// The maximum number of bots allowed on the field based on division and cards.
	MaxAllowedBots uint32
	// The team has submitted an intent to substitute one or more robots at the next chance.
	BotSubstitutionIntent bool
	// Indicate if the team reached the maximum allowed ball placement failures and is thus not allowed to place the ball anymore.
	BallPlacementFailuresReached bool
	// The team is allowed to substitute one or more robots currently.
	BotSubstitutionAllowed bool
	// The number of bot substitutions left by the team in this halftime.
	BotSubstitutionsLeft uint32
	// The number of microseconds left for current bot substitution.
	BotSubstitutionTimeLeft uint32
}

// String method for TeamInfo
func (t *TeamInfo) String() string {
	// Format YellowCardTimes into a comma-separated string
	yellowCardTimesStr := strings.Trim(strings.Replace(fmt.Sprint(t.YellowCardTimes), " ", ", ", -1), "[]")

	// Format the TeamInfo struct into a readable string
	return fmt.Sprintf(
		"Team Name: %s\n"+
			"  Score: %d\n"+
			"  Red Cards: %d\n"+
			"  Yellow Card Times: [%s]\n"+
			"  Yellow Cards: %d\n"+
			"  Timeouts: %d\n"+
			"  Timeout Time (microseconds): %d\n"+
			"  Goalkeeper Number: %d\n"+
			"  Foul Counter: %d\n"+
			"  Ball Placement Failures: %d\n"+
			"  Can Place Ball: %t\n"+
			"  Max Allowed Bots: %d\n"+
			"  Bot Substitution Intent: %t\n"+
			"  Ball Placement Failures Reached: %t\n"+
			"  Bot Substitution Allowed: %t\n"+
			"  Bot Substitutions Left: %d\n"+
			"  Bot Substitution Time Left (microseconds): %d",
		t.Name,
		t.Score,
		t.RedCards,
		yellowCardTimesStr,
		t.YellowCards,
		t.Timeouts,
		t.TimeoutTime,
		t.Goalkeeper,
		t.FoulCounter,
		t.BallPlacementFailures,
		t.CanPlaceBall,
		t.MaxAllowedBots,
		t.BotSubstitutionIntent,
		t.BallPlacementFailuresReached,
		t.BotSubstitutionAllowed,
		t.BotSubstitutionsLeft,
		t.BotSubstitutionTimeLeft,
	)
}

// Setters for TeamInfo fields
func (t *TeamInfo) SetName(name string) {
	t.Name = name
}

func (t *TeamInfo) SetScore(score uint32) {
	t.Score = score
}

func (t *TeamInfo) SetRedCards(redCards uint32) {
	t.RedCards = redCards
}

func (t *TeamInfo) SetYellowCardTimes(yellowCardTimes []uint32) {
	t.YellowCardTimes = yellowCardTimes
}

func (t *TeamInfo) SetYellowCards(yellowCards uint32) {
	t.YellowCards = yellowCards
}

func (t *TeamInfo) SetTimeouts(timeouts uint32) {
	t.Timeouts = timeouts
}

func (t *TeamInfo) SetTimeoutTime(timeoutTime uint32) {
	t.TimeoutTime = timeoutTime
}

func (t *TeamInfo) SetGoalkeeper(goalkeeper uint32) {
	t.Goalkeeper = goalkeeper
}

func (t *TeamInfo) SetFoulCounter(foulCounter uint32) {
	t.FoulCounter = foulCounter
}

func (t *TeamInfo) SetBallPlacementFailures(ballPlacementFailures uint32) {
	t.BallPlacementFailures = ballPlacementFailures
}

func (t *TeamInfo) SetCanPlaceBall(canPlaceBall bool) {
	t.CanPlaceBall = canPlaceBall
}

func (t *TeamInfo) SetMaxAllowedBots(maxAllowedBots uint32) {
	t.MaxAllowedBots = maxAllowedBots
}

func (t *TeamInfo) SetBotSubstitutionIntent(botSubstitutionIntent bool) {
	t.BotSubstitutionIntent = botSubstitutionIntent
}

func (t *TeamInfo) SetBallPlacementFailuresReached(ballPlacementFailuresReached bool) {
	t.BallPlacementFailuresReached = ballPlacementFailuresReached
}

func (t *TeamInfo) SetBotSubstitutionAllowed(botSubstitutionAllowed bool) {
	t.BotSubstitutionAllowed = botSubstitutionAllowed
}

func (t *TeamInfo) SetBotSubstitutionsLeft(botSubstitutionsLeft uint32) {
	t.BotSubstitutionsLeft = botSubstitutionsLeft
}

func (t *TeamInfo) SetBotSubstitutionTimeLeft(botSubstitutionTimeLeft uint32) {
	t.BotSubstitutionTimeLeft = botSubstitutionTimeLeft
}

// Constructor for creating a new TeamInfo instance
func NewTeamInfo() *TeamInfo {
	return &TeamInfo{}
}
