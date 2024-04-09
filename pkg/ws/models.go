package ws

type UserResponse struct {
	MatchId string `json:"matchId"`
	Option  int    `json:"option"`
}

type MatchFoundResponse struct {
	MatchId string `json:"matchId"`
}
