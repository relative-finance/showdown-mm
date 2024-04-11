package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mmf/pkg/model"
	"net/http"
	"os"
	"strconv"
	"time"
)

// MatchRequestBodyD2 represents the structure of the JSON body for the REST call
type MatchRequestBodyD2 struct {
	TeamA       []int64 `json:"teamA"`
	TeamB       []int64 `json:"teamB"`
	LobbyConfig struct {
		GameName     string `json:"gameName"`
		ServerRegion int    `json:"serverRegion"`
		PassKey      string `json:"passKey"`
		GameMode     string `json:"gameMode"`
	} `json:"lobbyConfig"`
	StartTime string `json:"startTime"`
}

type TeamInfo struct {
	Name string `json:"name"`
}

type PlayerInfo struct {
	Team          string `json:"team"`
	SteamId64     string `json:"steam_id_64"`
	WalletAddress string `json:"walletAddress"`
}

type MatchRequestBodyCS2 struct {
	Region string `json:"region"`
	// matchId int `json:"matchId"`
	// tournamentId int `json:"tournamentId"`
	Team1Id        int      `json:"team1Id"`
	Team2Id        int      `json:"team2Id"`
	Team1          TeamInfo `json:"team1"`
	Team2          TeamInfo `json:"team2"`
	Map            string   `json:"map"`
	Currency       string   `json:"currency"`
	DeadlineEpoch  int64    `json:"deadlineEpoch"`
	StartEpoch     int64    `json:"startEpoch"`
	NumberOfRounds int32    `json:"numberOfRounds"`
	RoundTime      int32    `json:"roundTime"`
	// Prize int64 `json:"prize"`
	Players []PlayerInfo `json:"players"`
}

func ScheduleMatch(url string, requestBody interface{}) error {
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
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
		LobbyConfig: struct {
			GameName     string `json:"gameName"`
			ServerRegion int    `json:"serverRegion"`
			PassKey      string `json:"passKey"`
			GameMode     string `json:"gameMode"`
		}{
			GameName:     "Relative Game Test",
			ServerRegion: 3,
			PassKey:      "test",
			GameMode:     "AP",
		},
		StartTime: "", // If sent as empty string, the match will be scheduled immediately
	}

	if err := ScheduleMatch(url, requestBody); err != nil {
		log.Println("Error making REST call:", err)
		return
	}
}

func ScheduleCS2Match(tickets1 []model.Ticket, tickets2 []model.Ticket) {
	log.Println("Scheduling CS2 match")

	url := os.Getenv("CS2API") + "/v1/match"
	requestBody := MatchRequestBodyCS2{
		Region:  "stockholm",
		Team1Id: 1,
		Team2Id: 2,
		Team1: TeamInfo{
			Name: "Team 1",
		},
		Team2: TeamInfo{
			Name: "Team 2",
		},
		Map:            "de_dust2",
		Currency:       "0x0000000000000000000000000000000000000000",
		DeadlineEpoch:  time.Now().Add(1 * time.Minute).Unix(),
		StartEpoch:     time.Now().Add(10 * time.Minute).Unix(),
		NumberOfRounds: 15,
		RoundTime:      1000,
		Players:        []PlayerInfo{},
	}

	for _, ticket := range tickets1 {
		requestBody.Players = append(requestBody.Players, PlayerInfo{
			Team:          "team1",
			SteamId64:     ticket.Member,
			WalletAddress: "0x26deC32bF104E11DB3d969671cdc3f167D68040C",
		})
	}

	for _, ticket := range tickets2 {
		requestBody.Players = append(requestBody.Players, PlayerInfo{
			Team:          "team2",
			SteamId64:     ticket.Member,
			WalletAddress: "0x1270026872A6A38b4b0868552521E56C2d14D227",
		})
	}

	if err := ScheduleMatch(url, requestBody); err != nil {
		log.Println("Error making REST call:", err)
		return
	}
}
