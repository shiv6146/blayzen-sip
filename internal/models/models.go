// Package models defines the domain models for blayzen-sip
package models

import (
	"time"
)

// Account represents a tenant/user account
type Account struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	APIKey    string    `json:"-" db:"api_key"` // Never expose API key in JSON
	Active    bool      `json:"active" db:"active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Route represents an inbound SIP routing rule
type Route struct {
	ID                  string                 `json:"id" db:"id"`
	AccountID           string                 `json:"account_id" db:"account_id"`
	Name                string                 `json:"name" db:"name"`
	Priority            int                    `json:"priority" db:"priority"`
	MatchToUser         *string                `json:"match_to_user,omitempty" db:"match_to_user"`
	MatchFromUser       *string                `json:"match_from_user,omitempty" db:"match_from_user"`
	MatchSIPHeader      *string                `json:"match_sip_header,omitempty" db:"match_sip_header"`
	MatchSIPHeaderValue *string                `json:"match_sip_header_value,omitempty" db:"match_sip_header_value"`
	WebSocketURL        string                 `json:"websocket_url" db:"websocket_url"`
	CustomData          map[string]interface{} `json:"custom_data,omitempty" db:"custom_data" swaggertype:"object"`
	Active              bool                   `json:"active" db:"active"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
}

// Trunk represents an outbound SIP trunk configuration
type Trunk struct {
	ID               string    `json:"id" db:"id"`
	AccountID        string    `json:"account_id" db:"account_id"`
	Name             string    `json:"name" db:"name"`
	Host             string    `json:"host" db:"host"`
	Port             int       `json:"port" db:"port"`
	Transport        string    `json:"transport" db:"transport"`
	Username         *string   `json:"username,omitempty" db:"username"`
	Password         *string   `json:"-" db:"password"` // Never expose password
	FromUser         *string   `json:"from_user,omitempty" db:"from_user"`
	FromHost         *string   `json:"from_host,omitempty" db:"from_host"`
	Register         bool      `json:"register" db:"register"`
	RegisterInterval int       `json:"register_interval" db:"register_interval"`
	Active           bool      `json:"active" db:"active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// CallStatus represents the state of a call
type CallStatus string

const (
	CallStatusInitiated CallStatus = "initiated"
	CallStatusRinging   CallStatus = "ringing"
	CallStatusAnswered  CallStatus = "answered"
	CallStatusCompleted CallStatus = "completed"
	CallStatusFailed    CallStatus = "failed"
	CallStatusCancelled CallStatus = "cancelled"
)

// CallDirection represents whether a call is inbound or outbound
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// CallLog represents a call detail record (CDR)
type CallLog struct {
	ID              string                 `json:"id" db:"id"`
	AccountID       *string                `json:"account_id,omitempty" db:"account_id"`
	CallID          string                 `json:"call_id" db:"call_id"`
	Direction       CallDirection          `json:"direction" db:"direction"`
	FromURI         string                 `json:"from_uri" db:"from_uri"`
	ToURI           string                 `json:"to_uri" db:"to_uri"`
	FromUser        string                 `json:"from_user" db:"from_user"`
	ToUser          string                 `json:"to_user" db:"to_user"`
	RouteID         *string                `json:"route_id,omitempty" db:"route_id"`
	TrunkID         *string                `json:"trunk_id,omitempty" db:"trunk_id"`
	WebSocketURL    string                 `json:"websocket_url" db:"websocket_url"`
	Status          CallStatus             `json:"status" db:"status"`
	InitiatedAt     time.Time              `json:"initiated_at" db:"initiated_at"`
	RingingAt       *time.Time             `json:"ringing_at,omitempty" db:"ringing_at"`
	AnsweredAt      *time.Time             `json:"answered_at,omitempty" db:"answered_at"`
	EndedAt         *time.Time             `json:"ended_at,omitempty" db:"ended_at"`
	DurationSeconds *int                   `json:"duration_seconds,omitempty" db:"duration_seconds"`
	HangupCause     *string                `json:"hangup_cause,omitempty" db:"hangup_cause"`
	HangupParty     *string                `json:"hangup_party,omitempty" db:"hangup_party"`
	CustomData      map[string]interface{} `json:"custom_data,omitempty" db:"custom_data" swaggertype:"object"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// Matches checks if the route matches the given criteria
func (r *Route) Matches(toUser, fromUser string, headers map[string]string) bool {
	// Check To User match
	if r.MatchToUser != nil && *r.MatchToUser != "" {
		if toUser != *r.MatchToUser {
			return false
		}
	}

	// Check From User match
	if r.MatchFromUser != nil && *r.MatchFromUser != "" {
		if fromUser != *r.MatchFromUser {
			return false
		}
	}

	// Check custom header match
	if r.MatchSIPHeader != nil && *r.MatchSIPHeader != "" {
		headerValue, exists := headers[*r.MatchSIPHeader]
		if !exists {
			return false
		}
		if r.MatchSIPHeaderValue != nil && *r.MatchSIPHeaderValue != "" {
			if headerValue != *r.MatchSIPHeaderValue {
				return false
			}
		}
	}

	return true
}

