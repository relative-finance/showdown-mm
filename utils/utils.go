package utils

import (
	"encoding/json"
	"log"
	"mmf/internal/constants"
	"mmf/internal/model"
	"mmf/internal/redis"
	ws "mmf/internal/server/websockets"
	"mmf/pkg/client"
	"time"

	r "github.com/go-redis/redis"
)

func WaitingForMatchThread(matchId string, queue constants.QueueType, tickets1 []model.Ticket, tickets2 []model.Ticket, timeToCancelMatch int) {
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
				MatchFailedReturnPlayersToMM(queue, matchId, true)
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
				client.ScheduleLichessMatch(tickets1, tickets2, matchId)
			}
			log.Println("Match scheduled")

			DisconnectAllUsers(matchId)
			ret := redis.RedisClient.Del(matchId)
			if ret.Err() != nil {
				log.Println("Error deleting match from redis: ", ret.Err())
			}
			break
		}
	}

	MatchFailedReturnPlayersToMM(queue, matchId, false)
}

func DisconnectAllUsers(matchId string) {
	for _, redisPlayer := range redis.RedisClient.HGetAll(matchId).Val() {
		matchPlayer := model.UnmarshalMatchPlayer([]byte(redisPlayer))
		log.Println("Disconnecting user: ", matchPlayer.SteamId)
		ws.DisconnectUser(matchPlayer.SteamId)
	}
}

func MatchFailedReturnPlayersToMM(queue constants.QueueType, matchId string, denied bool) {
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
