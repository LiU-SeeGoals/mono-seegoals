// Package action defines the commands the AI gives individual robots.
//
// Actions are plain domain values. They know nothing about wire formats:
// client.BaseStationClient encodes them for the real robots, client.SimClient
// turns them into simulator commands, and the GameViewer server builds its
// JSON from them.
package action

// Action is a command for one robot.
type Action interface {
	RobotID() int
}
