package call

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shiv6146/blayzen-sip/internal/config"
	"github.com/shiv6146/blayzen-sip/internal/models"
	"github.com/shiv6146/blayzen-sip/internal/store"
	"github.com/shiv6146/blayzen/pkg/protocol/exotel"
)

// Session represents an active call session
type Session struct {
	CallID       string
	StreamSID    string
	FromURI      string
	ToURI        string
	FromUser     string
	ToUser       string
	Route        *models.Route
	WebSocketURL string

	// SIP transaction
	tx sip.ServerTransaction

	// RTP
	rtpConn    *net.UDPConn
	rtpPort    int
	remoteAddr *net.UDPAddr

	// WebSocket connection to agent
	wsConn *websocket.Conn
	wsMu   sync.Mutex

	// State
	config     *config.Config
	store      *store.PostgresStore
	closed     bool
	closeMu    sync.Mutex
	stopChan   chan struct{}
	chunkCount int
}

// SetTransaction stores the SIP transaction for later use
func (s *Session) SetTransaction(tx sip.ServerTransaction) {
	s.tx = tx
}

// allocateRTPPorts allocates UDP ports for RTP
func (s *Session) allocateRTPPorts() error {
	// Find an available port in the configured range
	for port := s.config.RTPPortMin; port <= s.config.RTPPortMax; port++ {
		addr := &net.UDPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: port,
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			continue // Port in use, try next
		}

		s.rtpConn = conn
		s.rtpPort = port
		s.StreamSID = uuid.New().String()
		s.stopChan = make(chan struct{})

		log.Printf("[Session] Allocated RTP port %d for call %s", port, s.CallID)
		return nil
	}

	return fmt.Errorf("no available RTP ports in range %d-%d", s.config.RTPPortMin, s.config.RTPPortMax)
}

// GenerateSDP generates an SDP answer for the call
func (s *Session) GenerateSDP() string {
	localIP := getLocalIP()

	sdp := fmt.Sprintf(`v=0
o=blayzen-sip %d %d IN IP4 %s
s=blayzen-sip
c=IN IP4 %s
t=0 0
m=audio %d RTP/AVP 0
a=rtpmap:0 PCMU/8000
a=ptime:20
a=sendrecv
`,
		time.Now().Unix(),
		time.Now().Unix(),
		localIP,
		localIP,
		s.rtpPort,
	)

	return sdp
}

// ConnectAgent establishes WebSocket connection to the Blayzen agent
func (s *Session) ConnectAgent(ctx context.Context) error {
	log.Printf("[Session] Connecting to agent: %s", s.WebSocketURL)

	// Connect with timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, s.WebSocketURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to agent: %w", err)
	}

	s.wsConn = conn

	// Send connected message
	connectedMsg := exotel.NewConnectedMessage()
	if err := s.sendWSMessage(connectedMsg); err != nil {
		return fmt.Errorf("failed to send connected message: %w", err)
	}

	// Send start message with call metadata
	startMsg := exotel.NewStartMessage(
		s.StreamSID,
		s.CallID,
		s.Route.AccountID,
		s.FromUser,
		s.ToUser,
	)

	// Add custom data from route
	if s.Route.CustomData != nil {
		startMsg.CustomData = s.Route.CustomData
	}

	if err := s.sendWSMessage(startMsg); err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	log.Printf("[Session] Agent connected for call %s", s.CallID)

	// Start receiving agent responses
	go s.receiveFromAgent()

	return nil
}

// StartMedia starts the media streaming between RTP and WebSocket
func (s *Session) StartMedia() {
	log.Printf("[Session] Starting media for call %s", s.CallID)

	// Update call status
	ctx := context.Background()
	if err := s.store.UpdateCallStatus(ctx, s.CallID, models.CallStatusAnswered); err != nil {
		log.Printf("[Session] Failed to update call status: %v", err)
	}

	// Start RTP receiver
	go s.receiveRTP()
}

