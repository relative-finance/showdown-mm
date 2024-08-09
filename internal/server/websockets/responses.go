package ws

import "mmf/internal/model"

type UserResponse struct {
	MatchId string `json:"matchId"`
	Option  int    `json:"option"`
}

type UserConfirmation struct {
	MatchId string `json:"matchId"`
	TxnHash string `json:"txnHash"`
}

type MatchFoundResponse struct {
	MatchId string         `json:"matchId"`
	TeamA   []model.Ticket `json:"teamA"`
	TeamB   []model.Ticket `json:"teamB"`
}

func GenerateMatchFoundResponse(tickets []model.Ticket, matchId string) MatchFoundResponse {
	mess := MatchFoundResponse{MatchId: matchId}
	mid := len(tickets) / 2
	mess.TeamA = tickets[:mid]
	mess.TeamB = tickets[mid:]
	return mess
}
