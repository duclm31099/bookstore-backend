.PHONY: help run build test migrate-up migrate-down migrate-create docker-up docker-down clean dev dev-stop dev-logs dev-all dev-db


# Variables
APP_NAME=bookstore-backend
VERSION=1.0.0
BUILD_DIR=bin
GO=go
GOFLAGS=-v

# ========================================
# HELP
# ========================================

help: ## Show this help
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-25s %s\n", $$1, $$2}'

# ========================================
# SETUP
# ========================================

install: ## Install dependencies
	$(GO) mod download
	$(GO) mod tidy

# ========================================
# DEVELOPMENT - HOT RELOAD (AIR)
# ========================================

dev: ## Run with hot reload using Air (requires: go install github.com/air-verse/air@latest)
	@echo "🔥 Starting development server with hot reload..."
	$(shell go env GOPATH)/bin/air

dev-db: ## Start Docker containers (PostgreSQL + Redis)
	@echo "🚀 Starting Docker containers..."
	docker compose up -d postgres redis
	@echo "✅ Docker containers started"

dev-stop: ## Stop Docker containers
	@echo "🛑 Stopping Docker containers..."
	docker compose down
	@echo "✅ Containers stopped"

dev-logs: ## View Docker logs (real-time)
	docker compose logs -f

dev-help: ## Development workflow instructions
	@echo "Development Workflow:"
	@echo "1. Terminal 1: make dev-db     (start Docker)"
	@echo "2. Terminal 2: make dev        (start app with hot reload)"
	@echo "3. Terminal 3: make test       (run tests)"
	@echo ""
	@echo "When you edit Go files, app automatically restarts!"
	@echo "Stop with: make dev-stop"

# ========================================
# RUNNING
# ========================================

run: ## Run the application (without hot reload)
	$(GO) run cmd/api/main.go

run-worker: ## Run background worker
	$(GO) run cmd/worker/main.go

build: ## Build the application
	mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/api cmd/api/main.go
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/worker cmd/worker/main.go
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/migrate cmd/migrate/main.go

# ========================================
# DATABASE MIGRATIONS - DEV
# ========================================

DB_URL=postgresql://bookstore:secret@localhost:5439/bookstore_dev?sslmode=disable

migrate-up: ## Run database migrations (dev)
	migrate -path ./migrations -database "$(DB_URL)" up

migrate-down: ## Rollback last migration (dev)
	migrate -path ./migrations -database "$(DB_URL)" down 1

migrate-down-all: ## Rollback all migrations (dev)
	migrate -path ./migrations -database "$(DB_URL)" down

migrate-create: ## Create new migration (usage: make migrate-create name=create_books_table)
	migrate create -ext sql -dir migrations -seq $(name)

migrate-version: ## Show current migration version (dev)
	migrate -path ./migrations -database "$(DB_URL)" version

migrate-force: ## Force migration version (usage: make migrate-force version=1)
	migrate -path ./migrations -database "$(DB_URL)" force $(version)

# ========================================
# DATABASE MIGRATIONS - TEST
# ========================================

migrate-test-up: ## Run migrations on test database
	migrate -path migrations \
		-database "postgresql://bookstore:secret@localhost:5439/bookstore_test?sslmode=disable" \
		up

migrate-test-down: ## Rollback test migrations
	migrate -path migrations \
		-database "postgresql://bookstore:secret@localhost:5439/bookstore_test?sslmode=disable" \
		down 1

migrate-test-reset: ## Reset test database (drop + up)
	migrate -path migrations \
		-database "postgresql://bookstore:secret@localhost:5439/bookstore_test?sslmode=disable" \
		drop -f
	migrate -path migrations \
		-database "postgresql://bookstore:secret@localhost:5439/bookstore_test?sslmode=disable" \
		up

# ========================================
# TESTING
# ========================================

test: ## Run all tests
	go test ./tests/... -v

test-coverage: ## Run tests with coverage
	go test ./tests/... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

test-clean: ## Clean test database and run tests
	@make migrate-test-reset
	go test ./tests/... -v -count=1

# ========================================
# DOCKER
# ========================================

docker-up: ## Start all Docker containers
	docker compose up -d

docker-down: ## Stop all Docker containers
	docker compose down

docker-logs: ## View Docker logs (real-time)
	docker compose logs -f

docker-ps: ## List running containers
	docker compose ps

# ========================================
# DATABASE UTILITIES
# ========================================

seed: ## Seed database with initial data
	$(GO) run scripts/seed.go

db-shell: ## Access PostgreSQL shell (dev database)
	docker exec -it bookstore_postgres psql -U bookstore -d bookstore_dev

db-table: ## Show users table schema
	docker exec -it bookstore_postgres psql -U bookstore -d bookstore_dev -c "\d users"

# ========================================
# CODE QUALITY
# ========================================

fmt: ## Format code
	$(GO) fmt ./...
	goimports -w .

lint: ## Run linters
	golangci-lint run

# ========================================
# CLEANUP
# ========================================

clean: ## Clean build artifacts and temp files
	rm -rf $(BUILD_DIR)
	rm -rf tmp/
	rm -f coverage.out coverage.html
	$(GO) clean

.DEFAULT_GOAL := help
