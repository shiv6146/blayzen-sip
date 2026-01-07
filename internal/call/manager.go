// Package call manages active call sessions
package call

import (
	"context"
	"log"
	"sync"

	"github.com/emiago/sipgo/sip"
	"github.com/shiv6146/blayzen-sip/internal/config"
	"github.com/shiv6146/blayzen-sip/internal/models"
	"github.com/shiv6146/blayzen-sip/internal/store"
)

// Manager manages active call sessions
type Manager struct {
	config   *config.Config
	store    *store.PostgresStore
	cache    *store.Cache
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a new call manager
func NewManager(cfg *config.Config, store *store.PostgresStore, cache *store.Cache) *Manager {
	return &Manager{
		config:   cfg,
		store:    store,
		cache:    cache,
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new call session
func (m *Manager) CreateSession(ctx context.Context, callID string, req *sip.Request, route *models.Route) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract call details
	toURI := req.To().Address
	fromURI := req.From().Address

	session := &Session{
		CallID:       callID,
		FromURI:      fromURI.String(),
		ToURI:        toURI.String(),
		FromUser:     fromURI.User,
		ToUser:       toURI.User,
		Route:        route,
		WebSocketURL: route.WebSocketURL,
		config:       m.config,
		store:        m.store,
	}

	// Allocate RTP ports
	if err := session.allocateRTPPorts(); err != nil {
		return nil, err
	}

	// Create call log entry
	callLog := &models.CallLog{
		AccountID:    &route.AccountID,
		CallID:       callID,
		Direction:    models.CallDirectionInbound,
		FromURI:      session.FromURI,
		ToURI:        session.ToURI,
		FromUser:     session.FromUser,
		ToUser:       session.ToUser,
		RouteID:      &route.ID,
		WebSocketURL: route.WebSocketURL,
		Status:       models.CallStatusInitiated,
	}

	if _, err := m.store.CreateCallLog(ctx, callLog); err != nil {
		log.Printf("[Call] Failed to create call log: %v", err)
		// Don't fail the call, just log the error
	}

	// Track in cache
	if m.cache != nil {
		_ = m.cache.SetActiveCall(ctx, callID, map[string]string{
			"from":   session.FromUser,
			"to":     session.ToUser,
			"status": string(models.CallStatusInitiated),
		})
	}

	m.sessions[callID] = session
	log.Printf("[Call] Session created: %s", callID)

	return session, nil
}

// GetSession returns a session by call ID
func (m *Manager) GetSession(callID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[callID]
}

// RemoveSession removes a session
func (m *Manager) RemoveSession(callID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[callID]; ok {
		session.Close()
		delete(m.sessions, callID)

		// Update call status
		ctx := context.Background()
		if err := m.store.UpdateCallStatus(ctx, callID, models.CallStatusCompleted); err != nil {
			log.Printf("[Call] Failed to update call status: %v", err)
		}

		// Remove from cache
		if m.cache != nil {
			_ = m.cache.RemoveActiveCall(ctx, callID)
		}

		log.Printf("[Call] Session removed: %s", callID)
	}
}

// CloseAll closes all active sessions
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for callID, session := range m.sessions {
		session.Close()
		delete(m.sessions, callID)
	}

	log.Println("[Call] All sessions closed")
}

// ActiveCount returns the number of active sessions
func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

