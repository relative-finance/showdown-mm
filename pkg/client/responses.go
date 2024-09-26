package client

import "mmf/internal/model"

type MatchResponseBodyCS2 struct {
	ConnectionString string  `json:"connection_string"`
	MatchId          MatchId `json:"match_id"`
}

type MatchId struct {
	Id           string `json:"id"`
	GameServerId string `json:"game_server_id"`
}

type TestPairResponse struct {
	Team1 []model.Ticket `json:"team1"`
	Team2 []model.Ticket `json:"team2"`
}
