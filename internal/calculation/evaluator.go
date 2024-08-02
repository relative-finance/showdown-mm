package calculation

import (
	"mmf/config"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	ws "mmf/internal/server/websockets"
	"mmf/utils"
	"strconv"
	"time"
)

func EvaluateTickets(config config.MMRConfig, queue constants.QueueType) bool {
	var gameTickets []model.Ticket
	tickets := redis.RedisClient.ZRangeWithScores(constants.GetIndexNameQueue(queue), 0, -1)

	if len(tickets.Val()) < config.TeamSize*2 {
		return false
	}

	for _, ticket := range tickets.Val() {
		gameTickets = append(gameTickets, model.Ticket{Member: ticket.Member.(string), Score: ticket.Score})
	}

	for i := 0; i < len(gameTickets); i++ {
		// If sliding window is out of bounds, break
		if i+config.TeamSize*2 > len(gameTickets) {
			break
		}

		// If the difference between the highest and lowest in sliding window MMR is too high, skip
		if gameTickets[i+config.TeamSize*2-1].Score-gameTickets[i].Score > 100 { //TODO: make this value dynamic based off the mmr range
			continue
		}

		matchTickets := gameTickets[i : i+config.TeamSize*2]
		tickets1, tickets2 := getTeams(matchTickets)
		matchQuality := getMatchQuality(tickets1, tickets2, config.Mode)
		if matchQuality > config.Treshold {
			matchId := "match_" + strconv.Itoa(int(time.Now().UnixMilli()))
			utils.AddMatchToRedis(matchId, tickets1, tickets2, queue)
			sent := ws.SendMatchFoundToPlayers(matchId, matchTickets)
			if !sent {
				return false
			}
			go utils.WaitingForMatchThread(matchId, queue, tickets1, tickets2, config.TimeToCancelMatch)
			return true
		}
	}

	return false
}

func getTeams(tickets []model.Ticket) ([]model.Ticket, []model.Ticket) {
	var tickets1, tickets2 []model.Ticket
	mid := len(tickets) / 2
	length := len(tickets)
	for i := 0; i < mid; i++ {
		if i%2 == 0 {
			tickets1 = append(tickets1, tickets[i], tickets[length-i-1])
		} else {
			tickets2 = append(tickets2, tickets[i], tickets[length-i-1])
		}
	}

	// When there are 8 tickets, the mid is 4
	// First team will have tickets 0, 9, 2, 7, 4
	// Second team will have tickets 1, 8, 3, 6, 5
	if length%4 != 0 {
		tickets1 = tickets1[:len(tickets1)-1]
		tickets2 = append(tickets2, tickets[mid])
	}

	return tickets1, tickets2
}
