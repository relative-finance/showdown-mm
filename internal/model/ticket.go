package model

import (
	"encoding/json"
)

type SubmitTicketRequest struct {
	Id                string             `json:"steamId"`
	Elo               float64            `json:"elo"`
	WalletAddress     string             `json:"walletAddress"`
	LichessCustomData *LichessCustomData `json:"lichessCustomData"`
}

type Ticket struct {
	Member MemberData `json:"member"`
	Score  float64    `json:"score"`
}

type MemberData struct {
	Id                string             `json:"id"`
	WalletAddress     string             `json:"walletAddress"`
	LichessCustomData *LichessCustomData `json:"lichessCustomData"`
}

type Collateral string

const (
	SP   Collateral = "SP"
	SUSD Collateral = "SUSD"
)

type LichessCustomData struct {
	Time       int        `json:"time"`
	Increment  int        `json:"increment"`
	Collateral Collateral `json:"collateral"`
}

func (md *MemberData) MarshalBinary() ([]byte, error) {
	return json.Marshal(md)
}

func (md *MemberData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, md)
}
