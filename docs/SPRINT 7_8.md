<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# TODO LIST CHI TIẾT CHO BACKEND DEVELOPER - SPRINT 7-8: ADMIN PANEL \& POLISH

Dựa trên URD, dưới đây là danh sách công việc chi tiết và đầy đủ cho backend developer trong Sprint 7-8 (Phase 1, 2 tuần - 14 ngày làm việc).[^1]

## 1. RBAC Middleware (P1-T029)

### Mô tả

Xây dựng middleware Role-Based Access Control để kiểm soát quyền truy cập dựa trên vai trò người dùng.[^1]

### Yêu cầu kỹ thuật

- Hỗ trợ các vai trò: `guest`, `user`, `seller`, `warehouse`, `cskh`, `admin`[^1]
- Middleware phải kiểm tra JWT token và role từ claims[^1]
- Trả về HTTP 403 với error code `AUTH003` khi không đủ quyền[^1]


### Công việc cụ thể

- Tạo file `internal/shared/middleware/rbac.go`[^1]
- Implement function `RequireRole(roles ...string) gin.HandlerFunc`[^1]
- Extract user info từ JWT middleware (P1-T012)[^1]
- Validate role của user có trong danh sách roles cho phép[^1]
- Log audit trail khi access denied[^1]
- Viết unit test với mock JWT claims[^1]


### Acceptance Criteria

- Chỉ user có role phù hợp mới truy cập được endpoint[^1]
- Trả về error response chuẩn khi unauthorized[^1]
- Test coverage ≥ 80%[^1]


### Dependencies

- P1-T012: JWT middleware[^1]


### Effort

2 ngày[^1]

***

## 2. Admin: Create/Edit/Delete Books (P1-T030)

### Mô tả

Xây dựng đầy đủ CRUD APIs cho admin quản lý sách.[^1]

### API Endpoints

- `POST /v1/admin/books` - Tạo sách mới[^1]
- `PUT /v1/admin/books/:id` - Cập nhật thông tin sách[^1]
- `DELETE /v1/admin/books/:id` - Xóa mềm sách (soft delete)[^1]
- `GET /v1/admin/books` - List books với filter nâng cao[^1]


### Database Schema

Làm việc với bảng `books`:[^1]

```sql
CREATE TABLE books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    ebook_file_url TEXT,
    is_active BOOLEAN DEFAULT true,
    search_vector tsvector,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
)
```


### Công việc cụ thể

#### 2.1 Create Book API

- Tạo handler `internal/domains/book/handler/admin_handler.go`[^1]
- Validate input với ozzo-validation:[^1]
    - Title: required, min 1, max 500 chars[^1]
    - ISBN: unique, format ISBN-10/13[^1]
    - Price: required, > 0[^1]
    - Author_id: required, exists in authors table[^1]
- Auto-generate slug từ title (transliterate tiếng Việt)[^1]
- Upload cover image lên S3/MinIO nếu có[^1]
- Insert vào database với transaction[^1]
- Auto update search_vector trigger[^1]


#### 2.2 Update Book API

- Validate book exists và chưa bị xóa[^1]
- Partial update (chỉ update fields được gửi lên)[^1]
- Re-generate slug nếu title thay đổi, check unique[^1]
- Update `updated_at` timestamp[^1]
- Log thay đổi vào audit_logs[^1]


#### 2.3 Delete Book API

- Soft delete: set `deleted_at = NOW()`[^1]
- Kiểm tra không có orders đang pending với sách này[^1]
- Giữ lại dữ liệu để audit[^1]


#### 2.4 List Books Admin

- Support filter: `?status=active/inactive&category_id=xxx&search=xxx`[^1]
- Pagination: `?page=1&limit=20`[^1]
- Sort: `?sort=created_at:desc`[^1]
- Include soft deleted records với flag `?include_deleted=true`[^1]


### Validation Rules (ozzo-validation)

```go
validation.ValidateStruct(&book,
    validation.Field(&book.Title, validation.Required, validation.Length(1, 500)),
    validation.Field(&book.ISBN, validation.Match(regexp.MustCompile(`^[0-9-]{10,17}$`))),
    validation.Field(&book.Price, validation.Required, validation.Min(0.0)),
    validation.Field(&book.Format, validation.In("paperback", "hardcover", "ebook")),
)
```


### Acceptance Criteria

