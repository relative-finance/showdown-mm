package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mmf/config"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	ws "mmf/internal/server/websockets"
	"mmf/pkg/client"
	"net/http"
	"os"
	"time"

	r "github.com/go-redis/redis"
)

func WaitingForMatchThread(matchId string, queue constants.QueueType, tickets1 []model.Ticket, tickets2 []model.Ticket) {
	mmCfg := config.GlobalConfig.MMRConfig
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	allTickets := append(tickets1, tickets2...)
	timeToAccept := time.Now().Add(time.Duration(mmCfg.TimeToAccept) * time.Second)
	ws.SendMatchFoundToPlayers(matchId, allTickets, timeToAccept.Format(time.RFC3339))

	for range ticker.C {
		if time.Now().After(timeToAccept) {
			MatchFailedReturnPlayersToMM(queue, matchId, false)
			return
		}

		allAccepted := true
		for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
			matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

			if matchPlayer.Option == 0 {
				ticker.Stop()
				MatchFailedReturnPlayersToMM(queue, matchId, true)
				return
			}

			if matchPlayer.Option == 1 {
				allAccepted = false
			}
		}

		if allAccepted {
			break
		}
	}

	log.Println("Creating match on chain")
	_, err := createLichessMatchShowdown(tickets1, tickets2, matchId)
	if err != nil {
		// logic to return players to matchmaking
		log.Println("Error while creating match on showdown ", err.Error())
		return
	}

	userState := model.UserGlobalState{State: model.PaymentPending, MatchId: matchId}
	for _, ticket := range allTickets {
		cmd := redis.RedisClient.HSet("user_state", ticket.Member.Id, userState.Marshal())
		if cmd.Err() != nil {
			log.Println("Error setting user state to payment pending: ", cmd.Err())
		}
	}

	end := time.Now().Add(time.Duration(mmCfg.TimeToCancelMatch) * time.Second)
	paymentResponse := ws.PaymentResponse{MatchId: matchId, TimeToPay: end.Format(time.RFC3339)}
	for _, ticket := range allTickets {
		ws.SendJSONToUser(ticket.Member.Id, ws.Info, paymentResponse)
	}

	for range ticker.C {
		if time.Now().After(end) {
			ticker.Stop()
			MatchFailedReturnPlayersToMM(queue, matchId, false, true)
			return
		}

		allPaid := true
		if queue == constants.LCQueue {
			for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
				matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

				if !matchPlayer.Paid {
					allPaid = false
					break
				}
			}
		}

		if allPaid {
			break
		}
	}

	ticker.Stop()
	switch queue {
	case constants.D2Queue:
		client.ScheduleDota2Match(tickets1, tickets2)
	case constants.CS2Queue:
		client.ScheduleCS2Match(tickets1, tickets2)
	case constants.LCQueue:
		_, err := client.ScheduleLichessMatch(tickets1, tickets2, matchId)
		if err != nil {
			log.Println("Error scheduling lichess match: ", err)
			// MatchFailedReturnPlayersToMM(queue, matchId, false)
			// return
		}
	}
	log.Println("Match scheduled")

	DisconnectAllUsers(matchId)
	ret := redis.RedisClient.Del(matchId)
	if ret.Err() != nil {
		log.Println("Error deleting match from redis: ", ret.Err())
	}

	cmd := redis.RedisClient.HDel("user_state", tickets1[0].Member.Id, tickets2[0].Member.Id)
	if cmd.Err() != nil {
		log.Println("Error deleting user state from redis: ", cmd.Err())
	}
}

func DisconnectAllUsers(matchId string) {
	for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
		matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))
		log.Println("Disconnecting user: ", matchPlayer.Id)
		ws.DisconnectUser(matchPlayer.Id)
	}
}

