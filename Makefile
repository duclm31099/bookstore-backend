.PHONY: help install dev dev-worker dev-db dev-stop dev-logs dev-all \
        run run-worker build test test-coverage test-clean \
        docker-build docker-up docker-down docker-restart docker-logs docker-ps \
        migrate-up migrate-down migrate-create migrate-version \
        seed clean-seed db-shell db-reset \
        asynq-stats asynq-dashboard \
        fmt lint clean

# ========================================
# VARIABLES
# ========================================
APP_NAME=bookstore-backend
VERSION=1.0.0
BUILD_DIR=bin
GO=go
GOFLAGS=-v

# Database connection
DB_HOST ?= localhost
DB_PORT ?= 5439
DB_NAME ?= bookstore_dev
DB_USER ?= bookstore
DB_PASSWORD ?= secret
DB_URL=postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
PSQL=PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1

# ========================================
# HELP
# ========================================
help: ## Show available commands
	@echo "📚 $(APP_NAME) - Development Commands"
	@echo ""
	@echo "🚀 Quick Start (Development):"
	@echo "  1. make install          # Install Go dependencies"
	@echo "  2. make dev-db           # Start infrastructure (DB, Redis, MailHog)"
	@echo "  3. make migrate-up       # Run database migrations"
	@echo "  4. make dev              # Start API with hot reload (Terminal 1)"
	@echo "  5. make dev-worker       # Start Worker (Terminal 2)"
	@echo ""
	@echo "📋 Available Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "📍 Service URLs:"
	@echo "  API:           http://localhost:8080"
	@echo "  MailHog UI:    http://localhost:8025"
	@echo "  Asynqmon:      http://localhost:8081"
	@echo "  MinIO Console: http://localhost:9001"

# ========================================
# SETUP
# ========================================
install: ## Install Go dependencies and Air
	@echo "📦 Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo ""
	@echo "🔥 Installing Air (hot reload tool)..."
	@if ! command -v air > /dev/null; then \
		go install github.com/air-verse/air@latest; \
		echo "✅ Air installed"; \
	else \
		echo "✅ Air already installed"; \
	fi
	@echo ""
	@echo "✅ Setup complete!"

# ========================================
# DEVELOPMENT - LOCAL (Hot Reload)
# ========================================
dev-db: ## Start infrastructure (PostgreSQL, Redis, MailHog, MinIO, Asynqmon)
	@echo "🐳 Starting infrastructure containers..."
	docker compose up -d postgres redis mailhog minio asynqmon
	@echo "⏳ Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "✅ Infrastructure started!"


dev: ## Start API server with hot reload (Air)
	@echo "🔥 Starting API server with hot reload..."
	$(shell go env GOPATH)/bin/air

dev-worker: ## Start background worker (local)
	@echo "🔋 Starting background worker..."
	$(GO) run cmd/worker/main.go

dev-stop: ## Stop all Docker containers
	@echo "🛑 Stopping Docker containers..."
	docker compose down
	@echo "✅ Containers stopped"

dev-logs: ## View Docker container logs
	docker compose logs -f

dev-all: ## Setup full development environment
	@echo "🚀 Setting up full development environment..."
	@make dev-db
	@sleep 3
	@make migrate-up
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "📝 Next steps:"
	@echo "  Terminal 1: make dev         # Start API with hot reload"
	@echo "  Terminal 2: make dev-worker  # Start Worker"
	@echo ""
	@echo "📍 Service URLs:"
	@echo "  API:        http://localhost:8080"
	@echo "  MailHog:    http://localhost:8025"
	@echo "  Asynqmon:   http://localhost:8081"

# ========================================
# RUNNING (Without hot reload)
# ========================================
run: ## Run API server (production mode)
	@echo "🚀 Starting API server..."
	$(GO) run cmd/api/main.go

run-worker: ## Run background worker (production mode)
	@echo "🔋 Starting background worker..."
	$(GO) run cmd/worker/main.go

build: ## Build API and Worker binaries
	@echo "🔨 Building binaries..."
	mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/api ./cmd/api
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/worker ./cmd/worker
	@echo "✅ Binaries built in $(BUILD_DIR)/"

# ========================================
# DOCKER (Production-like)
# ========================================
docker-build: ## Build Docker images (for production testing)
	@echo "🐳 Building Docker images..."
	docker compose -f docker-compose.prod.yml build
	@echo "✅ Images built successfully"

docker-up: ## Start all services in Docker (production-like)
	@echo "🐳 Starting all Docker services..."
	docker compose -f docker-compose.prod.yml up -d
	@echo "✅ All services started!"

docker-down: ## Stop all Docker containers
	@echo "🛑 Stopping Docker containers..."
	docker compose -f docker-compose.prod.yml down
	@echo "✅ Containers stopped"

