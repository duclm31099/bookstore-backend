# Bookstore Backend

This is the backend for the Bookstore application.

## Setup

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Run the API server:
   ```bash
   go run cmd/api/main.go
   ```

# Start all services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f

# View logs của 1 service cụ thể
docker compose logs -f postgres

# Stop all services
docker compose down

# Stop và xóa volumes (reset everything)
docker compose down -v

# Restart một service
docker compose restart postgres

# Test connect từ command line
psql -h localhost -p 5439 -U bookstore -d bookstore_dev
# Password: secret

docker compose ps

# Nếu postgres không healthy:
docker compose logs postgres
