package api

import (
	"context"
	"time"

	"mmf/pkg/services"
	"mmf/wires"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func RegisterTicket(router *gin.Engine, ctx context.Context) {
	tickets := router.Group("/tickets")
	{
		tickets.POST("/submit", submitTicket)
		tickets.GET("/fetch", fetchTickets)
	}

	router.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			msg := wires.Instance.TicketService.EvaluateTickets(c)
			conn.WriteMessage(websocket.TextMessage, []byte(msg[0]))
			time.Sleep(1 * time.Second)
		}
	})

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

func fetchTickets(c *gin.Context) {
	tickets := wires.Instance.TicketService.GetAllTickets(c)
	c.JSON(200, gin.H{
		"tickets": tickets,
	})
}
