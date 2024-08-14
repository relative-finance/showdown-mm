package model

import "encoding/json"

type SubmitTicketRequest struct {
	SteamID           string             `json:"steamId"`
	Elo               float64            `json:"elo"`
	WalletAddress     string             `json:"walletAddress"`
	LichessCustomData *LichessCustomData `json:"lichessCustomData"`
}

type Ticket struct {
	Member MemberData `json:"member"`
	Score  float64    `json:"elo"`
}

type MemberData struct {
	SteamID           string             `json:"steamId"`
	WalletAddress     string             `json:"walletAddress"`
	LichessCustomData *LichessCustomData `json:"lichessCustomData"`
}

type Collateral string

const (
	Showdown Collateral = "showdown"
	Practice Collateral = "practice"
)

type LichessCustomData struct {
	ApiKey     string     `json:"apiKey"`
	Time       int        `json:"interval"`
	Increment  int        `json:"increment"`
	Collateral Collateral `json:"collateral"`
}

func (md *MemberData) MarshalBinary() ([]byte, error) {
	return json.Marshal(md)
}

func (md *MemberData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, md)
}
