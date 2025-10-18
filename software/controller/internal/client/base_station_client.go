package client

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/action"
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
	threadMutex   sync.Mutex
	queue         []*robot_action.Command
	hasBeenInited bool
}

func NewBaseStationClient(address string) *BaseStationClient {
	var err error = nil
	connection, _ := net.Dial("udp", address)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
	}

	return &BaseStationClient{
		connection:    connection,
		address:       address,
		queue:         make([]*robot_action.Command, 0),
		hasBeenInited: false,
	}
}

func (b *BaseStationClient) Init() {
	go b.sendCommands()
	b.hasBeenInited = true
	b.threadMutex.Lock()
}

// Goroutine function for processing and sending commands
func (b *BaseStationClient) sendCommands() {
	for {

		if len(b.queue) == 0 {
			// Wait to be unlocked
			b.threadMutex.Lock()
			continue
		}

		// Process the first command in the queue
		b.queueMutex.Lock()
		cmd := b.queue[0]
		b.queue = b.queue[1:]
		b.queueMutex.Unlock()

		// Send the command
		serializedCmd, _ := proto.Marshal(cmd) // Add error handling
		b.sendMessage(serializedCmd)           // Add error handling
	}
}

// send list of actions to the base station
func (b *BaseStationClient) SendActions(actions []action.Action) {
	if !b.hasBeenInited {
		fmt.Println("\033[0m Base station client has not been inited\033[33m")
	}
	for _, action := range actions {
		b.queueMutex.Lock()
		b.queue = append(b.queue, action.TranslateReal())
		b.queueMutex.Unlock()
	}
	b.threadMutex.Unlock()
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
