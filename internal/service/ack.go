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

func SendFinalAck(ack model.Ack, failed bool, cfg *config.Config) {
	status := "success"
	if failed {
		status = "error"
	}

	final := model.FinalAck{
		MessageID: ack.MessageID,
		Status:    status,
	}

	data, _ := json.Marshal(final)
	url := cfg.AppEarthURL + "/receiveAck"

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("[ACK] Failed to send final ACK: %v", err)
		return
	}

	log.Printf("[ACK] Final ACK sent to app-earth: %s (status=%s)", ack.MessageID, status)
}
