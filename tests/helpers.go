package tests

import (
	"encoding/json"
	"errors"
	"io"
	ws "mmf/internal/server/websockets"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func callWS(url string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}

	wsConn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	_, mess, err := wsConn.ReadMessage()
	if err != nil {
		return nil, err
	}

	// if mess doesn't start with "Hello," then return error
	if !strings.HasPrefix(string(mess), "Hello,") {
		return nil, errors.New("invalid message")
	}

	return wsConn, nil
}

func getMatchId(wsConn *websocket.Conn) (*string, error) {
	_, mess, err := wsConn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var matchId struct {
		MatchId string `json:"matchId"`
	}

	err = json.Unmarshal(mess, &matchId)
	if err != nil {
		return nil, err
	}

	return &matchId.MatchId, err
}

func sendResponseWS(wsConn *websocket.Conn, resp ws.UserResponse) error {
	message, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	return wsConn.WriteMessage(websocket.TextMessage, message)
}

func httpCall(method, url string, body string) (*http.Response, error) {
	bodyReader := io.Reader(strings.NewReader(body))
	if body == "" {
		bodyReader = nil
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
