// Package store provides database and cache access
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shiv6146/blayzen-sip/internal/models"
)

// PostgresStore implements database operations
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

// Close closes the connection pool
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// =============================================================================
// Account Operations
// =============================================================================

// ValidateAPIKey validates an API key and returns the account
func (s *PostgresStore) ValidateAPIKey(ctx context.Context, accountID, apiKey string) (*models.Account, error) {
	var account models.Account
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, api_key, active, created_at, updated_at
		FROM accounts
		WHERE id = $1 AND api_key = $2 AND active = true
	`, accountID, apiKey).Scan(
		&account.ID, &account.Name, &account.APIKey,
		&account.Active, &account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, err
	}
	return &account, nil
}

// GetAccount returns an account by ID
func (s *PostgresStore) GetAccount(ctx context.Context, id string) (*models.Account, error) {
	var account models.Account
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, api_key, active, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`, id).Scan(
		&account.ID, &account.Name, &account.APIKey,
		&account.Active, &account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// =============================================================================
// Route Operations
// =============================================================================

// ListRoutes returns all routes for an account
func (s *PostgresStore) ListRoutes(ctx context.Context, accountID string) ([]*models.Route, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, account_id, name, priority, 
		       match_to_user, match_from_user, match_sip_header, match_sip_header_value,
		       websocket_url, custom_data, active, created_at, updated_at
		FROM sip_routes
		WHERE account_id = $1
		ORDER BY priority DESC, name ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []*models.Route
	for rows.Next() {
		var r models.Route
		err := rows.Scan(
			&r.ID, &r.AccountID, &r.Name, &r.Priority,
			&r.MatchToUser, &r.MatchFromUser, &r.MatchSIPHeader, &r.MatchSIPHeaderValue,
			&r.WebSocketURL, &r.CustomData, &r.Active, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		routes = append(routes, &r)
	}

	return routes, rows.Err()
}

// GetRoute returns a route by ID
func (s *PostgresStore) GetRoute(ctx context.Context, accountID, routeID string) (*models.Route, error) {
	var r models.Route
	err := s.pool.QueryRow(ctx, `
		SELECT id, account_id, name, priority,
		       match_to_user, match_from_user, match_sip_header, match_sip_header_value,
		       websocket_url, custom_data, active, created_at, updated_at
		FROM sip_routes
		WHERE id = $1 AND account_id = $2
	`, routeID, accountID).Scan(
		&r.ID, &r.AccountID, &r.Name, &r.Priority,
		&r.MatchToUser, &r.MatchFromUser, &r.MatchSIPHeader, &r.MatchSIPHeaderValue,
		&r.WebSocketURL, &r.CustomData, &r.Active, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRoute creates a new route
func (s *PostgresStore) CreateRoute(ctx context.Context, accountID string, route *models.Route) (*models.Route, error) {
	customData := route.CustomData
	if customData == nil {
		customData = make(map[string]interface{})
	}

	var r models.Route
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sip_routes (account_id, name, priority, match_to_user, match_from_user,
		                        match_sip_header, match_sip_header_value, websocket_url, custom_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, account_id, name, priority, match_to_user, match_from_user,
		          match_sip_header, match_sip_header_value, websocket_url, custom_data,
		          active, created_at, updated_at
	`, accountID, route.Name, route.Priority, route.MatchToUser, route.MatchFromUser,
		route.MatchSIPHeader, route.MatchSIPHeaderValue, route.WebSocketURL, customData,
	).Scan(
		&r.ID, &r.AccountID, &r.Name, &r.Priority,
		&r.MatchToUser, &r.MatchFromUser, &r.MatchSIPHeader, &r.MatchSIPHeaderValue,
		&r.WebSocketURL, &r.CustomData, &r.Active, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateRoute updates a route
func (s *PostgresStore) UpdateRoute(ctx context.Context, accountID string, route *models.Route) (*models.Route, error) {
	customData := route.CustomData
	if customData == nil {
		customData = make(map[string]interface{})
	}

	var r models.Route
	err := s.pool.QueryRow(ctx, `
		UPDATE sip_routes
		SET name = $3, priority = $4, match_to_user = $5, match_from_user = $6,
		    match_sip_header = $7, match_sip_header_value = $8, websocket_url = $9,
		    custom_data = $10, active = $11
		WHERE id = $1 AND account_id = $2
		RETURNING id, account_id, name, priority, match_to_user, match_from_user,
		          match_sip_header, match_sip_header_value, websocket_url, custom_data,
		          active, created_at, updated_at
	`, route.ID, accountID, route.Name, route.Priority, route.MatchToUser, route.MatchFromUser,
		route.MatchSIPHeader, route.MatchSIPHeaderValue, route.WebSocketURL, customData, route.Active,
	).Scan(
		&r.ID, &r.AccountID, &r.Name, &r.Priority,
		&r.MatchToUser, &r.MatchFromUser, &r.MatchSIPHeader, &r.MatchSIPHeaderValue,
		&r.WebSocketURL, &r.CustomData, &r.Active, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteRoute deletes a route
func (s *PostgresStore) DeleteRoute(ctx context.Context, accountID, routeID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM sip_routes WHERE id = $1 AND account_id = $2
	`, routeID, accountID)
	return err
}

// FindMatchingRoutes finds routes that could match the given criteria
func (s *PostgresStore) FindMatchingRoutes(ctx context.Context, toUser, fromUser string) ([]*models.Route, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, account_id, name, priority,
		       match_to_user, match_from_user, match_sip_header, match_sip_header_value,
		       websocket_url, custom_data, active, created_at, updated_at
		FROM sip_routes
		WHERE active = true
		  AND (match_to_user IS NULL OR match_to_user = '' OR match_to_user = $1)
		  AND (match_from_user IS NULL OR match_from_user = '' OR match_from_user = $2)
		ORDER BY priority DESC
	`, toUser, fromUser)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []*models.Route
	for rows.Next() {
		var r models.Route
		err := rows.Scan(
			&r.ID, &r.AccountID, &r.Name, &r.Priority,
			&r.MatchToUser, &r.MatchFromUser, &r.MatchSIPHeader, &r.MatchSIPHeaderValue,
			&r.WebSocketURL, &r.CustomData, &r.Active, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		routes = append(routes, &r)
	}

	return routes, rows.Err()
}

// =============================================================================
// Trunk Operations
// =============================================================================

// ListTrunks returns all trunks for an account
func (s *PostgresStore) ListTrunks(ctx context.Context, accountID string) ([]*models.Trunk, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, account_id, name, host, port, transport,
		       username, password, from_user, from_host,
		       register, register_interval, active, created_at, updated_at
		FROM sip_trunks
		WHERE account_id = $1
		ORDER BY name ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trunks []*models.Trunk
	for rows.Next() {
		var t models.Trunk
		err := rows.Scan(
			&t.ID, &t.AccountID, &t.Name, &t.Host, &t.Port, &t.Transport,
			&t.Username, &t.Password, &t.FromUser, &t.FromHost,
			&t.Register, &t.RegisterInterval, &t.Active, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		trunks = append(trunks, &t)
	}

	return trunks, rows.Err()
}

// GetTrunk returns a trunk by ID
func (s *PostgresStore) GetTrunk(ctx context.Context, accountID, trunkID string) (*models.Trunk, error) {
	var t models.Trunk
	err := s.pool.QueryRow(ctx, `
		SELECT id, account_id, name, host, port, transport,
		       username, password, from_user, from_host,
		       register, register_interval, active, created_at, updated_at
		FROM sip_trunks
		WHERE id = $1 AND account_id = $2
	`, trunkID, accountID).Scan(
		&t.ID, &t.AccountID, &t.Name, &t.Host, &t.Port, &t.Transport,
		&t.Username, &t.Password, &t.FromUser, &t.FromHost,
		&t.Register, &t.RegisterInterval, &t.Active, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateTrunk creates a new trunk
func (s *PostgresStore) CreateTrunk(ctx context.Context, accountID string, trunk *models.Trunk) (*models.Trunk, error) {
	var t models.Trunk
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sip_trunks (account_id, name, host, port, transport,
		                        username, password, from_user, from_host,
		                        register, register_interval)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, account_id, name, host, port, transport,
		          username, password, from_user, from_host,
		          register, register_interval, active, created_at, updated_at
	`, accountID, trunk.Name, trunk.Host, trunk.Port, trunk.Transport,
		trunk.Username, trunk.Password, trunk.FromUser, trunk.FromHost,
		trunk.Register, trunk.RegisterInterval,
	).Scan(
		&t.ID, &t.AccountID, &t.Name, &t.Host, &t.Port, &t.Transport,
		&t.Username, &t.Password, &t.FromUser, &t.FromHost,
		&t.Register, &t.RegisterInterval, &t.Active, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTrunk updates a trunk
func (s *PostgresStore) UpdateTrunk(ctx context.Context, accountID string, trunk *models.Trunk) (*models.Trunk, error) {
	var t models.Trunk
	err := s.pool.QueryRow(ctx, `
		UPDATE sip_trunks
		SET name = $3, host = $4, port = $5, transport = $6,
		    username = $7, password = $8, from_user = $9, from_host = $10,
		    register = $11, register_interval = $12, active = $13
		WHERE id = $1 AND account_id = $2
		RETURNING id, account_id, name, host, port, transport,
		          username, password, from_user, from_host,
		          register, register_interval, active, created_at, updated_at
	`, trunk.ID, accountID, trunk.Name, trunk.Host, trunk.Port, trunk.Transport,
		trunk.Username, trunk.Password, trunk.FromUser, trunk.FromHost,
		trunk.Register, trunk.RegisterInterval, trunk.Active,
	).Scan(
		&t.ID, &t.AccountID, &t.Name, &t.Host, &t.Port, &t.Transport,
		&t.Username, &t.Password, &t.FromUser, &t.FromHost,
		&t.Register, &t.RegisterInterval, &t.Active, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteTrunk deletes a trunk
func (s *PostgresStore) DeleteTrunk(ctx context.Context, accountID, trunkID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM sip_trunks WHERE id = $1 AND account_id = $2
	`, trunkID, accountID)
	return err
}

// =============================================================================
// Call Log Operations
// =============================================================================

// CreateCallLog creates a new call log entry
func (s *PostgresStore) CreateCallLog(ctx context.Context, call *models.CallLog) (*models.CallLog, error) {
	customData := call.CustomData
	if customData == nil {
		customData = make(map[string]interface{})
	}

	var c models.CallLog
	err := s.pool.QueryRow(ctx, `
		INSERT INTO call_logs (account_id, call_id, direction, from_uri, to_uri,
		                       from_user, to_user, route_id, trunk_id, websocket_url,
		                       status, custom_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, account_id, call_id, direction, from_uri, to_uri,
		          from_user, to_user, route_id, trunk_id, websocket_url,
		          status, initiated_at, created_at
	`, call.AccountID, call.CallID, call.Direction, call.FromURI, call.ToURI,
		call.FromUser, call.ToUser, call.RouteID, call.TrunkID, call.WebSocketURL,
		call.Status, customData,
	).Scan(
		&c.ID, &c.AccountID, &c.CallID, &c.Direction, &c.FromURI, &c.ToURI,
		&c.FromUser, &c.ToUser, &c.RouteID, &c.TrunkID, &c.WebSocketURL,
		&c.Status, &c.InitiatedAt, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateCallStatus updates the status of a call
func (s *PostgresStore) UpdateCallStatus(ctx context.Context, callID string, status models.CallStatus) error {
	now := time.Now()
	var query string
	var args []interface{}

	switch status {
	case models.CallStatusRinging:
		query = `UPDATE call_logs SET status = $1, ringing_at = $2 WHERE call_id = $3`
		args = []interface{}{status, now, callID}
	case models.CallStatusAnswered:
		query = `UPDATE call_logs SET status = $1, answered_at = $2 WHERE call_id = $3`
		args = []interface{}{status, now, callID}
	case models.CallStatusCompleted, models.CallStatusFailed, models.CallStatusCancelled:
		query = `
			UPDATE call_logs 
			SET status = $1, ended_at = $2, 
			    duration_seconds = EXTRACT(EPOCH FROM ($2 - COALESCE(answered_at, initiated_at)))::INT
			WHERE call_id = $3`
		args = []interface{}{status, now, callID}
	default:
		query = `UPDATE call_logs SET status = $1 WHERE call_id = $2`
		args = []interface{}{status, callID}
	}

	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

// ListCalls returns recent calls for an account
func (s *PostgresStore) ListCalls(ctx context.Context, accountID string, limit int) ([]*models.CallLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, account_id, call_id, direction, from_uri, to_uri,
		       from_user, to_user, route_id, trunk_id, websocket_url,
		       status, initiated_at, ringing_at, answered_at, ended_at,
		       duration_seconds, hangup_cause, hangup_party, custom_data, created_at
		FROM call_logs
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []*models.CallLog
	for rows.Next() {
		var c models.CallLog
		err := rows.Scan(
			&c.ID, &c.AccountID, &c.CallID, &c.Direction, &c.FromURI, &c.ToURI,
			&c.FromUser, &c.ToUser, &c.RouteID, &c.TrunkID, &c.WebSocketURL,
			&c.Status, &c.InitiatedAt, &c.RingingAt, &c.AnsweredAt, &c.EndedAt,
			&c.DurationSeconds, &c.HangupCause, &c.HangupParty, &c.CustomData, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		calls = append(calls, &c)
	}

	return calls, rows.Err()
}

// GetCall returns a call by ID
func (s *PostgresStore) GetCall(ctx context.Context, accountID, callID string) (*models.CallLog, error) {
	var c models.CallLog
	err := s.pool.QueryRow(ctx, `
		SELECT id, account_id, call_id, direction, from_uri, to_uri,
		       from_user, to_user, route_id, trunk_id, websocket_url,
		       status, initiated_at, ringing_at, answered_at, ended_at,
		       duration_seconds, hangup_cause, hangup_party, custom_data, created_at
		FROM call_logs
		WHERE id = $1 AND account_id = $2
	`, callID, accountID).Scan(
		&c.ID, &c.AccountID, &c.CallID, &c.Direction, &c.FromURI, &c.ToURI,
		&c.FromUser, &c.ToUser, &c.RouteID, &c.TrunkID, &c.WebSocketURL,
		&c.Status, &c.InitiatedAt, &c.RingingAt, &c.AnsweredAt, &c.EndedAt,
		&c.DurationSeconds, &c.HangupCause, &c.HangupParty, &c.CustomData, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

