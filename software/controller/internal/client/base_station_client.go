package client

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"google.golang.org/protobuf/proto"
)

const MAX_SEND_SIZE = 2048

type Connection interface {
	Write(b []byte) (n int, err error)
	Close() error
}

type BaseStationClient struct {
	connection    Connection
	address       string
	queueMutex    sync.Mutex
	queue         []*robot_action.Command
	hasBeenInited bool
}

func NewBaseStationClient(address string) *BaseStationClient {
	multicastIP := config.GetBasestationAdress()
	multicastPort := config.GetBasestationPort()

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", multicastIP, multicastPort))
	if err != nil {
		panic(err)
	}

	iface, err := net.InterfaceByName(config.GetFetdatornInterface())
	if err != nil {
		panic(err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		panic(err)
	}

	var localIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			localIP = ipnet.IP
			break
		}
	}

	if localIP == nil {
		panic("no IPv4 address found on interface")
	}

	localAddr := &net.UDPAddr{IP: localIP, Port: 0}

	connection, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Multicast from %s via %s to %s:%s (basedstation)\n", localIP, iface.Name, multicastIP, multicastPort)

	return &BaseStationClient{
		connection:    connection,
		address:       "",
		queue:         make([]*robot_action.Command, 0),
		hasBeenInited: false,
	}
}

func (b *BaseStationClient) Init() {
	go b.sendCommands()
	b.hasBeenInited = true
}

func (b *BaseStationClient) sendCommands() {
	for {
		b.queueMutex.Lock()
		if len(b.queue) == 0 {
			b.queueMutex.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}

		cmd := b.queue[0]
		b.queue = b.queue[1:]
		b.queueMutex.Unlock()

		serializedCmd, _ := proto.Marshal(cmd)
		b.sendMessage(serializedCmd)
	}
}

func (b *BaseStationClient) SendActions(actions []action.Action) {
	if !b.hasBeenInited {
		fmt.Println("\033[0m Base station client has not been inited\033[33m")
		return
	}

	b.queueMutex.Lock()
	for _, action := range actions {
		b.queue = append(b.queue, action.TranslateReal())
	}
	b.queueMutex.Unlock()
}

func (b *BaseStationClient) sendMessage(input []byte) error {
	if len(input) > MAX_SEND_SIZE {
		fmt.Print("to big to send (if sent = Rasmus mad 😡)")
		return errors.New("too long message")
	}
	_, err := b.connection.Write(input)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return err
	}
	return nil
}

func (b *BaseStationClient) CloseConnection() {
	b.connection.Close()
}
