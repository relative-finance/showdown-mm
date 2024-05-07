package ws

import (
	"encoding/json"
	"log"
	"mmf/pkg/model"
	"mmf/pkg/redis"
	"mmf/wires"
	"net/http"
	"os"
	"sync"

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

	eloData := getDataFromRelay(steamId)
	wires.Instance.TicketService.SubmitTicket(c, model.SubmitTicketRequest{SteamID: steamId, Elo: eloData.Elo}, game)

	conn.WriteMessage(websocket.TextMessage, []byte("Hello, "+steamId))
	for {
		_, mess, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
		}

		var userResponse UserResponse
		if err = json.Unmarshal(mess, &userResponse); err != nil {
			log.Println(err)
			return
		}

		redisPlayer := redis.RedisClient.HGet(userResponse.MatchId, steamId).Val()
		matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

		matchPlayer.Option = userResponse.Option
		redis.RedisClient.HSet(userResponse.MatchId, steamId, matchPlayer.Marshal())
	}
}

func getDataFromRelay(steamId string) *model.EloData {
	relayAddress := os.Getenv("RELAY_ADDRESS")
	resp, err := http.Get(relayAddress + "/statistics/elo/" + steamId)
	if err != nil {
		log.Println("Error getting elo from relay")
		return &model.EloData{Elo: 1500}
	}
	if resp.StatusCode != 200 {
		log.Println("Error getting elo from relay")
		return &model.EloData{Elo: 1500}
	}

	var eloData model.EloData

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&eloData)
	if err != nil {
		log.Println("Error decoding elo data")
		return &model.EloData{Elo: 1500}
	}
	return &eloData
}
