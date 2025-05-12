package main

import (
	"bytes"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/processSegment", handleProcessSegment)
	http.HandleFunc("/processAck", handleProcessAck)

	port := getEnv("CHANNEL_PORT", "5000")
	log.Println("Mock Channel Level started on port :" + port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))

}

func handleProcessSegment(w http.ResponseWriter, r *http.Request) {
	if rand.N(5) == 0 {
		log.Println("Segment lost")
		return
	}
	forward(r.Body, getEnv("FORWARD_SEGMENT_TO", "http://localhost:4000/transferSegment"))
	w.WriteHeader(http.StatusOK)
}

func handleProcessAck(w http.ResponseWriter, r *http.Request) {
	forward(r.Body, getEnv("FORWARD_ACK_TO", "http://localhost:4000/transferAck"))
	w.WriteHeader(http.StatusOK)
}

func forward(body io.ReadCloser, targetURL string) {
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		log.Printf("read error: %v", err)
		return
	}

	resp, err := http.Post(targetURL, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("forward error to %s: %v", targetURL, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("forwarded to %s -> %d", targetURL, resp.StatusCode)

}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
