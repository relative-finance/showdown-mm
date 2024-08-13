package services

import (
	"log"
	"mmf/config"
	"mmf/internal/constants"
	"mmf/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

type TicketServiceImpl struct {
	Redis     *redis.Client
	MMRConfig config.MMRConfig
}

func (s *TicketServiceImpl) SubmitTicket(g *gin.Context, submitTicketRequest model.SubmitTicketRequest, queue string) error {
	if submitTicketRequest.ApiKey != "" {
		s.Redis.ZAdd(constants.GetIndexNameStr(queue), redis.Z{Score: float64(submitTicketRequest.Elo), Member: &model.MemberData{
			WalletAddress: submitTicketRequest.WalletAddress,
			SteamID:       submitTicketRequest.SteamID,
			ApiKey:        submitTicketRequest.ApiKey,
		}})

		return nil
	}
	resp := s.Redis.ZAdd(constants.GetIndexNameStr(queue), redis.Z{Score: float64(submitTicketRequest.Elo), Member: &model.MemberData{
		WalletAddress: submitTicketRequest.WalletAddress,
		SteamID:       submitTicketRequest.SteamID,
	}})
	log.Print(resp)
	return nil
}

func (s *TicketServiceImpl) GetAllTickets(g *gin.Context, queue string) []model.Ticket {
	tickets := s.Redis.ZRangeWithScores(constants.GetIndexNameStr(queue), 0, -1) // Includes second limit
	if tickets.Err() != nil {
		log.Println("Error fetching tickets", tickets.Err())
		return nil
	}
	var gameTickets []model.Ticket
	for _, ticket := range tickets.Val() {
		gameTickets = append(gameTickets, model.Ticket{Member: ticket.Member.(model.MemberData), Score: ticket.Score})
	}
	return gameTickets
}