- Admin tạo được sách với đầy đủ thông tin[^1]
- Update được từng field riêng lẻ[^1]
- Soft delete không ảnh hưởng data integrity[^1]
- Search_vector tự động cập nhật khi title/description thay đổi[^1]
- Audit log ghi lại mọi thay đổi[^1]


### Dependencies

- P1-T029: RBAC middleware[^1]


### Effort

3 ngày[^1]

***

## 3. Admin: View Orders (P1-T031)

### Mô tả

API cho admin xem danh sách và chi tiết đơn hàng của toàn hệ thống.[^1]

### API Endpoints

- `GET /v1/admin/orders` - List orders với filter nâng cao[^1]
- `GET /v1/admin/orders/:id` - Chi tiết order[^1]


### Database Schema

Bảng `orders` đã được partition theo tháng:[^1]

```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    order_number TEXT UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    status TEXT NOT NULL,
    payment_method TEXT NOT NULL,
    payment_status TEXT DEFAULT 'pending',
    total NUMERIC(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (created_at);
```


### Công việc cụ thể

#### 3.1 List Orders API

- Handler: `internal/domains/order/handler/admin_handler.go`[^1]
- Filter parameters:[^1]
    - `status`: pending, confirmed, processing, shipped, delivered, completed, cancelled, refunded[^1]
    - `payment_status`: pending, paid, failed, refunded[^1]
    - `payment_method`: cod, vnpay, momo[^1]
    - `date_range`: `?from=2025-10-01&to=2025-10-31`[^1]
    - `user_id`: filter theo user cụ thể[^1]
    - `order_number`: tìm theo mã đơn hàng[^1]
- Pagination: 20 items/page default[^1]
- Sort: `?sort=created_at:desc,total:asc`[^1]
- Response bao gồm:[^1]
    - Order summary info[^1]
    - User info (email, fullname)[^1]
    - Items count[^1]
    - Total value[^1]


#### 3.2 Order Detail API

- Include full information:[^1]
    - Order info (tất cả fields)[^1]
    - User info (join users table)[^1]
    - Shipping address (join addresses table)[^1]
    - Order items (join order_items + books)[^1]
    - Status history (join order_status_history)[^1]
    - Promotion info nếu có[^1]
- Query optimization với indexes:[^1]
    - `idx_orders_status`[^1]
    - `idx_orders_user`[^1]
    - `idx_orders_number`[^1]


### Response Format

```json
{
  "success": true,
  "data": {
    "orders": [...],
    "meta": {
      "page": 1,
      "limit": 20,
      "total": 150,
      "total_revenue": 50000000
    }
  }
}
```


### Acceptance Criteria

- Admin xem được tất cả orders trong hệ thống[^1]
- Filter chính xác theo nhiều tiêu chí[^1]
- Performance: P95 < 200ms với 10,000 orders[^1]
- Export được data sang CSV (optional)[^1]


### Dependencies

- P1-T029: RBAC middleware[^1]


### Effort

1 ngày[^1]

***

## 4. Admin: Update Order Status (P1-T032)

### Mô tả

API cho admin cập nhật trạng thái đơn hàng và xử lý workflow.[^1]

### API Endpoint

- `PATCH /v1/admin/orders/:id/status`[^1]


### Request Body

```json
{
  "status": "confirmed",
  "note": "Đã xác nhận đơn hàng",
  "tracking_number": "GHN123456789",
  "shipping_provider": "GHN"
}
```


### Order Status Flow

```
pending → confirmed → processing → shipped → delivered → completed
       ↘ cancelled (before processing)
       ↘ payment_failed
```


### Công việc cụ thể

#### 4.1 Status Validation

- Implement status transition rules:[^1]
    - pending → confirmed, cancelled, payment_failed[^1]
    - confirmed → processing, cancelled[^1]
    - processing → shipped[^1]
    - shipped → delivered, return_requested[^1]
    - delivered → completed[^1]
- Validate không cho phép transition không hợp lệ[^1]


#### 4.2 Business Logic

- **confirmed**: Trừ inventory reserved[^1]
- **cancelled**: Release inventory reserved[^1]
- **shipped**: Required tracking_number và shipping_provider[^1]
- **completed**: Trigger review request email[^1]
- **refunded**: Trigger refund process job[^1]


#### 4.3 Audit Trail

- Insert vào `order_status_history`:[^1]

```sql
INSERT INTO order_status_history 
(order_id, old_status, new_status, note, changed_by, created_at)
VALUES (?, ?, ?, ?, ?, NOW());
```


