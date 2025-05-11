package service

import (
	"transport/internal/model"
)

func SplitMessageToSegments(req model.SendMessageRequest, maxSegmentSize int) []model.Segment {
	msg := []byte(req.Message)
	total := (len(msg) + maxSegmentSize - 1) / maxSegmentSize
	segments := make([]model.Segment, 0, total)

	for i := 0; i < total; i++ {
		start := i * maxSegmentSize
		end := start + maxSegmentSize
		if end > len(msg) {
			end = len(msg)
		}

		segment := model.Segment{
			Sender:        req.Sender,
			MessageID:     req.MessageID,
			SegmentIndex:  i,
			TotalSegments: total,
			Payload:       string(msg[start:end]),
		}
		segments = append(segments, segment)
	}

	return segments
}
