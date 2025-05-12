package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppMarsURL    string
	AppEarthURL   string
	ChannelURL    string
	KafkaBrokers  string
	SegmentSize   int
	Timeout       time.Duration
	CheckInterval time.Duration
	TransportPort string
}

func Load() *Config {
	segmentSize, err := strconv.Atoi(getEnv("SEGMENT_SIZE", "120"))
	if err != nil {
		log.Fatalf("[Config] Invalid SEGMENT_SIZE: %v", err)
	}

	timeout, err := time.ParseDuration(getEnv("TIMEOUT_DURATION", "10s"))
	if err != nil {
		log.Fatalf("[Config] Invalid TIMEOUT_DURATION: %v", err)
	}

	interval, err := time.ParseDuration(getEnv("CHECK_INTERVAL", "5s"))
	if err != nil {
		log.Fatalf("[Config] Invalid CHECK_INTERVAL: %v", err)
	}

	cfg := &Config{
		AppMarsURL:    os.Getenv("APP_MARS_URL"),
		AppEarthURL:   os.Getenv("APP_EARTH_URL"),
		ChannelURL:    os.Getenv("CHANNEL_URL"),
		KafkaBrokers:  os.Getenv("KAFKA_BROKERS"),
		SegmentSize:   segmentSize,
		Timeout:       timeout,
		CheckInterval: interval,
		TransportPort: getEnv("TRANSPORT_PORT", "4000"),
	}

	log.Println("[Config] Loaded configuration:")
	log.Printf("  APP_MARS_URL:    %s", cfg.AppMarsURL)
	log.Printf("  APP_EARTH_URL:   %s", cfg.AppEarthURL)
	log.Printf("  CHANNEL_URL:     %s", cfg.ChannelURL)
	log.Printf("  KAFKA_BROKERS:   %s", cfg.KafkaBrokers)
	log.Printf("  SEGMENT_SIZE:    %d", cfg.SegmentSize)
	log.Printf("  TIMEOUT:         %v", cfg.Timeout)
	log.Printf("  CHECK_INTERVAL:  %v", cfg.CheckInterval)
	log.Printf("  TRANSPORT_PORT:  %s", cfg.TransportPort)

	return cfg
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Printf("[Config] %s not set, using default: %s", key, fallback)
		return fallback
	}
	return val
}
