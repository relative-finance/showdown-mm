package ws

import (
	"encoding/json"
	"log"
	"mmf/pkg/model"
	"mmf/wires"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Map for user connection
var userConnections = make(map[string]*websocket.Conn)
var userConnectionsMutex sync.Mutex

func StartWebSocket(game string, steamId string, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	userConnectionsMutex.Lock()
	userConnections[steamId] = conn
	userConnectionsMutex.Unlock()

	defer func() {
		userConnectionsMutex.Lock()
		delete(userConnections, steamId)
		userConnectionsMutex.Unlock()
	}()

	relayAddress := os.Getenv("RELAY_ADDRESS")
	resp, err := http.Get(relayAddress + "/statistics/elo/" + steamId)
	if err != nil {
		log.Println("Error getting elo from relay")
		return
	}
	if resp.StatusCode != 200 {
		log.Println("Error getting elo from relay")
		log.Panicln(resp.Body)
		return
	}
	var eloData struct {
		Elo float64 `json:"elo"`
	}

	err = json.NewDecoder(resp.Body).Decode(&eloData)
	if err != nil {
		log.Println("Error decoding elo data")
		return
	}

	wires.Instance.TicketService.SubmitTicket(c, model.SubmitTicketRequest{SteamID: steamId, Elo: eloData.Elo}, game)
	defer resp.Body.Close()

	for {
		conn.WriteMessage(websocket.TextMessage, []byte("Hello, "+steamId))
		time.Sleep(3 * time.Second)
	}
}

func SendMessageToUser(steamId string, message []byte) {
	userConnectionsMutex.Lock()
	defer userConnectionsMutex.Unlock()

	conn, ok := userConnections[steamId]
	if !ok {
		log.Println("User not connected")
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		log.Println(err)
	}
}

func DisconnectUser(steamId string) {
	userConnectionsMutex.Lock()
	defer userConnectionsMutex.Unlock()

	conn, ok := userConnections[steamId]
	if !ok {
		log.Println("User not connected")
		return
	}

	// Close the connection
	if err := conn.Close(); err != nil {
		log.Println("Error closing connection:", err)
	}

	// Remove the connection from the map
	delete(userConnections, steamId)
}
