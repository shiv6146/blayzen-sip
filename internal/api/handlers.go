// Package api provides the REST API handlers for blayzen-sip
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shiv6146/blayzen-sip/internal/models"
	"github.com/shiv6146/blayzen-sip/internal/store"
)

// Handler holds the API dependencies
type Handler struct {
	store *store.PostgresStore
	cache *store.Cache
}

// NewHandler creates a new API handler
func NewHandler(store *store.PostgresStore, cache *store.Cache) *Handler {
	return &Handler{
		store: store,
		cache: cache,
	}
}

// =============================================================================
// Request/Response DTOs
// =============================================================================

// CreateRouteRequest is the request body for creating a route
type CreateRouteRequest struct {
	Name                string                 `json:"name" binding:"required" example:"Support Line"`
	Priority            int                    `json:"priority" example:"10"`
	MatchToUser         *string                `json:"match_to_user,omitempty" example:"1000"`
	MatchFromUser       *string                `json:"match_from_user,omitempty" example:"+14155551234"`
	MatchSIPHeader      *string                `json:"match_sip_header,omitempty" example:"X-Customer-Tier"`
	MatchSIPHeaderValue *string                `json:"match_sip_header_value,omitempty" example:"vip"`
	WebSocketURL        string                 `json:"websocket_url" binding:"required" example:"ws://agent:8081/ws"`
	CustomData          map[string]interface{} `json:"custom_data,omitempty"`
}

// UpdateRouteRequest is the request body for updating a route
type UpdateRouteRequest struct {
	Name                string                 `json:"name" binding:"required" example:"Support Line"`
	Priority            int                    `json:"priority" example:"10"`
	MatchToUser         *string                `json:"match_to_user,omitempty" example:"1000"`
	MatchFromUser       *string                `json:"match_from_user,omitempty" example:"+14155551234"`
	MatchSIPHeader      *string                `json:"match_sip_header,omitempty" example:"X-Customer-Tier"`
	MatchSIPHeaderValue *string                `json:"match_sip_header_value,omitempty" example:"vip"`
	WebSocketURL        string                 `json:"websocket_url" binding:"required" example:"ws://agent:8081/ws"`
	CustomData          map[string]interface{} `json:"custom_data,omitempty"`
	Active              bool                   `json:"active" example:"true"`
}

// CreateTrunkRequest is the request body for creating a trunk
type CreateTrunkRequest struct {
	Name             string  `json:"name" binding:"required" example:"Primary Trunk"`
	Host             string  `json:"host" binding:"required" example:"sip.provider.com"`
	Port             int     `json:"port" example:"5060"`
	Transport        string  `json:"transport" example:"udp"`
	Username         *string `json:"username,omitempty" example:"user"`
	Password         *string `json:"password,omitempty" example:"secret"`
	FromUser         *string `json:"from_user,omitempty" example:"+14155551234"`
	FromHost         *string `json:"from_host,omitempty" example:"sip.provider.com"`
	Register         bool    `json:"register" example:"false"`
	RegisterInterval int     `json:"register_interval" example:"3600"`
}

// UpdateTrunkRequest is the request body for updating a trunk
type UpdateTrunkRequest struct {
	Name             string  `json:"name" binding:"required" example:"Primary Trunk"`
	Host             string  `json:"host" binding:"required" example:"sip.provider.com"`
	Port             int     `json:"port" example:"5060"`
	Transport        string  `json:"transport" example:"udp"`
	Username         *string `json:"username,omitempty" example:"user"`
	Password         *string `json:"password,omitempty" example:"secret"`
	FromUser         *string `json:"from_user,omitempty" example:"+14155551234"`
	FromHost         *string `json:"from_host,omitempty" example:"sip.provider.com"`
	Register         bool    `json:"register" example:"false"`
	RegisterInterval int     `json:"register_interval" example:"3600"`
	Active           bool    `json:"active" example:"true"`
}

// InitiateCallRequest is the request body for initiating an outbound call
type InitiateCallRequest struct {
	TrunkID      string                 `json:"trunk_id" binding:"required" example:"trunk-uuid"`
	To           string                 `json:"to" binding:"required" example:"+14155551234"`
	From         *string                `json:"from,omitempty" example:"+14155555678"`
	WebSocketURL string                 `json:"websocket_url" binding:"required" example:"ws://agent:8081/ws"`
	CustomData   map[string]interface{} `json:"custom_data,omitempty"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error" example:"Invalid request"`
	Details string `json:"details,omitempty" example:"Field 'name' is required"`
}

// SuccessResponse represents a success message
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// =============================================================================
// Route Handlers
// =============================================================================

