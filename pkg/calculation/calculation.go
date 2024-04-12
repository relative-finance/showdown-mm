package calculation

import (
	"mmf/config"
	"mmf/pkg/constants"
	"mmf/pkg/model"
	"mmf/pkg/redis"
	"mmf/pkg/ws"
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
			addMatchToRedis(matchId, tickets1, tickets2, queue)
			sent := ws.SendMatchFoundToPlayers(matchId, matchTickets)
			if !sent {
				return false
			}
			go waitingForMatchThread(matchId, queue, tickets1, tickets2, config.TimeToCancelMatch)
			return true
		}
	}

	return false
}
