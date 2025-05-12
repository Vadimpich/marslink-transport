package service

import (
	"log"
	"transport/internal/model"
)

func SplitMessageToSegments(req model.SendMessageRequest, maxSegmentSize int) []model.Segment {
	msg := []byte(req.Message)
	total := (len(msg) + maxSegmentSize - 1) / maxSegmentSize
	segments := make([]model.Segment, 0, total)

	log.Printf("[Segmenter] Splitting message %s from %s into %d segments (size=%d)", req.MessageID, req.Sender, total, maxSegmentSize)

	for i := 0; i < total; i++ {
		start := i * maxSegmentSize
		end := start + maxSegmentSize
		if end > len(msg) {
			end = len(msg)
		}

		payload := msg[start:end]

		segment := model.Segment{
			Sender:        req.Sender,
			MessageID:     req.MessageID,
			SegmentIndex:  i,
			TotalSegments: total,
			Payload:       string(payload),
		}

		log.Printf("[Segmenter] Segment %d: %d bytes", i, len(payload))
		segments = append(segments, segment)
	}

	return segments
}
