package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"strings"
	"time"
	"transport/internal/config"
	"transport/internal/handler"
	"transport/internal/kafka"
	"transport/internal/service"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}
}

func main() {
	cfg := config.Load()

	reassembler := service.NewReassembler(cfg)
	producer := kafka.NewProducer(strings.Split(cfg.KafkaBrokers, ","), "segment-topic")

	h := handler.NewTransportHandler(producer, reassembler, cfg)

	r := mux.NewRouter()
	r.HandleFunc("/sendMessage", h.SendMessage).Methods("POST")
	r.HandleFunc("/transferSegment", h.TransferSegment).Methods("POST")
	r.HandleFunc("/transferAck", h.TransferAck).Methods("POST")

	ctx := context.Background()

	consumer := kafka.NewConsumer(
		strings.Split(cfg.KafkaBrokers, ","),
		"segment-topic",
		"transport-group",
		cfg,
	)
	go consumer.Start(ctx)

	go func() {
		ticker := time.NewTicker(cfg.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				reassembler.CheckTimeoutsAndAssemble()
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Println("Transport service started on :4000")
	log.Fatal(http.ListenAndServe(":4000", r))
}
