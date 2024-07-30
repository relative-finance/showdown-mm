package ws

import (
	"encoding/json"
	"log"
	"mmf/internal/model"
	"mmf/internal/redis"
	"mmf/internal/wires"
	"net/http"
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
	var eloData *model.EloData

	switch game {
	case "lcqueue":
		rating, err := getGlicko(steamId, "blitz") // TODO: Make it so that elo is fetched for correct game mode
		if err != nil {
			log.Println("Error getting elo from lichess, using default elo 1500")
			eloData = &model.EloData{Elo: 1500}
		} else {
			eloData = &model.EloData{Elo: float64(rating)}
		}
	default:
		eloData = getDataFromRelay(steamId)
	}

	wires.Instance.TicketService.SubmitTicket(c, model.SubmitTicketRequest{SteamID: steamId, Elo: eloData.Elo}, game)

	conn.WriteMessage(websocket.TextMessage, []byte("Hello, "+steamId))
	for {
		_, mess, err := conn.ReadMessage()
		if err != nil {
			return
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
