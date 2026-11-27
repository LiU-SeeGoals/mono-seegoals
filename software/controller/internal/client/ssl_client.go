package client

import (
	"fmt"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

type SSLClient struct {
	vision           *SSLVisionClient
	referee          *SSLRefereeClient
	lastDetection    time.Time
	lastTimingReport [16]time.Time
}

const visionTimeout = 250 * time.Millisecond
const controlWakeInterval = 50 * time.Millisecond

// SSLTrackedClient receives tracker frames only; referee packets come
// through SSLClient.
type SSLTrackedClient struct {
	vision *SSLTrackedVisionClient
}

func NewSSLClient(visionAddress string) *SSLClient {
	loadVisionMask()
	return &SSLClient{
		vision:  NewSSLVisionClient(visionAddress),
		referee: NewSSLRefereeClient(config.GetGCClientAddress()),
	}
}

func NewSSLTrackedClient(visionAddress string) *SSLTrackedClient {
	loadVisionMask()
	return &SSLTrackedClient{vision: NewSSLTrackedVisionClient(visionAddress)}
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
		client.handleVision(packet, ok, gi)
		// Each camera sends its own packet; apply every one already queued so
		// a frame sees all cameras, not just the first to arrive.
		for drained := false; !drained && client.vision.ssl_channel != nil; {
			select {
			case packet, ok := <-client.vision.ssl_channel:
				client.handleVision(packet, ok, gi)
			default:
				drained = true
			}
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

func (client *SSLClient) handleVision(packet *ssl_vision.SSL_WrapperPacket, ok bool, gi *info.GameInfo) {
	if !ok {
		client.vision.ssl_channel = nil
		client.lastDetection = time.Time{}
		return
	}
	client.vision.handlePacket(packet, ok, gi, time.Now().UnixMilli())
	if detection := packet.GetDetection(); detection != nil {
		client.lastDetection = time.Now()
		cameraID := detection.GetCameraId()
		if cameraID < uint32(len(client.lastTimingReport)) {
			reportVisionTiming(fmt.Sprintf("raw camera %d", cameraID), detection.GetTCapture(), detection.GetTSent(), &client.lastTimingReport[cameraID])
		}
	}
}

func (client *SSLTrackedClient) UpdateState(gi *info.GameInfo, play_time int64) {
	client.vision.UpdateGameInfoTracked(gi, play_time)
}
