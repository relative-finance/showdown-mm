package client

import (
	"mmf/config"
	"mmf/internal/model"
)

type LobbyConfig struct {
	GameName     string `json:"gameName"`
	ServerRegion int    `json:"serverRegion"`
	PassKey      string `json:"passKey"`
	GameMode     string `json:"gameMode"`
}

// MatchRequestBodyD2 represents the structure of the JSON body for the REST call
type MatchRequestBodyD2 struct {
	TeamA       []int64     `json:"teamA"`
	TeamB       []int64     `json:"teamB"`
	LobbyConfig LobbyConfig `json:"lobbyConfig"`
	StartTime   string      `json:"startTime"`
}

// CS2
type Team struct {
	Name string `json:"name"`
}

type PlayerDatHost struct {
	Team      string `json:"team"`
	SteamId64 string `json:"steam_id_64"`
}
type GameSettings struct {
	Map                 string `json:"map"`
	ConnectTime         int    `json:"connect_time"`
	MatchBeginCountdown int    `json:"match_begin_countdown"`
	EnableTechPause     bool   `json:"enable_tech_pause"`
}
type Webhooks struct {
	MatchEndURL string `json:"match_end_url"`
	RoundEndURL string `json:"round_end_url"`
}
type MatchRequestBodyCS2 struct {
	Team1    Team            `json:"team1"`
	Team2    Team            `json:"team2"`
	Webhooks Webhooks        `json:"webhooks"`
	Settings GameSettings    `json:"settings"`
	Players  []PlayerDatHost `json:"players"`
}

// Lichess
type MatchRequestBodyLichess struct {
	TeamA Team `json:"team1"`
	TeamB Team `json:"team2"`
}

// LICHESS
type CreateLichessMatchRequest struct {
	Player1       string  `json:"player1"`           // API Access Key for the player1
	Player2       string  `json:"player2"`           // API Access Key for the player2
	Clock         Clock   `json:"clock,omitempty"`   // Clock for the match
	Variant       Variant `json:"variant"`           // Variant of the match
	Rated         bool    `json:"rated,omitempty"`   // Whether the match is rated or not
	Message       string  `json:"message,omitempty"` // Message to be sent to the opponent
	Rules         []Rules `json:"rules,omitempty"`   // Rules for the match
	PairAt        int     `json:"pairAt"`            // Time in seconds to wait before pairing
	StartClocksAt int     `json:"startClocksAt"`     // Time in seconds to wait before starting clocks
	Webhook       string  `json:"webhook,omitempty"` // Webhook to be called after the match ends
	Instant       bool    `json:"instant,omitempty"` // Instant to be called after the match ends
}

type StartLichessShowdownMatchRequest struct {
	MatchID   string `json:"matchId"`
	LichessID string `json:"lichessMatchId"`
}

type Clock struct {
	Increment int `json:"increment"`
	Limit     int `json:"limit"`
}

type Variant string

const (
	Standard      Variant = "standard"
	Chess960      Variant = "chess960"
	Crazyhouse    Variant = "crazyhouse"
	Antichess     Variant = "antichess"
	Atomic        Variant = "atomic"
	Horde         Variant = "horde"
	KingOfTheHill Variant = "kingOfTheHill"
	RacingKings   Variant = "racingKings"
	ThreeCheck    Variant = "threeCheck"
	FromPosition  Variant = "fromPosition"
)

var VariantValue = map[string]Variant{
	"standard":      Standard,
	"chess960":      Chess960,
	"crazyhouse":    Crazyhouse,
	"antichess":     Antichess,
	"atomic":        Atomic,
	"horde":         Horde,
	"kingOfTheHill": KingOfTheHill,
	"racingKings":   RacingKings,
	"threeCheck":    ThreeCheck,
	"fromPosition":  FromPosition,
}

type Rules string

const (
	NoAbort     Rules = "noAbort"
	NoRematch   Rules = "noRematch"
	NoGiveTime  Rules = "noGiveTime"
	NoClaimWin  Rules = "noClaimWin"
	NoEarlyDraw Rules = "noEarlyDraw"
)

var RulesValue = map[string]Rules{
	"noAbort":     NoAbort,
	"noRematch":   NoRematch,
	"noGiveTime":  NoGiveTime,
	"noClaimWin":  NoClaimWin,
	"noEarlyDraw": NoEarlyDraw,
}

type Color string

const (
	White  Color = "white"
	Black  Color = "black"
	Random Color = "random"
)

var ColorValue = map[string]Color{
	"white":  White,
	"black":  Black,
	"random": Random,
}

type TestPlayerRequest struct {
	Elo               float64                  `json:"elo"`
	LichessCustomData *model.LichessCustomData `json:"lichessCustomData"`
}

type TestMMRRequest struct {
	Players          []TestPlayerRequest `json:"players"`
	config.MMRConfig `json:",inline"`
}
