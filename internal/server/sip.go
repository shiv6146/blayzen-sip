// Package server provides the SIP server implementation
package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/google/uuid"
	"github.com/shiv6146/blayzen-sip/internal/call"
	"github.com/shiv6146/blayzen-sip/internal/config"
	"github.com/shiv6146/blayzen-sip/internal/routing"
	"github.com/shiv6146/blayzen-sip/internal/store"
)

// SIPServer handles SIP signaling
type SIPServer struct {
	config  *config.Config
	store   *store.PostgresStore
	cache   *store.Cache
	router  *routing.Router
	ua      *sipgo.UserAgent
	server  *sipgo.Server
	calls   *call.Manager
	mu      sync.RWMutex
	running bool
}

// NewSIPServer creates a new SIP server
func NewSIPServer(cfg *config.Config, store *store.PostgresStore, cache *store.Cache) (*SIPServer, error) {
	// Create user agent
	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent("blayzen-sip/1.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user agent: %w", err)
	}

	// Create SIP server
	server, err := sipgo.NewServer(ua)
	if err != nil {
		return nil, fmt.Errorf("failed to create SIP server: %w", err)
	}

	// Create routing engine
	router := routing.NewRouter(store, cache, cfg.DefaultWebSocketURL)

	// Create call manager
	callMgr := call.NewManager(cfg, store, cache)

	s := &SIPServer{
		config: cfg,
		store:  store,
		cache:  cache,
		router: router,
		ua:     ua,
		server: server,
		calls:  callMgr,
	}

	// Register SIP handlers
	s.registerHandlers()

	return s, nil
}

// registerHandlers sets up SIP message handlers
func (s *SIPServer) registerHandlers() {
	// Handle INVITE (incoming calls)
	s.server.OnInvite(s.handleInvite)

	// Handle ACK
	s.server.OnAck(s.handleAck)

	// Handle BYE (call termination)
	s.server.OnBye(s.handleBye)

	// Handle CANCEL
	s.server.OnCancel(s.handleCancel)

	// Handle OPTIONS (keep-alive / health check)
	s.server.OnOptions(s.handleOptions)
}

// handleInvite processes incoming INVITE requests
func (s *SIPServer) handleInvite(req *sip.Request, tx sip.ServerTransaction) {
	ctx := context.Background()
	callID := req.CallID().Value()

	log.Printf("[SIP] INVITE received: Call-ID=%s From=%s To=%s",
		callID, req.From().Value(), req.To().Value())

	// Extract call info
	toURI := req.To().Address
	fromURI := req.From().Address

	toUser := toURI.User
	fromUser := fromURI.User

	// Extract custom headers for routing
	headers := make(map[string]string)
	for _, h := range req.Headers() {
		name := h.Name()
		if len(name) > 2 && name[:2] == "X-" {
			headers[name] = h.Value()
		}
	}

	// Find matching route
	route, err := s.router.FindRoute(ctx, toUser, fromUser, headers)
	if err != nil {
		log.Printf("[SIP] No route found for call %s: %v", callID, err)
		// Send 404 Not Found
		resp := sip.NewResponseFromRequest(req, 404, "Not Found", nil)
		if err := tx.Respond(resp); err != nil {
			log.Printf("[SIP] Failed to send 404: %v", err)
		}
		return
	}

	log.Printf("[SIP] Route matched: %s -> %s", route.Name, route.WebSocketURL)

	// Send 100 Trying
	trying := sip.NewResponseFromRequest(req, 100, "Trying", nil)
	if err := tx.Respond(trying); err != nil {
		log.Printf("[SIP] Failed to send 100 Trying: %v", err)
	}

	// Create call session
	session, err := s.calls.CreateSession(ctx, callID, req, route)
	if err != nil {
		log.Printf("[SIP] Failed to create session: %v", err)
		// Send 500 Internal Server Error
		resp := sip.NewResponseFromRequest(req, 500, "Internal Server Error", nil)
		if err := tx.Respond(resp); err != nil {
			log.Printf("[SIP] Failed to send 500: %v", err)
		}
		return
	}

	// Store transaction for later use
	session.SetTransaction(tx)

	// Send 180 Ringing
	ringing := sip.NewResponseFromRequest(req, 180, "Ringing", nil)
	if err := tx.Respond(ringing); err != nil {
		log.Printf("[SIP] Failed to send 180 Ringing: %v", err)
	}

	// Connect to WebSocket agent (async)
	go func() {
		if err := session.ConnectAgent(ctx); err != nil {
			log.Printf("[SIP] Failed to connect to agent: %v", err)
			// Send 503 Service Unavailable
			resp := sip.NewResponseFromRequest(req, 503, "Service Unavailable", nil)
			if err := tx.Respond(resp); err != nil {
				log.Printf("[SIP] Failed to send 503: %v", err)
			}
			s.calls.RemoveSession(callID)
			return
		}

		// Agent connected, answer the call
		// Generate SDP for RTP
		sdp := session.GenerateSDP()

		// Send 200 OK with SDP
		ok := sip.NewResponseFromRequest(req, 200, "OK", []byte(sdp))
		ok.AppendHeader(sip.NewHeader("Content-Type", "application/sdp"))

		if err := tx.Respond(ok); err != nil {
			log.Printf("[SIP] Failed to send 200 OK: %v", err)
			session.Close()
			s.calls.RemoveSession(callID)
			return
		}

		log.Printf("[SIP] Call %s answered", callID)
	}()
}

