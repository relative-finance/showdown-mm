package model

type SubmitTicketRequest struct {
	SteamID string  `json:"steamId"`
	Elo     float64 `json:"elo"`
}

type Ticket struct {
	Member string  `json:"steamId"`
	Score  float64 `json:"elo"`
}
