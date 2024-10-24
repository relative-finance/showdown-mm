package external

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"log"
	"mmf/config"
	"net/http"
)

type Event struct {
	Type  string `json:"type"`
	RefId string `json:"ref_id"`
}

type Notification struct {
	Content  string      `json:"content"`
	Metadata interface{} `json:"metadata"`
	UserIds  []string    `json:"user_ids"`
	Type     string      `json:"type"`
	Subtype  string      `json:"sub_type"`
	Category string      `json:"category"`
	RefId    string      `json:"ref_id"`
	Service  string      `json:"service"`
}

func SendNotification(notification Notification) {
	jsonData, err := json.Marshal(notification)
	if err != nil {
		log.Println("Error marshalling data: ", err)
	}

	url := config.GlobalConfig.Notifications.URL + "/v1/notification"

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error calling %s: %s", url, err.Error())
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Error calling %s: %d", url, resp.StatusCode)
	}

	log.Printf("Sent event on... %s", url)
}
