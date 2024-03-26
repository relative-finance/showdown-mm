package services

import (
	"log"
	"math"
	"strconv"

	"github.com/fasmat/trueskill"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

type TicketServiceImpl struct {
	Redis *redis.Client
}

type SubmitTicketRequest struct {
	SteamID string  `json:"steamId"`
	Elo     float64 `json:"elo"`
}

type Ticket struct {
	Member string  `json:"steamId"`
	Score  float64 `json:"elo"`
}

func (s *TicketServiceImpl) SubmitTicket(g *gin.Context, submitTicketRequest SubmitTicketRequest) error {
	s.Redis.ZAdd("player_elo", redis.Z{Score: float64(submitTicketRequest.Elo), Member: submitTicketRequest.SteamID})
	return nil
}

func (s *TicketServiceImpl) GetAllTickets(g *gin.Context) []Ticket {
	var gameTickets []Ticket
	tickets := s.Redis.ZRangeWithScores("player_elo", 0, -1)

	for _, ticket := range tickets.Val() {
		gameTickets = append(gameTickets, Ticket{Member: ticket.Member.(string), Score: ticket.Score})
	}

	log.Println("Calculation with TrueSkill:", CalculateMatchQuality(gameTickets[0:4]))          // doens't include second limit
	log.Println("Calculation with only Glicko:", CalcMatchQualityNonTrueSkill(gameTickets[0:4])) // doens't include second limit
	return gameTickets
}

func (s *TicketServiceImpl) EvaluateTickets(g *gin.Context) []string {
	gameTickets := s.Redis.ZRange("player_elo", 0, -1) // Includes second limit
	return gameTickets.Val()
}

// Return 1.0 if the tickets are a perfect match, 0.0 if they are a complete mismatch
func CalculateMatchQuality(tickets []Ticket) float64 {
	players := make([]trueskill.Player, 0)

	for i := 0; i < len(tickets); i++ {
		id, err := strconv.Atoi(tickets[i].Member)
		if err != nil {
			log.Println(err)
		}
		mu := tickets[i].Score
		sigma := mu / 3
		players = append(players, trueskill.NewPlayer(id, mu, sigma))
	}

	team1 := trueskill.NewTeam(players[0 : len(tickets)/2])
	team2 := trueskill.NewTeam(players[len(tickets)/2:])

	// calculate beta and tau
	// first we need to calculate the sum of all sigmas inside of a game
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

	log.Println("Team 1: ", formatPrintTeam(team1))
	log.Println("Team 2: ", formatPrintTeam(team2))

	result, err := game.CalcMatchQuality(teams)
	if err != nil {
		log.Println(err)
		return 0.0
	}

	return result
}

func CalcMatchQualityNonTrueSkill(tickets []Ticket) float64 {
	// Calculate the average elo of the two teams
	team1 := 0.0
	team2 := 0.0
	for i := 0; i < len(tickets); i++ {
		if i < len(tickets)/2 {
			team1 += tickets[i].Score
		} else {
			team2 += tickets[i].Score
		}
	}

	team1 = team1 / float64(len(tickets)/2)
	team2 = team2 / float64(len(tickets)/2)

	// Calculate the difference in elo between the two teams
	eloDiff := team1 - team2

	// Calculate the expected win probability of the higher elo team
	// 1 / (1 + 10^((eloDiff) / 400))
	winProb := 1 / (1 + math.Pow(10, (eloDiff/400)))

	// Should return 1.0 if the teams are a perfect match, 0.0 if they are a complete mismatch
	winProb = 1.5 - winProb

	return winProb
}

func formatPrintTeam(t trueskill.Team) string {
	team := ""
	for i := 0; i < len(t.GetPlayers()); i++ {
		team += strconv.Itoa(t.GetPlayers()[i].GetID()) + " "
	}
	return team
}