// ListRoutes godoc
// @Summary List all routes
// @Description Get all SIP routing rules for the account
// @Tags Routes
// @Accept json
// @Produce json
// @Security BasicAuth
// @Success 200 {array} models.Route
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/routes [get]
func (h *Handler) ListRoutes(c *gin.Context) {
	accountID := c.GetString("account_id")

	routes, err := h.store.ListRoutes(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to fetch routes", Details: err.Error()})
		return
	}

	if routes == nil {
		routes = []*models.Route{}
	}

	c.JSON(http.StatusOK, routes)
}

// GetRoute godoc
// @Summary Get a route
// @Description Get a specific SIP routing rule by ID
// @Tags Routes
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "Route ID"
// @Success 200 {object} models.Route
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/routes/{id} [get]
func (h *Handler) GetRoute(c *gin.Context) {
	accountID := c.GetString("account_id")
	routeID := c.Param("id")

	route, err := h.store.GetRoute(c.Request.Context(), accountID, routeID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Route not found"})
		return
	}

	c.JSON(http.StatusOK, route)
}

// CreateRoute godoc
// @Summary Create a route
// @Description Create a new SIP routing rule
// @Tags Routes
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param route body CreateRouteRequest true "Route configuration"
// @Success 201 {object} models.Route
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/routes [post]
func (h *Handler) CreateRoute(c *gin.Context) {
	accountID := c.GetString("account_id")

	var req CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}

	route := &models.Route{
		Name:                req.Name,
		Priority:            req.Priority,
		MatchToUser:         req.MatchToUser,
		MatchFromUser:       req.MatchFromUser,
		MatchSIPHeader:      req.MatchSIPHeader,
		MatchSIPHeaderValue: req.MatchSIPHeaderValue,
		WebSocketURL:        req.WebSocketURL,
	}

	created, err := h.store.CreateRoute(c.Request.Context(), accountID, route)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create route", Details: err.Error()})
		return
	}

	// Invalidate route cache
	if h.cache != nil {
		_ = h.cache.InvalidateRouteCache(c.Request.Context())
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateRoute godoc
// @Summary Update a route
// @Description Update an existing SIP routing rule
// @Tags Routes
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "Route ID"
// @Param route body UpdateRouteRequest true "Route configuration"
// @Success 200 {object} models.Route
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/routes/{id} [put]
func (h *Handler) UpdateRoute(c *gin.Context) {
	accountID := c.GetString("account_id")
	routeID := c.Param("id")

	var req UpdateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}

	route := &models.Route{
		ID:                  routeID,
		Name:                req.Name,
		Priority:            req.Priority,
		MatchToUser:         req.MatchToUser,
		MatchFromUser:       req.MatchFromUser,
		MatchSIPHeader:      req.MatchSIPHeader,
		MatchSIPHeaderValue: req.MatchSIPHeaderValue,
		WebSocketURL:        req.WebSocketURL,
		Active:              req.Active,
	}

	updated, err := h.store.UpdateRoute(c.Request.Context(), accountID, route)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update route", Details: err.Error()})
		return
	}

	// Invalidate route cache
	if h.cache != nil {
		_ = h.cache.InvalidateRouteCache(c.Request.Context())
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteRoute godoc
// @Summary Delete a route
// @Description Delete a SIP routing rule
// @Tags Routes
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "Route ID"
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/routes/{id} [delete]
func (h *Handler) DeleteRoute(c *gin.Context) {
	accountID := c.GetString("account_id")
	routeID := c.Param("id")

	if err := h.store.DeleteRoute(c.Request.Context(), accountID, routeID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete route", Details: err.Error()})
		return
	}

	// Invalidate route cache
	if h.cache != nil {
		_ = h.cache.InvalidateRouteCache(c.Request.Context())
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Route deleted successfully"})
}

// =============================================================================
// Trunk Handlers
// =============================================================================

// ListTrunks godoc
// @Summary List all trunks
// @Description Get all SIP trunks for the account
// @Tags Trunks
// @Accept json
// @Produce json
// @Security BasicAuth
// @Success 200 {array} models.Trunk
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/trunks [get]
func (h *Handler) ListTrunks(c *gin.Context) {
	accountID := c.GetString("account_id")

	trunks, err := h.store.ListTrunks(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to fetch trunks", Details: err.Error()})
		return
	}

	if trunks == nil {
		trunks = []*models.Trunk{}
	}

	c.JSON(http.StatusOK, trunks)
}

// GetTrunk godoc
// @Summary Get a trunk
// @Description Get a specific SIP trunk by ID
// @Tags Trunks
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "Trunk ID"
// @Success 200 {object} models.Trunk
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/trunks/{id} [get]
func (h *Handler) GetTrunk(c *gin.Context) {
	accountID := c.GetString("account_id")
	trunkID := c.Param("id")

	trunk, err := h.store.GetTrunk(c.Request.Context(), accountID, trunkID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Trunk not found"})
		return
	}

	c.JSON(http.StatusOK, trunk)
}

// CreateTrunk godoc
// @Summary Create a trunk
// @Description Create a new SIP trunk
// @Tags Trunks
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param trunk body CreateTrunkRequest true "Trunk configuration"
// @Success 201 {object} models.Trunk
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/trunks [post]
func (h *Handler) CreateTrunk(c *gin.Context) {
	accountID := c.GetString("account_id")

	var req CreateTrunkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}

	port := req.Port
	if port == 0 {
		port = 5060
	}

	transport := req.Transport
	if transport == "" {
		transport = "udp"
	}

	trunk := &models.Trunk{
		Name:             req.Name,
		Host:             req.Host,
		Port:             port,
		Transport:        transport,
		Username:         req.Username,
		Password:         req.Password,
		FromUser:         req.FromUser,
		FromHost:         req.FromHost,
		Register:         req.Register,
		RegisterInterval: req.RegisterInterval,
	}

	created, err := h.store.CreateTrunk(c.Request.Context(), accountID, trunk)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create trunk", Details: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateTrunk godoc
// @Summary Update a trunk
// @Description Update an existing SIP trunk
// @Tags Trunks
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "Trunk ID"
// @Param trunk body UpdateTrunkRequest true "Trunk configuration"
// @Success 200 {object} models.Trunk
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/trunks/{id} [put]
func (h *Handler) UpdateTrunk(c *gin.Context) {
	accountID := c.GetString("account_id")
	trunkID := c.Param("id")

	var req UpdateTrunkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}

	port := req.Port
	if port == 0 {
		port = 5060
	}

	transport := req.Transport
	if transport == "" {
		transport = "udp"
	}

	trunk := &models.Trunk{
		ID:               trunkID,
		Name:             req.Name,
		Host:             req.Host,
		Port:             port,
		Transport:        transport,
		Username:         req.Username,
		Password:         req.Password,
		FromUser:         req.FromUser,
		FromHost:         req.FromHost,
		Register:         req.Register,
		RegisterInterval: req.RegisterInterval,
		Active:           req.Active,
	}

	updated, err := h.store.UpdateTrunk(c.Request.Context(), accountID, trunk)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update trunk", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteTrunk godoc
// @Summary Delete a trunk
// @Description Delete a SIP trunk
// @Tags Trunks
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "Trunk ID"
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/trunks/{id} [delete]
func (h *Handler) DeleteTrunk(c *gin.Context) {
	accountID := c.GetString("account_id")
	trunkID := c.Param("id")

	if err := h.store.DeleteTrunk(c.Request.Context(), accountID, trunkID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete trunk", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Trunk deleted successfully"})
}

// =============================================================================
// Call Handlers
// =============================================================================

// ListCalls godoc
// @Summary List recent calls
// @Description Get recent call detail records for the account
// @Tags Calls
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param limit query int false "Maximum number of records" default(100)
// @Success 200 {array} models.CallLog
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls [get]
func (h *Handler) ListCalls(c *gin.Context) {
	accountID := c.GetString("account_id")

	calls, err := h.store.ListCalls(c.Request.Context(), accountID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to fetch calls", Details: err.Error()})
		return
	}

	if calls == nil {
		calls = []*models.CallLog{}
	}

	c.JSON(http.StatusOK, calls)
}

// GetCall godoc
// @Summary Get a call
// @Description Get a specific call detail record by ID
// @Tags Calls
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "Call ID"
// @Success 200 {object} models.CallLog
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls/{id} [get]
func (h *Handler) GetCall(c *gin.Context) {
	accountID := c.GetString("account_id")
	callID := c.Param("id")

	call, err := h.store.GetCall(c.Request.Context(), accountID, callID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Call not found"})
		return
	}

	c.JSON(http.StatusOK, call)
}

// InitiateCall godoc
// @Summary Initiate an outbound call
// @Description Start a new outbound call via SIP trunk
// @Tags Calls
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param call body InitiateCallRequest true "Call configuration"
// @Success 202 {object} models.CallLog
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/calls [post]
func (h *Handler) InitiateCall(c *gin.Context) {
	// This is a placeholder - actual implementation requires the SIP server
	c.JSON(http.StatusNotImplemented, ErrorResponse{Error: "Outbound calling not yet implemented"})
}

// =============================================================================
// Health Check
// =============================================================================

// HealthCheck godoc
// @Summary Health check
// @Description Check if the service is healthy
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "blayzen-sip",
	})
}

