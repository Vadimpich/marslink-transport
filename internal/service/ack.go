package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"transport/internal/config"
	"transport/internal/model"
)

func SendFinalAck(ack model.Ack, failed bool, cfg *config.Config) error {
	status := "success"
	if failed {
		status = "error"
	}

	final := model.FinalAck{
		MessageID: ack.MessageID,
		Status:    status,
	}

	data, err := json.Marshal(final)
	if err != nil {
		return fmt.Errorf("failed to marshal final ACK: %w", err)
	}

	url := cfg.AppEarthURL + "/receiveAck"
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send final ACK to app-earth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("final ACK not accepted, status code: %d", resp.StatusCode)
	}

	return nil
}
