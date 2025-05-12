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
	Sender        string
	Segments      map[int]string
	ReceivedAt    time.Time
	TotalSegments int
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
	r.mu.Lock()
	defer r.mu.Unlock()

	buf, ok := r.buffer[seg.MessageID]
	if !ok {
		buf = &bufferedMessage{
			Sender:        seg.Sender,
			Segments:      make(map[int]string),
			TotalSegments: seg.TotalSegments,
			ReceivedAt:    time.Now(),
		}
		r.buffer[seg.MessageID] = buf
		log.Printf("[Reassembler] New message %s from %s. Total segments: %d",
			seg.MessageID, seg.Sender, seg.TotalSegments)
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
		lastConfirmed := calculateLastConfirmedIndex(buf.Segments, buf.TotalSegments)

		if lastConfirmed == buf.TotalSegments-1 {
			log.Printf("[Reassembler] Message %s fully received. Assembling...", messageID)

			msg := model.Message{
				Sender:    buf.Sender,
				Content:   joinSegments(buf.Segments, buf.TotalSegments),
				Timestamp: time.Now(),
				HasError:  false,
			}
			go sendMessageToAppMars(msg, r.cfg)

			ack := model.Ack{
				MessageID:            messageID,
				LastConfirmedSegment: lastConfirmed,
			}
			go sendAckToChannel(ack, r.cfg)

			delete(r.buffer, messageID)
			continue
		}

		if now.Sub(buf.ReceivedAt) > r.timeout {
			log.Printf("[Reassembler] Message %s timed out. Partial segments received. Last confirmed: %d",
				messageID, lastConfirmed)

			ack := model.Ack{
				MessageID:            messageID,
				LastConfirmedSegment: lastConfirmed,
			}
			go sendAckToChannel(ack, r.cfg)

			delete(r.buffer, messageID)
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
	log.Printf("[Reassembler] Message from %s successfully sent to app-mars", msg.Content)
}

func sendAckToChannel(ack model.Ack, cfg *config.Config) {
	data, _ := json.Marshal(ack)
	url := cfg.ChannelURL + "/processAck"
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("[Reassembler] Failed to send ACK for message %s: %v", ack.MessageID, err)
		return
	}
	log.Printf("[Reassembler] ACK sent to channel: messageId=%s, lastConfirmed=%d", ack.MessageID, ack.LastConfirmedSegment)
}
