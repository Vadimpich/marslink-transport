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
}

func Load() *Config {
	segmentSize, err := strconv.Atoi(getEnv("SEGMENT_SIZE", "120"))
	if err != nil {
		log.Fatalf("invalid SEGMENT_SIZE: %v", err)
	}

	timeout, err := time.ParseDuration(getEnv("TIMEOUT_DURATION", "10s"))
	if err != nil {
		log.Fatalf("invalid TIMEOUT_DURATION: %v", err)
	}

	interval, err := time.ParseDuration(getEnv("CHECK_INTERVAL", "5s"))
	if err != nil {
		log.Fatalf("invalid CHECK_INTERVAL: %v", err)
	}

	return &Config{
		AppMarsURL:    os.Getenv("APP_MARS_URL"),
		AppEarthURL:   os.Getenv("APP_EARTH_URL"),
		ChannelURL:    os.Getenv("CHANNEL_URL"),
		KafkaBrokers:  os.Getenv("KAFKA_BROKERS"),
		SegmentSize:   segmentSize,
		Timeout:       timeout,
		CheckInterval: interval,
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
