package wires

import (
	"mmf/config"
	"mmf/internal/redis"
	"mmf/internal/services"
)

type Wires struct {
	TicketService services.TicketServiceImpl
}

var Instance *Wires

func Init(config *config.Config) {
	Instance = &Wires{
		TicketService: services.TicketServiceImpl{
			Redis:     redis.RedisClient,
			MMRConfig: config.MMRConfig,
		},
	}
}
