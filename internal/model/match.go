package model

import (
	"encoding/json"
	"log"
)

type EloData struct {
	Elo float64 `json:"elo"`
}

type MatchPlayer struct {
	Id                string  `json:"id"`
	Option            int     `json:"option"`
	Team              int     `json:"team"`
	Score             float64 `json:"score"`
	TxnHash           string  `json:"txnHash"`
	Paid              bool    `json:"paid"`
	ApiKey            string
	WalletAddress     string              `json:"walletAddress"`
	LichessCustomData []LichessCustomData `json:"lichessCustomData"`
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

type UserState string

const (
	NoState        UserState = "noState"
	MatchFound     UserState = "matchFound"
	MatchAccepted  UserState = "matchAccepted"
	PaymentPending UserState = "paymentPending"
	Paid           UserState = "paid"
	RejoinQueue    UserState = "rejoinQueue"
)

var UserStateValue = map[string]UserState{
	"matchFound":     MatchFound,
	"matchAccepted":  MatchAccepted,
	"paymentPending": PaymentPending,
	"paid":           Paid,
	"noState":        NoState,
}

type UserGlobalState struct {
	State      UserState   `json:"state"`
	MatchId    string      `json:"matchId,omitempty"`
	MemberData *MemberData `json:"memberData,omitempty"`
	ExpiryTime int64       `json:"expiryTime,omitempty"`
}

func (ugs *UserGlobalState) Marshal() []byte {
	marshalled, err := json.Marshal(ugs)
	if err != nil {
		log.Println(err)
		return nil
	}

	return marshalled
}

func UnmarshalUserGlobalState(data []byte) *UserGlobalState {
	var ugs UserGlobalState
	err := json.Unmarshal(data, &ugs)
	if err != nil {
		log.Println(err)
		return nil
	}

	return &ugs
}
