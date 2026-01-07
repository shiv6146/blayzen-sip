// Package main is the entry point for blayzen-sip
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shiv6146/blayzen-sip/internal/api"
	"github.com/shiv6146/blayzen-sip/internal/config"
	"github.com/shiv6146/blayzen-sip/internal/server"
	"github.com/shiv6146/blayzen-sip/internal/store"

	_ "github.com/shiv6146/blayzen-sip/docs" // Import generated swagger docs
)

// @title blayzen-sip API
// @version 1.0
// @description SIP Server for Blayzen Voice Agents
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/shiv6146/blayzen-sip
// @contact.email support@blayzen.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.basic BasicAuth

func main() {
	log.Println("Starting blayzen-sip...")

	// Load configuration
	cfg := config.Load()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	pgStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgStore.Close()
	log.Println("PostgreSQL connected")

	// Connect to Valkey (optional)
	var cache *store.Cache
	if cfg.ValkeyURL != "" {
		log.Println("Connecting to Valkey...")
		cache, err = store.NewCache(ctx, cfg.ValkeyURL, cfg.ValkeyPassword, cfg.ValkeyDB, cfg.CacheRouteTTL)
		if err != nil {
			log.Printf("Warning: Failed to connect to Valkey: %v (continuing without cache)", err)
			cache = nil
		} else {
			defer cache.Close()
			log.Println("Valkey connected")
		}
	}

	// Create and start SIP server
	log.Println("Starting SIP server...")
	sipServer, err := server.NewSIPServer(cfg, pgStore, cache)
	if err != nil {
		log.Fatalf("Failed to create SIP server: %v", err)
	}

	if err := sipServer.Start(ctx); err != nil {
		log.Fatalf("Failed to start SIP server: %v", err)
	}
	log.Printf("SIP server listening on %s:%d (%s)", cfg.SIPHost, cfg.SIPPort, cfg.SIPTransport)

	// Create and start API server
	log.Println("Starting REST API server...")
	apiServer := api.NewServer(cfg, pgStore, cache)

	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()
	log.Printf("REST API server listening on %s:%d", cfg.APIHost, cfg.APIPort)
	log.Printf("Swagger UI: http://%s:%d/swagger/index.html", cfg.APIHost, cfg.APIPort)

	// Print startup summary
	log.Println("")
	log.Println("========================================")
	log.Println("blayzen-sip is running!")
	log.Println("========================================")
	log.Printf("SIP:      %s:%d (%s)", cfg.SIPHost, cfg.SIPPort, cfg.SIPTransport)
	log.Printf("REST API: http://%s:%d/api/v1", cfg.APIHost, cfg.APIPort)
	log.Printf("Swagger:  http://%s:%d/swagger/index.html", cfg.APIHost, cfg.APIPort)
	log.Printf("Health:   http://%s:%d/health", cfg.APIHost, cfg.APIPort)
	log.Println("========================================")
	log.Println("")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received, stopping services...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop API server
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}

	// Stop SIP server
	if err := sipServer.Stop(); err != nil {
		log.Printf("SIP server shutdown error: %v", err)
	}

	cancel()
	log.Println("blayzen-sip stopped")
}

