package referee

import (
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
)

// Halt stops every robot. The executor independently refuses to send motion
// while halted, so this only keeps the activities consistent.
type Halt struct {
	RefereeInfo
}

// Timeout leaves the robots idle until STOP resumes match preparation.
type Timeout struct {
	Halt
}

func (s *Halt) Initialize() {}

func (s *Halt) Update() state_machine.EventName {
	for _, id := range s.activeRobots {
		s.activityHandler.AddActivity(activity.NewStop(id))
	}
	return state_machine.NoEvent
}

// BallPlacement keeps every robot clear of the ball and the placement target
// while the other team (or a human) places the ball. The goalie returns to its
// goal and, if the next command is our free kick, the kicker already moves
// behind the ball.
type BallPlacement struct {
	RefereeInfo
}

func (s *BallPlacement) Initialize() {}

func (s *BallPlacement) Update() state_machine.EventName {
	goalieID, hasGoalie := moveGoalieToPosition(s.activeRobots, s.team, s.gi, s.activityHandler)
	preparedKickerID, preparingFreeKick := PrepareKickerForUpcomingFreeKick(
		s.gi,
		s.team,
		s.activeRobots,
		s.activityHandler,
	)

	for _, id := range s.activeRobots {
		if hasGoalie && id == goalieID {
			continue
		}
		if preparingFreeKick && id == preparedKickerID {
			continue
		}
		s.activityHandler.AddActivity(activity.NewRefStop(s.team, id))
	}
	return state_machine.NoEvent
}
