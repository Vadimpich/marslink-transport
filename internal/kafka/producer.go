package kafka

import (
	"context"
	"encoding/json"
	"log"

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
		log.Printf("[Kafka] Failed to marshal segment %d of message %s: %v",
			segment.SegmentIndex, segment.MessageID, err)
		return err
	}

	msg := kafka.Message{
		Key:   []byte(segment.MessageID),
		Value: data,
	}

	err = p.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		log.Printf("[Kafka] Failed to send segment %d of message %s: %v",
			segment.SegmentIndex, segment.MessageID, err)
		return err
	}

	log.Printf("[Kafka] Segment %d/%d of message %s successfully sent",
		segment.SegmentIndex, segment.TotalSegments, segment.MessageID)

	return nil
}
