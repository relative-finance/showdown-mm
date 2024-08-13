package model

import (
	"encoding/json"
	"log"
)

type EloData struct {
	Elo float64 `json:"elo"`
}

type MatchPlayer struct {
	SteamId string  `json:"steamId"`
	Option  int     `json:"option"`
	Team    int     `json:"team"`
	Score   float64 `json:"score"`
	TxnHash string  `json:"txnHash"`
	Payed   bool    `json:"payed"`
	ApiKey  string
}

func (mp *MatchPlayer) Marshal() []byte {
	marshalled, err := json.Marshal(mp)
	if err != nil {
		log.Println(err)
		return nil
	}

	return marshalled
}

func UnmarshalMatchPlayer(data []byte) *MatchPlayer {
	var mp MatchPlayer
	err := json.Unmarshal(data, &mp)
	if err != nil {
		log.Println(err)
		return nil
	}

	return &mp
}