#### 4.4 Background Jobs

- Trigger Asynq jobs sau khi update:[^1]
    - `SendOrderConfirmation` khi status = confirmed[^1]
    - `ProcessRefund` khi status = refunded[^1]
    - `SendShippingNotification` khi status = shipped[^1]


### Acceptance Criteria

- Chỉ admin mới update được status[^1]
- Validate status transition đúng flow[^1]
- Inventory được update đồng bộ[^1]
- Audit log ghi lại đầy đủ[^1]
- Background job được trigger tự động[^1]


### Dependencies

- P1-T029: RBAC middleware[^1]
- P1-T031: Admin view orders[^1]
- P1-T033: Email templates (để trigger notification)[^1]


### Effort

2 ngày[^1]

***

## 5. Email Templates + Sender (AWS SES) (P1-T033)

### Mô tả

Setup AWS SES và tạo email templates cho các sự kiện trong hệ thống.[^1]

### Email Templates Required

1. **Order Confirmation** - Khi order created/confirmed[^1]
2. **Payment Reminder** - Khi order pending > 10 phút[^1]
3. **Shipping Notification** - Khi order shipped[^1]
4. **Delivery Confirmation** - Khi order delivered[^1]
5. **Low Stock Alert** - Gửi admin khi stock < threshold[^1]

### AWS SES Configuration

#### 5.1 Environment Variables

```bash
AWS_SES_REGION=ap-southeast-1
AWS_SES_FROM_EMAIL=noreply@bookstore.com
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx
```


#### 5.2 Infrastructure Setup

- Tạo file `internal/infrastructure/email/ses.go`[^1]
- Implement interface:[^1]

```go
type EmailService interface {
    SendOrderConfirmation(to string, orderID string, orderData OrderData) error
    SendLowStockAlert(to string, bookID string, stockData StockData) error
}
```


#### 5.3 AWS SES Client

```go
import "github.com/aws/aws-sdk-go/service/ses"

func NewSESClient() *ses.SES {
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String(os.Getenv("AWS_SES_REGION")),
    }))
    return ses.New(sess)
}
```


### Công việc cụ thể

#### 5.4 Template Files

- Tạo folder `templates/emails/`[^1]
- Template HTML với Go template syntax:[^1]
    - `order_confirmation.html`[^1]
    - `payment_reminder.html`[^1]
    - `shipping_notification.html`[^1]
    - `low_stock_alert.html`[^1]


#### 5.5 Template Variables

**Order Confirmation Template**:

```go
type OrderConfirmationData struct {
    OrderNumber   string
    CustomerName  string
    Items         []OrderItem
    Total         string
    ShippingAddr  string
    PaymentMethod string
}
```


#### 5.6 Email Service Implementation

- Parse HTML template[^1]
- Inject data vào template[^1]
- Send via SES với retry logic (3 attempts)[^1]
- Log email sending status[^1]
- Handle SES errors (rate limit, bounce, complaint)[^1]


#### 5.7 Testing

- Mock SES client cho unit test[^1]
- Test với SES Sandbox environment[^1]
- Verify email được format đúng[^1]


### Acceptance Criteria

- Email templates responsive trên mobile \& desktop[^1]
- Gửi thành công qua AWS SES[^1]
- Retry khi failed, log errors[^1]
- Test coverage ≥ 70%[^1]
- Email delivery time < 30s[^1]


### Dependencies

- None (độc lập)[^1]


### Effort

2 ngày[^1]

***

## 6. Send Order Confirmation Email (P1-T034)

### Mô tả

Tích hợp email service vào order flow để gửi email xác nhận tự động.[^1]

### Integration Points

- Sau khi order created thành công (COD)[^1]
- Sau khi payment confirmed (VNPay/Momo)[^1]
- Khi admin update status sang "confirmed"[^1]


### Công việc cụ thể

#### 6.1 Asynq Job Setup

- Tạo job `SendOrderConfirmation` trong Asynq[^1]
- Priority: **High**[^1]
- Timeout: 30s[^1]
- Retry: 3 lần[^1]

```go
// Job payload
type SendOrderConfirmationPayload struct {
    OrderID string `json:"order_id"`
}

// Handler
func (h *EmailHandler) SendOrderConfirmation(ctx context.Context, t *asynq.Task) error {
    var p SendOrderConfirmationPayload
    json.Unmarshal(t.Payload(), &p)
    
    // Fetch order details
    order := h.orderRepo.FindByID(ctx, p.OrderID)
    
    // Send email
    err := h.emailService.SendOrderConfirmation(order.User.Email, order.OrderNumber, order)
    return err
}
```


