package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mmf/pkg/model"
	"net/http"
	"strconv"
)

// MatchRequestBody represents the structure of the JSON body for the REST call
type MatchRequestBody struct {
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

func ScheduleMatch(url string, requestBody MatchRequestBody) error {
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

	url := "http://d2-api:8080/v1/match" // TODO: Move endpoints and urls to .env/config
	requestBody := MatchRequestBody{
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
