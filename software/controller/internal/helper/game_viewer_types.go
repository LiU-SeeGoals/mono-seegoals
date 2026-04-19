package helper

// Command types
const (
	CHANGE_SCENARIO = "CHANGE_SCENARIO"
	MOVE_ROBOT      = "MOVE_ROBOT"
)

type GameViewerCommand struct {
	CommandType string `json:"Command"`
	X           int32  `json:"x"`
	Y           int32  `json:"y"`
	Id          int    `json:"Id"`
	Type        string `json:"Type"`
}
