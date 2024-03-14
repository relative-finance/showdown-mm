package api

import (
	"context"

	"mmf/pkg/services"
	"mmf/wires"

	"github.com/gin-gonic/gin"
)

func RegisterTicket(router *gin.Engine, ctx context.Context) {
	tickets := router.Group("/tickets")
	{
		tickets.POST("/submit", submitTicket)
	}
}

func submitTicket(c *gin.Context) {
	var submitTicketRequest services.SubmitTicketRequest
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
