package tests

import (
	"context"
	"encoding/json"
	"mmf/config"
	"mmf/pkg/model"
	"mmf/pkg/redis"
	"mmf/pkg/server"
	"mmf/pkg/server/api"
	"mmf/pkg/ws"
	"mmf/wires"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var (
	testServer *httptest.Server
)

func TestMain(m *testing.M) {
	setup()
	exitCode := m.Run()
	os.Exit(exitCode)
}

var wsURL string

func setup() {
	// TODO: This should be read from test .env file
	config := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     "6378",
			DB:       0,
			Password: "redis",
		},
		MMRConfig: config.MMRConfig{
			Mode:              "glicko",
			Interval:          "5",
			TeamSize:          1,
			Treshold:          0.8,
			TimeToCancelMatch: 15,
		},
	}

	server.InitCrawler(config.MMRConfig)
	redis.Init(config, context.Background())
	wires.Init(config)

	r := gin.Default()
	api.RegisterVersion(r, context.Background())

	testServer = httptest.NewServer(r)
	wsURL = strings.Replace(testServer.URL, "http", "ws", 1) + "/ws"
}

func TestFetchTickets(t *testing.T) {
	t.Log("Test Fetch Tickets")
	wsConn, err := callWS(wsURL + "/d2queue/1")
	assert.NoError(t, err, "Error connecting to WebSocket")
	defer wsConn.Close()

	resp, err := httpCall("GET", testServer.URL+"/tickets/fetch/d2queue", "")
	assert.NoError(t, err, "Error making HTTP request")
	defer resp.Body.Close()

	var tickets []model.Ticket
	err = json.NewDecoder(resp.Body).Decode(&tickets)
	assert.NoError(t, err, "Error decoding response body")

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
	assert.Equal(t, 1, len(tickets), "Unexpected number of tickets")
	assert.Equal(t, "1", tickets[0].Member, "Unexpected ticket member")
}

func TestWebsocketsConnection(t *testing.T) {
	t.Log("Test Websockets Connection")

	url := wsURL + "/d2queue/1"
	wsConn, err := callWS(url)
	assert.NoError(t, err, "Error connecting to WebSocket")
	defer wsConn.Close()

	done := make(chan struct{})

	// Start reading messages from the WebSocket in a goroutine
	go func() {
		defer close(done)
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				t.Logf("Error: %v", err)
			}
			return // return or break based on your error handling
		}

		expectedMessage := "Hello, 1"
		assert.Equal(t, expectedMessage, string(message), "Unexpected message received")
	}()

	// Write a message (if your test case requires sending a message to the server)
	err = wsConn.WriteMessage(websocket.TextMessage, []byte("Test message"))
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Wait for done signal from the read goroutine, with a timeout to prevent hanging tests
	select {
	case <-done:
	case <-time.After(3 * time.Second): // Adjust the timeout as needed
		t.Fatal("Test timed out waiting for WebSocket response")
	}

	redis.RedisClient.FlushAll()
}

func TestMatchMakingFlowEveryoneAccepts(t *testing.T) {
	t.Log("Test MatchMaking Flow Everyone Accepts")
	wsConn1, err := callWS(wsURL + "/d2queue/1")
	assert.NoError(t, err, "Error connecting to WebSocket 1")
	defer wsConn1.Close()

	wsConn2, err := callWS(wsURL + "/d2queue/2")
	assert.NoError(t, err, "Error connecting to WebSocket 2")
	defer wsConn2.Close()

	time.Sleep(1 * time.Second)

	// Get matchId from both connections
	matchId1, err := getMatchId(wsConn1)
	assert.NoError(t, err, "Error getting matchId from WebSocket 1")
	matchId2, err := getMatchId(wsConn2)
	assert.NoError(t, err, "Error getting matchId from WebSocket 2")

	// Get match details from Redis
	matchDetails := redis.RedisClient.HGetAll(*matchId1)
	assert.NoError(t, matchDetails.Err(), "Error fetching match details")

	assert.Equal(t, 2, len(matchDetails.Val()), "Unexpected number of players in the match")

	// Send response to both connections
	err = sendResponseWS(wsConn1, ws.UserResponse{MatchId: *matchId1, Option: 2})
	assert.NoError(t, err, "Error sending response to WebSocket 1")

	err = sendResponseWS(wsConn2, ws.UserResponse{MatchId: *matchId2, Option: 2})
	assert.NoError(t, err, "Error sending response to WebSocket 2")

	redis.RedisClient.FlushAll()
}

func TestOneGuysDoesntRespond(t *testing.T) {
	t.Log("Test One Guys Doesn't Respond")
	wsConn1, err := callWS(wsURL + "/d2queue/3")
	assert.NoError(t, err, "Error connecting to WebSocket 3")
	defer wsConn1.Close()

	wsConn2, err := callWS(wsURL + "/d2queue/4")
	assert.NoError(t, err, "Error connecting to WebSocket 4")
	defer wsConn2.Close()

	time.Sleep(1 * time.Second)

	// Get matchId from both connections
	matchId1, err := getMatchId(wsConn1)
	assert.NoError(t, err, "Error getting matchId from WebSocket 1")

	// Get match details from Redis
	matchDetails := redis.RedisClient.HGetAll(*matchId1)
	assert.NoError(t, matchDetails.Err(), "Error fetching match details")

	// Only first sents response
	err = sendResponseWS(wsConn1, ws.UserResponse{MatchId: *matchId1, Option: 2})
	assert.NoError(t, err, "Error sending response to WebSocket 1")

	// Wait for the match to be cancelled
	time.Sleep(20 * time.Second)
	matchExists := redis.RedisClient.Exists(*matchId1)
	assert.Equal(t, int64(0), matchExists.Val(), "Match should have been cancelled")

	firstPlayerBackToQueue := redis.RedisClient.ZRangeWithScores("d2queue", 0, -1)
	t.Log(firstPlayerBackToQueue.Val())
	assert.NoError(t, firstPlayerBackToQueue.Err(), "Error fetching player from queue")

	redis.RedisClient.FlushAll()
}
