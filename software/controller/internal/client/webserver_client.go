package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/config"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/gorilla/websocket"
)

//----------------------------------------------------------------------------------------------
// Start of WebServer class
//----------------------------------------------------------------------------------------------

// Define the WebServer class
type WebServer struct {
	websocketConnections      []*websocket.Conn
	websocketConnectionsMutex sync.Mutex

	websocketupgrader *websocket.Upgrader

	gameStatePacketQueue []([]byte)
	incomingData         []GameViewerCommand
	dataLookup           map[string]*GameViewerCommand
	gameStateQueueMutex  sync.Mutex
	// broadcastThreadMutex sync.Mutex
	receivedDataMutex sync.Mutex
}

var (
	webserverInstance *WebServer
	Once              sync.Once
)

// Method to get the singleton instance of the WebServer class
func getInstance() *WebServer {
	Once.Do(startWebServer)
	return webserverInstance
}

// Constructor for the WebServer class
func startWebServer() {
	webserverInstance = &WebServer{
		gameStatePacketQueue: make([]([]byte), 0),
		dataLookup:           make(map[string]*GameViewerCommand),
	}

	webserverInstance.websocketupgrader = webserverInstance.getUpgrader()

	http.HandleFunc("/ws", webserverInstance.handleGameStateRequest)
	go func() {
		if err := http.ListenAndServe(config.GetGameViewerPort(), nil); err != nil {
			Logger.Errorw("Webserver stopped", "error", err)
		}
	}()
	go webserverInstance.sendGameState()
	fmt.Println("Gameviewer socket online at", config.GetGameViewerPort())
	Logger.Info("Webserver online at", config.GetGameViewerPort())
}

func (server *WebServer) getUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (server *WebServer) handleGameStateRequest(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a WebSocket
	ws, err := server.websocketupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.websocketConnectionsMutex.Lock()
	defer server.websocketConnectionsMutex.Unlock() // unlock after function returns
	server.websocketConnections = append(server.websocketConnections, ws)
	fmt.Println("Gameviewer client connected")

	go server.receiveData(ws)
}

func (server *WebServer) sendGameState() {
	var gameStateJSON []byte
	for {
		server.gameStateQueueMutex.Lock()
		if len(server.gameStatePacketQueue) == 0 {
			server.gameStateQueueMutex.Unlock()
			time.Sleep(time.Millisecond * 10)
			continue
		}

		gameStateJSON = server.gameStatePacketQueue[0]
		server.gameStatePacketQueue = server.gameStatePacketQueue[1:]
		server.gameStateQueueMutex.Unlock()

		// Creating a copy of the connections. This prevents locking other threads if the connection takes too long
		server.websocketConnectionsMutex.Lock()
		connectionsCopy := make([]*websocket.Conn, len(server.websocketConnections))
		copy(connectionsCopy, server.websocketConnections)
		server.websocketConnectionsMutex.Unlock()

		for _, ws := range connectionsCopy {
			if err := ws.WriteMessage(websocket.TextMessage, gameStateJSON); err != nil {
				fmt.Println(err)
				server.removeConnection(ws)
			}
		}
	}
}

// Method to receive data from all connected clients
func (server *WebServer) receiveData(ws *websocket.Conn) {
	defer func() {
		server.removeConnection(ws)
		ws.Close()
	}()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		var receivedData GameViewerCommand
		err_unmarshal := json.Unmarshal(message, &receivedData)
		if err_unmarshal != nil {
			log.Println("Error unmarshalling message:", err_unmarshal)
			continue
		}

		server.receivedDataMutex.Lock()
		log.Println("Received data:", receivedData)
		server.incomingData = append(server.incomingData, receivedData)
		server.dataLookup[receivedData.CommandType] = &receivedData
		server.receivedDataMutex.Unlock()
	}
}

func (server *WebServer) removeConnection(ws *websocket.Conn) {
	server.websocketConnectionsMutex.Lock()
	defer server.websocketConnectionsMutex.Unlock()

	for i, connection := range server.websocketConnections {
		if connection == ws {
			server.websocketConnections = append(server.websocketConnections[:i], server.websocketConnections[i+1:]...)
			return
		}
	}
}

//----------------------------------------------------------------------------------------------
// End of WebServer class
//----------------------------------------------------------------------------------------------

// How to use the WebServer class:
// Only use the functions under this comment to interact with the WebServer class
// The WebServer class is a singleton class, so you can only have one instance of it,
// and the functions under handles all of it so multiple instances are not created

func actionsToJson(actions []action.Action) []byte {
	output, err := json.Marshal(actions)
	if err != nil {
		Logger.Error("The action packet could not be marshalled to JSON.")
	}
	return output
}

func StartGameViewerServer() {
	getInstance()
}

// Returns a list of all new incoming actions
func GetAllIncoming() []GameViewerCommand {
	webserver := getInstance()
	webserver.receivedDataMutex.Lock()
	defer webserver.receivedDataMutex.Unlock()
	// Return a copy of the incomingActions slice
	actionsCopy := make([]GameViewerCommand, len(webserver.incomingData))
	copy(actionsCopy, webserver.incomingData)
	webserver.incomingData = nil // Empty the incomingActions slice
	return actionsCopy
}

// Returns a list of all new incoming actions
func GetCommand(commandName string) *GameViewerCommand {
	webserver := getInstance()
	webserver.receivedDataMutex.Lock()
	defer webserver.receivedDataMutex.Unlock()
	command, ok := webserver.dataLookup[commandName]
	if !ok {
		return nil
	}
	delete(webserver.dataLookup, commandName)
	return command
}

func BroadcastActions(actions []action.Action) {
	actionsJson := actionsToJson(actions)
	webserver := getInstance()
	webserver.gameStateQueueMutex.Lock()
	webserver.gameStatePacketQueue = append(webserver.gameStatePacketQueue, actionsJson)
	webserver.gameStateQueueMutex.Unlock()
}

// Command types that are recieved from gameviewer

const (
	CHANGE_SCENARIO = "CHANGE_SCENARIO"
	MOVE_ROBOT      = "MOVE_ROBOT"
)

type GameViewerCommand struct {
	CommandType string `json:"Command"`
	X           int32  `json:"x"`
	Y           int32  `json:"y"`
	Id          int    `json:"Id"`
	Type        string `json:"Type"`
}
