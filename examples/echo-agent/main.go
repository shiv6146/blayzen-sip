// Package main provides a simple echo agent for testing blayzen-sip
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/shiv6146/blayzen/pkg/protocol/exotel"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/health", handleHealth)

	server := &http.Server{
		Addr: ":" + port,
	}

	go func() {
		log.Printf("Echo agent listening on :%s", port)
		log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
		log.Printf("Error writing health response: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	log.Println("New WebSocket connection")

	var wsMu sync.Mutex
	var callActive bool

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		msg, err := exotel.ParseMessage(data)
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		switch m := msg.(type) {
		case *exotel.ConnectedMessage:
			log.Println("Received: connected")

		case *exotel.StartMessage:
			log.Printf("Received: start - Call from %s to %s", m.From, m.To)
			callActive = true

		case *exotel.MediaMessage:
			if !callActive {
				continue
			}

			// Echo the audio back (simple echo agent)
			// Decode and re-encode to create echo response
			audio, err := m.DecodeAudio()
			if err != nil {
				log.Printf("Failed to decode audio: %v", err)
				continue
			}

			// Create outbound media message (simpler format)
			response := map[string]interface{}{
				"event":     exotel.EventMedia,
				"media":     m.Media.Payload, // Echo back same audio
				"timestamp": m.Media.Timestamp,
				"chunk":     m.Media.Chunk,
			}

			wsMu.Lock()
			responseBytes, _ := json.Marshal(response)
			if err := conn.WriteMessage(websocket.TextMessage, responseBytes); err != nil {
				log.Printf("Failed to send echo: %v", err)
			}
			wsMu.Unlock()

			// Log every 100th chunk to reduce noise
			if m.Media.Chunk%100 == 0 {
				log.Printf("Echo: chunk %d (%d bytes)", m.Media.Chunk, len(audio))
			}

		case *exotel.StopMessage:
			log.Println("Received: stop")
			callActive = false
			return

		case *exotel.DTMFMessage:
			log.Printf("Received DTMF: %s", m.DTMF)
		}
	}
}

