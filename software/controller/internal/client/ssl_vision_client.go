package client

import (
	"fmt"
	"net"

	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
	"google.golang.org/protobuf/proto"
)

const (
	// Read buffer size
	READ_BUFFER_SIZE = 8192
)

// SSL Vision receiver
type SSLConnection struct {
	// Connection
	conn *net.UDPConn
	// UDP address
	addr *net.UDPAddr
	// Read buffer
	buff []byte
	// SSL lets not heap allocate this every time
	packet ssl_vision.SSL_WrapperPacket
}

// Create a new SSL vision receiver.
// Address should be <ip>:<port>
func NewSSLConnection(addr string) *SSLConnection {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}

	return &SSLConnection{
		conn: nil,
		addr: udpAddr,
		buff: make([]byte, READ_BUFFER_SIZE),
	}
}

// Connect/subscribe receiver to UDP multicast.
// Note, this will NOT block.
func (r *SSLConnection) Connect() {
	conn, err := net.ListenMulticastUDP("udp", nil, r.addr)
	if err != nil {
		panic(err)
	}

	r.conn = conn
}

// Start receiving packets.
// This function should be run in a goroutine:
//
//	go recv.Receive()
//
// Parsed packets are transferred using packetChan.
func (r *SSLConnection) Receive(packetChan chan *ssl_vision.SSL_WrapperPacket) {
	for {
		sz, err := r.conn.Read(r.buff)
		if err != nil {
			// fmt.Printf("Unable to receive packet: %s", err)
			Logger.Errorf("Unable to receive packet: %v", err)
			continue
		}

		err = proto.Unmarshal(r.buff[:sz], &r.packet)
		if err != nil {
			// fmt.Printf("Unable to unmarshal packet: %s", err)
			Logger.Errorf("Unable to unmarshal packet: %v", err)
			continue
		}
		// packetChan <- &r.packet
		helper.NB_Send[ssl_vision.SSL_WrapperPacket](packetChan, &r.packet)
	}
}

type SSLVisionClient struct {
	ssl         *SSLConnection
	ssl_channel chan *ssl_vision.SSL_WrapperPacket
}

func unpack(packet *ssl_vision.SSL_WrapperPacket, gi *info.GameInfo, play_time int64) {
	detect := packet.GetDetection()
	gi.State.SetMessageReceivedTime(play_time)

	for _, robot := range detect.GetRobotsBlue() {
		x := float64(robot.GetX())
		y := float64(robot.GetY())
		angle := float64(robot.GetOrientation())
		//fmt.Println("Robot", robot.GetRobotId(), "x:", x, "y:", y, "angle:", angle)

		gi.State.SetBlueRobot(robot.GetRobotId(), x, y, angle, play_time)
	}

	for _, robot := range detect.GetRobotsYellow() {
		//fmt.Println("Robot", robot.GetRobotId(), "x:", robot.GetX(), "y:", robot.GetY(), "angle:", robot.GetOrientation())
		x := float64(robot.GetX())
		y := float64(robot.GetY())
		angle := float64(robot.GetOrientation())
		gi.State.SetYellowRobot(robot.GetRobotId(), x, y, angle, play_time)

	}

	// TOOD: Here we loop over all balls, setting the one last in the list
	// meaning error balls have 100% chance of fing it up!
	if detect.GetBalls() != nil {
		for _, ball := range detect.GetBalls() {
			//fmt.Println("Ball", ball.GetX(), ball.GetY(), ball.GetZ())
			x := float64(ball.GetX())
			y := float64(ball.GetY())
			z := float64(ball.GetZ())

			gi.State.SetBall(x, y, z, play_time)
		}
	}

	gi.State.SetValid(true)
	gi.State.Update()

	field := packet.GetGeometry().GetField()

	if gi.HasField() == false {
		gi.SetField(field)
	}
}

func (receiver *SSLVisionClient) handlePacket(packet *ssl_vision.SSL_WrapperPacket, ok bool, gi *info.GameInfo, play_time int64) {
	if !ok {
		fmt.Println("SSL Channel closed")
		return
	}

	unpack(packet, gi, play_time)
}

func (receiver *SSLVisionClient) UpdateGameInfo(gi *info.GameInfo, play_time int64) {
	packet, ok := <-receiver.ssl_channel
	receiver.handlePacket(packet, ok, gi, play_time)
}

// Start a SSL Vision receiver, returns a channel from
// which SSL wrapper packets can be obtained.
func (receiver *SSLVisionClient) Connect() {
	receiver.ssl.Connect()
	go receiver.ssl.Receive(receiver.ssl_channel)
}

func NewSSLVisionClient(sslReceiverAddress string) *SSLVisionClient {
	ssl_channel := make(chan *ssl_vision.SSL_WrapperPacket)
	receiver := &SSLVisionClient{
		ssl:         NewSSLConnection(sslReceiverAddress),
		ssl_channel: ssl_channel,
	}
	receiver.Connect()
	return receiver
}
