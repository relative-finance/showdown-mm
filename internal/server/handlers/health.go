package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
)

func RegisterHealth(router *gin.Engine, ctx context.Context) {
	router.GET("/health", healthHandler)
}

func healthHandler(c *gin.Context) {
	c.Status(200)
}
