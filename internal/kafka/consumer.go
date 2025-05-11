package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"transport/internal/config"

	"github.com/segmentio/kafka-go"
	"transport/internal/model"
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
	log.Println("Kafka consumer started...")

	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("error reading message: %v", err)
			continue
		}

		var segment model.Segment
		if err := json.Unmarshal(m.Value, &segment); err != nil {
			log.Printf("invalid segment: %v", err)
			continue
		}

		go c.forwardToChannel(segment)
	}
}

func (c *Consumer) forwardToChannel(segment model.Segment) {
	data, _ := json.Marshal(segment)
	url := c.cfg.ChannelURL + "/processSegment"
	resp, err := c.client.Post(url, "application/json",
		bytes.NewReader(data))

	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("failed to forward segment to channel: %v", err)
	}
}
