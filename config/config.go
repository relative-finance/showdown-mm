package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Redis               RedisConfig
	Server              ServerConfig
	MMRConfig           MMRConfig
	EthRpc              ExternalApiConfig
	ShowdownUserService ExternalApiConfig
	LichessApi          ExternalApiConfig
	CS2Api              ExternalApiConfig
	D2Api               ExternalApiConfig
	ShowdownStatsRelay  ExternalApiConfig
	LichessBaseUrl      ExternalApiConfig
	ShowdownApi         ExternalApiConfig
	MatchEndWebhook     ExternalApiConfig
	Subgraph            ExternalApiConfig
}

type ServerConfig struct {
	Port string
}

type MMRConfig struct {
	Mode              string
	Interval          int
	TeamSize          int
	Treshold          float64
	Range             int
	TimeToCancelMatch int
	TimeToAccept      int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type ExternalApiConfig struct {
	URL    string
	ApiKey string
}

var GlobalConfig *Config

func NewConfig() *Config {
	db, err := strconv.Atoi(readEnvVar("REDIS_DB"))
	if err != nil {
		db = 0
	}

	interval, err := strconv.Atoi(readEnvVar("MMR_INTERVAL"))
	if err != nil {
		interval = 5 // default
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

	timeToAccept, err := strconv.Atoi(readEnvVar("MMR_TIME_TO_ACCEPT"))
	if err != nil {
		timeToAccept = 30 // default
	}

	rangeInt, err := strconv.Atoi(readEnvVar("MMR_RANGE"))
	if err != nil {
		rangeInt = 100 // default
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
			Interval:          interval,
			TeamSize:          teamSize,
			Treshold:          treshold,
			TimeToCancelMatch: timeToCancelMatch,
			TimeToAccept:      timeToAccept,
			Range:             rangeInt,
		},
		EthRpc: ExternalApiConfig{
			URL: readEnvVar("ETH_RPC_URL"),
		},
		ShowdownUserService: ExternalApiConfig{
			URL:    readEnvVar("SHOWDOWN_API"),
			ApiKey: readEnvVar("SHOWDOWN_API_KEY"),
		},
		LichessApi: ExternalApiConfig{
			URL: readEnvVar("LICHESSAPI"),
		},
		CS2Api: ExternalApiConfig{
			URL: readEnvVar("CS2API"),
		},
		D2Api: ExternalApiConfig{
			URL: readEnvVar("D2API"),
		},
		ShowdownStatsRelay: ExternalApiConfig{
			URL: readEnvVar("RELAY_ADDRESS"),
		},
		LichessBaseUrl: ExternalApiConfig{
			URL: readEnvVar("LICHESS_BASE_URL"),
		},
		ShowdownApi: ExternalApiConfig{
			URL: readEnvVar("SHOWDOWN_RELAY"),
		},
		MatchEndWebhook: ExternalApiConfig{
			URL: readEnvVar("WEBHOOK_ENDPOINT"),
		},
		Subgraph: ExternalApiConfig{
			URL: readEnvVar("SUBGRAPH_URL"),
		},
	}

	return GlobalConfig
}

func readEnvVar(name string) string {
	godotenv.Load(".env")
	return os.Getenv(name)
}