#### 6.2 Trigger Points

**Trong Order Service**:

```go
// Sau khi create order
client := asynq.NewClient(redisOpt)
task := asynq.NewTask("email:order_confirmation", payload)
client.Enqueue(task, asynq.Queue("high"))
```

**Trong Admin Update Status**:

```go
if newStatus == "confirmed" {
    // Enqueue email job
}
```


#### 6.3 Email Content

- Order number, date[^1]
- Customer info[^1]
- Items list (book title, quantity, price)[^1]
- Subtotal, discount, shipping, total[^1]
- Shipping address[^1]
- Payment method[^1]
- Tracking link (nếu có)[^1]
- Customer support contact[^1]


### Acceptance Criteria

- Email gửi trong 30s sau order created[^1]
- Hiển thị đầy đủ thông tin order[^1]
- Link valid 24h[^1]
- Failed job được retry, log error[^1]


### Dependencies

- P1-T033: Email templates + AWS SES[^1]
- P1-T025: Order creation API[^1]


### Effort

1 ngày[^1]

***

## 7. Rate Limiting Middleware (P1-T035)

### Mô tả

Implement rate limiting middleware để bảo vệ APIs khỏi abuse và DDoS.[^1]

### Rate Limit Rules

| Endpoint Type | Limit | Window |
| :-- | :-- | :-- |
| Login/Register | 5 requests | 1 minute |
| Checkout | 10 requests | 1 minute |
| General API | 100 requests | 1 minute |
| Admin API | 1000 requests | 1 minute |

### Technology

- Redis-based rate limiting[^1]
- Algorithm: **Token Bucket** hoặc **Sliding Window**[^1]


### Công việc cụ thể

#### 7.1 Middleware Implementation

- Tạo file `internal/shared/middleware/rate_limiter.go`[^1]
- Sử dụng Redis INCR và EXPIRE[^1]

```go
func RateLimiter(limit int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Key: IP hoặc UserID
        key := fmt.Sprintf("ratelimit:%s:%s", c.ClientIP(), c.FullPath())
        
        // Redis INCR
        count, _ := redisClient.Incr(ctx, key).Result()
        if count == 1 {
            redisClient.Expire(ctx, key, window)
        }
        
        if count > limit {
            c.JSON(429, gin.H{
                "error": "Rate limit exceeded. Please try again later.",
            })
            c.Abort()
            return
        }
        
        // Set headers
        c.Header("X-RateLimit-Limit", fmt.Sprint(limit))
        c.Header("X-RateLimit-Remaining", fmt.Sprint(limit - count))
        
        c.Next()
    }
}
```


#### 7.2 Apply to Routes

```go
// Auth routes
authGroup := r.Group("/v1/auth")
authGroup.Use(RateLimiter(5, time.Minute))
{
    authGroup.POST("/login", handler.Login)
    authGroup.POST("/register", handler.Register)
}

// Checkout
r.POST("/v1/orders", RateLimiter(10, time.Minute), handler.CreateOrder)

// Admin
adminGroup := r.Group("/v1/admin")
adminGroup.Use(RateLimiter(1000, time.Minute))
```


#### 7.3 Redis Key Design

- Key pattern: `ratelimit:{ip}:{endpoint}:{window}`[^1]
- TTL = window duration[^1]
- Auto cleanup bởi Redis expiration[^1]


#### 7.4 Response Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1698765432
Retry-After: 60 (khi 429)
```


### Acceptance Criteria

- Requests vượt limit bị chặn với HTTP 429[^1]
- Reset counter sau window time[^1]
- Headers hiển thị đúng limit info[^1]
- Performance overhead < 5ms[^1]
- Test với concurrent requests[^1]


### Dependencies

- P1-T008: Basic middlewares[^1]
- P1-T004: Redis setup[^1]


### Effort

1 ngày[^1]

***

## 8. Input Validation (ozzo-validation) (P1-T036)

### Mô tả

Áp dụng ozzo-validation cho tất cả API inputs để đảm bảo data integrity.[^1]

### Library

- **ozzo-validation v4**: Flexible, no struct tags required[^1]


### Công việc cụ thể

#### 8.1 Validation Package Setup

- Tạo folder `pkg/validator/`[^1]
- Custom validation rules:[^1]
    - ISBN format validator[^1]
    - Vietnamese phone number (+84)[^1]
    - Price range validator[^1]
    - Slug format validator[^1]

```go
package validator

