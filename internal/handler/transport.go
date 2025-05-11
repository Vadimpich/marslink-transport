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
}

func NewTransportHandler(prod *kafka.Producer, reas *service.Reassembler, cfg *config.Config) *TransportHandler {
	return &TransportHandler{
		Producer:    prod,
		Reassembler: reas,
		Config:      cfg,
	}
}

func (h *TransportHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req model.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	segments := service.SplitMessageToSegments(req, h.Config.SegmentSize)

	for _, segment := range segments {
		err := h.Producer.SendSegment(segment)
		if err != nil {
			log.Printf("failed to send segment: %v", err)
			http.Error(w, "failed to send", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TransportHandler) TransferSegment(w http.ResponseWriter, r *http.Request) {
	var seg model.Segment
	if err := json.NewDecoder(r.Body).Decode(&seg); err != nil {
		http.Error(w, "invalid segment", http.StatusBadRequest)
		return
	}

	h.Reassembler.AddSegment(seg)

	w.WriteHeader(http.StatusOK)
}

func (h *TransportHandler) TransferAck(w http.ResponseWriter, r *http.Request) {
	var ack model.Ack
	if err := json.NewDecoder(r.Body).Decode(&ack); err != nil {
		http.Error(w, "invalid ACK", http.StatusBadRequest)
		return
	}

	go service.ForwardAckToAppEarth(ack, h.Config)

	w.WriteHeader(http.StatusOK)
}
