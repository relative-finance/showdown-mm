package ws

import (
	"log"
	"mmf/internal/model"
)

func SendMatchFoundToPlayers(matchId string, matchTickets []model.Ticket) bool {
	mess := GenerateMatchFoundResponse(matchTickets, matchId)

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

	if err := conn.WriteJSON(map[string]interface{}{
		"eventType": event,
		"message":   message,
	}); err != nil {
		log.Println(err)
	}
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
