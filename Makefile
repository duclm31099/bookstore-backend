build:
	@bash scripts/build.sh

run-api:
	go run cmd/api/main.go

run-worker:
	go run cmd/worker/main.go

run-migrate:
	go run cmd/migrate/main.go
