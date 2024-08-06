package client

type MatchResponseBodyCS2 struct {
	ConnectionString string  `json:"connection_string"`
	MatchId          MatchId `json:"match_id"`
}

type MatchId struct {
	Id           string `json:"id"`
	GameServerId string `json:"game_server_id"`
}

type ShowdownApiResponse struct {
	Key string `json:"key"`
}
