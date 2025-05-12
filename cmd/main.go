package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"transport/internal/config"
	"transport/internal/handler"
	"transport/internal/kafka"
	"transport/internal/service"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("[Init] No .env file found, using system env")
	} else {
		log.Println("[Init] .env loaded")
	}
}

func main() {
	cfg := config.Load()

	// Инициализация компонентов
	tracker := service.NewAckTracker(cfg)
	reassembler := service.NewReassembler(cfg)
	producer := kafka.NewProducer(strings.Split(cfg.KafkaBrokers, ","), "segment-topic")
	handler := handler.NewTransportHandler(producer, reassembler, cfg, tracker)

	// Маршруты
	router := mux.NewRouter()
	router.HandleFunc("/sendMessage", handler.SendMessage).Methods("POST")
	router.HandleFunc("/transferSegment", handler.TransferSegment).Methods("POST")
	router.HandleFunc("/transferAck", handler.TransferAck).Methods("POST")

	// Kafka consumer
	ctx := context.Background()
	consumer := kafka.NewConsumer(strings.Split(cfg.KafkaBrokers, ","), "segment-topic", "transport-group", cfg)

	go func() {
		log.Println("[Kafka] Starting consumer...")
		consumer.Start(ctx)
	}()

	// Периодическая сборка сообщений
	go func() {
		ticker := time.NewTicker(cfg.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				reassembler.CheckTimeoutsAndAssemble()
			case <-ctx.Done():
				log.Println("[System] Shutting down ticker")
				return
			}
		}
	}()

	// Запуск HTTP сервера
	addr := ":" + cfg.TransportPort
	log.Printf("[HTTP] Transport service listening on %s (public)", addr)
	if err := http.ListenAndServe("0.0.0.0"+addr, router); err != nil {
		log.Fatalf("[FATAL] HTTP server error: %v", err)
	}
}
