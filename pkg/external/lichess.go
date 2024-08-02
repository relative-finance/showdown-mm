package external

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Glicko struct {
	Rating    float64 `json:"rating"`
	Deviation float64 `json:"deviation"`
}

type Perf struct {
	Glicko   Glicko `json:"glicko"`
	Nb       int    `json:"nb"`
	Progress int    `json:"progress"`
}

type User struct {
	Name string `json:"name"`
}

type LichessResponse struct {
	User       User    `json:"user"`
	Perf       Perf    `json:"perf"`
	Rank       *int    `json:"rank"`
	Percentile float64 `json:"percentile"`
}

type Performance struct {
	Games  int  `json:"games"`
	Rating int  `json:"rating"`
	RD     int  `json:"rd"`
	Prog   int  `json:"prog"`
	Prov   bool `json:"prov"`
	Runs   int  `json:"runs,omitempty"`
	Score  int  `json:"score,omitempty"`
}

type LichessAccount struct {
	Username string                 `json:"username"`
	Perfs    map[string]Performance `json:"perfs"`
}

func GetGlicko(username, perf string) (int, error) {

	apiKey := username
	if apiKey == "" {
		return 0, fmt.Errorf("LICHESS_API_KEY not found in environment variables")
	}

	// url := fmt.Sprintf("https://lichess.org/api/user/%s/perf/%s", username, perf)
	url := os.Getenv("LICHESS_BASE_URL") + "/api/account"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %v", err)
	}

	var lr LichessAccount
	err = json.Unmarshal(body, &lr)
	if err != nil {
		return 0, fmt.Errorf("error parsing JSON response: %v", err)
	}

	prf, ok := lr.Perfs[perf]
	if !ok {
		return 0, fmt.Errorf("no performance data for %s", perf)
	}

	log.Println("Rating for ", username, " is ", prf.Rating)

	return prf.Rating, nil
}
