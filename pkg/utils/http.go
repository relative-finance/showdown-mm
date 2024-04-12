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

// CS2
type Team struct {
	Name string `json:"name"`
}

type PlayerDatHost struct {
	Team      string `json:"team"`
	SteamId64 string `json:"steam_id_64"`
}
type GameSettings struct {
	Map                 string `json:"map"`
	Password            string `json:"password"`
	ConnectTime         int    `json:"connect_time"`
	MatchBeginCountdown int    `json:"match_begin_countdown"`
	EnableTechPause     bool   `json:"enable_tech_pause"`
}
type Webhooks struct {
	MatchEndURL string `json:"match_end_url"`
	RoundEndURL string `json:"round_end_url"`
}
type MatchRequestBodyCS2 struct {
	Team1    Team            `json:"team1"`
	Team2    Team            `json:"team2"`
	Webhooks Webhooks        `json:"webhooks"`
	Settings GameSettings    `json:"settings"`
	Players  []PlayerDatHost `json:"players"`
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
			Password:            "test",
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

	if err := ScheduleMatch(url, requestBody); err != nil {
		log.Println("Error making REST call:", err)
		return
	}
}
