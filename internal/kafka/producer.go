package kafka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"transport/internal/model"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) SendSegment(segment model.Segment) error {
	data, err := json.Marshal(segment)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(segment.MessageID),
		Value: data,
	}

	return p.writer.WriteMessages(context.Background(), msg)
}
