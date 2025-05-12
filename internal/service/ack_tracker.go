package service

import (
	"sync"
	"time"
	"transport/internal/model"
)

type AckTracker struct {
	mu       sync.Mutex
	messages map[string]*TrackedMessage
}

type TrackedMessage struct {
	Segments      []model.Segment
	LastConfirmed int
	RetryCount    int
	TotalSegments int
	SentAt        time.Time
}

func NewAckTracker() *AckTracker {
	return &AckTracker{
		messages: make(map[string]*TrackedMessage),
	}
}

func (a *AckTracker) Track(msgID string, segments []model.Segment) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.messages[msgID] = &TrackedMessage{
		Segments:      segments,
		LastConfirmed: -1,
		RetryCount:    0,
		TotalSegments: len(segments),
		SentAt:        time.Now(),
	}
}

func (a *AckTracker) HandleAck(ack model.Ack) (resend []model.Segment, done bool, fail bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tracked, ok := a.messages[ack.MessageID]
	if !ok {
		return nil, false, false
	}

	if ack.LastConfirmedSegment == tracked.TotalSegments-1 {
		delete(a.messages, ack.MessageID)
		return nil, true, false
	}

	if ack.LastConfirmedSegment == tracked.LastConfirmed {
		tracked.RetryCount++
		if tracked.RetryCount >= 2 {
			delete(a.messages, ack.MessageID)
			return nil, false, true
		}
	} else {
		tracked.RetryCount = 0
	}

	tracked.LastConfirmed = ack.LastConfirmedSegment

	for i := ack.LastConfirmedSegment + 1; i < tracked.TotalSegments; i++ {
		resend = append(resend, tracked.Segments[i])
	}

	return resend, false, false
}
