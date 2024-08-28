package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mmf/config"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	"mmf/internal/wires"
	"mmf/pkg/external"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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

func StartLichessWebSocket(game string, id string, walletAddress string, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	userConnectionsMutex.Lock()
	userConnections[id] = conn
	userConnectionsMutex.Unlock()

	var memberData *model.MemberData
	defer func() {
		userConnectionsMutex.Lock()
		defer userConnectionsMutex.Unlock()
		delete(userConnections, id)
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

	conn.WriteJSON(GetMessage(Info, "Hello, "+id))
	waitForNewMsg := true
	for {
		_, mess, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if !waitForNewMsg {
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
			var payload *UserPayment
			if err := mapstructure.Decode(userResponse.Payload, &payload); err != nil || payload == nil {
				conn.WriteJSON(GetMessage(Error, "Error parsing payload"))
				continue
			}

			payed := checkTransactionOnChain(payload, memberData)
			if !payed {
				conn.WriteJSON(GetMessage(Error, "Error processing payment"))
				continue
			}

			redisPlayer := redis.RedisClient.HGet(payload.MatchId, id).Val()
			matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))
			if matchPlayer == nil {
				conn.WriteJSON(GetMessage(Error, "Error getting match player"))
				continue
			}

			matchPlayer.Payed = true
			redis.RedisClient.HSet(payload.MatchId, id, matchPlayer.Marshal())
			conn.WriteJSON(GetMessage(Info, "Payment processed"))
		case SendOption:
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
			matchPlayer.Payed = checkTransactionOnChain(&userConfirmation, memberData)
			if !matchPlayer.Payed {
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

const abiJSON = `[{
	"inputs": [
		{
			"internalType": "string",
			"name": "_matchId",
			"type": "string"
		},
		{
			"internalType": "string",
			"name": "_chessUsername",
			"type": "string"
		},
		{
			"internalType": "uint256",
			"name": "_amount",
			"type": "uint256"
		}
	],
	"name": "joinMatch",
	"outputs": [],
	"stateMutability": "nonpayable",
	"type": "function"
}]`

func checkTransactionOnChain(userConfirmation *UserPayment, ticket *model.MemberData) bool {
	txHash := common.HexToHash(userConfirmation.TxnHash)

	client, err := ethclient.Dial(config.GlobalConfig.EthRpc.URL)
	if err != nil {
		log.Println("Error connecting to eth client: ", err)
		return false
	}

	tx, _, err := client.TransactionByHash(context.Background(), txHash)
	if err != nil {
		log.Println("Error getting transaction by hash: ", err)
		return false
	}

	if tx == nil {
		return false
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		log.Println("Error parsing abi: ", err)
		return false
	}

	data := tx.Data()

	method, err := parsedABI.MethodById(data[:4])
	if err != nil {
		log.Println("Error getting method by id: ", err)
		return false
	}

	params, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		log.Println("Error unpacking data: ", err)
		return false
	}

	if len(params) != 3 {
		log.Println("Invalid number of parameters")
		return false
	}

	// if params[0].(string) != userConfirmation.MatchId {
	// 	log.Println("Invalid match id")
	// 	return false
	// }

	if params[1].(string) != ticket.Id {
		log.Println("Invalid username")
		return false
	}

	return true
}
