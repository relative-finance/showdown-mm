package redis

import (
	"context"
	"log"
	"mmf/config"

	"github.com/go-redis/redis"
)

var RedisClient *redis.Client

func Init(config *config.Config, ctx context.Context) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: config.Redis.Host + ":" + config.Redis.Port,
		DB:   config.Redis.DB,
	})

	_, err := RedisClient.Ping().Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %s", err)
	}
	log.Println("Connected to Redis")
}
