package info

import (
	"container/list"
	"fmt"
)

type rawRobotPos struct {
	pos  Position
	time int64
}
type rawRobot struct {
	active          bool
	id              ID
	team            Team
	history         *list.List
	historyCapacity int
}

func (r *rawRobot) SetPositionTime(x, y, angle float64, time int64) {
	r.active = true
	if r.history.Len() >= r.historyCapacity {
		element := r.history.Back()
		r.history.Remove(element)

		robot := element.Value.(*rawRobotPos)

		robot.pos.X = x
		robot.pos.Y = y
		robot.pos.Angle = angle
		robot.time = time

		r.history.PushFront(robot)
	} else {
		pos := Position{x, y, 0, angle}
		r.history.PushFront(&rawRobotPos{pos, time})
	}
}

func (r *rawRobot) GetPositionTime() (Position, int64, error) {
	if r.history.Len() == 0 {
		return Position{}, 0, fmt.Errorf("No position in history for robot %d %s", r.id, r.team.String())
		// panic("No position in history for robot " + fmt.Sprint(r.id) + " " + r.team.String())
	}

	element := r.history.Front()
	robot := element.Value.(*rawRobotPos)
	return robot.pos, robot.time, nil
}

func (r *rawRobot) GetPosition() (Position, error) {
	pos, _, err := r.GetPositionTime()
	return pos, err
}

func (r *rawRobot) GetID() ID {
	return r.id
}
func (r *rawRobot) SetActive(active bool) {
	r.active = active
}

func (r *rawRobot) IsActive() bool {
	return r.active
}
