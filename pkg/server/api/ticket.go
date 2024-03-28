package api

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"mmf/pkg/model"
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

// Map for user connection
var userConnections = make(map[string]*websocket.Conn)
var userConnectionsMutex sync.Mutex

func RegisterTicket(router *gin.Engine, ctx context.Context) {
	tickets := router.Group("/tickets")
	{
		tickets.POST("/submit", submitTicket)
		tickets.GET("/fetch", fetchTickets)
	}

	router.GET("/ws/:steamId", func(c *gin.Context) {
		steamId := c.Param("steamId")
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		userConnectionsMutex.Lock()
		userConnections[steamId] = conn
		userConnectionsMutex.Unlock()

		defer func() {
			userConnectionsMutex.Lock()
			delete(userConnections, steamId)
			userConnectionsMutex.Unlock()
		}()

		// Keep player connected
		for {
			// TODO: Read elo form relay
			conn.WriteMessage(websocket.TextMessage, []byte("Hello, "+steamId))
			time.Sleep(3 * time.Second)
		}
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

func SendMessageToUser(steamId string, message []byte) {
	userConnectionsMutex.Lock()
	defer userConnectionsMutex.Unlock()

	conn, ok := userConnections[steamId]
	if !ok {
		log.Println("User not connected")
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		log.Println(err)
	}
}

func DisconnectUser(steamId string) {
	userConnectionsMutex.Lock()
	defer userConnectionsMutex.Unlock()

	conn, ok := userConnections[steamId]
	if !ok {
		log.Println("User not connected")
		return
	}

	// Close the connection
	if err := conn.Close(); err != nil {
		log.Println("Error closing connection:", err)
	}

	// Remove the connection from the map
	delete(userConnections, steamId)
}
