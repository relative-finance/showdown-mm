package server

import (
	"context"
	"log"
	"time"

	"mmf/config"
	"mmf/internal/redis"
	"mmf/internal/redis/crawler"
	"mmf/internal/wires"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config *config.Config
}

func NewServer(config *config.Config) *Server {
	return &Server{
		config: config,
	}
}

func (server *Server) Start() {
	InitCrawler(server.config.MMRConfig)

	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	wires.Init(server.config)
	redis.Init(server.config, context.Background())
	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	r.Use(gin.Recovery())

	RegisterVersion(r, context.Background())

	err := r.Run(":" + server.config.Server.Port)

	if err != nil {
		log.Fatal("Could not start the server" + err.Error())
		return
	}

	println("Starting server on port: " + server.config.Server.Port)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length")
		c.Writer.Header().Set("Access-Allow-Methods", "POST, GET")

		c.Next()
	}
}

func InitCrawler(config config.MMRConfig) bool {

	ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
	quit := make(chan struct{})
	go func() bool {
		for {
			select {
			case <-ticker.C:
				flag := crawler.StartCrawler(config)

				if !flag {
					return false
				}
			case <-quit:
				ticker.Stop()
				return true
			}
		}
	}()

	return true
}
