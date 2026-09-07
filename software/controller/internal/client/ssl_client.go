package client

import (
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"time"
)

type SSLClient struct {
	vision        *SSLVisionClient
	referee       *SSLRefereeClient
	lastDetection time.Time
}

const visionTimeout = 250 * time.Millisecond
const controlWakeInterval = 50 * time.Millisecond

type SSLTrackedClient struct {
	vision  *SSLTrackedVisionClient
	referee *SSLRefereeClient
}

func NewSSLClient(visionAddress string) *SSLClient {
	return &SSLClient{
		vision:  NewSSLVisionClient(visionAddress),
		referee: NewSSLRefereeClient(config.GetGCClientAddress()),
	}
}

func NewSSLTrackedClient(visionAddress string) *SSLTrackedClient {
	return &SSLTrackedClient{
		vision:  NewSSLTrackedVisionClient(visionAddress),
		referee: NewSSLRefereeClient(config.GetGCClientAddress()),
	}
}

func (client *SSLClient) UpdateState(gi *info.GameInfo, play_time int64) {
	client.vision.UpdateGameInfo(gi, play_time)
	client.referee.UpdateGameInfo(gi)
}

// WaitForVision also wakes for referee commands and vision loss. Its result
// tells the caller whether motion can still use a recent detection frame.
func (client *SSLClient) WaitForVision(gi *info.GameInfo) bool {
	timer := time.NewTimer(controlWakeInterval)
	defer timer.Stop()
	select {
	case packet, ok := <-client.vision.ssl_channel:
		if ok {
			client.vision.handlePacket(packet, ok, gi, time.Now().UnixMilli())
			if packet.GetDetection() != nil {
				client.lastDetection = time.Now()
			}
		} else {
			client.vision.ssl_channel = nil
			client.lastDetection = time.Time{}
		}
	case packet, ok := <-client.referee.gc_channel:
		if ok {
			client.referee.handlePacket(packet, ok, gi)
		} else {
			client.referee.gc_channel = nil
		}
	case <-timer.C:
	}
	// Drain an already queued referee update even when vision won the select.
	client.referee.UpdateGameInfo(gi)
	return !client.lastDetection.IsZero() && time.Since(client.lastDetection) < visionTimeout
}

func (client *SSLTrackedClient) UpdateState(gi *info.GameInfo, play_time int64) {
	client.vision.UpdateGameInfoTracked(gi, play_time)
}
