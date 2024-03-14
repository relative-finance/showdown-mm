package services

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

type TicketServiceImpl struct {
	Redis *redis.Client
}

type SubmitTicketRequest struct {
	SteamID string `json:"steamId"`
	Elo     int    `json:"elo"`
}

func (s *TicketServiceImpl) SubmitTicket(g *gin.Context, submitTicketRequest SubmitTicketRequest) error {
	s.Redis.ZAdd("player_elo", redis.Z{Score: float64(submitTicketRequest.Elo), Member: submitTicketRequest.SteamID})
	log.Println("Ticket submitted")

	return nil
}
