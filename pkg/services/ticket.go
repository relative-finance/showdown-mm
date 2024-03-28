package services

import (
	"mmf/config"
	"mmf/pkg/model"
	"strconv"

	"github.com/fasmat/trueskill"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

type TicketServiceImpl struct {
	Redis     *redis.Client
	MMRConfig config.MMRConfig
}

func (s *TicketServiceImpl) SubmitTicket(g *gin.Context, submitTicketRequest model.SubmitTicketRequest) error {
	s.Redis.ZAdd("player_elo", redis.Z{Score: float64(submitTicketRequest.Elo), Member: submitTicketRequest.SteamID})
	return nil
}

func (s *TicketServiceImpl) GetAllTickets(g *gin.Context) []model.Ticket {
	tickets := s.Redis.ZRangeWithScores("player_elo", 0, -1) // Includes second limit
	var gameTickets []model.Ticket
	for _, ticket := range tickets.Val() {
		gameTickets = append(gameTickets, model.Ticket{Member: ticket.Member.(string), Score: ticket.Score})
	}
	return gameTickets
}

func formatPrintTeam(t trueskill.Team) string {
	team := ""
	for i := 0; i < len(t.GetPlayers()); i++ {
		team += strconv.Itoa(t.GetPlayers()[i].GetID()) + " "
	}
	return team
}
