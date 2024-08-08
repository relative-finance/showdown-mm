package utils

import (
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
)

func AddMatchToRedis(matchId string, tickets1 []model.Ticket, tickets2 []model.Ticket, queue constants.QueueType) {
	for _, ticket := range tickets1 {
		redis.RedisClient.ZRem(constants.GetIndexNameQueue(queue), ticket.Member)
	}

	for _, ticket := range tickets2 {
		redis.RedisClient.ZRem(constants.GetIndexNameQueue(queue), ticket.Member)
	}

	matchPlayer := model.MatchPlayer{SteamId: "", Score: 0, Option: 1, Team: 1}
	for _, ticket := range tickets1 {
		matchPlayer.SteamId = ticket.Member.SteamID
		matchPlayer.Score = ticket.Score
		redis.RedisClient.HSet(matchId, ticket.Member.SteamID, matchPlayer.Marshal())
	}

	matchPlayer.Team = 2
	for _, ticket := range tickets2 {
		matchPlayer.SteamId = ticket.Member.SteamID
		matchPlayer.Score = ticket.Score
		redis.RedisClient.HSet(matchId, ticket.Member.SteamID, matchPlayer.Marshal())
	}
}
