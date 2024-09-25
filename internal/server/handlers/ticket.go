package handlers

import (
	"context"

	ws "mmf/internal/server/websockets"
	"mmf/internal/wires"

	"github.com/gin-gonic/gin"
)

func RegisterTicket(router *gin.Engine, ctx context.Context) {
	tickets := router.Group("/tickets")
	{
		// tickets.POST("/submit/:game", submitTicket)
		tickets.GET("/fetch/:queue", fetchTickets)
	}

	router.GET("/ws/:queue/:id/:walletAddress", wsGet)
	router.GET("/ws/:queue/:id", wsGetLichess)
}

func wsGetLichess(c *gin.Context) {
	queue := c.Param("queue")
	id := c.Param("id")

	if queue == "" || id == "" {
		c.JSON(400, gin.H{"error": "missing required parameters"})
		return
	}

	if queue != "lcqueue" {
		c.JSON(400, gin.H{"error": "other than lcqueue is on different endpoint"})
		return
	}
	ws.StartLichessWebSocket(queue, id, c)
}

func wsGet(c *gin.Context) {
	queue := c.Param("queue")
	id := c.Param("id")
	walletAddress := c.Param("walletAddress")

	if queue == "" || id == "" || walletAddress == "" {
		c.JSON(400, gin.H{"error": "missing required parameters"})
		return
	}

	if queue == "lcqueue" {
		c.JSON(400, gin.H{"error": "lcqueue is on different endpoint"})
		return
	}
	ws.StartWebSocket(queue, id, walletAddress, c)
}

func fetchTickets(c *gin.Context) {
	queue := c.Param("queue")
	tickets := wires.Instance.TicketService.GetAllTickets(queue)
	c.JSON(200, tickets)
}
