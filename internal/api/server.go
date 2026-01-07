package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shiv6146/blayzen-sip/internal/config"
	"github.com/shiv6146/blayzen-sip/internal/store"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the REST API server
type Server struct {
	config     *config.Config
	store      *store.PostgresStore
	cache      *store.Cache
	handler    *Handler
	router     *gin.Engine
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, store *store.PostgresStore, cache *store.Cache) *Server {
	gin.SetMode(cfg.GinMode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	handler := NewHandler(store, cache)

	s := &Server{
		config:  cfg,
		store:   store,
		cache:   cache,
		handler: handler,
		router:  router,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check (no auth required)
	s.router.GET("/health", s.handler.HealthCheck)

	// Swagger documentation
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := s.router.Group("/api/v1")

	// Apply authentication middleware if enabled
	if s.config.APIAuthEnabled {
		v1.Use(s.authMiddleware())
	}

	// Routes
	routes := v1.Group("/routes")
	{
		routes.GET("", s.handler.ListRoutes)
		routes.GET("/:id", s.handler.GetRoute)
		routes.POST("", s.handler.CreateRoute)
		routes.PUT("/:id", s.handler.UpdateRoute)
		routes.DELETE("/:id", s.handler.DeleteRoute)
	}

	// Trunks
	trunks := v1.Group("/trunks")
	{
		trunks.GET("", s.handler.ListTrunks)
		trunks.GET("/:id", s.handler.GetTrunk)
		trunks.POST("", s.handler.CreateTrunk)
		trunks.PUT("/:id", s.handler.UpdateTrunk)
		trunks.DELETE("/:id", s.handler.DeleteTrunk)
	}

	// Calls
	calls := v1.Group("/calls")
	{
		calls.GET("", s.handler.ListCalls)
		calls.GET("/:id", s.handler.GetCall)
		calls.POST("", s.handler.InitiateCall)
	}
}

// authMiddleware validates Basic Auth credentials against the database
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, apiKey, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="blayzen-sip"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Authentication required",
			})
			return
		}

		account, err := s.store.ValidateAPIKey(c.Request.Context(), accountID, apiKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid credentials",
			})
			return
		}

		// Store account info in context
		c.Set("account_id", account.ID)
		c.Set("account_name", account.Name)

		c.Next()
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.APIHost, s.config.APIPort)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("REST API server starting on %s", addr)
	log.Printf("Swagger UI available at http://%s/swagger/index.html", addr)

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Router returns the underlying gin router (for testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}

