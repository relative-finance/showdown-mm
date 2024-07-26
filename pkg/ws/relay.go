package ws

import (
	"encoding/json"
	"log"
	"mmf/pkg/model"
	"net/http"
	"os"
)

func getDataFromRelay(steamId string) *model.EloData {
	relayAddress := os.Getenv("RELAY_ADDRESS")
	resp, err := http.Get(relayAddress + "/statistics/elo/" + steamId)
	if err != nil {
		log.Println("Error getting elo from relay")
		return &model.EloData{Elo: 1500}
	}
	if resp.StatusCode != 200 {
		log.Println("Error getting elo from relay")
		return &model.EloData{Elo: 1500}
	}

	var eloData model.EloData

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&eloData)
	if err != nil {
		log.Println("Error decoding elo data")
		return &model.EloData{Elo: 1500}
	}
	return &eloData
}
