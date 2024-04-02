package constants

type GameType string

const (
	CounterStrike2 GameType = "cs2"
	Dota2          GameType = "dota2"
)

func (g *GameType) String() string {
	return string(*g)
}

func GetAllGameTypes() []GameType {
	return []GameType{CounterStrike2, Dota2}
}

func GetIndexName(game GameType) string {
	return "players_" + game.String()
}
