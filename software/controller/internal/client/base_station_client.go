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

const MAX_SEND_SIZE = 32

type Connection interface {
	Write(b []byte) (n int, err error)
	Close() error
}

type BaseStationClient struct {
	connections   []Connection
	address       string
	queueMutex    sync.Mutex
	queue         []*robot_action.Command
	hasBeenInited bool
}

func NewBaseStationClient(address string) *BaseStationClient {
	// If a specific address is provided, prefer unicast UDP to that destination.
	// This is useful for tests and local setups that don't use multicast/config.
	if address != "" {
		remoteAddr, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			panic(err)
		}

		conn, err := net.DialUDP("udp", nil, remoteAddr)
		if err != nil {
			panic(err)
		}

		return &BaseStationClient{
			connections:   []Connection{conn},
			address:       address,
			queue:         make([]*robot_action.Command, 0),
			hasBeenInited: false,
		}
	}

	multicastIP := config.GetBasestationAdress()
	multicastPort := config.GetBasestationPort()
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", multicastIP, multicastPort))
	if err != nil {
		panic(err)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	foundInterface := false;

	var connections []Connection
	for _, iface := range ifaces {
		if config.GetAIMulticastInterface() != "" && config.GetAIMulticastInterface() != iface.Name {
			continue
		}

		foundInterface = true;

		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, ifaceAddr := range addrs {
			ipnet, ok := ifaceAddr.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}

			localAddr := &net.UDPAddr{IP: ipnet.IP, Port: 0}
			connection, err := net.DialUDP("udp", localAddr, remoteAddr)
			if err != nil {
				continue
			}

			fmt.Printf("Multicast from %s via %s to %s:%s (basedstation)\n",
				ipnet.IP, iface.Name, multicastIP, multicastPort)
			connections = append(connections, connection)
		}
	}

	if len(connections) == 0 || foundInterface == false {
		panic("no suitable interfaces found")
	}

	return &BaseStationClient{
		connections:   connections,
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

	for _, conn := range b.connections {
		_, err := conn.Write(input)
		if err != nil {
			fmt.Printf("Some error %v\n", err)
		}
	}

	return nil
}

func (b *BaseStationClient) CloseConnection() {
	for _, conn := range b.connections {
		conn.Close()
	}
}
