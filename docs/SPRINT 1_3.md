<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# **BACKEND DEVELOPER TO-DO LIST**

## **Hệ Thống E-commerce Bán Sách Online**


***

## **PHASE 1: MVP - CORE E-COMMERCE (8 tuần)**

### **SPRINT 1: Project Foundation (Tuần 1-2)**

#### **Ngày 1-2: Project Setup**

**☐ Task 1.1: Khởi tạo Go Project**

- [ ] Tạo repository Git mới
- [ ] Init Go modules: `go mod init bookstore-backend`
- [ ] Tạo `.gitignore` (exclude `/vendor`, `.env`, `*.log`)
- [ ] Setup branch strategy (main, develop, feature/*)
- [ ] Tạo `README.md` với hướng dẫn setup

**☐ Task 1.2: Tạo Folder Structure**

```
bookstore-backend/
├── cmd/
│   ├── api/main.go
│   ├── worker/main.go
│   └── migrate/main.go
├── internal/
│   ├── domains/
│   │   ├── user/
│   │   ├── book/
│   │   └── order/
│   └── shared/
│       ├── middleware/
│       └── dto/
├── pkg/
│   ├── config/
│   ├── logger/
│   └── db/
├── migrations/
├── .env.example
└── go.mod
```

**☐ Task 1.3: Install Dependencies**

```bash
go get -u github.com/gin-gonic/gin
go get -u github.com/jackc/pgx/v5
go get -u github.com/redis/go-redis/v9
go get -u github.com/spf13/viper
go get -u github.com/rs/zerolog
go get -u github.com/golang-jwt/jwt/v5
go get -u github.com/go-ozzo/ozzo-validation/v4
go get -u github.com/google/uuid
go get -u golang.org/x/crypto/bcrypt
```


#### **Ngày 3-4: Configuration \& Logging**

**☐ Task 1.4: Setup Config Management**

File: `pkg/config/config.go`

```go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    App      AppConfig
    Database DBConfig
    Redis    RedisConfig
    JWT      JWTConfig
}

type AppConfig struct {
    Env  string
    Port string
}

type DBConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
    MaxConns int
}

// Thêm các config khác...

func Load() (*Config, error) {
    viper.SetConfigFile(".env")
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    cfg := &Config{
        App: AppConfig{
            Env:  viper.GetString("APP_ENV"),
            Port: viper.GetString("APP_PORT"),
        },
        // Map các config khác...
    }
    
    return cfg, nil
}
```

**☐ Task 1.5: Setup Logger**

File: `pkg/logger/logger.go`

```go
package logger

import (
    "os"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func Init(env string) {
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
    
    if env == "development" {
        log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
    }
    
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func Info(msg string, fields map[string]interface{}) {
    log.Info().Fields(fields).Msg(msg)
}

func Error(msg string, err error) {
    log.Error().Err(err).Msg(msg)
}
```

**☐ Task 1.6: Create .env File**

```bash
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=debug

DB_HOST=localhost
DB_PORT=5432
DB_USER=bookstore
DB_PASSWORD=secret
DB_NAME=bookstore_dev
DB_MAX_CONNECTIONS=25

REDIS_HOST=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=your-super-secret-key-change-in-production
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=72h
```


#### **Ngày 5-7: Database Setup**

**☐ Task 1.7: Install \& Setup PostgreSQL**

- [ ] Install PostgreSQL 15 (local hoặc Docker)
- [ ] Tạo database: `CREATE DATABASE bookstore_dev;`
- [ ] Tạo user: `CREATE USER bookstore WITH PASSWORD 'secret';`
- [ ] Grant permissions: `GRANT ALL PRIVILEGES ON DATABASE bookstore_dev TO bookstore;`

**☐ Task 1.8: Setup Database Connection Pool**

File: `pkg/db/postgres.go`

```go
package db

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(host, port, user, password, dbname string, maxConns int) (*pgxpool.Pool, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable pool_max_conns=%d",
        host, port, user, password, dbname, maxConns,
    )
    
    config, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, err
    }
    
    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        return nil, err
    }
    
    // Test connection
    if err := pool.Ping(context.Background()); err != nil {
        return nil, err
    }
    
    return pool, nil
}
```

**☐ Task 1.9: Setup Migration Tool**

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create migration helper script
```

File: `scripts/migrate.sh`

```bash
#!/bin/bash
migrate -path migrations -database "postgresql://bookstore:secret@localhost:5436/bookstore_dev?sslmode=disable" $1
```

**☐ Task 1.10: Create Core Tables Migration**

File: `migrations/000001_create_users_table.up.sql`

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL,
    phone TEXT,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin', 'warehouse', 'cskh')),
    points INT NOT NULL DEFAULT 0,
    is_verified BOOLEAN DEFAULT false,
    verification_token TEXT,
    verification_sent_at TIMESTAMPTZ,
    reset_token TEXT,
    reset_token_expires_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role);

-- Trigger auto update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

File: `migrations/000001_create_users_table.down.sql`

```sql
DROP TRIGGER IF EXISTS users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS users;
```

**☐ Task 1.11: Create Authors \& Publishers Tables**

File: `migrations/000002_create_authors_publishers.up.sql`

```sql
CREATE TABLE authors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    bio TEXT,
    photo_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_authors_slug ON authors(slug);

CREATE TABLE publishers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    website TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_publishers_slug ON publishers(slug);
```

**☐ Task 1.12: Create Categories Table**

File: `migrations/000003_create_categories.up.sql`

```sql
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    parent_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_categories_parent ON categories(parent_id);
CREATE INDEX idx_categories_slug ON categories(slug);
```

**☐ Task 1.13: Create Books Table**

File: `migrations/000004_create_books.up.sql`

```sql
CREATE TABLE books (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    author_id UUID NOT NULL REFERENCES authors(id),
    publisher_id UUID REFERENCES publishers(id),
    category_id UUID REFERENCES categories(id),
    isbn TEXT UNIQUE,
    price NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    compare_at_price NUMERIC(10,2),
    cover_url TEXT,
    description TEXT,
    pages INT,
    language TEXT DEFAULT 'vi',
    published_year INT,
    format TEXT CHECK (format IN ('paperback', 'hardcover', 'ebook')),
    is_active BOOLEAN DEFAULT true,
    view_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_books_slug ON books(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_books_author ON books(author_id);
CREATE INDEX idx_books_category ON books(category_id);
CREATE INDEX idx_books_isbn ON books(isbn) WHERE isbn IS NOT NULL;
CREATE INDEX idx_books_price ON books(price) WHERE is_active = true;

CREATE TRIGGER books_updated_at BEFORE UPDATE ON books
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

**☐ Task 1.14: Run Migrations**

```bash
chmod +x scripts/migrate.sh
./scripts/migrate.sh up
```


#### **Ngày 8-9: Redis Setup**

**☐ Task 1.15: Install \& Setup Redis**

- [ ] Install Redis 7 (local hoặc Docker)
- [ ] Start Redis: `redis-server`
- [ ] Test connection: `redis-cli ping`

**☐ Task 1.16: Create Redis Client**

File: `pkg/db/redis.go`

```go
package db

import (
    "context"
    "github.com/redis/go-redis/v9"
)

func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })
    
    // Test connection
    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, err
    }
    
    return client, nil
}
```


#### **Ngày 10: Basic Middlewares**

**☐ Task 1.17: Request ID Middleware**

File: `internal/shared/middleware/request_id.go`

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)
        c.Next()
    }
}
```

**☐ Task 1.18: Logger Middleware**

File: `internal/shared/middleware/logger.go`

```go
package middleware

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog/log"
)

func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        
        c.Next()
        
        latency := time.Since(start)
        status := c.Writer.Status()
        
        log.Info().
            Str("request_id", c.GetString("request_id")).
            Str("method", c.Request.Method).
            Str("path", path).
            Int("status", status).
            Dur("latency_ms", latency).
            Str("ip", c.ClientIP()).
            Msg("HTTP Request")
    }
}
```

**☐ Task 1.19: CORS Middleware**

File: `internal/shared/middleware/cors.go`

```go
package middleware

import (
    "github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}
```

**☐ Task 1.20: Panic Recovery Middleware**

File: `internal/shared/middleware/recovery.go`

```go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog/log"
)

func Recovery() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                log.Error().
                    Str("request_id", c.GetString("request_id")).
                    Interface("error", err).
                    Msg("Panic recovered")
                
                c.JSON(http.StatusInternalServerError, gin.H{
                    "success": false,
                    "error": gin.H{
                        "code":    "SYS_001",
                        "message": "Internal server error",
                    },
                })
                c.Abort()
            }
        }()
        
        c.Next()
    }
}
```

**☐ Task 1.21: Setup Main Server**

File: `cmd/api/main.go`

```go
package main

import (
    "log"
    "bookstore-backend/pkg/config"
    "bookstore-backend/pkg/logger"
    "bookstore-backend/pkg/db"
    "bookstore-backend/internal/shared/middleware"
    "github.com/gin-gonic/gin"
)

func main() {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Init logger
    logger.Init(cfg.App.Env)
    
    // Connect to PostgreSQL
    pgPool, err := db.NewPostgresPool(
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.User,
        cfg.Database.Password,
        cfg.Database.DBName,
        cfg.Database.MaxConns,
    )
    if err != nil {
        log.Fatal("Failed to connect to PostgreSQL:", err)
    }
    defer pgPool.Close()
    
    // Connect to Redis
    redisClient, err := db.NewRedisClient(
        cfg.Redis.Host,
        cfg.Redis.Password,
        cfg.Redis.DB,
    )
    if err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }
    defer redisClient.Close()
    
    // Setup Gin
    if cfg.App.Env == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    r := gin.New()
    
    // Apply middlewares
    r.Use(middleware.Recovery())
    r.Use(middleware.RequestID())
    r.Use(middleware.Logger())
    r.Use(middleware.CORS())
    
    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })
    
    // API routes
    v1 := r.Group("/v1")
    {
        // Routes sẽ được thêm ở sprint sau
    }
    
    // Start server
    if err := r.Run(":" + cfg.App.Port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
```

**☐ Task 1.22: Test Server**

```bash
go run cmd/api/main.go
# Test: curl http://localhost:8080/health
```


***

### **SPRINT 2: Authentication \& Authorization (Tuần 3-4)**

#### **Ngày 1-2: User Domain - Models \& DTOs**

**☐ Task 2.1: Create User Model**

File: `internal/domains/user/model/user.go`

```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID                   uuid.UUID  `json:"id"`
    Email                string     `json:"email"`
    PasswordHash         string     `json:"-"` // Never expose
    FullName             string     `json:"full_name"`
    Phone                *string    `json:"phone"`
    Role                 string     `json:"role"`
    Points               int        `json:"points"`
    IsVerified           bool       `json:"is_verified"`
    VerificationToken    *string    `json:"-"`
    VerificationSentAt   *time.Time `json:"-"`
    ResetToken           *string    `json:"-"`
    ResetTokenExpiresAt  *time.Time `json:"-"`
    LastLoginAt          *time.Time `json:"last_login_at"`
    CreatedAt            time.Time  `json:"created_at"`
    UpdatedAt            time.Time  `json:"updated_at"`
    DeletedAt            *time.Time `json:"-"`
}
```

**☐ Task 2.2: Create DTOs**

File: `internal/domains/user/dto/auth.go`

```go
package dto

import (
    validation "github.com/go-ozzo/ozzo-validation/v4"
    "github.com/go-ozzo/ozzo-validation/v4/is"
)

type RegisterRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    FullName string `json:"full_name"`
    Phone    string `json:"phone"`
}

func (r RegisterRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Email, validation.Required, is.Email),
        validation.Field(&r.Password, validation.Required, validation.Length(8, 100)),
        validation.Field(&r.FullName, validation.Required, validation.Length(2, 100)),
        validation.Field(&r.Phone, validation.Match(regexp.MustCompile(`^0[0-9]{9}$`))),
    )
}

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Email, validation.Required, is.Email),
        validation.Field(&r.Password, validation.Required),
    )
}

type AuthResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int64  `json:"expires_in"` // seconds
    User         UserResponse `json:"user"`
}

type UserResponse struct {
    ID         string `json:"id"`
    Email      string `json:"email"`
    FullName   string `json:"full_name"`
    Phone      string `json:"phone"`
    Role       string `json:"role"`
    IsVerified bool   `json:"is_verified"`
}
```


#### **Ngày 3-4: User Repository**

**☐ Task 2.3: Create Repository Interface**

File: `internal/domains/user/repository/repository.go`

```go
package repository

import (
    "context"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/user/model"
)

type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
    UpdateVerificationToken(ctx context.Context, userID uuid.UUID, token string) error
    VerifyEmail(ctx context.Context, token string) error
    UpdateResetToken(ctx context.Context, email string, token string, expiresAt time.Time) error
    ResetPassword(ctx context.Context, token string, newPasswordHash string) error
}
```

**☐ Task 2.4: Implement PostgreSQL Repository**

File: `internal/domains/user/repository/postgres.go`

```go
package repository

import (
    "context"
    "errors"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/user/model"
)

type postgresRepository struct {
    db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) UserRepository {
    return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, user *model.User) error {
    query := `
        INSERT INTO users (email, password_hash, full_name, phone, role, verification_token, verification_sent_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at, updated_at
    `
    
    return r.db.QueryRow(ctx, query,
        user.Email,
        user.PasswordHash,
        user.FullName,
        user.Phone,
        user.Role,
        user.VerificationToken,
        user.VerificationSentAt,
    ).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    query := `
        SELECT id, email, password_hash, full_name, phone, role, points, 
               is_verified, last_login_at, created_at, updated_at
        FROM users
        WHERE email = $1 AND deleted_at IS NULL
    `
    
    user := &model.User{}
    err := r.db.QueryRow(ctx, query, email).Scan(
        &user.ID,
        &user.Email,
        &user.PasswordHash,
        &user.FullName,
        &user.Phone,
        &user.Role,
        &user.Points,
        &user.IsVerified,
        &user.LastLoginAt,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, nil // User not found
        }
        return nil, err
    }
    
    return user, nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
    query := `
        SELECT id, email, password_hash, full_name, phone, role, points, 
               is_verified, last_login_at, created_at, updated_at
        FROM users
        WHERE id = $1 AND deleted_at IS NULL
    `
    
    user := &model.User{}
    err := r.db.QueryRow(ctx, query, id).Scan(
        &user.ID,
        &user.Email,
        &user.PasswordHash,
        &user.FullName,
        &user.Phone,
        &user.Role,
        &user.Points,
        &user.IsVerified,
        &user.LastLoginAt,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    
    return user, nil
}

func (r *postgresRepository) VerifyEmail(ctx context.Context, token string) error {
    query := `
        UPDATE users
        SET is_verified = true, verification_token = NULL, verification_sent_at = NULL
        WHERE verification_token = $1
        RETURNING id
    `
    
    var id uuid.UUID
    err := r.db.QueryRow(ctx, query, token).Scan(&id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return errors.New("invalid or expired token")
        }
        return err
    }
    
    return nil
}

// Implement các methods khác tương tự...
```


#### **Ngày 5-6: JWT Utilities \& Auth Service**

**☐ Task 2.5: Create JWT Helper**

File: `pkg/auth/jwt.go`

```go
package auth

import (
    "errors"
    "time"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type JWTClaims struct {
    UserID uuid.UUID `json:"user_id"`
    Email  string    `json:"email"`
    Role   string    `json:"role"`
    jwt.RegisteredClaims
}

type TokenPair struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int64
}

func GenerateTokenPair(userID uuid.UUID, email, role, secret string, accessExpiry, refreshExpiry time.Duration) (*TokenPair, error) {
    // Access Token
    accessClaims := JWTClaims{
        UserID: userID,
        Email:  email,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessTokenString, err := accessToken.SignedString([]byte(secret))
    if err != nil {
        return nil, err
    }
    
    // Refresh Token
    refreshClaims := JWTClaims{
        UserID: userID,
        Email:  email,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshTokenString, err := refreshToken.SignedString([]byte(secret))
    if err != nil {
        return nil, err
    }
    
    return &TokenPair{
        AccessToken:  accessTokenString,
        RefreshToken: refreshTokenString,
        ExpiresIn:    int64(accessExpiry.Seconds()),
    }, nil
}

func ValidateToken(tokenString, secret string) (*JWTClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return []byte(secret), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, errors.New("invalid token")
}
```

**☐ Task 2.6: Create Auth Service**

File: `internal/domains/user/service/auth_service.go`

```go
package service

import (
    "context"
    "errors"
    "time"
    "golang.org/x/crypto/bcrypt"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/user/model"
    "bookstore-backend/internal/domains/user/repository"
    "bookstore-backend/internal/domains/user/dto"
    "bookstore-backend/pkg/auth"
)

type AuthService struct {
    repo           repository.UserRepository
    jwtSecret      string
    accessExpiry   time.Duration
    refreshExpiry  time.Duration
}

func NewAuthService(repo repository.UserRepository, jwtSecret string, accessExpiry, refreshExpiry time.Duration) *AuthService {
    return &AuthService{
        repo:          repo,
        jwtSecret:     jwtSecret,
        accessExpiry:  accessExpiry,
        refreshExpiry: refreshExpiry,
    }
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) error {
    // Validate input
    if err := req.Validate(); err != nil {
        return err
    }
    
    // Check if email exists
    existing, err := s.repo.FindByEmail(ctx, req.Email)
    if err != nil {
        return err
    }
    if existing != nil {
        return errors.New("email already registered")
    }
    
    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    
    // Generate verification token
    verificationToken := uuid.New().String()
    
    // Create user
    user := &model.User{
        Email:              req.Email,
        PasswordHash:       string(hashedPassword),
        FullName:           req.FullName,
        Phone:              &req.Phone,
        Role:               "user",
        VerificationToken:  &verificationToken,
        VerificationSentAt: func() *time.Time { t := time.Now(); return &t }(),
    }
    
    if err := s.repo.Create(ctx, user); err != nil {
        return err
    }
    
    // TODO: Send verification email (will implement in background job)
    
    return nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
    // Validate input
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    // Find user
    user, err := s.repo.FindByEmail(ctx, req.Email)
    if err != nil {
        return nil, err
    }
    if user == nil {
        return nil, errors.New("invalid credentials")
    }
    
    // Check password
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        return nil, errors.New("invalid credentials")
    }
    
    // Generate tokens
    tokens, err := auth.GenerateTokenPair(
        user.ID,
        user.Email,
        user.Role,
        s.jwtSecret,
        s.accessExpiry,
        s.refreshExpiry,
    )
    if err != nil {
        return nil, err
    }
    
    // Update last login
    // TODO: Update last_login_at in background
    
    return &dto.AuthResponse{
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresIn:    tokens.ExpiresIn,
        User: dto.UserResponse{
            ID:         user.ID.String(),
            Email:      user.Email,
            FullName:   user.FullName,
            Phone:      *user.Phone,
            Role:       user.Role,
            IsVerified: user.IsVerified,
        },
    }, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
    return s.repo.VerifyEmail(ctx, token)
}

// Implement RefreshToken, ForgotPassword, ResetPassword...
```


#### **Ngày 7-8: Auth Handlers \& Routes**

**☐ Task 2.7: Create Auth Handler**

File: `internal/domains/user/handler/auth_handler.go`

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "bookstore-backend/internal/domains/user/service"
    "bookstore-backend/internal/domains/user/dto"
)

type AuthHandler struct {
    authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
    return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req dto.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "VAL_001",
                "message": "Invalid request body",
                "details": err.Error(),
            },
        })
        return
    }
    
    if err := h.authService.Register(c.Request.Context(), req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "AUTH_004",
                "message": err.Error(),
            },
        })
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "message": "Registration successful. Please check your email to verify your account.",
    })
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req dto.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "VAL_001",
                "message": "Invalid request body",
            },
        })
        return
    }
    
    resp, err := h.authService.Login(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "AUTH_001",
                "message": err.Error(),
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    resp,
    })
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
    token := c.Query("token")
    if token == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "VAL_001",
                "message": "Token is required",
            },
        })
        return
    }
    
    if err := h.authService.VerifyEmail(c.Request.Context(), token); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "AUTH_005",
                "message": err.Error(),
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Email verified successfully",
    })
}
```

**☐ Task 2.8: Create Auth Middleware**

File: `internal/shared/middleware/auth.go`

```go
package middleware

import (
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
    "bookstore-backend/pkg/auth"
)

func AuthRequired(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "error": gin.H{
                    "code":    "AUTH_002",
                    "message": "Authorization header required",
                },
            })
            c.Abort()
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "error": gin.H{
                    "code":    "AUTH_002",
                    "message": "Invalid authorization header format",
                },
            })
            c.Abort()
            return
        }
        
        claims, err := auth.ValidateToken(parts[1], jwtSecret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "error": gin.H{
                    "code":    "AUTH_002",
                    "message": "Invalid or expired token",
                },
            })
            c.Abort()
            return
        }
        
        // Set user info in context
        c.Set("user_id", claims.UserID.String())
        c.Set("user_email", claims.Email)
        c.Set("user_role", claims.Role)
        
        c.Next()
    }
}

func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole := c.GetString("user_role")
        
        allowed := false
        for _, role := range roles {
            if userRole == role {
                allowed = true
                break
            }
        }
        
        if !allowed {
            c.JSON(http.StatusForbidden, gin.H{
                "success": false,
                "error": gin.H{
                    "code":    "AUTH_003",
                    "message": "Insufficient permissions",
                },
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

**☐ Task 2.9: Setup Auth Routes**

File: `cmd/api/main.go` (cập nhật)

```go
// ... existing code ...

func main() {
    // ... existing setup ...
    
    // Initialize repositories
    userRepo := userRepository.NewPostgresRepository(pgPool)
    
    // Initialize services
    authService := userService.NewAuthService(
        userRepo,
        cfg.JWT.Secret,
        15*time.Minute,  // access token
        72*time.Hour,    // refresh token
    )
    
    // Initialize handlers
    authHandler := userHandler.NewAuthHandler(authService)
    
    // Routes
    v1 := r.Group("/v1")
    {
        auth := v1.Group("/auth")
        {
            auth.POST("/register", authHandler.Register)
            auth.POST("/login", authHandler.Login)
            auth.POST("/verify-email", authHandler.VerifyEmail)
        }
    }
    
    // ... start server ...
}
```


#### **Ngày 9-10: Testing \& Documentation**

**☐ Task 2.10: Create Unit Tests**

File: `internal/domains/user/service/auth_service_test.go`

```go
package service_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "bookstore-backend/internal/domains/user/dto"
    "bookstore-backend/internal/domains/user/service"
)

// Mock repository
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.User), args.Error(1)
}

// ... implement other methods ...

func TestAuthService_Register_Success(t *testing.T) {
    mockRepo := new(MockUserRepository)
    authService := service.NewAuthService(mockRepo, "secret", time.Minute, time.Hour)
    
    req := dto.RegisterRequest{
        Email:    "test@example.com",
        Password: "password123",
        FullName: "Test User",
        Phone:    "0123456789",
    }
    
    mockRepo.On("FindByEmail", mock.Anything, req.Email).Return(nil, nil)
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
    
    err := authService.Register(context.Background(), req)
    
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_EmailExists(t *testing.T) {
    mockRepo := new(MockUserRepository)
    authService := service.NewAuthService(mockRepo, "secret", time.Minute, time.Hour)
    
    req := dto.RegisterRequest{
        Email:    "existing@example.com",
        Password: "password123",
        FullName: "Test User",
        Phone:    "0123456789",
    }
    
    existingUser := &model.User{Email: "existing@example.com"}
    mockRepo.On("FindByEmail", mock.Anything, req.Email).Return(existingUser, nil)
    
    err := authService.Register(context.Background(), req)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "already registered")
}
```

**☐ Task 2.11: Test APIs với Postman/cURL**

```bash
# Register
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "full_name": "Test User",
    "phone": "0123456789"
  }'

# Login
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

**☐ Task 2.12: Create API Documentation**

- [ ] Tạo Postman collection cho tất cả auth endpoints
- [ ] Export collection to JSON
- [ ] Thêm example requests/responses
- [ ] Document error codes

***

### **SPRINT 3: Book Management (Tuần 5-6)**

#### **Ngày 1-2: Book Domain Setup**

**☐ Task 3.1: Create Book Models**

File: `internal/domains/book/model/book.go`

```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Book struct {
    ID              uuid.UUID  `json:"id"`
    Title           string     `json:"title"`
    Slug            string     `json:"slug"`
    AuthorID        uuid.UUID  `json:"author_id"`
    PublisherID     *uuid.UUID `json:"publisher_id"`
    CategoryID      *uuid.UUID `json:"category_id"`
    ISBN            *string    `json:"isbn"`
    Price           float64    `json:"price"`
    CompareAtPrice  *float64   `json:"compare_at_price"`
    CoverURL        *string    `json:"cover_url"`
    Description     *string    `json:"description"`
    Pages           *int       `json:"pages"`
    Language        string     `json:"language"`
    PublishedYear   *int       `json:"published_year"`
    Format          string     `json:"format"`
    IsActive        bool       `json:"is_active"`
    ViewCount       int        `json:"view_count"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
    DeletedAt       *time.Time `json:"-"`
    
    // Relationships (populated via joins)
    Author    *Author    `json:"author,omitempty"`
    Publisher *Publisher `json:"publisher,omitempty"`
    Category  *Category  `json:"category,omitempty"`
}

type Author struct {
    ID       uuid.UUID `json:"id"`
    Name     string    `json:"name"`
    Slug     string    `json:"slug"`
    Bio      *string   `json:"bio"`
    PhotoURL *string   `json:"photo_url"`
}

type Publisher struct {
    ID      uuid.UUID `json:"id"`
    Name    string    `json:"name"`
    Slug    string    `json:"slug"`
    Website *string   `json:"website"`
}

type Category struct {
    ID         uuid.UUID   `json:"id"`
    Name       string      `json:"name"`
    Slug       string      `json:"slug"`
    ParentID   *uuid.UUID  `json:"parent_id"`
    SortOrder  int         `json:"sort_order"`
    Children   []Category  `json:"children,omitempty"`
}
```

**☐ Task 3.2: Create Book DTOs**

File: `internal/domains/book/dto/book.go`

```go
package dto

import (
    validation "github.com/go-ozzo/ozzo-validation/v4"
)

type BookListRequest struct {
    Search      string  `form:"search"`
    CategoryID  string  `form:"category"`
    AuthorID    string  `form:"author"`
    PriceMin    float64 `form:"price_min"`
    PriceMax    float64 `form:"price_max"`
    Language    string  `form:"language"`
    Format      string  `form:"format"`
    SortBy      string  `form:"sort"` // price:asc, title:asc, created_at:desc
    Page        int     `form:"page"`
    Limit       int     `form:"limit"`
}

func (r *BookListRequest) SetDefaults() {
    if r.Page < 1 {
        r.Page = 1
    }
    if r.Limit < 1 || r.Limit > 100 {
        r.Limit = 20
    }
    if r.SortBy == "" {
        r.SortBy = "created_at:desc"
    }
}

type BookResponse struct {
    ID             string   `json:"id"`
    Title          string   `json:"title"`
    Slug           string   `json:"slug"`
    Author         AuthorBrief `json:"author"`
    Publisher      *PublisherBrief `json:"publisher"`
    Category       *CategoryBrief `json:"category"`
    ISBN           string   `json:"isbn"`
    Price          float64  `json:"price"`
    CompareAtPrice float64  `json:"compare_at_price"`
    Discount       float64  `json:"discount_percentage"`
    CoverURL       string   `json:"cover_url"`
    Description    string   `json:"description"`
    Pages          int      `json:"pages"`
    Language       string   `json:"language"`
    PublishedYear  int      `json:"published_year"`
    Format         string   `json:"format"`
    AverageRating  float64  `json:"average_rating"`
    ReviewCount    int      `json:"review_count"`
}

type AuthorBrief struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Slug string `json:"slug"`
}

type PublisherBrief struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type CategoryBrief struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Slug string `json:"slug"`
}

type BookListResponse struct {
    Books []BookResponse `json:"books"`
    Meta  PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}
```

**☐ Task 3.3: Seed Sample Data**

File: `seeds/001_sample_data.sql`

```sql
-- Insert authors
INSERT INTO authors (id, name, slug, bio) VALUES
('a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1', 'Nguyễn Nhật Ánh', 'nguyen-nhat-anh', 'Nhà văn nổi tiếng Việt Nam'),
('b2b2b2b2-b2b2-b2b2-b2b2-b2b2b2b2b2b2', 'Tô Hoài', 'to-hoai', 'Nhà văn Việt Nam'),
('c3c3c3c3-c3c3-c3c3-c3c3-c3c3c3c3c3c3', 'Paulo Coelho', 'paulo-coelho', 'Brazilian author');

-- Insert publishers
INSERT INTO publishers (id, name, slug) VALUES
('d4d4d4d4-d4d4-d4d4-d4d4-d4d4d4d4d4d4', 'NXB Trẻ', 'nxb-tre'),
('e5e5e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e5', 'NXB Kim Đồng', 'nxb-kim-dong');

-- Insert categories
INSERT INTO categories (id, name, slug, parent_id, sort_order) VALUES
('f6f6f6f6-f6f6-f6f6-f6f6-f6f6f6f6f6f6', 'Văn học', 'van-hoc', NULL, 1),
('g7g7g7g7-g7g7-g7g7-g7g7-g7g7g7g7g7g7', 'Tiểu thuyết', 'tieu-thuyet', 'f6f6f6f6-f6f6-f6f6-f6f6-f6f6f6f6f6f6', 1),
('h8h8h8h8-h8h8-h8h8-h8h8-h8h8h8h8h8h8', 'Thiếu nhi', 'thieu-nhi', NULL, 2);

-- Insert books
INSERT INTO books (id, title, slug, author_id, publisher_id, category_id, isbn, price, compare_at_price, description, pages, language, published_year, format, is_active) VALUES
(uuid_generate_v4(), 'Mắt Biếc', 'mat-biec', 'a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1', 'd4d4d4d4-d4d4-d4d4-d4d4-d4d4d4d4d4d4', 'g7g7g7g7-g7g7-g7g7-g7g7-g7g7g7g7g7g7', '9786041096578', 95000, 120000, 'Câu chuyện tình đầu của Ngạn và Hà Lan', 350, 'vi', 2015, 'paperback', true),
(uuid_generate_v4(), 'Cho Tôi Xin Một Vé Đi Tuổi Thơ', 'cho-toi-xin-mot-ve-di-tuoi-tho', 'a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1', 'd4d4d4d4-d4d4-d4d4-d4d4-d4d4d4d4d4d4', 'g7g7g7g7-g7g7-g7g7-g7g7-g7g7g7g7g7g7', '9786041020269', 80000, 100000, 'Hồi ức tuổi thơ đẹp đẽ', 280, 'vi', 2012, 'paperback', true),
(uuid_generate_v4(), 'Dế Mèn Phiêu Lưu Ký', 'de-men-phieu-luu-ky', 'b2b2b2b2-b2b2-b2b2-b2b2-b2b2b2b2b2b2', 'e5e5e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e5', 'h8h8h8h8-h8h8-h8h8-h8h8-h8h8h8h8h8h8', '9786042003456', 65000, NULL, 'Cuộc phiêu lưu của Dế Mèn', 220, 'vi', 1941, 'paperback', true);
```

Chạy seed:

```bash
psql -h localhost -U bookstore -d bookstore_dev -f seeds/001_sample_data.sql
```


#### **Ngày 3-5: Book Repository \& Service**

**☐ Task 3.4: Create Book Repository**

File: `internal/domains/book/repository/book_repository.go`

```go
package repository

import (
    "context"
    "fmt"
    "strings"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/book/model"
    "bookstore-backend/internal/domains/book/dto"
)

type BookRepository interface {
    List(ctx context.Context, req dto.BookListRequest) ([]model.Book, int64, error)
    FindByID(ctx context.Context, id uuid.UUID) (*model.Book, error)
    FindBySlug(ctx context.Context, slug string) (*model.Book, error)
    IncrementViewCount(ctx context.Context, id uuid.UUID) error
}

type postgresRepository struct {
    db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) BookRepository {
    return &postgresRepository{db: db}
}

func (r *postgresRepository) List(ctx context.Context, req dto.BookListRequest) ([]model.Book, int64, error) {
    // Build WHERE clause
    var conditions []string
    var args []interface{}
    argCounter := 1
    
    conditions = append(conditions, "b.deleted_at IS NULL AND b.is_active = true")
    
    if req.Search != "" {
        conditions = append(conditions, fmt.Sprintf("(b.title ILIKE $%d OR a.name ILIKE $%d)", argCounter, argCounter))
        args = append(args, "%"+req.Search+"%")
        argCounter++
    }
    
    if req.CategoryID != "" {
        conditions = append(conditions, fmt.Sprintf("b.category_id = $%d", argCounter))
        categoryUUID, _ := uuid.Parse(req.CategoryID)
        args = append(args, categoryUUID)
        argCounter++
    }
    
    if req.PriceMin > 0 {
        conditions = append(conditions, fmt.Sprintf("b.price >= $%d", argCounter))
        args = append(args, req.PriceMin)
        argCounter++
    }
    
    if req.PriceMax > 0 {
        conditions = append(conditions, fmt.Sprintf("b.price <= $%d", argCounter))
        args = append(args, req.PriceMax)
        argCounter++
    }
    
    whereClause := "WHERE " + strings.Join(conditions, " AND ")
    
    // Build ORDER BY
    orderBy := "ORDER BY b.created_at DESC"
    if req.SortBy != "" {
        parts := strings.Split(req.SortBy, ":")
        if len(parts) == 2 {
            column := parts[0]
            direction := strings.ToUpper(parts[1])
            orderBy = fmt.Sprintf("ORDER BY b.%s %s", column, direction)
        }
    }
    
    // Count query
    countQuery := fmt.Sprintf(`
        SELECT COUNT(*)
        FROM books b
        LEFT JOIN authors a ON b.author_id = a.id
        %s
    `, whereClause)
    
    var total int64
    err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
    if err != nil {
        return nil, 0, err
    }
    
    // Main query
    offset := (req.Page - 1) * req.Limit
    args = append(args, req.Limit, offset)
    
    query := fmt.Sprintf(`
        SELECT 
            b.id, b.title, b.slug, b.price, b.compare_at_price, b.cover_url,
            b.description, b.pages, b.language, b.published_year, b.format,
            a.id, a.name, a.slug,
            p.id, p.name, p.slug,
            c.id, c.name, c.slug
        FROM books b
        LEFT JOIN authors a ON b.author_id = a.id
        LEFT JOIN publishers p ON b.publisher_id = p.id
        LEFT JOIN categories c ON b.category_id = c.id
        %s
        %s
        LIMIT $%d OFFSET $%d
    `, whereClause, orderBy, argCounter, argCounter+1)
    
    rows, err := r.db.Query(ctx, query, args...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()
    
    var books []model.Book
    for rows.Next() {
        var book model.Book
        var author model.Author
        var publisher model.Publisher
        var category model.Category
        
        err := rows.Scan(
            &book.ID, &book.Title, &book.Slug, &book.Price, &book.CompareAtPrice,
            &book.CoverURL, &book.Description, &book.Pages, &book.Language,
            &book.PublishedYear, &book.Format,
            &author.ID, &author.Name, &author.Slug,
            &publisher.ID, &publisher.Name, &publisher.Slug,
            &category.ID, &category.Name, &category.Slug,
        )
        if err != nil {
            return nil, 0, err
        }
        
        book.Author = &author
        book.Publisher = &publisher
        book.Category = &category
        
        books = append(books, book)
    }
    
    return books, total, nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Book, error) {
    query := `
        SELECT 
            b.id, b.title, b.slug, b.isbn, b.price, b.compare_at_price, 
            b.cover_url, b.description, b.pages, b.language, b.published_year, 
            b.format, b.view_count, b.created_at,
            a.id, a.name, a.slug, a.bio,
            p.id, p.name, p.slug,
            c.id, c.name, c.slug
        FROM books b
        LEFT JOIN authors a ON b.author_id = a.id
        LEFT JOIN publishers p ON b.publisher_id = p.id
        LEFT JOIN categories c ON b.category_id = c.id
        WHERE b.id = $1 AND b.deleted_at IS NULL
    `
    
    var book model.Book
    var author model.Author
    var publisher model.Publisher
    var category model.Category
    
    err := r.db.QueryRow(ctx, query, id).Scan(
        &book.ID, &book.Title, &book.Slug, &book.ISBN, &book.Price,
        &book.CompareAtPrice, &book.CoverURL, &book.Description, &book.Pages,
        &book.Language, &book.PublishedYear, &book.Format, &book.ViewCount,
        &book.CreatedAt,
        &author.ID, &author.Name, &author.Slug, &author.Bio,
        &publisher.ID, &publisher.Name, &publisher.Slug,
        &category.ID, &category.Name, &category.Slug,
    )
    
    if err != nil {
        return nil, err
    }
    
    book.Author = &author
    book.Publisher = &publisher
    book.Category = &category
    
    return &book, nil
}

func (r *postgresRepository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE books SET view_count = view_count + 1 WHERE id = $1`
    _, err := r.db.Exec(ctx, query, id)
    return err
}
```

**☐ Task 3.5: Create Book Service**

File: `internal/domains/book/service/book_service.go`

```go
package service

import (
    "context"
    "math"
    "github.com/google/uuid"
    "bookstore-backend/internal/domains/book/model"
    "bookstore-backend/internal/domains/book/repository"
    "bookstore-backend/internal/domains/book/dto"
)

type BookService struct {
    repo repository.BookRepository
}

func NewBookService(repo repository.BookRepository) *BookService {
    return &BookService{repo: repo}
}

func (s *BookService) ListBooks(ctx context.Context, req dto.BookListRequest) (*dto.BookListResponse, error) {
    req.SetDefaults()
    
    books, total, err := s.repo.List(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Convert to response DTOs
    bookResponses := make([]dto.BookResponse, len(books))
    for i, book := range books {
        bookResponses[i] = s.toBookResponse(book)
    }
    
    totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
    
    return &dto.BookListResponse{
        Books: bookResponses,
        Meta: dto.PaginationMeta{
            Page:       req.Page,
            Limit:      req.Limit,
            Total:      total,
            TotalPages: totalPages,
        },
    }, nil
}

func (s *BookService) GetBookByID(ctx context.Context, id string) (*dto.BookResponse, error) {
    bookUUID, err := uuid.Parse(id)
    if err != nil {
        return nil, err
    }
    
    book, err := s.repo.FindByID(ctx, bookUUID)
    if err != nil {
        return nil, err
    }
    
    // Increment view count in background (non-blocking)
    go s.repo.IncrementViewCount(context.Background(), bookUUID)
    
    response := s.toBookResponse(*book)
    return &response, nil
}

func (s *BookService) toBookResponse(book model.Book) dto.BookResponse {
    resp := dto.BookResponse{
        ID:            book.ID.String(),
        Title:         book.Title,
        Slug:          book.Slug,
        Price:         book.Price,
        CoverURL:      *book.CoverURL,
        Language:      book.Language,
        Format:        book.Format,
    }
    
    if book.CompareAtPrice != nil && *book.CompareAtPrice > book.Price {
        resp.CompareAtPrice = *book.CompareAtPrice
        resp.Discount = ((*book.CompareAtPrice - book.Price) / *book.CompareAtPrice) * 100
    }
    
    if book.Author != nil {
        resp.Author = dto.AuthorBrief{
            ID:   book.Author.ID.String(),
            Name: book.Author.Name,
            Slug: book.Author.Slug,
        }
    }
    
    if book.Publisher != nil {
        resp.Publisher = &dto.PublisherBrief{
            ID:   book.Publisher.ID.String(),
            Name: book.Publisher.Name,
        }
    }
    
    if book.Category != nil {
        resp.Category = &dto.CategoryBrief{
            ID:   book.Category.ID.String(),
            Name: book.Category.Name,
            Slug: book.Category.Slug,
        }
    }
    
    // TODO: Calculate average rating from reviews table
    
    return resp
}
```


#### **Ngày 6-7: Book Handlers \& Routes**

**☐ Task 3.6: Create Book Handler**

File: `internal/domains/book/handler/book_handler.go`

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "bookstore-backend/internal/domains/book/service"
    "bookstore-backend/internal/domains/book/dto"
)

type BookHandler struct {
    bookService *service.BookService
}

func NewBookHandler(bookService *service.BookService) *BookHandler {
    return &BookHandler{bookService: bookService}
}

func (h *BookHandler) ListBooks(c *gin.Context) {
    var req dto.BookListRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "VAL_001",
                "message": "Invalid query parameters",
            },
        })
        return
    }
    
    resp, err := h.bookService.ListBooks(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "SYS_001",
                "message": "Failed to fetch books",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    resp.Books,
        "meta":    resp.Meta,
    })
}

func (h *BookHandler) GetBookByID(c *gin.Context) {
    id := c.Param("id")
    
    resp, err := h.bookService.GetBookByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error": gin.H{
                "code":    "RES_001",
                "message": "Book not found",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    resp,
    })
}
```

**☐ Task 3.7: Setup Book Routes**

File: `cmd/api/main.go` (cập nhật)

```go
// ... trong main() ...

// Initialize book repository & service
bookRepo := bookRepository.NewPostgresRepository(pgPool)
bookService := bookService.NewBookService(bookRepo)
bookHandler := bookHandler.NewBookHandler(bookService)

// Routes
v1 := r.Group("/v1")
{
    // Auth routes...
    
    // Book routes
    books := v1.Group("/books")
    {
        books.GET("", bookHandler.ListBooks)
        books.GET("/:id", bookHandler.GetBookByID)
    }
}
```


#### **Ngày 8-9: Categories \& Search**

**☐ Task 3.8: Create Category Repository**

File: `internal/domains/book/repository/category_repository.go`

```go
package repository

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "bookstore-backend/internal/domains/book/model"
)

type CategoryRepository interface {
    GetTree(ctx context.Context) ([]model.Category, error)
}

type categoryRepository struct {
    db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) CategoryRepository {
    return &categoryRepository{db: db}
}

func (r *categoryRepository) GetTree(ctx context.Context) ([]model.Category, error) {
    query := `
        SELECT id, name, slug, parent_id, sort_order
        FROM categories
        ORDER BY sort_order, name
    `
    
    rows, err := r.db.Query(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var categories []model.Category
    categoryMap := make(map[string]*model.Category)
    
    for rows.Next() {
        var cat model.Category
        err := rows.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.ParentID, &cat.SortOrder)
        if err != nil {
            return nil, err
        }
        
        cat.Children = []model.Category{}
        categoryMap[cat.ID.String()] = &cat
        categories = append(categories, cat)
    }
    
    // Build tree structure
    var tree []model.Category
    for _, cat := range categories {
        if cat.ParentID == nil {
            tree = append(tree, cat)
        } else {
            parent, exists := categoryMap[cat.ParentID.String()]
            if exists {
                parent.Children = append(parent.Children, cat)
            }
        }
    }
    
    return tree, nil
}
```

**☐ Task 3.9: Add Full-text Search**

File: `migrations/000005_add_fulltext_search.up.sql`

```sql
-- Add tsvector column
ALTER TABLE books ADD COLUMN search_vector tsvector;

-- Create index
CREATE INDEX idx_books_search_vector ON books USING GIN(search_vector);

-- Update existing rows
UPDATE books SET search_vector = 
    to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''));

-- Trigger to auto-update
CREATE OR REPLACE FUNCTION books_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', coalesce(NEW.title, '') || ' ' || coalesce(NEW.description, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER books_search_vector_trigger
BEFORE INSERT OR UPDATE ON books
FOR EACH ROW EXECUTE FUNCTION books_search_vector_update();
```

**☐ Task 3.10: Update Repository cho Full-text Search**

Cập nhật `List()` method trong `book_repository.go`:

```go
if req.Search != "" {
    // Use full-text search instead of ILIKE
    conditions = append(conditions, fmt.Sprintf("b.search_vector @@ plainto_tsquery('english', $%d)", argCounter))
    args = append(args, req.Search)
    argCounter++
}
```


#### **Ngày 10: Testing \& Optimization**

**☐ Task 3.11: Create Integration Tests**

File: `internal/domains/book/handler/book_handler_test.go`

```go
package handler_test

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestBookHandler_ListBooks(t *testing.T) {
    // Setup test database with testcontainers
    // Setup handler
    // Make request
    // Assert response
}
```

**☐ Task 3.12: Performance Testing**

```bash
# Test với k6 hoặc Apache Bench
ab -n 1000 -c 10 http://localhost:8080/v1/books
```

**☐ Task 3.13: Add Redis Caching cho Book List**

File: `internal/infrastructure/cache/book_cache.go`

```go
package cache

import (
    "context"
    "encoding/json"
    "time"
    "github.com/redis/go-redis/v9"
    "bookstore-backend/internal/domains/book/dto"
)

type BookCache struct {
    redis *redis.Client
}

func NewBookCache(redis *redis.Client) *BookCache {
    return &BookCache{redis: redis}
}

func (c *BookCache) GetBookList(ctx context.Context, key string) (*dto.BookListResponse, error) {
    data, err := c.redis.Get(ctx, "book:list:"+key).Result()
    if err != nil {
        return nil, err
    }
    
    var resp dto.BookListResponse
    if err := json.Unmarshal([]byte(data), &resp); err != nil {
        return nil, err
    }
    
    return &resp, nil
}

func (c *BookCache) SetBookList(ctx context.Context, key string, data *dto.BookListResponse, ttl time.Duration) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }
    
    return c.redis.Set(ctx, "book:list:"+key, jsonData, ttl).Err()
}
```


***

Vì nội dung quá dài, tôi đã tạo chi tiết cho **3 sprint đầu tiên (6 tuần)**. Bạn có muốn tôi tiếp tục với:

1. **Sprint 4-5: Cart \& Checkout** (tuần 7-10)
2. **Sprint 6-8: Admin Panel \& Payment** (tuần 11-16)
3. **Các phase tiếp theo**

Hay bạn cần tôi xuất toàn bộ nội dung hiện tại thành file Markdown để lưu trữ trước?

