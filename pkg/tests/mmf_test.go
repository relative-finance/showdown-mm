package tests

import (
	"context"
	"mmf/config"
	"mmf/pkg/redis"
	"mmf/pkg/server"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	config := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     "6378",
			DB:       0,
			Password: "redis",
		},
		Server: config.ServerConfig{
			Port: "9877",
		},
		MMRConfig: config.MMRConfig{
			Mode:     "glicko",
			Interval: "5",
			TeamSize: 2,
			Treshold: 0.8,
		},
	}
	redis.Init(config, context.Background())
	server := server.NewServer(config)
	server.Start()
}

func TestFetchTickets(t *testing.T) {
	t.Log("TestFetchTickets")

	req, err := http.NewRequest("GET", "http://localhost:9877/fetch/d2queue", nil)
	assert.NoError(t, err, "Error creating request")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "Error sending request")
	defer resp.Body.Close()

	// Check the response status code
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
}

func TestWebsockets(t *testing.T) {
	// TODO: write test
}