// receiveRTP receives RTP packets and forwards to WebSocket
func (s *Session) receiveRTP() {
	buffer := make([]byte, 1500)

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		// Set read deadline
		if err := s.rtpConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			continue
		}

		n, addr, err := s.rtpConn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("[Session] RTP read error: %v", err)
			continue
		}

		// Store remote address for sending RTP back
		if s.remoteAddr == nil {
			s.remoteAddr = addr
			log.Printf("[Session] Remote RTP address: %s", addr.String())
		}

		// Parse RTP header (12 bytes minimum)
		if n < 12 {
			continue
		}

		// Extract audio payload (skip RTP header)
		payload := buffer[12:n]

		// Send to agent via WebSocket
		s.chunkCount++
		msg := exotel.NewMediaMessage(s.StreamSID, payload, s.chunkCount, time.Now().UnixMilli())

		if err := s.sendWSMessage(msg); err != nil {
			log.Printf("[Session] Failed to send media: %v", err)
		}
	}
}

// receiveFromAgent receives messages from the WebSocket agent
func (s *Session) receiveFromAgent() {
	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		_, data, err := s.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[Session] WebSocket read error: %v", err)
			}
			return
		}

		msg, err := exotel.ParseMessage(data)
		if err != nil {
			log.Printf("[Session] Failed to parse agent message: %v", err)
			continue
		}

		switch m := msg.(type) {
		case *exotel.MediaMessage:
			// Decode audio and send via RTP
			audio, err := m.DecodeAudio()
			if err != nil {
				log.Printf("[Session] Failed to decode audio: %v", err)
				continue
			}
			s.sendRTP(audio)

		case *exotel.ClearMessage:
			// Clear audio buffer (for barge-in)
			log.Printf("[Session] Clear buffer requested")

		case *exotel.StopMessage:
			// Agent requested call end
			log.Printf("[Session] Agent requested stop")
			go s.Close()
			return
		}
	}
}

// sendRTP sends audio data via RTP
func (s *Session) sendRTP(payload []byte) {
	if s.remoteAddr == nil || s.rtpConn == nil {
		return
	}

	// Build RTP packet
	// Version: 2, Padding: 0, Extension: 0, CSRC count: 0
	// Marker: 0, Payload type: 0 (PCMU)
	rtpHeader := []byte{
		0x80,                                        // Version 2, no padding, no extension, no CSRC
		0x00,                                        // Marker 0, payload type 0 (PCMU)
		byte(s.chunkCount >> 8), byte(s.chunkCount), // Sequence number
		0x00, 0x00, 0x00, 0x00, // Timestamp (placeholder)
		0x00, 0x00, 0x00, 0x01, // SSRC
	}

	// Combine header and payload
	packet := append(rtpHeader, payload...)

	if _, err := s.rtpConn.WriteToUDP(packet, s.remoteAddr); err != nil {
		log.Printf("[Session] RTP write error: %v", err)
	}
}

// sendWSMessage sends a message to the WebSocket agent
func (s *Session) sendWSMessage(msg interface{}) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	if s.wsConn == nil {
		return fmt.Errorf("websocket not connected")
	}

	return s.wsConn.WriteJSON(msg)
}

// Close closes the session and releases resources
func (s *Session) Close() {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return
	}
	s.closed = true
	s.closeMu.Unlock()

	log.Printf("[Session] Closing session: %s", s.CallID)

	// Signal stop
	close(s.stopChan)

	// Send stop message to agent
	if s.wsConn != nil {
		stopMsg := exotel.NewStopMessage(s.StreamSID)
		_ = s.sendWSMessage(stopMsg)

		// Close WebSocket
		s.wsMu.Lock()
		_ = s.wsConn.Close()
		s.wsConn = nil
		s.wsMu.Unlock()
	}

	// Close RTP connection
	if s.rtpConn != nil {
		_ = s.rtpConn.Close()
		s.rtpConn = nil
	}
}

// getLocalIP returns the local IP address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}
