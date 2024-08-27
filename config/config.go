package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Redis     RedisConfig
	Server    ServerConfig
	MMRConfig MMRConfig
	EthRpc    EthRpcConfig
}

type ServerConfig struct {
	Port string
}

type MMRConfig struct {
	Mode              string
	Interval          string
	TeamSize          int
	Treshold          float64
	TimeToCancelMatch int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type EthRpcConfig struct {
	URL string
}

var GlobalConfig *Config

func NewConfig() *Config {
	db, err := strconv.Atoi(readEnvVar("REDIS_DB"))
	if err != nil {
		db = 0
	}

	teamSize, err := strconv.Atoi(readEnvVar("MMR_TEAM_SIZE"))
	if err != nil {
		teamSize = 5 // default
	}

	treshold, err := strconv.ParseFloat(readEnvVar("MMR_TRESHOLD"), 64)
	if err != nil {
		treshold = 0.8 // default
	}

	timeToCancelMatch, err := strconv.Atoi(readEnvVar("MMR_TIME_TO_CANCEL_MATCH"))
	if err != nil {
		timeToCancelMatch = 60 // default
	}

	GlobalConfig = &Config{
		Redis: RedisConfig{
			Host:     readEnvVar("REDIS_HOST"),
			Port:     readEnvVar("REDIS_PORT"),
			Password: readEnvVar("REDIS_PASSWORD"),
			DB:       db,
		},
		Server: ServerConfig{
			Port: readEnvVar("SERVER_PORT"),
		},
		MMRConfig: MMRConfig{
			Mode:              readEnvVar("MMR_MODE"),
			Interval:          readEnvVar("MMR_INTERVAL"),
			TeamSize:          teamSize,
			Treshold:          treshold,
			TimeToCancelMatch: timeToCancelMatch,
		},
		EthRpc: EthRpcConfig{
			URL: readEnvVar("ETH_RPC_URL"),
		},
	}

	return GlobalConfig
}

func readEnvVar(name string) string {
	godotenv.Load(".env")
	return os.Getenv(name)
}
