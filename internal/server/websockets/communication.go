package ws

import (
	"log"
	"mmf/internal/model"
	"time"

	"github.com/gorilla/websocket"
)

func SendMatchFoundToPlayers(matchId string, matchTickets []model.Ticket, timeToAccept int64) bool {
	mess := GenerateMatchFoundResponse(matchTickets, matchId, timeToAccept)

	for _, ticket := range matchTickets {
		SendJSONToUser(ticket.Member.Id, Info, mess)
	}
	return true
}

func SendMessageToUser(id string, event EventType, message string) {
	userConnectionsMutex.Lock()
	defer userConnectionsMutex.Unlock()

	conn, ok := userConnections[id]
	if !ok {
		log.Println("User not connected")
		return
	}

	if err := conn.WriteJSON(GetMessage(event, message)); err != nil {
		log.Println(err)
	}
}

func SendJSONToUser(id string, event EventType, message interface{}) {
	userConnectionsMutex.Lock()
	defer userConnectionsMutex.Unlock()

	conn, ok := userConnections[id]
	if !ok {
		log.Println("User not connected")
		return
	}

	SendJSON(conn, event, message)
}

func DisconnectUser(steamId string) {
	userConnectionsMutex.Lock()
	defer userConnectionsMutex.Unlock()

	conn, ok := userConnections[steamId]
	if !ok {
		log.Println("User not connected")
		return
	}

	// Close the connection
	if err := conn.Close(); err != nil {
		log.Println("Error closing connection:", err)
	}

	// Remove the connection from the map
	delete(userConnections, steamId)
}

// Function to periodically send pings
func sendPings(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Error sending ping:", err)
				conn.Close()
				return
			}
		}
	}
}

func SendJSON(conn *websocket.Conn, eventType EventType, message interface{}) {
	if err := conn.WriteJSON(map[string]interface{}{
		"eventType": eventType,
		"message":   message,
	}); err != nil {
		log.Println(err)
	}
}
