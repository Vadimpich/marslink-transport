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
	Segments      map[int]string
	ReceivedAt    time.Time
	TotalSegments int
	Sender        string
}

type Reassembler struct {
	mu     sync.Mutex
	buffer map[string]*bufferedMessage
	cfg    *config.Config
}

func NewReassembler(cfg *config.Config) *Reassembler {
	return &Reassembler{
		buffer: make(map[string]*bufferedMessage),
		cfg:    cfg,
	}
}

func (r *Reassembler) AddSegment(seg model.Segment) {
	r.mu.Lock()
	defer r.mu.Unlock()

	buf, ok := r.buffer[seg.MessageID]
	if !ok {
		buf = &bufferedMessage{
			Segments:      make(map[int]string),
			TotalSegments: seg.TotalSegments,
			ReceivedAt:    time.Now(),
			Sender:        seg.Sender,
		}
		r.buffer[seg.MessageID] = buf
	}

	buf.Segments[seg.SegmentIndex] = seg.Payload
	buf.ReceivedAt = time.Now()
}

func (r *Reassembler) CheckTimeoutsAndAssemble() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for messageID, buf := range r.buffer {
		if len(buf.Segments) == buf.TotalSegments || now.Sub(buf.ReceivedAt) > r.cfg.Timeout {
			content := make([]string, buf.TotalSegments)
			hasError := false

			for i := 0; i < buf.TotalSegments; i++ {
				payload, ok := buf.Segments[i]
				if !ok {
					hasError = true
					payload = "[MISSING]"
				}
				content[i] = payload
			}

			final := model.Message{
				Sender:    buf.Sender,
				Content:   joinSegments(content),
				HasError:  hasError,
				Timestamp: time.Now(),
			}

			go sendToAppLevel(final, r.cfg)
			delete(r.buffer, messageID)
		}
	}
}

func joinSegments(parts []string) string {
	var buf bytes.Buffer
	for _, p := range parts {
		buf.WriteString(p)
	}
	return buf.String()
}

func sendToAppLevel(msg model.Message, cfg *config.Config) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}

	url := cfg.AppMarsURL + "/receiveMessage"
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("failed to send to app level: %v", err)
		return
	}
}
