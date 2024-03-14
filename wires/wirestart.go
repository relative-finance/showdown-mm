package wires

import (
	"mmf/pkg/redis"
	"mmf/pkg/services"
)

type Wires struct {
	TicketService services.TicketServiceImpl
}

var Instance *Wires

func Init() {
	Instance = &Wires{
		TicketService: services.TicketServiceImpl{
			Redis: redis.RedisClient,
		},
	}
}
