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
	"mmf/internal/wires"
	"mmf/pkg/client"
	"mmf/pkg/external"
	"net/http"
	"strings"
	"time"
)

func WaitingForMatchThread(matchId string, queue constants.QueueType, tickets1 []model.Ticket, tickets2 []model.Ticket) {
	mmCfg := config.GlobalConfig.MMRConfig
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	matchInfoLog := fmt.Sprintf("Queue: %s, MatchId: %s, Player1: %s, Player2: %s", queue, matchId, tickets1[0].Member.Id, tickets2[0].Member.Id)

	allTickets := append(tickets1, tickets2...)
	timeToAccept := time.Now().Add(time.Duration(mmCfg.TimeToAccept) * time.Second)

	userState := model.UserGlobalState{State: model.MatchFound, MatchId: matchId, ExpiryTime: timeToAccept.Unix()}
	for _, ticket := range allTickets {
		if err := SetUserStateInRedis(ticket.Member.Id, &userState); err != nil {
			log.Println("Error setting user state to match found: ", err)
		}
	}

	ws.SendMatchFoundToPlayers(matchId, allTickets, timeToAccept.Unix())

	for range ticker.C {
		if time.Now().After(timeToAccept) {
			log.Println("Players failed to accept in time ", matchInfoLog)
			MatchFailedReturnPlayersToMM(queue, matchId, false, false)
			return
		}

		allAccepted := true
		for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
			matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

			if matchPlayer.Option == 0 {
				ticker.Stop()
				log.Printf("Player %s did not accept - %s \n", matchPlayer.Id, matchInfoLog)
				MatchFailedReturnPlayersToMM(queue, matchId, false, false)
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

	log.Println("Creating match on chain ", matchInfoLog)
	_, err := createLichessMatchShowdown(tickets1, tickets2, matchId)
	if err != nil {
		// logic to return players to matchmaking
		log.Println("Error while creating match on showdown ", err.Error())
		return
	}
	end := time.Now().Add(time.Duration(mmCfg.TimeToCancelMatch) * time.Second)

	userState = model.UserGlobalState{State: model.PaymentPending, MatchId: matchId, ExpiryTime: end.Unix()}
	for _, ticket := range allTickets {
		if err := SetUserStateInRedis(ticket.Member.Id, &userState); err != nil {
			log.Println("Error setting user state to payment pending: ", err)
		}
	}

	paymentResponse := ws.PaymentResponse{MatchId: matchId, ExpiryTime: end.Unix(), State: model.PaymentPending}
	for _, ticket := range allTickets {
		ws.SendJSONToUser(ticket.Member.Id, ws.Info, paymentResponse)
	}

	noOfChecks := 1

	for range ticker.C {
		if time.Now().After(end) {
			ticker.Stop()
			log.Println("Players failed to pay in time ", matchInfoLog)
			MatchFailedReturnPlayersToMM(queue, matchId, true, false)
			return
		}

		unPaidPlayersList := []*model.MatchPlayer{}
		if queue == constants.LCQueue {
			for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
				matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

				if !matchPlayer.Paid {
					unPaidPlayersList = append(unPaidPlayersList, matchPlayer)
				}
			}

			// make subgraph calls once every 6 seconds
			if len(unPaidPlayersList) != 0 && noOfChecks%3 == 0 {
				// check user's payment status from subgraph as well
				playersPaymentStatus := client.GetQPUsersPaymentStatusFromSubgraph(matchId)
				for _, playerInfo := range unPaidPlayersList {
					playerWalletAddress := strings.ToLower(playerInfo.WalletAddress)
					if playersPaymentStatus[playerWalletAddress] {
						playerInfo.Paid = true
						redis.RedisClient.HSet(matchId, playerInfo.Id, playerInfo.Marshal())

						userPlayerInfo := ws.GetUserState(playerInfo.Id)
						userPlayerInfo.State = model.Paid
						ws.UpdateUserState(playerInfo.Id, userPlayerInfo)
					}
				}

			}
		}

		if len(unPaidPlayersList) == 0 {
			break
		} else {
			noOfChecks += 1
		}
	}

	ticker.Stop()
	switch queue {
	case constants.D2Queue:
		client.ScheduleDota2Match(tickets1, tickets2)
	case constants.CS2Queue:
		client.ScheduleCS2Match(tickets1, tickets2)
	case constants.LCQueue:
		log.Println("Players paid, scheduling lichess match ", matchInfoLog)
		_, err := client.ScheduleLichessMatch(tickets1, tickets2, matchId)
		// TODO: Cancel the match øn the contract
		if err != nil {
			log.Println("Error scheduling lichess match: ", err)
			MatchFailedReturnPlayersToMM(queue, matchId, false, true)
			return
		}
	}
	log.Println("Match scheduled successfully - disconnecting users", matchInfoLog)

	DisconnectAllUsers(matchId)
	ret := redis.RedisClient.Del(matchId)
	if ret.Err() != nil {
		log.Println("Error deleting match from redis: ", ret.Err())
	}

	cmd := redis.RedisClient.HDel("user_state", tickets1[0].Member.Id, tickets2[0].Member.Id)
	if cmd.Err() != nil {
		log.Println("Error deleting user state from redis: ", cmd.Err())
	}

	if queue == constants.LCQueue {
		go func() {
			type Metadata struct {
				Opponent string `json:"opponent"`
				Amount   string `json:"amount"`
				Currency string `json:"currency"`
			}

			md := Metadata{
				Opponent: tickets2[0].Member.Id,
				Amount:   "0.1",
				Currency: string(tickets2[0].Member.LichessCustomData[0].Collateral),
			}

			notification := external.Notification{
				Content:  "Quickplay match created with <opponent> for <amount> <currency>\n Best of luck!",
				Metadata: md,
				UserIds:  []string{tickets1[0].Member.Id},
				Type:     "chess_quickplay",
				Subtype:  "MATCH_CREATED",
				RefId:    matchId,
			}

			external.SendNotification(notification)

			md.Opponent = tickets1[0].Member.Id
			notification.UserIds = []string{tickets2[0].Member.Id}
			notification.Metadata = md

			external.SendNotification(notification)
		}()
	}
}

func DisconnectAllUsers(matchId string) {
	for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
		matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))
		log.Println("Disconnecting user: ", matchPlayer.Id)
		ws.DisconnectUser(matchPlayer.Id)
	}
}

