package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
	"transport/internal/config"
	"transport/internal/model"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	client *http.Client
	cfg    *config.Config
}

func NewConsumer(brokers []string, topic string, groupID string, cfg *config.Config) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		client: &http.Client{Timeout: 5 * time.Second},
		cfg:    cfg,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("[Kafka] Consumer started...")

	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("[Kafka] Error reading message: %v", err)
			continue
		}

		var segment model.Segment
		if err := json.Unmarshal(m.Value, &segment); err != nil {
			log.Printf("[Kafka] Invalid segment JSON: %v", err)
			continue
		}

		log.Printf("[Kafka] Consumed segment %d/%d from message %s",
			segment.SegmentIndex, segment.TotalSegments, segment.MessageID)

		go c.forwardToChannel(segment)
	}
}

func (c *Consumer) forwardToChannel(segment model.Segment) {
	data, _ := json.Marshal(segment)
	url := c.cfg.ChannelURL + "/processSegment"

	log.Printf("[Channel] Forwarding segment %d/%d of message %s to %s",
		segment.SegmentIndex, segment.TotalSegments, segment.MessageID, url)

	resp, err := c.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("[Channel] Failed to forward segment: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Channel] Bad response: status=%d, body=%s", resp.StatusCode, string(body))
	} else {
		log.Printf("[Channel] Segment %d of message %s successfully forwarded", segment.SegmentIndex, segment.MessageID)
	}
}
