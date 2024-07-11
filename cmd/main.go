package main

import (
	"context"
	"mmf/config"
	"mmf/pkg/redis"
	"mmf/pkg/server"
)

func main() {
	config := config.NewConfig()
	redis.Init(config, context.Background())
	server := server.NewServer(config)
	server.Start()
}
