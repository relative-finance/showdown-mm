package ws

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	"mmf/internal/wires"
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

type ShowdownApiResponse struct {
	Token string `json:"lichessToken"`
}

// Map for user connection
var userConnections = make(map[string]*websocket.Conn)
var userConnectionsMutex sync.Mutex

func usernameToKey(username string) (*string, error) {
	showdownApi := os.Getenv("SHOWDOWN_API")
	showdownKey := os.Getenv("SHOWDOWN_API_KEY")

	url := fmt.Sprintf("%s/get_lichess_token?userID=%s", showdownApi, username)
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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var apiResponse ShowdownApiResponse
	err = json.Unmarshal(body, &apiResponse)

	if err != nil {
		return nil, err
	}

	return &apiResponse.Token, nil
}

func StartWebSocket(game string, steamId string, walletAddress string, lichessData *model.LichessCustomData, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	userConnectionsMutex.Lock()
	userConnections[steamId] = conn
	userConnectionsMutex.Unlock()

	var memberData *model.MemberData
	defer func() {
		userConnectionsMutex.Lock()
		delete(userConnections, steamId)
		if memberData != nil {
			memberJSON, err := json.Marshal(memberData)
			if err != nil {
				log.Println("Error serializing MemberData:", err)
				return
			}

			redis.RedisClient.ZRem(constants.GetIndexNameStr(game), memberJSON)
		}
		userConnectionsMutex.Unlock()
	}()

	var eloData *model.EloData
	switch game {
	case "lcqueue":
		apiKey, err := usernameToKey(steamId)

		if err != nil {
			log.Println("Error getting token from showdown api ")
			log.Println(err.Error())
			return
		}

		rating, err := external.GetGlicko(*apiKey, "blitz") // TODO: Make it so that elo is fetched for correct game mode
		if err != nil {
			log.Println("Error getting elo from lichess, using default elo 1500")
			eloData = &model.EloData{Elo: 1500}
		} else {
			eloData = &model.EloData{Elo: float64(rating)}
		}

		lichessData.ApiKey = *apiKey
	default:
		eloData = external.GetDataFromRelay(steamId)
	}

	memberData, err = wires.Instance.TicketService.SubmitTicket(c, model.SubmitTicketRequest{
		SteamID:           steamId,
		Elo:               eloData.Elo,
		WalletAddress:     walletAddress,
		LichessCustomData: lichessData,
	}, game)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error submitting ticket"))
		log.Println("Error submitting ticket")
		log.Println(err.Error())
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte("Hello, "+steamId))
	waitForNewMsg := true
	for {
		_, mess, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if !waitForNewMsg {
			continue
		}

		var userResponse UserResponse
		if err = json.Unmarshal(mess, &userResponse); err != nil || userResponse.Option == 0 {
			log.Println(err)

			var userConfirmation UserConfirmation

			if err = json.Unmarshal(mess, &userConfirmation); err != nil {
				log.Println(err)
				return
			}

			redisPlayer := redis.RedisClient.HGet(userConfirmation.MatchId, steamId).Val()
			matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

			if matchPlayer.Option != 2 {
				//match declined
				return
			}

			matchPlayer.TxnHash = userConfirmation.TxnHash
			matchPlayer.Payed = true
			redis.RedisClient.HSet(userResponse.MatchId, steamId, matchPlayer.Marshal())

			waitForNewMsg = false
			continue
		}

		redisPlayer := redis.RedisClient.HGet(userResponse.MatchId, steamId).Val()
		matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

		matchPlayer.Option = userResponse.Option
		redis.RedisClient.HSet(userResponse.MatchId, steamId, matchPlayer.Marshal())
	}

}
