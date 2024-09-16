package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mmf/internal/model"
	ws "mmf/internal/server/websockets"

	"net/http"
	"os"
	"strconv"
	"time"
)

func ScheduleMatch(url string, requestBody interface{}) (*io.ReadCloser, error) {
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, with error: %s", resp.StatusCode, string(err))
	}

	return &resp.Body, nil
}

func ScheduleDota2Match(tickets1 []model.Ticket, tickets2 []model.Ticket) {
	log.Println("Scheduling Dota 2 match")

	// map tickets1 to TeamA
	teamA := []int64{}
	for _, ticket := range tickets1 {
		player, err := strconv.ParseInt(ticket.Member.Id, 10, 64)
		if err != nil {
			log.Println("Error parsing ticket member:", err)
			return
		}
		teamA = append(teamA, player)
	}
	// map tickets2 to TeamB

	teamB := []int64{}
	for _, ticket := range tickets2 {
		player, err := strconv.ParseInt(ticket.Member.Id, 10, 64)
		if err != nil {
			log.Println("Error parsing ticket member:", err)
			return
		}
		teamB = append(teamB, player)
	}

	url := os.Getenv("D2API") + "/v1/match"
	requestBody := MatchRequestBodyD2{
		TeamA: teamA,
		TeamB: teamB,
		LobbyConfig: LobbyConfig{
			GameName:     "Relative Game Test",
			ServerRegion: 3,
			PassKey:      "test",
			GameMode:     "AP",
		},
		StartTime: "", // If sent as empty string, the match will be scheduled immediately
	}

	if _, err := ScheduleMatch(url, requestBody); err != nil {
		log.Println("Error making REST call:", err)
		return
	}
}

func ScheduleCS2Match(tickets1 []model.Ticket, tickets2 []model.Ticket) {
	log.Println("Scheduling CS2 match")

	url := os.Getenv("CS2API") + "/v1/start-match"
	requestBody := MatchRequestBodyCS2{
		Team1: Team{
			Name: "team1",
		},
		Team2: Team{
			Name: "team2",
		},
		Players: []PlayerDatHost{},
		Settings: GameSettings{
			Map:                 "de_dust2",
			ConnectTime:         120,
			MatchBeginCountdown: 10,
			EnableTechPause:     false,
		},
		Webhooks: Webhooks{
			MatchEndURL: "",
			RoundEndURL: "",
		},
	}

	for _, ticket := range tickets1 {
		requestBody.Players = append(requestBody.Players, PlayerDatHost{
			Team:      "team1",
			SteamId64: ticket.Member.Id,
		})
	}

	for _, ticket := range tickets2 {
		requestBody.Players = append(requestBody.Players, PlayerDatHost{
			Team:      "team2",
			SteamId64: ticket.Member.Id,
		})
	}

	resp, err := ScheduleMatch(url, requestBody)
	if err != nil {
		log.Println("Error making REST call:", err)
		return
	}
	defer (*resp).Close()

	var matchResponse MatchResponseBodyCS2
	if err := json.NewDecoder(*resp).Decode(&matchResponse); err != nil {
		log.Println("Error decoding response:", err)
		return
	}

	for _, ticket := range append(tickets1, tickets2...) {
		ws.SendJSONToUser(ticket.Member.Id, ws.Info, matchResponse)
	}
}

func ScheduleLichessMatch(tickets1 []model.Ticket, tickets2 []model.Ticket, matchId string) (*CreateLichessMatchRequest, error) {
	log.Println("Scheduling Lichess match")

	if len(tickets1) == 0 || len(tickets2) == 0 {
		log.Println("Insufficient players to schedule a match")
		return nil, errors.New("insufficient players to schedule a match")
	}

	// Sending team data to players - needs pulling username
	type Teams struct {
		YourTeam []string `json:"your_team"`
		Opponent []string `json:"opponent_team"`
	}

	var tickets1team, tickets2team Teams
	for _, ticket := range tickets1 {

		tickets1team.YourTeam = append(tickets1team.YourTeam, ticket.Member.Id)
		tickets2team.Opponent = append(tickets2team.Opponent, ticket.Member.Id)
	}

	for _, ticket := range tickets2 {

		tickets1team.Opponent = append(tickets1team.Opponent, ticket.Member.Id)
		tickets2team.YourTeam = append(tickets2team.YourTeam, ticket.Member.Id)
	}

	for _, ticket := range tickets1 {
		ws.SendJSONToUser(ticket.Member.Id, ws.Info, tickets1team)
	}

	for _, ticket := range tickets2 {
		ws.SendJSONToUser(ticket.Member.Id, ws.Info, tickets2team)
	}

	// Assuming the first ticket in each list represents the player for the match
	player1 := tickets1[0].Member.Id // steamId for player1
	player2 := tickets2[0].Member.Id // steamId for player2

	url := os.Getenv("LICHESSAPI") + "/v1/match"

	requestBody := CreateLichessMatchRequest{
		Player1: player1,
		Player2: player2,
		Variant: Standard,
		Clock: Clock{
			Increment: tickets1[0].Member.LichessCustomData.Increment,
			Limit:     tickets1[0].Member.LichessCustomData.Time * 60,
		},
		Rated:         false,
		Rules:         []Rules{},
		PairAt:        int(time.Now().Add(1 * time.Minute).UnixMilli()),
		StartClocksAt: int(time.Now().Add(6 * time.Minute).UnixMilli()),
		Webhook:       fmt.Sprint(os.Getenv("WEBHOOK_ENDPOINT"), "/", matchId),
		Instant:       true,
	}

	type LichessId struct {
		Id string `json:"lichessId"`
	}

	body, err := ScheduleMatch(url, requestBody)
	if err != nil {
		ws.SendMessageToUser(tickets1[0].Member.Id, ws.Error, "Error scheduling match")
		ws.SendMessageToUser(tickets2[0].Member.Id, ws.Error, "Error scheduling match")

		log.Println("Error making REST call:", err)
		return nil, err
	}

	var lichessId LichessId
	if err := json.NewDecoder(*body).Decode(&lichessId); err != nil {
		log.Println("Error decoding response:", err)
		return nil, err
	}

	// Send lichess id to players
	for _, ticket := range append(tickets1, tickets2...) {
		ws.SendJSONToUser(ticket.Member.Id, ws.Info, lichessId)
	}

	log.Println("Lichess match scheduled successfully")

	// Notify showdown-api of match status
	go notifyShowdownAPI(matchId, lichessId.Id)

	return &requestBody, nil
}

func notifyShowdownAPI(matchId, lichessId string) {
	showdownReq := &StartLichessShowdownMatchRequest{
		MatchID:   matchId,
		LichessID: lichessId,
	}

	showdownApi := os.Getenv("SHOWDOWN_RELAY")

	url := fmt.Sprintf("%s/chess/start_chess_match", showdownApi)
	log.Println(url)
	client := &http.Client{}

	jsonData, err := json.Marshal(showdownReq)
	if err != nil {
		log.Println("Error marshalling showdown request:", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating showdown api request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error making showdown REST call:", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		err, _ := io.ReadAll(resp.Body)
		log.Println("Failed to start match on Showdown Api - Unexpected status code:", resp.StatusCode, "with error:", string(err))
		return
	}

	log.Println("Showdown API notified")
}
