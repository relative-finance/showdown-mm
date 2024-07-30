package api

import (
	"context"

	"github.com/gin-gonic/gin"
)

func RegisterVersion(router *gin.Engine, ctx context.Context) {
	RegisterTicket(router, ctx)
}
