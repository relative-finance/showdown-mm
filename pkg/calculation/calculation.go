package calculation

import (
	"log"
	"math"
	"mmf/pkg/model"
	"mmf/pkg/redis"
	"strconv"

	"github.com/fasmat/trueskill"
)

// Return 1.0 if the tickets are a perfect match, 0.0 if they are a complete mismatch
func CalculateMatchQuality(tickets1 []model.Ticket, tickets2 []model.Ticket) float64 {
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

func CalcMatchQualityNonTrueSkill(tickets1 []model.Ticket, tickets2 []model.Ticket) float64 {
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

func EvaluateTickets(mode string) bool {
	var gameTickets []model.Ticket
	tickets := redis.RedisClient.ZRangeWithScores("player_elo", 0, -1)

	if len(tickets.Val()) < 10 {
		return false
	}

	for _, ticket := range tickets.Val() {
		gameTickets = append(gameTickets, model.Ticket{Member: ticket.Member.(string), Score: ticket.Score})
	}

	for i := 0; i < len(gameTickets); i++ {
		if gameTickets[i+10-1].Score-gameTickets[i].Score > 100 {
			continue
		}

		matchTickets := gameTickets[i : i+10]
		tickets1, tickets2 := GetTeams(matchTickets)
		matchQuality := GetMatchQuality(tickets1, tickets2, mode)
		log.Printf("%d", matchQuality)
		if matchQuality > 0.8 {
			RemoveTickets(matchTickets)
			return true
		}
	}

	return false
}

func GetTeams(tickets []model.Ticket) ([]model.Ticket, []model.Ticket) {
	var tickets1, tickets2 []model.Ticket
	if len(tickets)%4 == 0 {
		for i := 0; i < len(tickets)/2; i++ {
			if i%2 == 0 {
				tickets1 = append(tickets1, tickets[i])
				tickets1 = append(tickets1, tickets[len(tickets)-i-1])
			} else {
				tickets2 = append(tickets2, tickets[i])
				tickets2 = append(tickets2, tickets[len(tickets)-i-1])
			}
		}
	} else {
		for i := 0; i < len(tickets)/2-1; i++ {
			if i%2 == 0 {
				tickets1 = append(tickets1, tickets[i])
				tickets1 = append(tickets1, tickets[len(tickets)-i-1])
			} else {
				tickets2 = append(tickets2, tickets[i])
				tickets2 = append(tickets2, tickets[len(tickets)-i-1])
			}
		}

		tickets1 = append(tickets1, tickets[len(tickets)/2])
		tickets2 = append(tickets2, tickets[len(tickets)/2+1])
	}

	return tickets1, tickets2
}

func GetMatchQuality(tickets1 []model.Ticket, tickets2 []model.Ticket, mode string) float64 {
	switch mode {
	case "trueskill":
		return CalculateMatchQuality(tickets1, tickets2)
	case "glicko":
		return CalcMatchQualityNonTrueSkill(tickets1, tickets2)
	default:
		return CalculateMatchQuality(tickets1, tickets2)
	}
}

func RemoveTickets(tickets []model.Ticket) {
	for _, ticket := range tickets {
		redis.RedisClient.ZRem("player_elo", ticket.Member)
	}
}
