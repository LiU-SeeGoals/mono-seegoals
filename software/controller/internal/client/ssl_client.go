package client

import (
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type SSLClient struct {
	vision  *SSLVisionClient
	referee *SSLRefereeClient
}

func NewSSLClient(visionAddress string) *SSLClient {
	return &SSLClient{
		vision:  NewSSLVisionClient(visionAddress),
		referee: NewSSLRefereeClient(config.GetGCClientAddress()),
	}
}

func (client *SSLClient) UpdateState(gi *info.GameInfo, play_time int64) {
	client.vision.UpdateGameInfo(gi, play_time)
	client.referee.UpdateGameInfo(gi)
}
