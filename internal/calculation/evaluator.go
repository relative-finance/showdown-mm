package calculation

import (
	"encoding/json"
	"fmt"
	"log"
	"mmf/config"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	"mmf/pkg/client"
	"mmf/utils"
	"strconv"
	"time"
)

func EvaluateTickets(config config.MMRConfig, queue constants.QueueType, testData *[]client.TestPairResponse) bool {
	gameBuckets := make(map[string][]model.Ticket)
	tickets := redis.RedisClient.ZRangeWithScores(constants.GetIndexNameQueue(queue), 0, -1)
	log.Print(tickets)
	if len(tickets.Val()) < config.TeamSize*2 {
		return false
	}

	for _, ticket := range tickets.Val() {
		var memberData model.MemberData
		memberRaw := ticket.Member.(string)
		if err := json.Unmarshal([]byte(memberRaw), &memberData); err != nil {
			return false
		}

		if queue == constants.LCQueue || queue == constants.LCQueueTest {
			if memberData.LichessCustomData == nil {
				continue
			}
			key := fmt.Sprintf("%d_%d_%s", memberData.LichessCustomData.Time, memberData.LichessCustomData.Increment, memberData.LichessCustomData.Collateral)
			gameBuckets[key] = append(gameBuckets[key], model.Ticket{Member: memberData, Score: ticket.Score})
		} else {
			gameBuckets["all"] = append(gameBuckets["all"], model.Ticket{Member: memberData, Score: ticket.Score})
		}
	}

	teamSize := config.TeamSize
	if queue == constants.LCQueue {
		teamSize = 1
	}

	for _, gameTickets := range gameBuckets {
		for i := 0; i < len(gameTickets); i++ {
			// If sliding window is out of bounds, break
			if i+teamSize*2 > len(gameTickets) {
				break
			}

			// If the difference between the highest and lowest in sliding window MMR is too high, skip
			if gameTickets[i+config.TeamSize*2-1].Score-gameTickets[i].Score > float64(config.Range) { //TODO: make this value dynamic based off the mmr range
				continue
			}

			matchTickets := gameTickets[i : i+config.TeamSize*2]
			tickets1, tickets2 := getTeams(matchTickets)
			matchQuality := getMatchQuality(tickets1, tickets2, config.Mode)
			if matchQuality > config.Treshold {
				if testData == nil {
					matchId := "match_" + strconv.Itoa(int(time.Now().UnixMilli()))
					utils.AddMatchToRedis(matchId, tickets1, tickets2, queue)

					go utils.WaitingForMatchThread(matchId, queue, tickets1, tickets2)
				} else {
					*testData = append(*testData, client.TestPairResponse{Team1: tickets1, Team2: tickets2})
				}

				i++
			}
		}
	}

	return true
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
