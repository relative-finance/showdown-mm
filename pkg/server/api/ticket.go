package api

import (
	"context"

	"mmf/pkg/model"
	"mmf/pkg/ws"
	"mmf/wires"

	"github.com/gin-gonic/gin"
)

func RegisterTicket(router *gin.Engine, ctx context.Context) {
	tickets := router.Group("/tickets")
	{
		tickets.POST("/submit", submitTicket)
		tickets.GET("/fetch", fetchTickets)
	}

	router.GET("/ws/:steamId", func(c *gin.Context) {
		steamId := c.Param("steamId")
		ws.StartWebSocket(steamId, c)
	})

}

func submitTicket(c *gin.Context) {
	var submitTicketRequest model.SubmitTicketRequest
	c.BindJSON(&submitTicketRequest)

	err := wires.Instance.TicketService.SubmitTicket(c, submitTicketRequest)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
	} else {
		c.JSON(200, gin.H{
			"message": "Ticket submitted",
		})
	}
}

func fetchTickets(c *gin.Context) {
	tickets := wires.Instance.TicketService.GetAllTickets(c)
	c.JSON(200, tickets)
}
