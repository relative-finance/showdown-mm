package api

import (
	"context"
	"net/http"
	"time"

	"mmf/pkg/services"
	"mmf/wires"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
			if len(msg) == 10 {
				conn.WriteJSON(msg)
				// TODO: Add scheduled task to check if all players are ready and begin the game
			} else {
				conn.WriteMessage(websocket.TextMessage, []byte("Waiting for more players to join the game."))
			}
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
	c.JSON(200, tickets)
}
