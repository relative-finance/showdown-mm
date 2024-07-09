package main

import (
	"context"
	"log"
	"mmf/config"
	"mmf/pkg/client"
	"mmf/pkg/redis"
	"mmf/pkg/server"
)

func main() {
	config := config.NewConfig()
	redis.Init(config, context.Background())

	elo, err := client.GetGlicko("thibault", "bullet")
	if err != nil {
		log.Fatalf("Error getting glicko: ", err)
	}
	log.Println("Elo: ", elo)

	server := server.NewServer(config)
	server.Start()
}