// handleAck processes ACK requests (call setup completion)
func (s *SIPServer) handleAck(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	log.Printf("[SIP] ACK received: Call-ID=%s", callID)

	session := s.calls.GetSession(callID)
	if session == nil {
		log.Printf("[SIP] No session found for ACK: %s", callID)
		return
	}

	// Start media streaming
	go session.StartMedia()
}

// handleBye processes BYE requests (call termination)
func (s *SIPServer) handleBye(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	log.Printf("[SIP] BYE received: Call-ID=%s", callID)

	session := s.calls.GetSession(callID)
	if session != nil {
		session.Close()
		s.calls.RemoveSession(callID)
	}

	// Send 200 OK
	ok := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(ok); err != nil {
		log.Printf("[SIP] Failed to send 200 OK for BYE: %v", err)
	}
}

// handleCancel processes CANCEL requests
func (s *SIPServer) handleCancel(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	log.Printf("[SIP] CANCEL received: Call-ID=%s", callID)

	session := s.calls.GetSession(callID)
	if session != nil {
		session.Close()
		s.calls.RemoveSession(callID)
	}

	// Send 200 OK
	ok := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(ok); err != nil {
		log.Printf("[SIP] Failed to send 200 OK for CANCEL: %v", err)
	}
}

// handleOptions processes OPTIONS requests (health check / keep-alive)
func (s *SIPServer) handleOptions(req *sip.Request, tx sip.ServerTransaction) {
	ok := sip.NewResponseFromRequest(req, 200, "OK", nil)
	ok.AppendHeader(sip.NewHeader("Allow", "INVITE, ACK, BYE, CANCEL, OPTIONS"))
	ok.AppendHeader(sip.NewHeader("Accept", "application/sdp"))

	if err := tx.Respond(ok); err != nil {
		log.Printf("[SIP] Failed to send OPTIONS response: %v", err)
	}
}

// Start starts the SIP server
func (s *SIPServer) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", s.config.SIPHost, s.config.SIPPort)

	// Start UDP listener
	if s.config.SIPTransport == "udp" || s.config.SIPTransport == "both" {
		go func() {
			log.Printf("[SIP] Starting UDP server on %s", addr)
			if err := s.server.ListenAndServe(ctx, "udp", addr); err != nil {
				log.Printf("[SIP] UDP server error: %v", err)
			}
		}()
	}

	// Start TCP listener
	if s.config.SIPTransport == "tcp" || s.config.SIPTransport == "both" {
		go func() {
			log.Printf("[SIP] Starting TCP server on %s", addr)
			if err := s.server.ListenAndServe(ctx, "tcp", addr); err != nil {
				log.Printf("[SIP] TCP server error: %v", err)
			}
		}()
	}

	log.Printf("[SIP] Server started on %s (%s)", addr, s.config.SIPTransport)
	return nil
}

// Stop stops the SIP server
func (s *SIPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// Close all active calls
	s.calls.CloseAll()

	log.Println("[SIP] Server stopped")
	return nil
}

// GetLocalIP returns the local IP address for SDP
func GetLocalIP() string {
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

// GenerateCallID generates a unique call ID
func GenerateCallID() string {
	return uuid.New().String()
}

