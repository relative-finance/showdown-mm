package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mmf/internal/model"
	ws "mmf/internal/websocket"

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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return &resp.Body, nil
}

func ScheduleDota2Match(tickets1 []model.Ticket, tickets2 []model.Ticket) {
	log.Println("Scheduling Dota 2 match")

	// map tickets1 to TeamA
	teamA := []int64{}
	for _, ticket := range tickets1 {
		player, err := strconv.ParseInt(ticket.Member, 10, 64)
		if err != nil {
			log.Println("Error parsing ticket member:", err)
			return
		}
		teamA = append(teamA, player)
	}
	// map tickets2 to TeamB

	teamB := []int64{}
	for _, ticket := range tickets2 {
		player, err := strconv.ParseInt(ticket.Member, 10, 64)
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
			SteamId64: ticket.Member,
		})
	}

	for _, ticket := range tickets2 {
		requestBody.Players = append(requestBody.Players, PlayerDatHost{
			Team:      "team2",
			SteamId64: ticket.Member,
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

	var message []byte
	if message, err = json.Marshal(matchResponse); err != nil {
		log.Println("Error marshalling message:", err)
		return
	}

	for _, ticket := range append(tickets1, tickets2...) {
		ws.SendMessageToUser(ticket.Member, message)
	}
}

func ScheduleLichessMatch(tickets1 []model.Ticket, tickets2 []model.Ticket) {
	log.Println("Scheduling Lichess match")

	if len(tickets1) == 0 || len(tickets2) == 0 {
		log.Println("Insufficient players to schedule a match")
		return
	}

	// Assuming the first ticket in each list represents the player for the match
	player1 := tickets1[0].Member // steamId for player1
	player2 := tickets2[0].Member // steamId for player2

	url := os.Getenv("LICHESSAPI") + "/v1/match"

	requestBody := CreateLichessMatchRequest{
		Player1: player1,
		Player2: player2,
		Variant: Standard,
		Clock: Clock{
			Increment: 0,
			Limit:     300,
		},
		Rated:         false,
		Rules:         []Rules{},
		PairAt:        int(time.Now().Add(1 * time.Minute).UnixMilli()),
		StartClocksAt: int(time.Now().Add(6 * time.Minute).UnixMilli()),
		Webhook:       "",
	}

	if _, err := ScheduleMatch(url, requestBody); err != nil {
		log.Println("Error making REST call:", err)
		return
	}
	log.Println("Lichess match scheduled successfully")
}
