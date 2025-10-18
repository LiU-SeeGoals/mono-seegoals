package client

import (
	"github.com/LiU-SeeGoals/controller/internal/action"
)

type Client interface {
	Init()
	SendActions(actions []action.Action)
	CloseConnection()
}