docker-logs: ## View Docker logs (all services)
	docker compose -f docker-compose.prod.yml logs -f

docker-ps: ## List running Docker containers
	docker compose ps

# ========================================
# DATABASE MIGRATIONS
# ========================================
migrate-up: ## Run all database migrations
	@echo "📈 Running migrations..."
	migrate -path ./migrations -database "$(DB_URL)" up
	@echo "✅ Migrations applied"

migrate-down: ## Rollback last migration
	@echo "📉 Rolling back last migration..."
	migrate -path ./migrations -database "$(DB_URL)" down 1
	@echo "✅ Migration rolled back"

migrate-create: ## Create new migration (usage: make migrate-create name=add_users_table)
	@if [ -z "$(name)" ]; then \
		echo "❌ Error: name parameter required"; \
		echo "Usage: make migrate-create name=add_users_table"; \
		exit 1; \
	fi
	@echo "📝 Creating migration: $(name)"
	migrate create -ext sql -dir migrations -seq $(name)
	@echo "✅ Migration files created in migrations/"

migrate-version: ## Show current migration version
	migrate -path ./migrations -database "$(DB_URL)" version

# ========================================
# DATABASE UTILITIES
# ========================================
db-shell: ## Access PostgreSQL shell
	docker exec -it bookstore_postgres psql -U $(DB_USER) -d $(DB_NAME)

db-reset: ## Reset database (drop all + migrate + seed)
	@echo "⚠️  Resetting database..."
	@make clean-seed
	@echo "✅ Database reset complete!"

seed: ## Run all seed files
	@echo "🌱 Seeding database..."
	@for i in 1 2 3 4 5 6 7 8; do \
		make -s seed-$$i; \
	done
	@echo "✅ All seeds completed!"

clean-seed: ## Clear all data from database
	@echo "🗑️  Truncating all tables..."
	@$(PSQL) -c "TRUNCATE TABLE reviews, refund_requests, payment_webhook_logs, payment_transactions, order_status_history, promotion_usage, order_items, orders, promotions, warehouse_inventory, warehouses, books, addresses, authors, publishers, categories, users RESTART IDENTITY CASCADE;" 2>/dev/null || echo "⚠️  Some tables may not exist yet"
	@echo "✅ All data cleared!"

seed-1: ## Seed users data
	@$(PSQL) -f seeds/001_users_seed.sql 2>/dev/null

seed-2: ## Seed books data
	@$(PSQL) -f seeds/002_book.sql 2>/dev/null

seed-3: ## Seed inventory data
	@$(PSQL) -f seeds/003_inventory.sql 2>/dev/null

seed-4: ## Seed promotions data
	@$(PSQL) -f seeds/004_promotion.sql 2>/dev/null

seed-5: ## Seed orders data
	@$(PSQL) -f seeds/005_order.sql 2>/dev/null

seed-6: ## Seed payments data
	@$(PSQL) -f seeds/006_payment.sql 2>/dev/null

seed-7: ## Seed refunds data
	@$(PSQL) -f seeds/007_refund.sql 2>/dev/null

seed-8: ## Seed reviews data
	@$(PSQL) -f seeds/008_review.sql 2>/dev/null

# ========================================
# ASYNQ MONITORING
# ========================================
asynq-stats: ## Show Asynq queue statistics
	@echo "📊 Asynq Queue Statistics:"
	@asynq stats 2>/dev/null || echo "⚠️  asynq CLI not installed. Run: go install github.com/hibiken/asynq/tools/asynq@latest"

asynq-dashboard: ## Open Asynqmon dashboard
	@echo "📊 Opening Asynqmon dashboard..."
	@echo "URL: http://localhost:8081"

# ========================================
# TESTING
# ========================================
test: ## Run all tests
	@echo "🧪 Running tests..."
	go test ./tests/... -v

test-coverage: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	go test ./tests/... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

test-clean: ## Reset test database and run tests
	@echo "🧹 Cleaning test database..."
	@echo "🧪 Running tests..."
	go test ./tests/... -v -count=1

# ========================================
# CODE QUALITY
# ========================================
fmt: ## Format Go code
	@echo "🎨 Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Code formatted"

lint: ## Run linters
	@echo "🔍 Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
		echo "✅ Linting complete"; \
	else \
		echo "⚠️  golangci-lint not installed"; \
	fi

# ========================================
# CLEANUP
# ========================================
clean: ## Clean build artifacts and temp files
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf tmp/
	rm -f coverage.out coverage.html
	$(GO) clean
	@echo "✅ Cleaned"

clean-all: clean docker-down ## Clean everything including Docker
	@echo "🧹 Deep cleaning..."
	docker compose down -v
	@echo "✅ All cleaned (including Docker volumes)"

# ========================================
# DEFAULT GOAL
# ========================================
.DEFAULT_GOAL := help
