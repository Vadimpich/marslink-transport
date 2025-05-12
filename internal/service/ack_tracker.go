package service

import (
	"log"
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
	RetryCount    int
	TotalSegments int
	SentAt        time.Time
	timer         *time.Timer
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
		RetryCount:    0,
		TotalSegments: len(segments),
		SentAt:        time.Now(),
	}
	tracked.timer = time.AfterFunc(a.timeout, func() {
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

	// Обработка финального ACK от Марса
	if ack.Final {
		log.Printf("[AckTracker] Final ACK received for message %s. Ending tracking.", ack.MessageID)
		delete(a.messages, ack.MessageID)
		return nil, true, true
	}

	// Сброс таймера
	tracked.timer.Stop()
	tracked.timer.Reset(a.timeout)

	// Успешно доставлены все сегменты
	if ack.LastConfirmedSegment == tracked.TotalSegments-1 {
		log.Printf("[AckTracker] All segments confirmed for message %s", ack.MessageID)
		delete(a.messages, ack.MessageID)
		return nil, true, false
	}

	// Нет прогресса
	if ack.LastConfirmedSegment <= tracked.LastConfirmed {
		tracked.RetryCount++
		log.Printf("[AckTracker] No progress for %s. Retry #%d", ack.MessageID, tracked.RetryCount)

		if tracked.RetryCount >= 1 {
			log.Printf("[AckTracker] No progress after retry for %s. Sending Final ACK", ack.MessageID)
			delete(a.messages, ack.MessageID)

			go func() {
				err := SendFinalAck(model.Ack{
					MessageID:            ack.MessageID,
					LastConfirmedSegment: ack.LastConfirmedSegment,
					Final:                true,
				}, true, a.cfg)
				if err != nil {
					log.Printf("[AckTracker] Failed to send final ACK for %s: %v", ack.MessageID, err)
				}
			}()

			return nil, false, true
		}
	} else {
		// Есть прогресс
		tracked.LastConfirmed = ack.LastConfirmedSegment
		tracked.RetryCount = 0
		log.Printf("[AckTracker] Progress detected for %s. Resetting retry count.", ack.MessageID)
	}

	// Повторная отправка недостающих сегментов
	for i := ack.LastConfirmedSegment + 1; i < tracked.TotalSegments; i++ {
		resend = append(resend, tracked.Segments[i])
	}

	return resend, false, false
}

func (a *AckTracker) handleTimeout(messageID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tracked, exists := a.messages[messageID]
	if !exists {
		return
	}

	log.Printf("[AckTracker] Timeout exceeded for message %s. Sending Final=true ACK", messageID)
	delete(a.messages, messageID)

	go func() {
		err := SendFinalAck(model.Ack{
			MessageID:            messageID,
			LastConfirmedSegment: tracked.LastConfirmed,
			Final:                true,
		}, true, a.cfg)

		if err != nil {
			log.Printf("[AckTracker] Failed to send final timeout ACK for %s: %v", messageID, err)
		}
	}()
}
