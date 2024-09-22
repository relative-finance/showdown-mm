package ws

import "mmf/internal/model"

type UserResponse struct {
	MatchId string `json:"matchId"`
	Option  int    `json:"option"`
}

type MessageType string

const (
	JoinQueue   MessageType = "JOIN_QUEUE"
	LeaveQueue  MessageType = "LEAVE_QUEUE"
	SendPayment MessageType = "SEND_PAYMENT"
	SendOption  MessageType = "SEND_OPTION"
)

var MessageTypeValues = map[string]MessageType{
	"JOIN_QUEUE":   JoinQueue,
	"LEAVE_QUEUE":  LeaveQueue,
	"SEND_PAYMENT": SendPayment,
	"SEND_OPTION":  SendOption,
}

type UserMessage struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type UserJoinQueue struct {
	Type    MessageType             `json:"type"`
	Payload model.LichessCustomData `json:"payload"`
}

type UserPayment struct {
	MatchId string `json:"matchId"`
	TxnHash string `json:"txnHash"`
}

type MatchFoundResponse struct {
	MatchId    string          `json:"matchId"`
	ExpiryTime int64           `json:"expiryTime"`
	TeamA      []model.Ticket  `json:"teamA"`
	TeamB      []model.Ticket  `json:"teamB"`
	State      model.UserState `json:"state"`
}

type BackToMatchMakingResponse struct {
	Message string          `json:"message"`
	State   model.UserState `json:"state"`
}

func GenerateMatchFoundResponse(tickets []model.Ticket, matchId string, expiryTime int64) MatchFoundResponse {
	mess := MatchFoundResponse{MatchId: matchId, ExpiryTime: expiryTime, State: model.MatchFound}
	mid := len(tickets) / 2
	mess.TeamA = tickets[:mid]
	mess.TeamB = tickets[mid:]
	return mess
}

type PaymentResponse struct {
	MatchId    string          `json:"matchId"`
	ExpiryTime int64           `json:"expiryTime"`
	State      model.UserState `json:"state"`
}

type EventType string

const (
	Info       EventType = "INFO"
	Error      EventType = "ERROR"
	Success    EventType = "SUCCESS"
	Removed    EventType = "REMOVED_FROM_QUEUE"
	MatchState EventType = "MATCH_STATE"
)

type Message struct {
	EventType EventType `json:"eventType"`
	Message   string    `json:"message"`
}

func GetMessage(event EventType, message string) Message {
	return Message{EventType: event, Message: message}
}
