package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"transport/internal/config"
	"transport/internal/kafka"
	"transport/internal/model"
	"transport/internal/service"
)

type TransportHandler struct {
	Producer    *kafka.Producer
	Reassembler *service.Reassembler
	Config      *config.Config
	AckTracker  *service.AckTracker
}

func NewTransportHandler(prod *kafka.Producer, reas *service.Reassembler, cfg *config.Config, tracker *service.AckTracker) *TransportHandler {
	return &TransportHandler{
		Producer:    prod,
		Reassembler: reas,
		Config:      cfg,
		AckTracker:  tracker,
	}
}

func (h *TransportHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	log.Println("[HTTP] /sendMessage called")

	var req model.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Invalid request body: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Received message %s from %s", req.MessageID, req.Sender)

	segments := service.SplitMessageToSegments(req, h.Config.SegmentSize)

	for _, segment := range segments {
		err := h.Producer.SendSegment(segment)
		if err != nil {
			log.Printf("[ERROR] Failed to send segment %d: %v", segment.SegmentIndex, err)
			http.Error(w, "failed to send", http.StatusInternalServerError)
			return
		}
		log.Printf("[INFO] Segment %d/%d sent for message %s", segment.SegmentIndex, segment.TotalSegments, segment.MessageID)
	}

	h.AckTracker.Track(req.MessageID, segments)
	log.Printf("[INFO] Tracking started for message %s", req.MessageID)

	w.WriteHeader(http.StatusOK)
}

func (h *TransportHandler) TransferSegment(w http.ResponseWriter, r *http.Request) {
	log.Println("[HTTP] /transferSegment called")

	var seg model.Segment
	if err := json.NewDecoder(r.Body).Decode(&seg); err != nil {
		log.Printf("[ERROR] Failed to unmarshal segment: %v", err)
		http.Error(w, "invalid segment", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Received segment %d/%d for message %s", seg.SegmentIndex, seg.TotalSegments, seg.MessageID)

	h.Reassembler.AddSegment(seg)

	w.WriteHeader(http.StatusOK)
}

func (h *TransportHandler) TransferAck(w http.ResponseWriter, r *http.Request) {
	log.Println("[HTTP] /transferAck called")

	var ack model.Ack
	if err := json.NewDecoder(r.Body).Decode(&ack); err != nil {
		log.Printf("[ERROR] Invalid ACK: %v", err)
		http.Error(w, "invalid ack", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Received ACK for message %s, lastConfirmed=%d", ack.MessageID, ack.LastConfirmedSegment)

	resend, done, fail := h.AckTracker.HandleAck(ack)

	if done {
		log.Printf("[INFO] All segments confirmed for %s. Forwarding final ACK to app-earth.", ack.MessageID)
		go service.SendFinalAck(ack, false, h.Config)
		w.WriteHeader(http.StatusOK)
		return
	}

	if fail {
		log.Printf("[WARN] Max retries exceeded for message %s. Sending error ACK.", ack.MessageID)
		go service.SendFinalAck(ack, true, h.Config)
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, seg := range resend {
		log.Printf("[INFO] Resending segment %d of message %s", seg.SegmentIndex, seg.MessageID)
		_ = h.Producer.SendSegment(seg)
	}

	w.WriteHeader(http.StatusOK)
}