import (
    "github.com/go-ozzo/ozzo-validation/v4"
    "github.com/go-ozzo/ozzo-validation/v4/is"
)

// Custom validators
var (
    ISBN = validation.NewStringRule(isISBN, "must be valid ISBN-10 or ISBN-13")
    VietnamesePhone = validation.Match(regexp.MustCompile(`^(\+84|0)[0-9]{9}$`))
)
```


#### 8.2 Apply Validation to DTOs

**Create Book DTO**:

```go
type CreateBookRequest struct {
    Title       string  `json:"title"`
    AuthorID    string  `json:"author_id"`
    ISBN        string  `json:"isbn"`
    Price       float64 `json:"price"`
    Format      string  `json:"format"`
}

func (r CreateBookRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.Title, 
            validation.Required, 
            validation.Length(1, 500)),
        validation.Field(&r.AuthorID, 
            validation.Required, 
            is.UUIDv4),
        validation.Field(&r.ISBN, 
            validation.When(r.ISBN != "", ISBN)),
        validation.Field(&r.Price, 
            validation.Required, 
            validation.Min(0.0), 
            validation.Max(100000000.0)),
        validation.Field(&r.Format, 
            validation.In("paperback", "hardcover", "ebook")),
    )
}
```

**Order Creation DTO**:

```go
type CreateOrderRequest struct {
    AddressID string `json:"address_id"`
    PaymentMethod string `json:"payment_method"`
    PromoCode string `json:"promo_code"`
}

func (r CreateOrderRequest) Validate() error {
    return validation.ValidateStruct(&r,
        validation.Field(&r.AddressID, validation.Required, is.UUIDv4),
        validation.Field(&r.PaymentMethod, validation.In("cod", "vnpay", "momo")),
        validation.Field(&r.PromoCode, validation.Length(0, 50)),
    )
}
```


#### 8.3 Middleware Integration

```go
func (h *Handler) CreateBook(c *gin.Context) {
    var req CreateBookRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse("VAL001", "Invalid JSON"))
        return
    }
    
    // Validate
    if err := req.Validate(); err != nil {
        c.JSON(400, ErrorResponse("VAL001", err.Error()))
        return
    }
    
    // Process...
}
```


#### 8.4 SQL Injection Prevention

- Sử dụng **parameterized queries** exclusively[^1]
- Không concat SQL strings[^1]
- Validate input trước khi query[^1]

```go
// Good ✓
db.Exec("SELECT * FROM books WHERE id = $1", bookID)

// Bad ✗
db.Exec(fmt.Sprintf("SELECT * FROM books WHERE id = '%s'", bookID))
```


#### 8.5 XSS Prevention

- Sanitize HTML input nếu cho phép rich text[^1]
- Escape output trong templates[^1]


### Validation Coverage

Áp dụng cho tất cả endpoints:[^1]

- Auth: register, login, reset password[^1]
- Book: create, update[^1]
- Order: create, update status[^1]
- User: update profile, address[^1]
- Admin: all CRUD operations[^1]


### Acceptance Criteria

- Tất cả inputs được validate trước khi xử lý[^1]
- Error messages rõ ràng, specific[^1]
- SQL injection tests pass[^1]
- XSS tests pass[^1]


### Dependencies

- None (áp dụng rộng khắp)[^1]


### Effort

2 ngày[^1]

***

## 9. Unit Tests (Coverage > 60%) (P1-T037)

### Mô tả

Viết unit tests cho tất cả components với target coverage > 60%.[^1]

### Testing Strategy

- **Tools**: Go testing package, testify/assert, testify/mock[^1]
- **Coverage target**: 60% (Phase 1), 90% (Phase 4)[^1]


### Công việc cụ thể

#### 9.1 Test Structure

```
tests/
  unit/
    domains/
      book/
        service_test.go
        repository_test.go
      order/
        service_test.go
      user/
        service_test.go
    shared/
      middleware_test.go
      validator_test.go
