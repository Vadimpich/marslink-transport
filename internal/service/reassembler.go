package service

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
	"transport/internal/config"
	"transport/internal/model"
)

type bufferedMessage struct {
	Sender         string
	Segments       map[int]string
	ReceivedAt     time.Time
	TotalSegments  int
	LastConfirmed  int
	FailedAttempts int
}

type Reassembler struct {
	mu      sync.Mutex
	buffer  map[string]*bufferedMessage
	timeout time.Duration
	cfg     *config.Config
}

func NewReassembler(cfg *config.Config) *Reassembler {
	return &Reassembler{
		buffer:  make(map[string]*bufferedMessage),
		timeout: cfg.Timeout,
		cfg:     cfg,
	}
}

func (r *Reassembler) AddSegment(seg model.Segment) {
	if seg.MessageID == "" {
		log.Println("[ERROR] Received segment with empty messageId!")
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	buf, ok := r.buffer[seg.MessageID]
	if !ok {
		buf = &bufferedMessage{
			Sender:         seg.Sender,
			Segments:       make(map[int]string),
			TotalSegments:  seg.TotalSegments,
			ReceivedAt:     time.Now(),
			LastConfirmed:  -1,
			FailedAttempts: 0,
		}
		r.buffer[seg.MessageID] = buf
		log.Printf("[Reassembler] New message %s from %s. Total segments: %d", seg.MessageID, seg.Sender, seg.TotalSegments)
	}

	buf.Segments[seg.SegmentIndex] = seg.Payload
	buf.ReceivedAt = time.Now()
	log.Printf("[Reassembler] Stored segment %d of message %s", seg.SegmentIndex, seg.MessageID)
}

func (r *Reassembler) CheckTimeoutsAndAssemble() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for messageID, buf := range r.buffer {
		if messageID == "" {
			continue
		}

		currentConfirmed := calculateLastConfirmedIndex(buf.Segments, buf.TotalSegments)

		if currentConfirmed == buf.TotalSegments-1 {
			log.Printf("[Reassembler] Message %s fully received. Assembling...", messageID)

			msg := model.Message{
				Sender:    buf.Sender,
				Content:   joinSegments(buf.Segments, buf.TotalSegments),
				Timestamp: time.Now(),
				HasError:  false,
			}
			go sendMessageToAppMars(msg, r.cfg)

			go sendAckToChannel(model.Ack{
				MessageID:            messageID,
				LastConfirmedSegment: currentConfirmed,
				Final:                true,
			}, r.cfg)

			delete(r.buffer, messageID)
			continue
		}

		if now.Sub(buf.ReceivedAt) > r.timeout {
			if currentConfirmed > buf.LastConfirmed {
				log.Printf("[Reassembler] Progress for message %s. Resetting fail counter.", messageID)
				buf.FailedAttempts = 0
				buf.LastConfirmed = currentConfirmed
			} else {
				buf.FailedAttempts++
				log.Printf("[Reassembler] No progress for %s. Attempt %d", messageID, buf.FailedAttempts)
			}

			if buf.FailedAttempts >= 2 {
				log.Printf("[Reassembler] Message %s failed after %d timeouts. Sending final ACK", messageID, buf.FailedAttempts)

				go sendAckToChannel(model.Ack{
					MessageID:            messageID,
					LastConfirmedSegment: buf.LastConfirmed,
					Final:                true,
				}, r.cfg)

				delete(r.buffer, messageID)
			} else {
				log.Printf("[Reassembler] Message %s timed out. Sending intermediate ACK", messageID)

				go sendAckToChannel(model.Ack{
					MessageID:            messageID,
					LastConfirmedSegment: buf.LastConfirmed,
					Final:                false,
				}, r.cfg)

				buf.ReceivedAt = now
			}
		}
	}
}

func calculateLastConfirmedIndex(segments map[int]string, total int) int {
	for i := 0; i < total; i++ {
		if _, ok := segments[i]; !ok {
			return i - 1
		}
	}
	return total - 1
}

func joinSegments(parts map[int]string, total int) string {
	var buf bytes.Buffer
	for i := 0; i < total; i++ {
		buf.WriteString(parts[i])
	}
	return buf.String()
}

func sendMessageToAppMars(msg model.Message, cfg *config.Config) {
	data, _ := json.Marshal(msg)
	url := cfg.AppMarsURL + "/receiveMessage"
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("[Reassembler] Failed to send message %s to app-mars: %v", msg.Sender, err)
		return
	}
	log.Printf("[Reassembler] Message %s successfully sent to app-mars", msg.Content)
}

func sendAckToChannel(ack model.Ack, cfg *config.Config) {
	data, _ := json.Marshal(ack)
	url := cfg.ChannelURL + "/processAck"

	const maxRetries = 3
	const retryDelay = 3 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Printf("[Reassembler] ACK sent to channel: messageId=%s, lastConfirmed=%d, final=%v", ack.MessageID, ack.LastConfirmedSegment, ack.Final)
			return
		}
		log.Printf("[Reassembler] Failed to send ACK attempt #%d for message %s: %v", attempt, ack.MessageID, err)
		time.Sleep(retryDelay)
	}

	log.Printf("[Reassembler] ACK send aborted after %d attempts for message %s", maxRetries, ack.MessageID)
}
