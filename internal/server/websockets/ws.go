package ws

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mmf/internal/model"
	"mmf/internal/redis"
	"mmf/internal/wires"
	client2 "mmf/pkg/client"
	"mmf/pkg/external"
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

func usernameToKey(username string) (*string, error) {
	showdownApi := os.Getenv("SHOWDOWN_API")
	showdownKey := os.Getenv("SHOWDOWN_API_KEY")

	url := fmt.Sprintf("%s/get_liches_token?userID=%s", showdownApi, username)
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", showdownKey)

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var apiResponse *client2.ShowdownApiResponse
	err = json.Unmarshal(body, apiResponse)

	if err != nil {
		return nil, err
	}

	return &apiResponse.Key, nil
}

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
		// TODO: Write a func that converts steamId (which is lichess username) to API_KEY
		// by calling showdown-api example: "http://65.1.107.225:81/get_lichess_token?userID=tkumar994"

		apiKey, err := usernameToKey(steamId)

		if err != nil {
			log.Println("Error getting token from showdown api")
			return
		}

		rating, err := external.GetGlicko(*apiKey, "blitz") // TODO: Make it so that elo is fetched for correct game mode
		if err != nil {
			log.Println("Error getting elo from lichess, using default elo 1500")
			eloData = &model.EloData{Elo: 1500}
		} else {
			eloData = &model.EloData{Elo: float64(rating)}
		}
	default:
		eloData = external.GetDataFromRelay(steamId)
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
