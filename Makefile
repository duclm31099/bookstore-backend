.PHONY: clean-seed seed-1 seed-2 seed-3 seed-4 seed-5 seed-6 seed-7 seed-8 help run build test migrate-up migrate-down migrate-create docker-up docker-down clean dev dev-stop dev-logs dev-all dev-db


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
# Database connection info
DB_HOST ?= localhost
DB_PORT ?= 5439
DB_NAME ?= bookstore_dev
DB_USER ?= bookstore
DB_PASSWORD ?= secret
PSQL = PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1
clean: ## Clean build artifacts and temp files
	rm -rf $(BUILD_DIR)
	rm -rf tmp/
	rm -f coverage.out coverage.html
	$(GO) clean

.DEFAULT_GOAL := help
# Clean all data
clean-seed:
	@echo "🗑️  Truncating all tables..."
	@$(PSQL) -c "TRUNCATE TABLE reviews, refund_requests, payment_webhook_logs, payment_transactions, order_status_history, promotion_usage, order_items, orders, promotions, warehouse_inventory, warehouses, books, addresses, authors, publishers, categories, users RESTART IDENTITY CASCADE;"
	@echo "✅ All data cleared!"

# Run individual seed files
seed-1:
	@echo "🌱 Running: 001_users_seed.sql"
	@$(PSQL) -f seeds/001_users_seed.sql
	@echo "✅ Done!"

seed-2:
	@echo "🌱 Running: 002_book.sql"
	@$(PSQL) -f seeds/002_book.sql
	@echo "✅ Done!"

seed-3:
	@echo "🌱 Running: 003_inventory.sql"
	@$(PSQL) -f seeds/003_inventory.sql
	@echo "✅ Done!"

seed-4:
	@echo "🌱 Running: 004_promotion.sql"
	@$(PSQL) -f seeds/004_promotion.sql
	@echo "✅ Done!"

seed-5:
	@echo "🌱 Running: 005_order.sql"
	@$(PSQL) -f seeds/005_order.sql
	@echo "✅ Done!"

seed-6:
	@echo "🌱 Running: 006_payment.sql"
	@$(PSQL) -f seeds/006_payment.sql
	@echo "✅ Done!"

seed-7:
	@echo "🌱 Running: 007_refund.sql"
	@$(PSQL) -f seeds/007_refund.sql
	@echo "✅ Done!"

seed-8:
	@echo "🌱 Running: 008_review.sql"
	@$(PSQL) -f seeds/008_review.sql
	@echo "✅ Done!"