func MatchFailedReturnPlayersToMM(queue constants.QueueType, matchId string, denied bool, payment ...bool) {
	statusMarker := 1
	if denied {
		statusMarker = 0
	}

	for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
		var matchPlayer model.MatchPlayer
		if err := json.Unmarshal([]byte(redisPlayer), &matchPlayer); err != nil {
			log.Println(err)
			return
		}

		if matchPlayer.Option > statusMarker {
			if len(payment) > 0 && payment[0] && !matchPlayer.Paid {
				ws.SendMessageToUser(matchPlayer.Id, ws.Info, "Time for payment expired")
				continue
			}

			matchPlayer.Option = 1
			cmd := redis.RedisClient.ZAdd(constants.GetIndexNameQueue(queue), r.Z{Score: matchPlayer.Score, Member: matchPlayer.Marshal()})
			if cmd.Err() != nil {
				log.Println("Error adding player to queue: ", cmd.Err())
				continue
			}

			ws.SendMessageToUser(matchPlayer.Id, ws.Info, "Opponent didn't accept the match, back to matchmaking")
			continue
		}

		if matchPlayer.Option == 0 {
			ws.SendMessageToUser(matchPlayer.Id, ws.Info, "Match was canceled")
			continue
		}
		ws.SendMessageToUser(matchPlayer.Id, ws.Info, "Time for accepting the match expired")
	}

	redis.RedisClient.Del(matchId)
}

type CreateLichessMatchShowdownRequest struct {
	MatchID       string           `json:"match_id"`
	Player1ID     string           `json:"player1_lichess_id"`
	Player2ID     string           `json:"player2_lichess_id"`
	Player1Wallet string           `json:"player1_wallet_address"`
	Player2Wallet string           `json:"player2_wallet_address"`
	Collateral    model.Collateral `json:"collateral"`
	Increment     int              `json:"increment"`
	Time          int              `json:"time"`
	Variant       string           `json:"variant"`
	Rated         bool             `json:"rated"`
}

type QuickPlayResponse struct {
	Hash string `json:"txHash"`
}

func createLichessMatchShowdown(tickets1 []model.Ticket, tickets2 []model.Ticket, matchId string) (*string, error) {
	if len(tickets1) == 0 || len(tickets2) == 0 {
		log.Println("Insufficient players to schedule a match")
		return nil, errors.New("insufficient players to schedule a match")
	}

	// Sending team data to players - needs pulling username
	type Teams struct {
		YourTeam []string `json:"your_team"`
		Opponent []string `json:"opponent_team"`
	}

	var ticket1team, tickets2team Teams
	for _, ticket := range tickets1 {

		ticket1team.YourTeam = append(ticket1team.YourTeam, ticket.Member.Id)
		tickets2team.Opponent = append(tickets2team.Opponent, ticket.Member.Id)
	}

	for _, ticket := range tickets2 {
		ticket1team.Opponent = append(ticket1team.Opponent, ticket.Member.Id)
		tickets2team.YourTeam = append(tickets2team.YourTeam, ticket.Member.Id)
	}

	player1 := tickets1[0].Member.Id // steamId for player1
	player2 := tickets2[0].Member.Id // steamId for player2

	player1Wallet := tickets1[0].Member.WalletAddress
	player2Wallet := tickets2[0].Member.WalletAddress

	showdownReq := &CreateLichessMatchShowdownRequest{
		MatchID:       matchId,
		Player1ID:     player1,
		Player2ID:     player2,
		Player1Wallet: player1Wallet,
		Player2Wallet: player2Wallet,
		Collateral:    tickets1[0].Member.LichessCustomData.Collateral,
		Increment:     tickets1[0].Member.LichessCustomData.Increment,
		Time:          tickets1[0].Member.LichessCustomData.Time,
		Variant:       "blitz",
		Rated:         false,
	}

	showdownApi := os.Getenv("SHOWDOWN_RELAY")

	url := fmt.Sprintf("%s/chess/create_quickplay_match", showdownApi)
	log.Println(url)
	client := &http.Client{}

	jsonData, err := json.Marshal(showdownReq)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))

	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var quickPlayResponse QuickPlayResponse

	if err = json.Unmarshal(body, &quickPlayResponse); err != nil {
		return nil, err
	}

	log.Println("CREATED MATCH ON SHOWDOWN API")
	log.Println(quickPlayResponse.Hash)
	return &quickPlayResponse.Hash, nil
}