func MatchFailedReturnPlayersToMM(queue constants.QueueType, matchId string, isPaymentFlow bool, isPostPayment bool) {
	var playerIdsToClear []string
	var matchPlayersToAddToQueue []model.MatchPlayer
	// TODO: store this info in DB to track user's reluctance to pay/accept matches
	for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
		var matchPlayer model.MatchPlayer
		if err := json.Unmarshal([]byte(redisPlayer), &matchPlayer); err != nil {
			log.Println(err)
			return
		}

		if isPaymentFlow {
			// flow after user accepts the match and payment is in progress
			if !matchPlayer.Paid {
				ws.SendMessageToUser(matchPlayer.Id, ws.Removed, "Time for payment expired")
			} else {
				// Users that paid will be added to the queue
				// TODO: Notify showdown api to cancel the match
				matchPlayersToAddToQueue = append(matchPlayersToAddToQueue, matchPlayer)
			}
		} else {
			switch matchPlayer.Option {
			case 0:
				ws.SendMessageToUser(matchPlayer.Id, ws.Removed, "You've declined the match")
			case 1:
				ws.SendMessageToUser(matchPlayer.Id, ws.Removed, "Time for accepting the match expired")
			case 2:
				matchPlayersToAddToQueue = append(matchPlayersToAddToQueue, matchPlayer)
			}
		}
		playerIdsToClear = append(playerIdsToClear, matchPlayer.Id)
	}

	log.Println("Clearing Match ID:", matchId, " Queue: ", queue)
	ClearMatchData(matchId, &playerIdsToClear)

	// Add them back to queue after clearing match data
	for _, matchPlayer := range matchPlayersToAddToQueue {
		_, err := wires.Instance.TicketService.SubmitTicket(model.SubmitTicketRequest{
			Id:                matchPlayer.Id,
			Elo:               matchPlayer.Score,
			WalletAddress:     matchPlayer.WalletAddress,
			LichessCustomData: matchPlayer.LichessCustomData,
		}, queue.String())
		log.Println("Added player:", matchPlayer.Id, "back to", queue)

		if err != nil {
			log.Println("Error adding player to queue: ", err)
			continue
		}
		message := ws.BackToMatchMakingResponse{
			Message: "",
			State:   model.RejoinQueue,
		}
		if isPostPayment {
			// happens when schedule lichess match fails
			message.Message = "Couldn't create match, match is cancelled - back to matchmaking"
		} else {
			message.Message = "Opponent didn't accept the match, back to matchmaking"
		}
		ws.SendJSONToUser(matchPlayer.Id, ws.Info, message)

	}

}

type CreateLichessMatchShowdownRequest struct {
	MatchID       string           `json:"match_id"`
	Player1ID     string           `json:"player1_lichess_id"`
	Player2ID     string           `json:"player2_lichess_id"`
	Player1Wallet string           `json:"player1_wallet_address"`
	Player2Wallet string           `json:"player2_wallet_address"`
	Collateral    model.Collateral `json:"collateral_token"`
	Increment     int              `json:"increment"`
	Time          int              `json:"limit"`
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

	time, increment, collateral := client.FindTimeIncrAndColl(tickets1[0], tickets2[0])
	showdownReq := &CreateLichessMatchShowdownRequest{
		MatchID:       matchId,
		Player1ID:     player1,
		Player2ID:     player2,
		Player1Wallet: player1Wallet,
		Player2Wallet: player2Wallet,
		Collateral:    collateral,
		Increment:     increment,
		Time:          time,
		Variant:       "blitz",
		Rated:         false,
	}

	url := fmt.Sprintf("%s/chess/create_quickplay_match", config.GlobalConfig.ShowdownApi.URL)
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