```


#### 9.2 Service Layer Tests

**Book Service Test**:

```go
func TestBookService_Create(t *testing.T) {
    // Setup
    mockRepo := new(MockBookRepository)
    service := NewBookService(mockRepo)
    
    // Test data
    book := &Book{
        Title: "Test Book",
        Price: 100000,
    }
    
    // Mock expectations
    mockRepo.On("Create", mock.Anything, book).Return(nil)
    
    // Execute
    err := service.Create(context.Background(), book)
    
    // Assert
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}

func TestBookService_Create_ValidationError(t *testing.T) {
    service := NewBookService(nil)
    
    book := &Book{
        Title: "", // Invalid
        Price: -100, // Invalid
    }
    
    err := service.Create(context.Background(), book)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "validation")
}
```


#### 9.3 Repository Tests

- Sử dụng **testcontainers-go** để spin up PostgreSQL[^1]
- Test với real database queries[^1]

```go
func TestBookRepository_Create(t *testing.T) {
    // Setup container
    ctx := context.Background()
    postgresC, _ := postgres.RunContainer(ctx)
    defer postgresC.Terminate(ctx)
    
    // Connect DB
    db := connectDB(postgresC.ConnectionString())
    repo := NewBookRepository(db)
    
    // Test
    book := &Book{Title: "Test", Price: 100000}
    err := repo.Create(ctx, book)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, book.ID)
}
```


#### 9.4 Handler Tests

- Sử dụng `httptest.NewRecorder()`[^1]
- Mock service layer[^1]

```go
func TestBookHandler_Create(t *testing.T) {
    // Setup
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    
    mockService := new(MockBookService)
    handler := NewBookHandler(mockService)
    
    // Request body
    reqBody := `{"title":"Test Book","price":100000}`
    c.Request = httptest.NewRequest("POST", "/v1/admin/books", strings.NewReader(reqBody))
    c.Request.Header.Set("Content-Type", "application/json")
    
    // Mock
    mockService.On("Create", mock.Anything, mock.Anything).Return(nil)
    
    // Execute
    handler.Create(c)
    
    // Assert
    assert.Equal(t, 201, w.Code)
}
```


#### 9.5 Middleware Tests

**RBAC Middleware Test**:

```go
func TestRBACMiddleware_AdminOnly(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    
    r.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
        c.JSON(200, gin.H{"ok": true})
    })
    
    // Test 1: Admin user
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/admin", nil)
    req.Header.Set("Authorization", "Bearer "+adminToken)
    r.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
    
    // Test 2: Regular user
    w = httptest.NewRecorder()
    req = httptest.NewRequest("GET", "/admin", nil)
    req.Header.Set("Authorization", "Bearer "+userToken)
    r.ServeHTTP(w, req)
    assert.Equal(t, 403, w.Code)
}
```


#### 9.6 Coverage Report

```bash
# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage
go tool cover -func=coverage.out

# HTML report
go tool cover -html=coverage.out -o coverage.html
```


#### 9.7 CI Integration

- GitHub Actions run tests tự động[^1]
- Fail build nếu coverage < 60%[^1]

```yaml
- name: Run tests
  run: go test -v -race -coverprofile=coverage.out ./...
  
- name: Check coverage
  run: |
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$coverage < 60" | bc -l) )); then
      echo "Coverage $coverage% is below 60%"
      exit 1
    fi
```


### Test Categories

1. **Service layer** (business logic): 70% coverage target[^1]
2. **Repository layer** (database): 60% coverage target[^1]
3. **Handler layer** (HTTP): 50% coverage target[^1]
4. **Middleware**: 80% coverage target[^1]
5. **Utils/helpers**: 70% coverage target[^1]

### Acceptance Criteria

- Overall coverage > 60%[^1]
- Tất cả critical paths được test[^1]
- Tests pass trên CI/CD[^1]
- No flaky tests[^1]
- Test execution time < 5 minutes[^1]


### Dependencies

- All above tasks (P1-T001 → P1-T036)[^1]


### Effort

3 ngày[^1]

***

## 10. Dockerize Application (P1-T038)

### Mô tả

Containerize ứng dụng với Docker để dễ dàng deploy.[^1]

### Deliverables

- Dockerfile cho API server[^1]
- Dockerfile cho Worker[^1]
- docker-compose.yml cho local development[^1]


### Công việc cụ thể

#### 10.1 Dockerfile cho API

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary từ builder
COPY --from=builder /app/api .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy email templates
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./api"]
```


#### 10.2 Dockerfile cho Worker

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/worker .
COPY --from=builder /app/templates ./templates

