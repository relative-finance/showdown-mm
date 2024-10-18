package server

import (
	"context"
	"mmf/internal/server/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterVersion(router *gin.Engine, ctx context.Context) {
	handlers.RegisterTicket(router, ctx)
	handlers.RegisterHealth(router, ctx)
}
