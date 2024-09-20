package ws

import (
	"encoding/json"
	"log"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	"mmf/internal/wires"
	"mmf/pkg/external"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
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

func getUserState(id string) *model.UserGlobalState {
	state := redis.RedisClient.HGet("user_state", id)
	if state.Err() == nil && state.Val() != "" {
		return model.UnmarshalUserGlobalState([]byte(state.Val()))
	}
	return &model.UserGlobalState{State: model.NoState}
}

func isUserInMM(userState *model.UserGlobalState) bool {
	return userState != nil && userState.State != model.NoState
}

func StartLichessWebSocket(game string, id string, walletAddress string, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Set pong handler to reset read deadline
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(20 * time.Second)) // Reset read deadline when pong is received

		if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
			log.Println(err)
		}
		return nil
	})

	userState := getUserState(id)
	isUserInMmVar := isUserInMM(userState)
	if isUserInMmVar {
		// TODO: Include both the teams info
		SendJSON(conn, MatchState, *userState)
	} else {
		SendJSON(conn, MatchState, nil)
	}

	// Set up ping-pong handling
	// Send pings every 10 seconds, expecting a pong response within 5 seconds
	go sendPings(conn)

	userConnectionsMutex.Lock()
	userConnections[id] = conn
	userConnectionsMutex.Unlock()

	// TODO: Remove the use of memberData as this is only set when user joins the queue
	// if user gets disconnected, memberData is getting reset
	// Using id and walletAddress of the user
	var memberData *model.MemberData
	if userState.State != model.NoState && userState.MemberData != nil {
		memberData = userState.MemberData
	}

	defer func() {
		userConnectionsMutex.Lock()
		defer userConnectionsMutex.Unlock()
		delete(userConnections, id)
		// member data is only set when user joins queue
		// If user gets disconnected, this'll clear the user details
		// TODO: handle it better
		if memberData != nil {
			memberJSON, err := json.Marshal(memberData)
			if err != nil {
				log.Println("Error serializing MemberData:", err)
				return
			}

			redis.RedisClient.ZRem(constants.GetIndexNameStr(game), memberJSON)
		}

		// if userState != nil && userState.State != model.NoState && userState.State != model.Paid {
		// 	cmd := redis.RedisClient.HSet("user_state", id, userState.Marshal())
		// 	if cmd.Err() != nil {
		// 		log.Println("Error saving user state")
		// 		log.Println(cmd.Err())
		// 	}
		// }
	}()

	var eloData *model.EloData
	apiKey, err := usernameToKey(id)
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

	waitForNewMsg := true
	for {
		_, mess, err := conn.ReadMessage()
		stringifiedMessage := string(mess)
		if err != nil {
			return
		}

		if !waitForNewMsg || stringifiedMessage == "ping" {
			continue
		}

		var userResponse UserMessage
		if err = json.Unmarshal(mess, &userResponse); err != nil {
			conn.WriteJSON(GetMessage(Error, "Error parsing message"))
			continue
		}

		switch userResponse.Type {
		case JoinQueue:
			var payload *model.LichessCustomData
			if err := mapstructure.Decode(userResponse.Payload, &payload); err != nil || payload == nil {
				conn.WriteJSON(GetMessage(Error, "Error parsing payload"))
				continue
			}
			userState := getUserState(id)
			if isUserInMM(userState) {
				SendJSON(conn, Error, "User already part of Queue")
				continue
			}

			memberData, err = wires.Instance.TicketService.SubmitTicket(c, model.SubmitTicketRequest{
				Id:                id,
				Elo:               eloData.Elo,
				WalletAddress:     walletAddress,
				LichessCustomData: payload,
			}, game)
			if err != nil {
				conn.WriteJSON(GetMessage(Error, "Error submitting ticket"))
				log.Println("Error submitting ticket")
				log.Println(err.Error())
			}

			conn.WriteJSON(GetMessage(Info, "Joined queue"))
		case LeaveQueue:
			cmd := redis.RedisClient.ZRem(constants.GetIndexNameStr(game), memberData)
			if cmd.Err() != nil {
				log.Println("Error removing ticket from queue")
				log.Println(cmd.Err())
				conn.WriteJSON(GetMessage(Error, "Error leaving queue"))
				continue
			}

			conn.WriteJSON(GetMessage(Info, "Left queue"))
		case SendPayment:
			// TODO: Add event validation
			var payload *UserPayment
			if err := mapstructure.Decode(userResponse.Payload, &payload); err != nil || payload == nil {
				conn.WriteJSON(GetMessage(Error, "Error parsing payload"))
				continue
			}

			paid := checkTransactionOnChain(payload, memberData)
			if !paid {
				conn.WriteJSON(GetMessage(Error, "Error processing payment"))
				continue
			}

			redisPlayer := redis.RedisClient.HGet(payload.MatchId, id).Val()
			matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))
			if matchPlayer == nil {
				conn.WriteJSON(GetMessage(Error, "Error getting match player"))
				continue
			}

			matchPlayer.Paid = true
			redis.RedisClient.HSet(payload.MatchId, id, matchPlayer.Marshal())
			conn.WriteJSON(GetMessage(Info, "Payment processed"))
			userState.State = model.Paid
			userState.MatchId = payload.MatchId
			userState.MemberData = memberData
			redis.RedisClient.HSet("user_state", id, userState.Marshal())
		case SendOption:
			// TODO: Add event validation
			var payload *UserResponse
			if err := mapstructure.Decode(userResponse.Payload, &payload); err != nil || payload == nil {
				conn.WriteJSON(GetMessage(Error, "Error parsing payload"))
				continue
			}

			redisPlayer := redis.RedisClient.HGet(payload.MatchId, id).Val()
			matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))
			if matchPlayer == nil {
				conn.WriteJSON(GetMessage(Error, "Error getting match player"))
				continue
			}

			matchPlayer.Option = payload.Option
			redis.RedisClient.HSet(payload.MatchId, id, matchPlayer.Marshal())
			conn.WriteJSON(GetMessage(Info, "Send option successful"))
			if payload.Option == 2 {
				userState.State = model.MatchAccepted
				userState.MatchId = payload.MatchId
				userState.MemberData = memberData
				redis.RedisClient.HSet("user_state", id, userState.Marshal())
			} else {
				redis.RedisClient.HDel(payload.MatchId, id)
			}
		default:
			conn.WriteJSON(GetMessage(Error, "Invalid message type"))
		}
	}

}

func StartWebSocket(game string, steamId string, walletAddress string, c *gin.Context) {
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
		defer userConnectionsMutex.Unlock()
		delete(userConnections, steamId)
		if memberData != nil {
			memberJSON, err := json.Marshal(memberData)
			if err != nil {
				log.Println("Error serializing MemberData:", err)
				return
			}

			redis.RedisClient.ZRem(constants.GetIndexNameStr(game), memberJSON)
		}
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

	default:
		eloData = external.GetDataFromRelay(steamId)
	}

	memberData, err = wires.Instance.TicketService.SubmitTicket(c, model.SubmitTicketRequest{
		Id:            steamId,
		Elo:           eloData.Elo,
		WalletAddress: walletAddress,
	}, game)
	if err != nil {
		conn.WriteJSON(GetMessage(Error, "Error submitting ticket"))
		log.Println("Error submitting ticket")
		log.Println(err.Error())
		return
	}

	conn.WriteJSON(GetMessage(Info, "Hello, "+steamId))
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

			var userConfirmation UserPayment

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
			matchPlayer.Paid = checkTransactionOnChain(&userConfirmation, memberData)
			if !matchPlayer.Paid {
				continue
			}
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
