package client

import (
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/gc"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

func testVisionClient() *SSLClient {
	return &SSLClient{
		vision:  &SSLVisionClient{ssl_channel: make(chan *ssl_vision.SSL_WrapperPacket, 1)},
		referee: &SSLRefereeClient{gc_channel: make(chan *gc.Referee, 1)},
	}
}

func TestVisionWaitProcessesHaltWithoutVision(t *testing.T) {
	client := testVisionClient()
	gi := info.NewGameInfo(10)
	gi.Status.GetGameEvent().CurrentState = info.STATE_PLAYING
	command := gc.Referee_HALT
	client.referee.gc_channel <- &gc.Referee{Command: &command}
	if client.WaitForVision(gi) {
		t.Fatal("missing detections must not permit motion")
	}
	if gi.Status.GetGameEvent().CurrentState != info.STATE_HALTED {
		t.Fatal("HALT was not processed without vision")
	}
}

func TestVisionWaitExpiresAndRecovers(t *testing.T) {
	client := testVisionClient()
	gi := info.NewGameInfo(10)
	client.lastDetection = time.Now().Add(-visionTimeout)
	client.vision.ssl_channel <- &ssl_vision.SSL_WrapperPacket{}
	if client.WaitForVision(gi) {
		t.Fatal("a geometry-only packet must not refresh detection freshness")
	}
	client.vision.ssl_channel <- &ssl_vision.SSL_WrapperPacket{Detection: &ssl_vision.SSL_DetectionFrame{}}
	if !client.WaitForVision(gi) {
		t.Fatal("fresh detection did not restore motion")
	}
	client.lastDetection = time.Now().Add(-visionTimeout)
	started := time.Now()
	if client.WaitForVision(gi) {
		t.Fatal("stale detections must not permit motion")
	}
	if time.Since(started) > time.Second {
		t.Fatal("vision loss did not wake the control loop promptly")
	}
}
