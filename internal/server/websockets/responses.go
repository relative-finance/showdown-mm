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
	MatchId      string         `json:"matchId"`
	TimeToAccept string         `json:"timeToAccept"`
	TeamA        []model.Ticket `json:"teamA"`
	TeamB        []model.Ticket `json:"teamB"`
}

func GenerateMatchFoundResponse(tickets []model.Ticket, matchId string, timeToAccept string) MatchFoundResponse {
	mess := MatchFoundResponse{MatchId: matchId, TimeToAccept: timeToAccept}
	mid := len(tickets) / 2
	mess.TeamA = tickets[:mid]
	mess.TeamB = tickets[mid:]
	return mess
}

type PaymentResponse struct {
	MatchId   string `json:"matchId"`
	TimeToPay string `json:"timeToPay"`
}

type EventType string

const (
	Info    EventType = "INFO"
	Error   EventType = "ERROR"
	Success EventType = "SUCCESS"
)

type Message struct {
	EventType EventType `json:"eventType"`
	Message   string    `json:"message"`
}

func GetMessage(event EventType, message string) Message {
	return Message{EventType: event, Message: message}
}