CMD ["./worker"]
```


#### 10.3 docker-compose.yml

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: bookstore
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: bookstore_dev
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bookstore"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s

  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=development
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=bookstore
      - DB_PASSWORD=secret
      - DB_NAME=bookstore_dev
      - REDIS_HOST=redis:6379
      - JWT_SECRET=dev-secret-key
      - AWS_SES_REGION=ap-southeast-1
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./templates:/root/templates

  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    environment:
      - APP_ENV=development
      - DB_HOST=postgres
      - REDIS_HOST=redis:6379
      - AWS_SES_REGION=ap-southeast-1
    depends_on:
      - postgres
      - redis
    volumes:
      - ./templates:/root/templates

volumes:
  pgdata:
  redisdata:
```


#### 10.4 .dockerignore

```
.git
.github
*.md
coverage.out
*.log
tmp/
vendor/
.env
```


#### 10.5 Optimization

- Multi-stage build để giảm image size[^1]
- Image size target: < 50MB cho API, < 40MB cho worker[^1]
- Build time: < 2 minutes[^1]


#### 10.6 Health Check

```go
// internal/api/health.go
func (h *Handler) Health(c *gin.Context) {
    // Check DB
    if err := h.db.Ping(); err != nil {
        c.JSON(503, gin.H{"status": "unhealthy", "db": "down"})
        return
    }
    
    // Check Redis
    if err := h.redis.Ping(context.Background()).Err(); err != nil {
        c.JSON(503, gin.H{"status": "unhealthy", "redis": "down"})
        return
    }
    
    c.JSON(200, gin.H{"status": "healthy"})
}
```


#### 10.7 Running Instructions

```bash
# Build images
docker-compose build

# Run all services
docker-compose up -d

# Check logs
docker-compose logs -f api

# Run migrations
docker-compose exec api ./api migrate up

# Stop all
docker-compose down
```


#### 10.8 Production Considerations

- Environment-specific docker-compose files[^1]
- Secrets management (không hardcode trong Dockerfile)[^1]
- Resource limits (memory, CPU)[^1]
- Logging driver configuration[^1]


### Acceptance Criteria

- Docker image builds thành công[^1]
- `docker-compose up` chạy được toàn bộ stack[^1]
- API accessible tại `http://localhost:8080`[^1]
- Health check endpoint trả về 200[^1]
- Image size < 500MB (constraint từ URD)[^1]
- README có hướng dẫn build và run[^1]


### Dependencies

- P1-T001: Project setup[^1]


### Effort

1 ngày[^1]

***

## SUMMARY

### Total Effort Sprint 7-8

| Task ID | Task | Effort (days) |
| :-- | :-- | :-- |
| P1-T029 | RBAC middleware | 2 |
| P1-T030 | Admin Create/Edit/Delete books | 3 |
| P1-T031 | Admin View orders | 1 |
| P1-T032 | Admin Update order status | 2 |
| P1-T033 | Email templates + AWS SES | 2 |
| P1-T034 | Send order confirmation email | 1 |
| P1-T035 | Rate limiting middleware | 1 |
| P1-T036 | Input validation (ozzo-validation) | 2 |
| P1-T037 | Unit tests (coverage > 60%) | 3 |
| P1-T038 | Dockerize application | 1 |
| **TOTAL** |  | **18 days** |

**Sprint duration**: 2 tuần (10 ngày làm việc)[^1]
**Team size**: 2 backend developers (có thể song song hóa tasks)[^1]

### Parallelization Strategy

- **Week 1**:
    - Dev 1: P1-T029, P1-T030 (5 days)[^1]
    - Dev 2: P1-T033, P1-T035, P1-T036 (5 days)[^1]
- **Week 2**:
    - Dev 1: P1-T031, P1-T032, P1-T038 (4 days)[^1]
    - Dev 2: P1-T034, P1-T037 (4 days)[^1]


### Deliverables Checklist Sprint 7-8

- ✅ Admin có thể quản lý sách (CRUD)[^1]
- ✅ Admin có thể xem và cập nhật orders[^1]
- ✅ Email notifications hoạt động (AWS SES)[^1]
- ✅ Rate limiting bảo vệ APIs[^1]
- ✅ Input validation đầy đủ[^1]
- ✅ Unit tests coverage > 60%[^1]
- ✅ Docker ready cho deployment[^1]

<div align="center">⁂</div>

[^1]: USER-REQUIREMENTS-DOCUMENT-URD-PHIEN-BAN-HOA.docx

