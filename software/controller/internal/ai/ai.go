package ai

import (
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/action"
	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type planner interface {
	Init(incoming <-chan info.GameInfo, activities *[info.TEAM_SIZE]ai.Activity, lock *sync.Mutex, team info.Team)
}

type executor interface {
	Init(incoming <-chan info.GameInfo,
		activities *[info.TEAM_SIZE]ai.Activity,
		lock *sync.Mutex,
		outgoing chan<- []action.Action,
		team info.Team,
	)
}

type Ai struct {
	team             info.Team
	planner             planner
	executor         executor
	gameInfoSenderSB chan<- info.GameInfo
	gameInfoSenderFB chan<- info.GameInfo
	actionReceiver   chan []action.Action
	activities       *[info.TEAM_SIZE]ai.Activity // Shared slice of Activity
	activity_lock    *sync.Mutex                  // Shared mutex for synchronization
}

// Constructor for the AI
func NewAi(team info.Team, planner planner, executor executor) *Ai {
	// Create a shared slice of Activity and a mutex
	activities := &[info.TEAM_SIZE]ai.Activity{}
	lock := &sync.Mutex{}

	gameInfoSenderSB, gameInfoReceiverSB := helper.NB_KeepLatestChan[info.GameInfo]()
	gameInfoSenderFB, gameInfoReceiverFB := helper.NB_KeepLatestChan[info.GameInfo]()
	actionReceiver := make(chan []action.Action)

	// Initialize plan and executor with the shared resources
	planner.Init(gameInfoReceiverSB, activities, lock, team)
	executor.Init(gameInfoReceiverFB, activities, lock, actionReceiver, team)

	// Construct the AI object
	ai := &Ai{
		team:             team,
		planner:             planner,
		executor:         executor,
		activities:       activities,
		gameInfoSenderSB: gameInfoSenderSB,
		gameInfoSenderFB: gameInfoSenderFB,
		activity_lock:    lock,
		actionReceiver:   actionReceiver,
	}
	return ai
}

// Decides on new actions for the robots
func (ai *Ai) GetActions(gi *info.GameInfo) []action.Action {

	// Send the game state copy to the plan so its aware of the environment
	ai.gameInfoSenderSB <- *gi

	// Send the game state to the executor so it can execute gamestate aware activities (e.g. avoid obstacles)	
	ai.gameInfoSenderFB <- *gi

	// Get the actions from the executor, this will block until it has decided on actions
	actions := <-ai.actionReceiver
	if len(actions) > 0 {
	}
	return actions
}
