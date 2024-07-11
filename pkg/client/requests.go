package client

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
