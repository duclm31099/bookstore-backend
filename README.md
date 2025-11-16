# 🐳 Bước 1: Build Docker images
make docker-build
https://missav123.com/vi/actresses/Futaba%20Otani Futaba Otani
# 🚀 Bước 2: Start tất cả services
make docker-up



# 📊 Bước 3: Kiểm tra tất cả services đang chạy
docker compose ps

# Xem logs real-time (tất cả services):
docker compose logs -f


# Xem log Worker

docker compose logs -f --tail=100 api      # API logs
docker compose logs -f worker   # Worker logs
docker compose logs -f postgres # Database logs



# Test 4: Kiểm tra email trong MailHog
Mở trình duyệt: http://localhost:8025

# Test 5: Kiểm tra Asynq queue
Mở trình duyệt: http://localhost:8081


| Service       | URL                   | Mô tả                     |
| ------------- | --------------------- | ------------------------- |
| API Server    | http://localhost:8080 | Backend API endpoints     |
| MailHog UI    | http://localhost:8025 | Xem email test đã gửi     |
| Asynqmon      | http://localhost:8081 | Monitor background jobs   |
| MinIO Console | http://localhost:9001 | Object storage management |




# 🛑 Dừng tất cả services
make docker-down
# Hoặc: docker compose down


# 🔄 Restart toàn bộ hệ thống
make docker-restart


# 1. Worker có chạy không?
docker compose ps | grep worker

# 2. Xem log worker
docker compose logs worker

# 3. Kiểm tra Redis connection
docker exec -it bookstore_worker ping redis

# 4. Kiểm tra queue stats
docker exec -it bookstore_redis redis-cli
> KEYS asynq:*
