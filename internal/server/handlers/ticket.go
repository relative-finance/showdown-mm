package handlers

import (
	"context"
	"strconv"

	"mmf/internal/model"
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

	router.GET("/ws/:queue/:steamId/:walletAddress", wsGet)

}

func wsGet(c *gin.Context) {
	queue := c.Param("queue")
	steamId := c.Param("steamId")
	walletAddress := c.Param("walletAddress")
	var lichessData *model.LichessCustomData
	if queue == "lcqueue" {
		increment := c.DefaultQuery("increment", "0")
		time := c.DefaultQuery("time", "5")
		collateral := c.DefaultQuery("collateral", "showdown")

		incrementInt, err := strconv.Atoi(increment)
		if err != nil {
			c.JSON(400, gin.H{"error": "increment must be an integer"})
			return
		}

		timeInt, err := strconv.Atoi(time)
		if err != nil {
			c.JSON(400, gin.H{"error": "time must be an integer"})
			return
		}

		lichessData = &model.LichessCustomData{
			Increment:  incrementInt,
			Time:       timeInt,
			Collateral: model.Collateral(collateral),
		}
	}
	ws.StartWebSocket(queue, steamId, walletAddress, lichessData, c)
}

func fetchTickets(c *gin.Context) {
	queue := c.Param("queue")
	tickets := wires.Instance.TicketService.GetAllTickets(c, queue)
	c.JSON(200, tickets)
}
