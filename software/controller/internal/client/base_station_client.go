package client

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
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
	wake          chan struct{}
	hasBeenInited bool
}

func NewBaseStationClient(address string) *BaseStationClient {

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

	foundInterface := false

	var connections []Connection
	for _, iface := range ifaces {
		if config.GetAIMulticastInterface() != "" && config.GetAIMulticastInterface() != iface.Name {
			continue
		}

		foundInterface = true

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
		wake:          make(chan struct{}, 1),
		hasBeenInited: false,
	}
}

func (b *BaseStationClient) Init() {
	b.queueMutex.Lock()
	if b.wake == nil {
		b.wake = make(chan struct{}, 1)
	}
	b.hasBeenInited = true
	b.queueMutex.Unlock()
	go b.sendCommands()
}

func (b *BaseStationClient) sendCommands() {
	for {
		b.queueMutex.Lock()
		if len(b.queue) == 0 {
			b.queueMutex.Unlock()
			<-b.wake
			continue
		}
		cmd := b.queue[0]
		b.queue = b.queue[1:]
		b.queueMutex.Unlock()
		// Use custom binary encoder instead of protobuf
		encoded, err := action.EncodeCommand(cmd)
		if err != nil {
			fmt.Printf("Failed to encode command for robot %d: %v\n", cmd.RobotId, err)
			continue
		}
		if err := b.sendMessage(encoded); err != nil {
			fmt.Printf("Unable to send robot command: %v\n", err)
		}
	}
}

func (b *BaseStationClient) SendActions(actions []action.Action) {
	b.queueMutex.Lock()
	defer b.queueMutex.Unlock()
	if !b.hasBeenInited {
		fmt.Println("\033[0m Base station client has not been inited\033[33m")
		return
	}

	for _, robotAction := range actions {
		command := robotAction.TranslateReal()
		if command.GetCommandId() == robot_action.ActionType_STOP_ACTION {
			// A stop supersedes every command that has not yet been sent for this
			// robot. In particular, do not leave a pre-HALT kick ahead of it.
			queue := b.queue[:0]
			for _, queuedCommand := range b.queue {
				if queuedCommand.GetRobotId() != command.GetRobotId() {
					queue = append(queue, queuedCommand)
				}
			}
			b.queue = queue
		} else if isMotionCommand(command) {
			// A later position or velocity update supersedes unsent motion for
			// the same robot. Keep kicks, init, and stops in their original order.
			queue := b.queue[:0]
			for _, queuedCommand := range b.queue {
				if queuedCommand.GetRobotId() != command.GetRobotId() || !isMotionCommand(queuedCommand) {
					queue = append(queue, queuedCommand)
				}
			}
			b.queue = queue
		}
		b.queue = append(b.queue, command)
	}
	if len(b.queue) > 0 && b.wake != nil {
		select {
		case b.wake <- struct{}{}:
		default:
		}
	}
}

func isMotionCommand(command *robot_action.Command) bool {
	switch command.GetCommandId() {
	case robot_action.ActionType_MOVE_TO_ACTION,
		robot_action.ActionType_MOVE_ACTION,
		robot_action.ActionType_ROTATE_ACTION:
		return true
	default:
		return false
	}
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
