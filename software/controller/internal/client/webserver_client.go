package client

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/gorilla/websocket"
)

type WebServer struct {
	multicastConns []*net.UDPConn
	multicastAddr  *net.UDPAddr

	logPacketQueue []([]byte)
	logQueueMutex  sync.Mutex

	gameStatePacketQueue []([]byte)
	incomingActions      []action.ActionDTO
	gameStateQueueMutex  sync.Mutex
	receivedDataMutex    sync.Mutex
}

var (
	webserverInstance *WebServer
	Once              sync.Once
)

func getInstance() *WebServer {
	Once.Do(startWebServer)
	return webserverInstance
}

func startWebServer() {
	multicastIP := config.GetGameViewerAdress()
	multicastPort := config.GetGameViewerPort()

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", multicastIP, multicastPort))
	if err != nil {
		panic(err)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	var conns []*net.UDPConn
	for _, iface := range ifaces {
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
			conn, err := net.DialUDP("udp", localAddr, addr)
			if err != nil {
				continue
			}

			fmt.Printf("Multicast from %s via %s to %s:%d (GameViewer)\n",
				ipnet.IP, iface.Name, multicastIP, multicastPort)
			conns = append(conns, conn)
		}
	}

	if len(conns) == 0 {
		panic("no suitable interfaces found")
	}

	webserverInstance = &WebServer{
		multicastConns:       conns,
		multicastAddr:        addr,
		gameStatePacketQueue: make([][]byte, 0),
		logPacketQueue:       make([][]byte, 0),
	}

	go webserverInstance.sendGameState()
}

func (server *WebServer) getUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (server *WebServer) sendGameState() {
	for {
		if len(server.gameStatePacketQueue) == 0 {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		server.gameStateQueueMutex.Lock()
		gameStateJSON := server.gameStatePacketQueue[0]
		server.gameStatePacketQueue = server.gameStatePacketQueue[1:]
		server.gameStateQueueMutex.Unlock()

		for _, conn := range server.multicastConns {
			conn.Write(gameStateJSON)
		}
	}
}

type WebsiteDTO struct {
	RobotPositions [2 * info.TEAM_SIZE]info.RobotDTO
	BallPosition   info.BallDTO
	RobotActions   []action.ActionDTO
	TerminalLog    []string
}

func toJson(input WebsiteDTO) []byte {
	output, err := json.Marshal(input)
	if err != nil {
		Logger.Error("The WebsiteDTO packet could not be marshalled to JSON.")
	}
	return output
}

func actionsToJson(actions []action.Action) []byte {
	output, err := json.Marshal(actions)
	if err != nil {
		Logger.Error("The WebsiteDTO packet could not be marshalled to JSON.")
	}
	return output
}

func GetIncoming() []action.ActionDTO {
	webserver := getInstance()
	webserver.receivedDataMutex.Lock()
	defer webserver.receivedDataMutex.Unlock()
	actionsCopy := make([]action.ActionDTO, len(webserver.incomingActions))
	copy(actionsCopy, webserver.incomingActions)
	webserver.incomingActions = nil
	return actionsCopy
}

func UpdateWebLog(logs []byte) {
	Logger.Info("Updating web log")
	webserver := getInstance()
	webserver.logQueueMutex.Lock()
	webserver.logPacketQueue = append(webserver.logPacketQueue, []byte(logs))
	webserver.logQueueMutex.Unlock()
}

func BroadcastGameState(message WebsiteDTO) {
	gameStateJson := toJson(message)
	webserver := getInstance()
	webserver.gameStateQueueMutex.Lock()
	webserver.gameStatePacketQueue = append(webserver.gameStatePacketQueue, gameStateJson)
	webserver.gameStateQueueMutex.Unlock()
}

func BroadcastActions(actions []action.Action) {
	actionsJson := actionsToJson(actions)
	webserver := getInstance()
	webserver.gameStateQueueMutex.Lock()
	webserver.gameStatePacketQueue = append(webserver.gameStatePacketQueue, actionsJson)
	webserver.gameStateQueueMutex.Unlock()
}

func UpdateWebGUI(gs *info.GameState, actions []action.Action, terminal_messages []string) {
	var gamestate_DTO = gs.ToDTO()
	var actionTDO = make([]action.ActionDTO, len(actions))
	for i, obj := range actions {
		actionTDO[i] = obj.ToDTO()
	}
	var websiteMessage = WebsiteDTO{
		RobotPositions: gamestate_DTO.RobotPositions,
		BallPosition:   gamestate_DTO.BallPosition,
		RobotActions:   actionTDO,
		TerminalLog:    terminal_messages,
	}
	BroadcastGameState(websiteMessage)
}
