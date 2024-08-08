package model

import "encoding/json"

type SubmitTicketRequest struct {
	SteamID       string  `json:"steamId"`
	Elo           float64 `json:"elo"`
	WalletAddress string  `json:"walletAddress"`
}

type Ticket struct {
	Member MemberData `json:"member"`
	Score  float64    `json:"elo"`
}

type MemberData struct {
	SteamID       string `json:"steamId"`
	WalletAddress string `json:"walletAddress"`
}

func (md *MemberData) MarshalBinary() ([]byte, error) {
	return json.Marshal(md)
}

func (md *MemberData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, md)
}
