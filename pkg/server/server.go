package server

import (
	"context"
	"log"
	"strconv"
	"time"

	"mmf/config"
	"mmf/pkg/crawler"
	"mmf/pkg/redis"
	"mmf/pkg/server/api"
	"mmf/wires"

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
	initCrawler(server.config.MMRConfig)

	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	wires.Init(server.config)
	redis.Init(server.config, context.Background())
	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	r.Use(gin.Recovery())

	api.RegisterVersion(r, context.Background())

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

func initCrawler(config config.MMRConfig) bool {
	intv, err := strconv.Atoi(config.Interval)
	if err != nil {
		log.Println("Error converting interval to int")
		return false
	}

	ticker := time.NewTicker(time.Duration(intv) * time.Second)
	quit := make(chan struct{})
	go func() bool {
		for {
			select {
			case <-ticker.C:
				flag := crawler.StartCrawler(config.Mode)

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
