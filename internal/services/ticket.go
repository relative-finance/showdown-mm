package services

import (
	"encoding/json"
	"fmt"
	"log"
	"mmf/config"
	"mmf/internal/constants"
	"mmf/internal/model"

	"github.com/go-redis/redis"
)

type TicketServiceImpl struct {
	Redis     *redis.Client
	MMRConfig config.MMRConfig
}

func (s *TicketServiceImpl) SubmitTicket(submitTicketRequest model.SubmitTicketRequest, queue string) (*model.MemberData, error) {
	memberData := &model.MemberData{
		WalletAddress:     submitTicketRequest.WalletAddress,
		Id:                submitTicketRequest.Id,
		LichessCustomData: submitTicketRequest.LichessCustomData,
	}
	resp := s.Redis.ZAdd(constants.GetIndexNameStr(queue), redis.Z{Score: float64(submitTicketRequest.Elo), Member: memberData})
	if resp.Err() != nil {
		log.Println("Error adding ticket", resp.Err())
		return nil, resp.Err()
	}

	return memberData, nil
}

func (s *TicketServiceImpl) GetAllTickets(queue string) *[]model.Ticket {
	tickets, err := s.Redis.ZRangeWithScores(constants.GetIndexNameStr(queue), 0, -1).Result() // Includes second limit
	if err != nil {
		log.Println("Error fetching tickets", err)
		return nil
	}
	var gameTickets []model.Ticket
	for _, ticket := range tickets {
		ticketStr, ok := ticket.Member.(string)
		if !ok {
			log.Println("ticket is not a string")
			continue
		}

		var gameTicketMemberData model.MemberData
		err := json.Unmarshal([]byte(ticketStr), &gameTicketMemberData)
		if err != nil {
			log.Printf("Error unmarshaling member data: %s\n", err)
			continue
		}

		gameTicket := model.Ticket{Member: gameTicketMemberData, Score: ticket.Score}

		gameTickets = append(gameTickets, gameTicket)
	}
	return &gameTickets
}

func (s *TicketServiceImpl) DeleteTicket(queue string, userId string) error {
	tickets := s.GetAllTickets(queue)

	if tickets == nil {
		err := fmt.Errorf("couldn't delete ticket as tickets are empty")
		return err
	}

	var memberData model.MemberData

	for _, ticket := range *tickets {
		if ticket.Member.Id == userId {
			memberData = ticket.Member
			break
		}
	}

	marshalledMemberData, err := memberData.MarshalBinary()
	if err != nil {
		err := fmt.Errorf("couldn't delete marshal member data %s", err)
		return err
	}

	cmd := s.Redis.ZRem(constants.GetIndexNameStr(queue), marshalledMemberData)
	if cmd.Err() != nil {
		err := fmt.Errorf("error removing ticket from queue - %s", cmd.Err())
		return err
	}

	return nil
}
