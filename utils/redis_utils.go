package utils

import (
	"encoding/json"
	"log"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
)

func AddMatchToRedis(matchId string, tickets1 []model.Ticket, tickets2 []model.Ticket, queue constants.QueueType) {
	for _, ticket := range tickets1 {
		memberJSON, err := json.Marshal(ticket.Member)
		if err != nil {
			log.Println("Error serializing MemberData:", err)
			continue // skip this iteration if there's an error
		}
		redis.RedisClient.ZRem(constants.GetIndexNameQueue(queue), memberJSON)
	}

	for _, ticket := range tickets2 {
		memberJSON, err := json.Marshal(ticket.Member)
		if err != nil {
			log.Println("Error serializing MemberData:", err)
			continue // skip this iteration if there's an error
		}
		redis.RedisClient.ZRem(constants.GetIndexNameQueue(queue), memberJSON)
	}

	userState := model.UserGlobalState{State: model.MatchFound, MatchId: matchId}

	matchPlayer := model.MatchPlayer{Id: "", Score: 0, Option: 1, Team: 1}
	for _, ticket := range tickets1 {
		matchPlayer.Id = ticket.Member.Id
		matchPlayer.Score = ticket.Score
		redis.RedisClient.HSet(matchId, ticket.Member.Id, matchPlayer.Marshal())
		redis.RedisClient.HSet("user_state", ticket.Member.Id, userState.Marshal())
	}

	matchPlayer.Team = 2
	for _, ticket := range tickets2 {
		matchPlayer.Id = ticket.Member.Id
		matchPlayer.Score = ticket.Score
		redis.RedisClient.HSet(matchId, ticket.Member.Id, matchPlayer.Marshal())
		redis.RedisClient.HSet("user_state", ticket.Member.Id, userState.Marshal())
	}
}

func SetUserStateInRedis(userId string, userState *model.UserGlobalState) error {
	cmd := redis.RedisClient.HSet("user_state", userId, userState.Marshal())
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func SetMatchInfoInRedis(matchId, userId string, matchPlayer *model.MatchPlayer) error {
	cmd := redis.RedisClient.HSet(matchId, userId, matchPlayer.Marshal())
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func ClearMatchData(matchId string, playerIds *[]string) error {
	cmd := redis.RedisClient.Del(matchId)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	for _, playerId := range *playerIds {
		if err := DeleteUserState(playerId); err != nil {
			return err
		}
	}
	return nil
}

func DeleteUserState(userId string) error {
	cmd := redis.RedisClient.HDel("user_state", userId)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}
