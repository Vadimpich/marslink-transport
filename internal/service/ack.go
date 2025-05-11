package service

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"transport/internal/config"
	"transport/internal/model"
)

func ForwardAckToAppEarth(ack model.Ack, cfg *config.Config) {
	url := cfg.AppEarthURL + "/receiveAck"

	data, err := json.Marshal(ack)
	if err != nil {
		log.Printf("failed to marshal ACK: %v", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("failed to forward ACK to app-earth: %v", err)
	}
}
