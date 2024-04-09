package tests

import (
	"context"
	"mmf/config"
	"mmf/pkg/redis"
	"mmf/pkg/server"
	"mmf/pkg/server/api"
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
			Mode:     "glicko",
			Interval: "5",
			TeamSize: 2,
			Treshold: 0.8,
		},
	}

	server.InitCrawler(config.MMRConfig)
	redis.Init(config, context.Background())

	r := gin.Default()
	api.RegisterVersion(r, context.Background())

	testServer = httptest.NewServer(r)
}

func TestFetchTickets(t *testing.T) {
	t.Log("TestFetchTickets")

	req, err := http.NewRequest("GET", testServer.URL+"/fetch/d2queue", nil)
	assert.NoError(t, err, "Error creating request")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "Error sending request")
	defer resp.Body.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
}

func TestWebsocketsConnection(t *testing.T) {
	t.Log("TestWebsocketsConnection")
	// Replace "/ws/d2queue/steamid1" with your actual WebSocket endpoint
	wsURL := strings.Replace(testServer.URL, "http", "ws", 1) + "/ws/d2queue/steamid1"

	// Prepare a WebSocket dialer
	dialer := websocket.Dialer{}

	// Connect to the server
	wsConn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer wsConn.Close()

	done := make(chan struct{})

	// Start reading messages from the WebSocket in a goroutine
	go func() {
		defer close(done)
		for {
			_, message, err := wsConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					t.Logf("Error: %v", err)
				}
				return // return or break based on your error handling
			}

			expectedMessage := "Hello, steamid1"
			if string(message) != expectedMessage {
				t.Errorf("Expected message %q, got %q", expectedMessage, message)
			}

			// Assuming you want to test only the first message for simplicity
			break
		}
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
}
