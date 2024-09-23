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
		ws.StartLichessWebSocket(queue, id, walletAddress, c)
		return
	}
	ws.StartWebSocket(queue, id, walletAddress, c)
}

func fetchTickets(c *gin.Context) {
	queue := c.Param("queue")
	tickets := wires.Instance.TicketService.GetAllTickets(queue)
	c.JSON(200, tickets)
}
