package client

import (
	"strings"

	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

const visionMaskMarginMM = 250.0

var visionMaskTeam = info.UNKNOWN

func loadVisionMask() {
	setting := strings.TrimSpace(config.RestrictToOpponentHalfTeam())
	visionMaskTeam = info.UNKNOWN
	for _, team := range []info.Team{info.Yellow, info.Blue} {
		if strings.EqualFold(setting, team.String()) {
			visionMaskTeam = team
			Logger.Warnf("Ignoring vision detections on the %v half", team)
		}
	}
}

func visionMasked(gi *info.GameInfo, x float64) bool {
	if visionMaskTeam == info.UNKNOWN || gi == nil {
		return false
	}
	return gi.OwnHalfXSign(visionMaskTeam)*x > visionMaskMarginMM
}

func unmaskedBalls(gi *info.GameInfo, balls []*ssl_vision.SSL_DetectionBall) []*ssl_vision.SSL_DetectionBall {
	if visionMaskTeam == info.UNKNOWN {
		return balls
	}
	kept := make([]*ssl_vision.SSL_DetectionBall, 0, len(balls))
	for _, ball := range balls {
		if !visionMasked(gi, float64(ball.GetX())) {
			kept = append(kept, ball)
		}
	}
	return kept
}
