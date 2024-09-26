package handlers

import (
	"context"
	"strconv"

	"mmf/internal/calculation"
	"mmf/internal/constants"
	"mmf/internal/model"
	r "mmf/internal/redis"
	ws "mmf/internal/server/websockets"
	"mmf/internal/wires"
	"mmf/pkg/client"

	"github.com/gin-gonic/gin"
)

func RegisterTicket(router *gin.Engine, ctx context.Context) {
	tickets := router.Group("/tickets")
	{
		// tickets.POST("/submit/:game", submitTicket)
		tickets.GET("/fetch/:queue", fetchTickets)
		tickets.POST("/test/:queue", testTicket)
	}

	router.GET("/ws/:queue/:id/:walletAddress", wsGet)
	router.GET("/ws/:queue/:id", wsGetLichess)
}

func testTicket(c *gin.Context) {
	queue := c.Param("queue")
	if queue == "" {
		c.JSON(400, gin.H{"error": "missing required parameters"})
		return
	}

	var testReq client.TestMMRRequest
	if err := c.BindJSON(&testReq); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	if testReq.MMRConfig.Mode == "" {
		testReq.MMRConfig.Mode = "glicko"
	}
	if testReq.MMRConfig.Range == 0 {
		testReq.MMRConfig.Range = 100
	}
	if testReq.MMRConfig.TeamSize == 0 {
		testReq.MMRConfig.TeamSize = 1
	}
	if testReq.MMRConfig.Interval == 0 {
		testReq.MMRConfig.Interval = 5
	}
	if testReq.MMRConfig.Treshold == 0 {
		testReq.MMRConfig.Treshold = 0.8
	}

	for id, player := range testReq.Players {
		idStr := strconv.Itoa(id)
		ticket := model.SubmitTicketRequest{
			WalletAddress:     idStr,
			Id:                idStr,
			LichessCustomData: player.LichessCustomData,
			Elo:               player.Elo,
		}

		if player.LichessCustomData == nil && (queue == "lcqueue" || queue == "lcqueue_test") {
			ticket.LichessCustomData = &model.LichessCustomData{
				Time:       5,
				Increment:  0,
				Collateral: model.SP,
			}
		}

		if _, err := wires.Instance.TicketService.SubmitTicket(ticket, queue); err != nil {
			c.JSON(400, gin.H{"error": "error submitting ticket"})
			return
		}
	}

	pairs := make([]client.TestPairResponse, 0)
	calculation.EvaluateTickets(testReq.MMRConfig, constants.GetQueueType(queue), &pairs)
	r.RedisClient.Del(constants.GetIndexNameStr(queue))
	c.JSON(200, gin.H{"matches": pairs})
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
