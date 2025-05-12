package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"transport/internal/config"
	"transport/internal/model"
)

type AckTracker struct {
	mu       sync.Mutex
	messages map[string]*TrackedMessage
	timeout  time.Duration
	cfg      *config.Config
}

type TrackedMessage struct {
	Segments      []model.Segment
	LastConfirmed int
	RetryDone     bool
	TotalSegments int
	Timer         *time.Timer
}

func NewAckTracker(cfg *config.Config) *AckTracker {
	return &AckTracker{
		messages: make(map[string]*TrackedMessage),
		timeout:  cfg.AckTimeout,
		cfg:      cfg,
	}
}

func (a *AckTracker) Track(msgID string, segments []model.Segment) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tracked := &TrackedMessage{
		Segments:      segments,
		LastConfirmed: -1,
		RetryDone:     false,
		TotalSegments: len(segments),
	}

	tracked.Timer = time.AfterFunc(a.timeout, func() {
		a.handleTimeout(msgID)
	})

	a.messages[msgID] = tracked
	log.Printf("[AckTracker] Tracking started for message %s", msgID)
}

func (a *AckTracker) HandleAck(ack model.Ack) (resend []model.Segment, done bool, fail bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tracked, ok := a.messages[ack.MessageID]
	if !ok {
		return nil, false, false
	}

	tracked.Timer.Reset(a.timeout)

	if ack.LastConfirmedSegment == tracked.TotalSegments-1 {
		delete(a.messages, ack.MessageID)
		log.Printf("[AckTracker] All segments confirmed for %s", ack.MessageID)
		return nil, true, false
	}

	if ack.LastConfirmedSegment <= tracked.LastConfirmed {
		if tracked.RetryDone {
			delete(a.messages, ack.MessageID)
			log.Printf("[AckTracker] Retry already done for %s, marking as failed", ack.MessageID)
			return nil, false, true
		}

		log.Printf("[AckTracker] No progress for %s. Sending missing segments once", ack.MessageID)
		tracked.RetryDone = true
	} else {
		log.Printf("[AckTracker] Progress detected for %s", ack.MessageID)
		tracked.LastConfirmed = ack.LastConfirmedSegment
	}

	for i := ack.LastConfirmedSegment + 1; i < tracked.TotalSegments; i++ {
		resend = append(resend, tracked.Segments[i])
	}

	return resend, false, false
}

func (a *AckTracker) handleTimeout(msgID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	_, exists := a.messages[msgID]
	if !exists {
		return
	}

	log.Printf("[AckTracker] Timeout exceeded for message %s. Sending error ACK", msgID)
	delete(a.messages, msgID)

	err := SendFinalAck(model.Ack{
		MessageID:            msgID,
		LastConfirmedSegment: -1,
	}, true, a.cfg)

	if err != nil {
		log.Printf("[AckTracker] Failed to send error ACK for %s: %v", msgID, err)
	}
}

func SendFinalAck(ack model.Ack, failed bool, cfg *config.Config) error {
	status := "success"
	if failed {
		status = "error"
	}

	payload := model.FinalAck{
		MessageID: ack.MessageID,
		Status:    status,
	}

	body, _ := json.Marshal(payload)
	url := cfg.AppEarthURL + "/receiveAck"

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("final ACK not accepted, status code: %d", resp.StatusCode)
	}

	log.Printf("[ACK] Final ACK sent to app-earth: %s (status=%s)", ack.MessageID, status)
	return nil
}
