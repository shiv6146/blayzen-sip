# blayzen-sip Makefile

.PHONY: all build run test dev docker-up docker-down swagger migrate seed clean lint help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=blayzen-sip

# Build directory
BUILD_DIR=bin

# Default target
.DEFAULT_GOAL := help

##@ Development

build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/blayzen-sip

run: build ## Run the server locally
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run with hot reload (requires air)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

test: ## Run tests
	$(GOTEST) -v -race ./...

test-cover: ## Run tests with coverage
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

fmt: ## Format code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

##@ Swagger

swagger: ## Generate Swagger documentation
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/blayzen-sip/main.go -o docs
	@echo "Swagger docs generated in docs/"
	@echo "Access at http://localhost:8080/swagger/index.html"

##@ Docker

docker-build: ## Build Docker image
	docker-compose build

docker-up: ## Start all services with Docker Compose
	docker-compose up -d

docker-down: ## Stop all services
	docker-compose down

docker-logs: ## View logs
	docker-compose logs -f blayzen-sip

docker-clean: ## Remove volumes and containers
	docker-compose down -v

##@ Database

migrate: ## Run database migrations
	@docker-compose exec postgres psql -U blayzen -d blayzen_sip -f /docker-entrypoint-initdb.d/001_initial.sql

seed: ## Seed test data
	@docker-compose exec -T postgres psql -U blayzen -d blayzen_sip < scripts/seed.sql

psql: ## Connect to PostgreSQL
	@docker-compose exec postgres psql -U blayzen -d blayzen_sip

##@ Quick Start

quickstart: docker-up ## Quick start for developers
	@echo "Waiting for services to start..."
	@sleep 5
	@echo "Seeding test data..."
	@make seed
	@echo ""
	@echo "==================================="
	@echo "blayzen-sip is ready!"
	@echo "==================================="
	@echo "SIP Server:  localhost:5060"
	@echo "REST API:    http://localhost:8080/api/v1"
	@echo "Swagger UI:  http://localhost:8080/swagger/index.html"
	@echo ""
	@echo "Test credentials:"
	@echo "  Account ID: 00000000-0000-0000-0000-000000000001"
	@echo "  API Key:    test-api-key-12345"
	@echo "==================================="

##@ Dependencies

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOMOD) tidy
	$(GOGET) -u ./...

##@ Cleanup

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

clean-all: clean docker-clean ## Clean everything including Docker volumes

##@ Help

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

