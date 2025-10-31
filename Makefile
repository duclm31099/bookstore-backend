.PHONY: help run build test migrate-up migrate-down migrate-create docker-up docker-down clean

# Variables
APP_NAME=bookstore-backend
VERSION=1.0.0
BUILD_DIR=bin
GO=go
GOFLAGS=-v

help: ## Show this help
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

install: ## Install dependencies
	$(GO) mod download
	$(GO) mod tidy

run: ## Run the application
	$(GO) run cmd/api/main.go

run-worker: ## Run background worker
	$(GO) run cmd/worker/main.go

build: ## Build the application
	mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/api cmd/api/main.go
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/worker cmd/worker/main.go
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/migrate cmd/migrate/main.go

test: ## Run tests
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

test-unit: ## Run unit tests only
	$(GO) test -v ./tests/unit/...

test-integration: ## Run integration tests
	$(GO) test -v ./tests/integration/...

# Database connection string
DB_URL=postgresql://bookstore:secret@localhost:5439/bookstore_dev?sslmode=disable

migrate-up: ## Run database migrations
	migrate -path ./migrations -database "$(DB_URL)" up

migrate-down: ## Rollback last migration
	migrate -path ./migrations -database "$(DB_URL)" down 1

migrate-down-all: ## Rollback all migrations
	migrate -path ./migrations -database "$(DB_URL)" down

migrate-create: ## Create new migration (usage: make migrate-create name=create_books_table)
	migrate create -ext sql -dir migrations -seq $(name)

migrate-version: ## Show current migration version
	migrate -path ./migrations -database "$(DB_URL)" version

migrate-force: ## Force migration version (usage: make migrate-force version=1)
	migrate -path ./migrations -database "$(DB_URL)" force $(version)


docker-up: ## Start Docker containers
	docker compose up -d

docker-down: ## Stop Docker containers
	docker compose down

docker-logs: ## View Docker logs
	docker compose logs -f

docker-ps: ## List running containers
	docker compose ps


seed: ## Seed database with initial data
	$(GO) run scripts/seed.go

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean

fmt: ## Format code
	$(GO) fmt ./...
	goimports -w .

lint: ## Run linters
	golangci-lint run

.DEFAULT_GOAL := help
