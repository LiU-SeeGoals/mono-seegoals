package client

import (
	"fmt"
	"net"

	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
	"github.com/LiU-SeeGoals/proto_go/gc"
	"google.golang.org/protobuf/proto"
)

const READ_BUFFER_SIZE_TRACKED = 8192

type SSLTrackedConnection struct {
	conn *net.UDPConn
	addr *net.UDPAddr
	buff []byte
}

func NewSSLTrackedConnection(addr string) *SSLTrackedConnection {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}

	return &SSLTrackedConnection{
		conn: nil,
		addr: udpAddr,
		buff: make([]byte, READ_BUFFER_SIZE_TRACKED),
	}
}

func (r *SSLTrackedConnection) ConnectTracked() {
	conn, err := net.ListenMulticastUDP("udp", nil, r.addr)
	if err != nil {
		panic(err)
	}

	r.conn = conn
}

func (r *SSLTrackedConnection) ReceiveTracked(packetChan chan *ssl_vision.TrackerWrapperPacket) {
	for {
		sz, err := r.conn.Read(r.buff)
		if err != nil {
			continue
		}

		packet := &ssl_vision.TrackerWrapperPacket{}
		err = proto.Unmarshal(r.buff[:sz], packet)
		if err != nil {
			continue
		}

		helper.NB_Send[ssl_vision.TrackerWrapperPacket](packetChan, packet)
	}
}

type SSLTrackedVisionClient struct {
	ssl         *SSLTrackedConnection
	ssl_channel chan *ssl_vision.TrackerWrapperPacket
}

func unpackTracked(packet *ssl_vision.TrackerWrapperPacket, gi *info.GameInfo, play_time int64) {
	frame := packet.GetTrackedFrame()
	if frame == nil {
		return
	}

	ts := frame.GetTimestamp()
	gi.State.SetTimestamp(ts)
	gi.State.SetMessageReceivedTime(play_time)

	for _, robot := range frame.GetRobots() {
		pos := robot.GetPos()
		vel := robot.GetVel()
		robotId := robot.GetRobotId()

		p := info.Position{
			X: float64(pos.GetX()),
			Y: float64(pos.GetY()),
			Z: 0,
			Angle: float64(robot.GetOrientation()),
		}

		v := info.Position{}
		if vel != nil {
			v.X = float64(vel.GetX())
			v.Y = float64(vel.GetY())
		}

		vtheta := float64(robot.GetVelAngular())

		var team info.Team
		switch robotId.GetTeam() {
		case gc.Team_BLUE:
			team = info.Blue
		case gc.Team_YELLOW:
			team = info.Yellow
		default:
			team = info.UNKNOWN
		}

		gi.State.SetTrackedRobot(team, robotId.GetId(), p, v, p.Angle, vtheta, ts)
	}

	balls := frame.GetBalls()
	if len(balls) > 0 {
		b := balls[0]
		pos := b.GetPos()
		vel := b.GetVel()

		p := info.Position{
			X: float64(pos.GetX()*1000),
			Y: float64(pos.GetY()*1000),
			Z: float64(pos.GetZ()*1000),
			Angle: 0,
		}

		v := info.Position{}
		if vel != nil {
			v.X = float64(vel.GetX())
			v.Y = float64(vel.GetY())
			v.Z = float64(vel.GetZ())
		}

		gi.State.SetTrackedBall(p, v, ts)
	}

	kicked := frame.GetKickedBall()
	if kicked != nil {
		pos := kicked.GetPos()
		stopPos := kicked.GetStopPos()
		startTs := kicked.GetStartTimestamp()
		stopTs := kicked.GetStopTimestamp()
		rid := kicked.GetRobotId()

		var team info.Team
		switch rid.GetTeam() {
		case gc.Team_BLUE:
			team = info.Blue
		case gc.Team_YELLOW:
			team = info.Yellow
		default:
			team = info.UNKNOWN
		}

		gi.State.SetKickedBall(
			float64(pos.GetX()),
			float64(pos.GetY()),
			startTs,
			float64(stopPos.GetX()),
			float64(stopPos.GetY()),
			stopTs,
			team,
			rid.GetId(),
		)
	}

	gi.State.SetValid(true)
}

func (receiver *SSLTrackedVisionClient) handleTrackedPacket(packet *ssl_vision.TrackerWrapperPacket, ok bool, gi *info.GameInfo, play_time int64) {
	if !ok {
		fmt.Println("Tracked SSL Channel closed")
		return
	}

	unpackTracked(packet, gi, play_time)
}

func (receiver *SSLTrackedVisionClient) UpdateGameInfoTracked(gi *info.GameInfo, play_time int64) {
	select {
	case packet, ok := <-receiver.ssl_channel:
		receiver.handleTrackedPacket(packet, ok, gi, play_time)
	default:
	}
}

func (receiver *SSLTrackedVisionClient) ConnectTracked() {
	receiver.ssl.ConnectTracked()
	go receiver.ssl.ReceiveTracked(receiver.ssl_channel)
}

func NewSSLTrackedVisionClient(addr string) *SSLTrackedVisionClient {
	ch := make(chan *ssl_vision.TrackerWrapperPacket, 1)
	receiver := &SSLTrackedVisionClient{
		ssl:         NewSSLTrackedConnection(addr),
		ssl_channel: ch,
	}
	receiver.ConnectTracked()
	return receiver
}

