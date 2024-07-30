package calculation

import (
	"encoding/json"
	"log"
	"math"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	ws "mmf/internal/websocket"
	"mmf/pkg/client"
	"strconv"
	"time"

	r "github.com/go-redis/redis"

	"github.com/fasmat/trueskill"
)

func waitingForMatchThread(matchId string, queue constants.QueueType, tickets1 []model.Ticket, tickets2 []model.Ticket, timeToCancelMatch int) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	end := time.Now().Add(time.Duration(timeToCancelMatch) * time.Second)
	for range ticker.C {
		if time.Now().After(end) {
			break
		}

		allAccepted := true
		for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
			matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))

			if matchPlayer.Option == 0 {
				ticker.Stop()
				matchFailedReturnPlayersToMM(queue, matchId, true)
				return
			}

			if matchPlayer.Option == 1 {
				allAccepted = false
			}
		}

		log.Println("All players accepted: ", allAccepted)

		if allAccepted {
			ticker.Stop()
			switch queue {
			case constants.D2Queue:
				client.ScheduleDota2Match(tickets1, tickets2)
			case constants.CS2Queue:
				client.ScheduleCS2Match(tickets1, tickets2)
			case constants.LCQueue:
				client.ScheduleLichessMatch(tickets1, tickets2)
			}
			log.Println("Match scheduled")
			disconnectAllUsers(matchId)
			ret := redis.RedisClient.Del(matchId)
			if ret.Err() != nil {
				log.Println("Error deleting match from redis: ", ret.Err())
			}
			break
		}
	}

	matchFailedReturnPlayersToMM(queue, matchId, false)
}

func disconnectAllUsers(matchId string) {
	for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
		matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))
		log.Println("Disconnecting user: ", matchPlayer.SteamId)
		ws.DisconnectUser(matchPlayer.SteamId)
	}
}

func matchFailedReturnPlayersToMM(queue constants.QueueType, matchId string, denied bool) {
	statusMarker := 1
	if denied {
		statusMarker = 0
	}

	for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
		var matchPlayer model.MatchPlayer

		if err := json.Unmarshal([]byte(redisPlayer), &matchPlayer); err != nil {
			log.Println(err)
			return
		}

		if matchPlayer.Option > statusMarker {
			redis.RedisClient.ZAdd(constants.GetIndexNameQueue(queue), r.Z{Score: matchPlayer.Score, Member: matchPlayer.SteamId})
			continue
		}

		ws.DisconnectUser(matchPlayer.SteamId)
	}

	redis.RedisClient.Del(matchId)
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

func getMatchQuality(tickets1 []model.Ticket, tickets2 []model.Ticket, mode string) float64 {
	switch mode {
	case "trueskill":
		return calculateMatchQualityTrueSkill(tickets1, tickets2)
	case "glicko":
		return calculateMatchQualityGlicko(tickets1, tickets2)
	default:
		return calculateMatchQualityTrueSkill(tickets1, tickets2)
	}
}

// Return 1.0 if the tickets are a perfect match, 0.0 if they are a complete mismatch
func calculateMatchQualityTrueSkill(tickets1 []model.Ticket, tickets2 []model.Ticket) float64 {
	var players1, players2 []trueskill.Player

	for _, ticket := range tickets1 {
		id, err := strconv.Atoi(ticket.Member)
		if err != nil {
			log.Println(err)
		}
		mu := ticket.Score
		sigma := mu / 3
		players1 = append(players1, trueskill.NewPlayer(id, mu, sigma))
	}

	for _, ticket := range tickets2 {
		id, err := strconv.Atoi(ticket.Member)
		if err != nil {
			log.Println(err)
		}
		mu := ticket.Score
		sigma := mu / 3
		players2 = append(players2, trueskill.NewPlayer(id, mu, sigma))
	}

	team1 := trueskill.NewTeam(players1)
	team2 := trueskill.NewTeam(players2)

	// calculate beta and tau
	// first we need to calculate the sum of all sigmas inside of a game
	players := append(players1, players2...)
	sigma_sum := 0.0
	for i := 0; i < len(players); i++ {
		sigma_sum += players[i].GetSigma()
	}
	avg_sigma := sigma_sum / float64(len(players))

	beta := avg_sigma * 3 / 2 // beta is skill difference to expect higher skill player/team to win 76% of the time
	tau := beta / 100         // tau is normal for 2% of beta
	pDraw := 0.1              // there shouldn't be any draws in a cs match - to be changed in the future

	game := trueskill.NewGame(beta, tau, pDraw)
	teams := make([]trueskill.Team, 0) // this is a 2 team game - to be configurable in the future
	teams = append(teams, team1, team2)

	result, err := game.CalcMatchQuality(teams)
	if err != nil {
		log.Println(err)
		return 0.0
	}

	return result
}

func calculateMatchQualityGlicko(tickets1 []model.Ticket, tickets2 []model.Ticket) float64 {
	// Calculate the average elo of the two teams
	team1 := 0.0
	team2 := 0.0
	for _, ticket := range tickets1 {
		team1 += ticket.Score
	}

	for _, ticket := range tickets2 {
		team2 += ticket.Score
	}

	team1 = team1 / float64(len(tickets1))
	team2 = team2 / float64(len(tickets2))

	// Calculate the difference in elo between the two teams
	eloDiff := team1 - team2

	// Calculate the expected win probability of the higher elo team
	// 1 / (1 + 10^((eloDiff) / 400))
	winProb := 1 / (1 + math.Pow(10, (eloDiff/400)))

	// Should return 1.0 if the teams are a perfect match, 0.0 if they are a complete mismatch
	winProb = 1.5 - winProb

	return winProb
}
