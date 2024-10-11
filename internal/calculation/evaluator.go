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
	var tickets []model.Ticket
	queueZSet := redis.RedisClient.ZRangeWithScores(constants.GetIndexNameQueue(queue), 0, -1)
	log.Print(queueZSet)
	if len(queueZSet.Val()) < config.TeamSize*2 {
		return false
	}

	for _, ticket := range queueZSet.Val() {
		var memberData model.MemberData
		memberRaw := ticket.Member.(string)
		if err := json.Unmarshal([]byte(memberRaw), &memberData); err != nil {
			return false
		}

		// key := fmt.Sprintf("%d_%d_%s", memberData.LichessCustomData.Time, memberData.LichessCustomData.Increment, memberData.LichessCustomData.Collateral)
		tickets = append(tickets, model.Ticket{Member: memberData, Score: ticket.Score})
	}

	if queue == constants.LCQueue || queue == constants.LCQueueTest {
		return lichessEvaluate(tickets, testData)
	}

	teamSize := config.TeamSize

	for i := 0; i < len(tickets); i++ {
		// If sliding window is out of bounds, break
		if i+teamSize*2 > len(tickets) {
			break
		}

		// If the difference between the highest and lowest in sliding window MMR is too high, skip
		if tickets[i+config.TeamSize*2-1].Score-tickets[i].Score > float64(config.Range) { //TODO: make this value dynamic based off the mmr range
			continue
		}

		matchTickets := tickets[i : i+config.TeamSize*2]
		tickets1, tickets2 := getTeams(matchTickets)
		matchQuality := getMatchQuality(tickets1, tickets2, config.Mode)
		if matchQuality > config.Treshold {
			matchId := "match_" + strconv.Itoa(int(time.Now().UnixMilli()))
			utils.AddMatchToRedis(matchId, tickets1, tickets2, queue)

			go utils.WaitingForMatchThread(matchId, queue, tickets1, tickets2)

			i++
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

func lichessEvaluate(tickets []model.Ticket, testData *[]client.TestPairResponse) bool {
	if len(tickets) < 2 {
		return true
	}

	ticketsMap := make(map[string][]model.Ticket)

	for i := 0; i < len(tickets); i++ {
		player := tickets[i]
		if player.Member.LichessCustomData == nil || len(player.Member.LichessCustomData) == 0 {
			continue
		}

		elapsedTime := time.Now().Unix() - player.Member.LichessCustomData[0].Timestamp
		checkingValues := 1
		if elapsedTime > 60 {
			checkingValues = len(player.Member.LichessCustomData)
		}

		for j := 0; j < checkingValues; j++ {
			key := fmt.Sprintf("%d_%d_%s", player.Member.LichessCustomData[j].Time, player.Member.LichessCustomData[j].Increment, player.Member.LichessCustomData[j].Collateral)
			ticketsMap[key] = append(ticketsMap[key], player)
		}
	}

	for _, ticks := range ticketsMap {
		for i := 0; i < len(ticks); i++ {
			player := ticks[i]
			difference := getDifference(player.Member.LichessCustomData[0].Timestamp)

			for j := i + 1; j < len(ticks); j++ {
				otherPlayer := ticks[j]
				otherDifference := getDifference(otherPlayer.Member.LichessCustomData[0].Timestamp)
				if player.Member.Id == otherPlayer.Member.Id {
					continue
				}

				diff := player.Score - otherPlayer.Score
				if diff > float64(min(difference, otherDifference)) {
					if diff > float64(difference) {
						break
					}
					continue
				}

				if testData == nil {
					matchId := "match_" + strconv.Itoa(int(time.Now().UnixMilli()))
					utils.AddMatchToRedis(matchId, []model.Ticket{player}, []model.Ticket{otherPlayer}, constants.LCQueue)

					go utils.WaitingForMatchThread(matchId, constants.LCQueue, []model.Ticket{player}, []model.Ticket{otherPlayer})
					return true
				} else {
					*testData = append(*testData, client.TestPairResponse{Team1: []model.Ticket{player}, Team2: []model.Ticket{otherPlayer}})
					i++
				}

			}
		}
	}

	return true
}

func getDifference(timestamp int64) int {
	elapsedTime := time.Now().Unix() - timestamp
	difference := 50
	if elapsedTime > int64(difference) {
		difference = min(250, int(elapsedTime))
	}
	return difference
}
