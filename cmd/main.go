package main

import (
	"context"
	"mmf/config"
	"mmf/internal/redis"
	"mmf/internal/server"
)

func main() {
	config := config.NewConfig()
	redis.Init(config, context.Background())
	server := server.NewServer(config)
	server.Start()
}
