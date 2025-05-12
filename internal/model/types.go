package model

import (
	"bytes"
	"encoding/json"
	"time"
)

type SendMessageRequest struct {
	Sender    string `json:"sender"`
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type Segment struct {
	Sender        string `json:"sender"`
	MessageID     string `json:"messageId"`
	SegmentIndex  int    `json:"segmentIndex"`
	TotalSegments int    `json:"totalSegments"`
	Payload       string `json:"payload"`
}

type Message struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	HasError  bool      `json:"hasError"`
}

type Ack struct {
	MessageID            string `json:"messageId"`
	LastConfirmedSegment int    `json:"lastConfirmedSegment"`
	Final                bool   `json:"final"` // ← добавь это поле
}

type FinalAck struct {
	MessageID string `json:"messageID"`
	Status    string `json:"status"` // "success" или "error"
}

func ToReader(v interface{}) *bytes.Reader {
	data, _ := json.Marshal(v)
	return bytes.NewReader(data)
}
