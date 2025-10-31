<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# TODO LIST CHI TIẾT CHO BACKEND DEVELOPER - SPRINT 9-10: PAYMENT INTEGRATION

Dựa trên URD, dưới đây là danh sách công việc chi tiết và đầy đủ cho backend developer trong Sprint 9-10 (Phase 2, 2 tuần - 10 ngày làm việc).[^1]

## 1. Asynq Setup + Basic Jobs (P2-T008)

### Mô tả

Thiết lập hệ thống background job processing với Asynq trước khi implement payment flow.[^1]

### Yêu cầu kỹ thuật

- **Asynq version**: 0.24[^1]
- **Backend**: Redis[^1]
- **Priority queues**: critical, high, default, low[^1]
- **Concurrency**: critical: 10, high: 20, default: 15, low: 5 workers[^1]


### Công việc cụ thể

#### 1.1 Cài đặt Asynq Package

```bash
go get -u github.com/hibiken/asynq
```


#### 1.2 Asynq Client Setup

Tạo file `internal/infrastructure/queue/asynq_client.go`:[^1]

```go
package queue

import (
    "github.com/hibiken/asynq"
    "github.com/redis/go-redis/v9"
)

const (
    QueueCritical = "critical"
    QueueHigh     = "high"
    QueueDefault  = "default"
    QueueLow      = "low"
)

type Client struct {
    client *asynq.Client
}

func NewAsynqClient(redisAddr string) *Client {
    client := asynq.NewClient(asynq.RedisClientOpt{
        Addr: redisAddr,
    })
    return &Client{client: client}
}

func (c *Client) Enqueue(task *asynq.Task, queue string, opts ...asynq.Option) error {
    allOpts := append([]asynq.Option{asynq.Queue(queue)}, opts...)
    _, err := c.client.Enqueue(task, allOpts...)
    return err
}

func (c *Client) Close() error {
    return c.client.Close()
}
```


#### 1.3 Asynq Server Setup

Tạo file `cmd/worker/main.go`:[^1]

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/hibiken/asynq"
    "bookstore/internal/config"
    "bookstore/internal/infrastructure/queue"
)

func main() {
    cfg := config.Load()
    
    srv := asynq.NewServer(
        asynq.RedisClientOpt{Addr: cfg.RedisHost},
        asynq.Config{
            Concurrency: 50,
            Queues: map[string]int{
                queue.QueueCritical: 10,
                queue.QueueHigh:     20,
                queue.QueueDefault:  15,
                queue.QueueLow:      5,
            },
            ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
                log.Printf("ERROR: task=%s error=%v", task.Type(), err)
            }),
        },
    )
    
    mux := asynq.NewServeMux()
    
    // Register handlers (sẽ thêm sau)
    // mux.HandleFunc("email:order_confirmation", handlers.SendOrderConfirmation)
    
    if err := srv.Run(mux); err != nil {
        log.Fatal(err)
    }
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    srv.Shutdown()
}
```


#### 1.4 Job Definitions

Tạo file `internal/domains/payment/jobs/types.go`:[^1]

```go
package jobs

const (
    TypePaymentTimeout    = "payment:timeout"
    TypeOrderConfirmation = "email:order_confirmation"
    TypePaymentReminder   = "email:payment_reminder"
    TypeProcessRefund     = "payment:refund"
    TypeUpdateOrderStatus = "order:update_status"
)

// Payloads
type PaymentTimeoutPayload struct {
    OrderID string `json:"order_id"`
}

type OrderConfirmationPayload struct {
    OrderID string `json:"order_id"`
}

type PaymentReminderPayload struct {
    OrderID string `json:"order_id"`
}
```


#### 1.5 Asynq Monitoring Dashboard

- Asynq cung cấp Web UI built-in[^1]
- Chạy dashboard riêng:

```go
// cmd/asynqmon/main.go
package main

import (
    "github.com/hibiken/asynq"
    "github.com/hibiken/asynqmon"
    "net/http"
)

func main() {
    h := asynqmon.New(asynqmon.Options{
        RootPath:     "/monitoring",
        RedisConnOpt: asynq.RedisClientOpt{Addr: "localhost:6379"},
    })
    
    http.Handle(h.RootPath()+"/", h)
    http.ListenAndServe(":8081", nil)
}
```

Dashboard accessible tại: `http://localhost:8081/monitoring`[^1]

#### 1.6 Testing

```go
func TestAsynqEnqueue(t *testing.T) {
    // Use miniredis for testing
    s := miniredis.RunT(t)
    
    client := queue.NewAsynqClient(s.Addr())
    defer client.Close()
    
    task := asynq.NewTask("test:task", []byte(`{"id":"123"}`))
    err := client.Enqueue(task, queue.QueueDefault)
    
    assert.NoError(t, err)
}
```


### Acceptance Criteria

- Asynq server chạy được với 4 priority queues[^1]
- Client enqueue task thành công[^1]
- Web dashboard hiển thị jobs[^1]
- Graceful shutdown hoạt động đúng[^1]


### Dependencies

- P1-T004: Redis setup[^1]


### Effort

2 ngày[^1]

***

## 2. VNPay SDK Integration (P2-T001)

### Mô tả

Tích hợp VNPay Payment Gateway để hỗ trợ thanh toán online.[^1]

### VNPay Integration Requirements

- **Payment method**: VNPay QR, VNPay Mobile Banking, ATM Card[^1]
- **Flow**: Redirect-based payment[^1]
- **Security**: HMAC SHA256 signature[^1]
- **IPN**: Instant Payment Notification webhook[^1]
- **Retry**: 3 attempts for failed payments[^1]


### Công việc cụ thể

#### 2.1 VNPay Configuration

Tạo file `internal/infrastructure/payment/vnpay/config.go`:[^1]

```go
package vnpay

type Config struct {
    TmnCode   string // Mã website (Terminal ID)
    SecretKey string // Secret key
    PaymentURL string // https://sandbox.vnpayment.vn/paymentv2/vpcpay.html
    ReturnURL  string // https://yourdomain.com/v1/payments/vnpay/return
    IPNURL     string // https://yourdomain.com/webhooks/vnpay
    Version    string // "2.1.0"
    Command    string // "pay"
}

func LoadConfig() *Config {
    return &Config{
        TmnCode:    os.Getenv("VNPAY_TMN_CODE"),
        SecretKey:  os.Getenv("VNPAY_SECRET_KEY"),
        PaymentURL: "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html",
        ReturnURL:  os.Getenv("APP_URL") + "/v1/payments/vnpay/return",
        IPNURL:     os.Getenv("APP_URL") + "/webhooks/vnpay",
        Version:    "2.1.0",
        Command:    "pay",
    }
}
```


#### 2.2 VNPay Client Implementation

Tạo file `internal/infrastructure/payment/vnpay/client.go`:[^1]

```go
package vnpay

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "net/url"
    "sort"
    "strings"
    "time"
)

type Client struct {
    config *Config
}

func NewClient(cfg *Config) *Client {
    return &Client{config: cfg}
}

type PaymentRequest struct {
    OrderID     string
    Amount      int64  // VNPay yêu cầu amount * 100 (VND)
    OrderInfo   string
    OrderType   string // Default: "billpayment"
    IPAddr      string
    BankCode    string // Optional: "NCB", "VIETCOMBANK", etc.
    Locale      string // "vn" or "en"
}

func (c *Client) CreatePaymentURL(req *PaymentRequest) (string, error) {
    // Build parameters
    params := url.Values{}
    params.Set("vnp_Version", c.config.Version)
    params.Set("vnp_Command", c.config.Command)
    params.Set("vnp_TmnCode", c.config.TmnCode)
    params.Set("vnp_Amount", fmt.Sprintf("%d", req.Amount))
    params.Set("vnp_CreateDate", time.Now().Format("20060102150405"))
    params.Set("vnp_CurrCode", "VND")
    params.Set("vnp_IpAddr", req.IPAddr)
    params.Set("vnp_Locale", req.Locale)
    params.Set("vnp_OrderInfo", req.OrderInfo)
    params.Set("vnp_OrderType", req.OrderType)
    params.Set("vnp_ReturnUrl", c.config.ReturnURL)
    params.Set("vnp_TxnRef", req.OrderID) // Order reference
    
    if req.BankCode != "" {
        params.Set("vnp_BankCode", req.BankCode)
    }
    
    // Generate secure hash
    secureHash := c.generateSecureHash(params)
    params.Set("vnp_SecureHash", secureHash)
    
    // Build final URL
    paymentURL := c.config.PaymentURL + "?" + params.Encode()
    return paymentURL, nil
}

func (c *Client) generateSecureHash(params url.Values) string {
    // Sort parameters alphabetically
    keys := make([]string, 0, len(params))
    for k := range params {
        if k != "vnp_SecureHash" {
            keys = append(keys, k)
        }
    }
    sort.Strings(keys)
    
    // Build hash data
    var hashData []string
    for _, k := range keys {
        hashData = append(hashData, fmt.Sprintf("%s=%s", k, params.Get(k)))
    }
    
    rawData := strings.Join(hashData, "&")
    
    // HMAC SHA256
    h := hmac.New(sha256.New, []byte(c.config.SecretKey))
    h.Write([]byte(rawData))
    
    return hex.EncodeToString(h.Sum(nil))
}

func (c *Client) ValidateSignature(params url.Values) bool {
    receivedHash := params.Get("vnp_SecureHash")
    params.Del("vnp_SecureHash")
    params.Del("vnp_SecureHashType")
    
    expectedHash := c.generateSecureHash(params)
    
    return receivedHash == expectedHash
}
```


#### 2.3 Response Codes

Tạo file `internal/infrastructure/payment/vnpay/codes.go`:[^1]

```go
package vnpay

var ResponseCodes = map[string]string{
    "00": "Giao dịch thành công",
    "07": "Trừ tiền thành công. Giao dịch bị nghi ngờ (liên quan tới lừa đảo, giao dịch bất thường)",
    "09": "Giao dịch không thành công do: Thẻ/Tài khoản của khách hàng chưa đăng ký dịch vụ InternetBanking",
    "10": "Giao dịch không thành công do: Khách hàng xác thực thông tin thẻ/tài khoản không đúng quá 3 lần",
    "11": "Giao dịch không thành công do: Đã hết hạn chờ thanh toán",
    "12": "Giao dịch không thành công do: Thẻ/Tài khoản của khách hàng bị khóa",
    "24": "Giao dịch không thành công do: Khách hàng hủy giao dịch",
    "51": "Giao dịch không thành công do: Tài khoản của quý khách không đủ số dư để thực hiện giao dịch",
    "65": "Giao dịch không thành công do: Tài khoản của Quý khách đã vượt quá hạn mức giao dịch trong ngày",
    "75": "Ngân hàng thanh toán đang bảo trì",
    "79": "Giao dịch không thành công do: KH nhập sai mật khẩu thanh toán quá số lần quy định",
    "99": "Lỗi không xác định",
}

func GetResponseMessage(code string) string {
    if msg, ok := ResponseCodes[code]; ok {
        return msg
    }
    return "Lỗi không xác định"
}
```


#### 2.4 Database Schema for Payments

Tạo migration `migrations/000020_create_payments_table.up.sql`:[^1]

```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    payment_method TEXT NOT NULL CHECK (payment_method IN ('cod', 'vnpay', 'momo')),
    
    -- VNPay specific
    vnp_txn_ref TEXT, -- Order reference
    vnp_transaction_no TEXT, -- VNPay transaction ID
    vnp_bank_code TEXT,
    vnp_card_type TEXT,
    vnp_response_code TEXT,
    
    -- Momo specific  
    momo_trans_id TEXT,
    momo_request_id TEXT,
    
    -- Common fields
    amount NUMERIC(10,2) NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'success', 'failed', 'refunded')),
    
    payment_url TEXT,
    callback_data JSONB, -- Store full callback data
    
    paid_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payments_order ON payments(order_id);
CREATE INDEX idx_payments_vnp_txn ON payments(vnp_txn_ref) WHERE vnp_txn_ref IS NOT NULL;
CREATE INDEX idx_payments_status ON payments(status);
```


### Acceptance Criteria

- Generate được VNPay payment URL với signature đúng[^1]
- Validate signature từ VNPay callback[^1]
- Parse response codes chính xác[^1]
- Test với VNPay sandbox environment[^1]


### Dependencies

- P1-T025: Order creation API[^1]


### Effort

3 ngày[^1]

***

## 3. VNPay Payment Creation API (P2-T002)

### Mô tả

API để tạo payment request và redirect user tới VNPay gateway.[^1]

### API Endpoint

`POST /v1/payments/vnpay/create`[^1]

### Request Body

```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "bank_code": "NCB",
  "locale": "vn"
}
```


### Công việc cụ thể

#### 3.1 Payment Handler

Tạo file `internal/domains/payment/handler/vnpay_handler.go`:[^1]

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "bookstore/internal/domains/payment/service"
    "bookstore/pkg/errors"
)

type VNPayHandler struct {
    paymentService *service.PaymentService
}

func NewVNPayHandler(ps *service.PaymentService) *VNPayHandler {
    return &VNPayHandler{paymentService: ps}
}

type CreateVNPayPaymentRequest struct {
    OrderID  string `json:"order_id" binding:"required"`
    BankCode string `json:"bank_code"`
    Locale   string `json:"locale"`
}

func (h *VNPayHandler) CreatePayment(c *gin.Context) {
    var req CreateVNPayPaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, errors.NewValidationError(err.Error()))
        return
    }
    
    // Get user from JWT
    userID := c.GetString("user_id")
    
    // Get IP address
    ipAddr := c.ClientIP()
    
    // Create payment
    paymentURL, payment, err := h.paymentService.CreateVNPayPayment(c.Request.Context(), service.CreateVNPayPaymentParams{
        UserID:   userID,
        OrderID:  req.OrderID,
        BankCode: req.BankCode,
        Locale:   req.Locale,
        IPAddr:   ipAddr,
    })
    
    if err != nil {
        c.JSON(500, errors.NewInternalError(err.Error()))
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "payment_id":  payment.ID,
            "payment_url": paymentURL,
            "expires_at":  time.Now().Add(15 * time.Minute).Unix(),
        },
    })
}
```


#### 3.2 Payment Service

Tạo file `internal/domains/payment/service/payment_service.go`:[^1]

```go
package service

import (
    "context"
    "fmt"
    
    "bookstore/internal/domains/order/repository"
    "bookstore/internal/infrastructure/payment/vnpay"
    "bookstore/internal/infrastructure/queue"
)

type PaymentService struct {
    orderRepo      *repository.OrderRepository
    paymentRepo    *repository.PaymentRepository
    vnpayClient    *vnpay.Client
    queueClient    *queue.Client
}

type CreateVNPayPaymentParams struct {
    UserID   string
    OrderID  string
    BankCode string
    Locale   string
    IPAddr   string
}

func (s *PaymentService) CreateVNPayPayment(ctx context.Context, params CreateVNPayPaymentParams) (string, *Payment, error) {
    // 1. Validate order exists and belongs to user
    order, err := s.orderRepo.FindByID(ctx, params.OrderID)
    if err != nil {
        return "", nil, fmt.Errorf("order not found")
    }
    
    if order.UserID != params.UserID {
        return "", nil, fmt.Errorf("unauthorized")
    }
    
    // 2. Check order status
    if order.Status != "pending" {
        return "", nil, fmt.Errorf("order is not pending")
    }
    
    if order.PaymentStatus == "paid" {
        return "", nil, fmt.Errorf("order already paid")
    }
    
    // 3. Create payment record
    payment := &Payment{
        OrderID:       order.ID,
        PaymentMethod: "vnpay",
        Amount:        order.Total,
        Status:        "pending",
        VnpTxnRef:     order.OrderNumber, // Use order number as txn ref
    }
    
    err = s.paymentRepo.Create(ctx, payment)
    if err != nil {
        return "", nil, err
    }
    
    // 4. Generate VNPay payment URL
    paymentURL, err := s.vnpayClient.CreatePaymentURL(&vnpay.PaymentRequest{
        OrderID:   order.OrderNumber,
        Amount:    int64(order.Total * 100), // Convert to VNPay format
        OrderInfo: fmt.Sprintf("Thanh toan don hang %s", order.OrderNumber),
        OrderType: "billpayment",
        IPAddr:    params.IPAddr,
        BankCode:  params.BankCode,
        Locale:    params.Locale,
    })
    
    if err != nil {
        return "", nil, err
    }
    
    // 5. Update payment with URL
    payment.PaymentURL = paymentURL
    s.paymentRepo.Update(ctx, payment)
    
    // 6. Schedule payment timeout job (15 minutes)
    task := asynq.NewTask(jobs.TypePaymentTimeout, &jobs.PaymentTimeoutPayload{
        OrderID: order.ID,
    })
    
    s.queueClient.Enqueue(task, queue.QueueCritical, asynq.ProcessIn(15*time.Minute))
    
    return paymentURL, payment, nil
}
```


#### 3.3 Validation

- Order phải tồn tại và thuộc về user[^1]
- Order status = "pending"[^1]
- Order chưa được thanh toán[^1]
- Payment method của order = "vnpay"[^1]


#### 3.4 Response Format

```json
{
  "success": true,
  "data": {
    "payment_id": "uuid",
    "payment_url": "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?...",
    "expires_at": 1698765432
  }
}
```


### Acceptance Criteria

- User tạo được payment và nhận được redirect URL[^1]
- Payment record được lưu vào database[^1]
- Timeout job được schedule sau 15 phút[^1]
- Validate order ownership và status[^1]


### Dependencies

- P2-T001: VNPay SDK integration[^1]
- P2-T008: Asynq setup[^1]


### Effort

2 ngày[^1]

***

## 4. VNPay IPN Webhook Handler (P2-T003)

### Mô tả

Xử lý IPN (Instant Payment Notification) callback từ VNPay sau khi user thanh toán.[^1]

### API Endpoint

`POST /webhooks/vnpay` (IPN - server-to-server)[^1]

### Webhook Flow

```
User pays at VNPay → VNPay IPN → Backend webhook → Update order → Return response to VNPay
```


### Công việc cụ thể

#### 4.1 Webhook Handler

Tạo file `internal/domains/payment/handler/vnpay_webhook.go`:[^1]

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "net/url"
)

func (h *VNPayHandler) HandleIPN(c *gin.Context) {
    // Parse query parameters
    params := c.Request.URL.Query()
    
    // 1. Validate signature
    if !h.paymentService.ValidateVNPaySignature(params) {
        c.JSON(200, gin.H{
            "RspCode": "97",
            "Message": "Invalid signature",
        })
        return
    }
    
    // 2. Extract parameters
    txnRef := params.Get("vnp_TxnRef")          // Order number
    amount := params.Get("vnp_Amount")           // Amount * 100
    responseCode := params.Get("vnp_ResponseCode")
    transactionNo := params.Get("vnp_TransactionNo")
    bankCode := params.Get("vnp_BankCode")
    cardType := params.Get("vnp_CardType")
    
    // 3. Process payment (idempotent)
    err := h.paymentService.ProcessVNPayIPN(c.Request.Context(), ProcessVNPayIPNParams{
        TxnRef:        txnRef,
        Amount:        amount,
        ResponseCode:  responseCode,
        TransactionNo: transactionNo,
        BankCode:      bankCode,
        CardType:      cardType,
        CallbackData:  params,
    })
    
    if err != nil {
        // Log error nhưng vẫn return success để VNPay không retry
        log.Error("IPN processing error", "error", err, "txnRef", txnRef)
        
        c.JSON(200, gin.H{
            "RspCode": "99",
            "Message": "Unknown error",
        })
        return
    }
    
    // 4. Return success response
    c.JSON(200, gin.H{
        "RspCode": "00",
        "Message": "Confirm Success",
    })
}
```


#### 4.2 Process IPN Service

```go
func (s *PaymentService) ProcessVNPayIPN(ctx context.Context, params ProcessVNPayIPNParams) error {
    // 1. Find order by txnRef (order number)
    order, err := s.orderRepo.FindByOrderNumber(ctx, params.TxnRef)
    if err != nil {
        return fmt.Errorf("order not found: %s", params.TxnRef)
    }
    
    // 2. Find payment record
    payment, err := s.paymentRepo.FindByOrderID(ctx, order.ID)
    if err != nil {
        return err
    }
    
    // 3. Idempotency check - nếu đã xử lý rồi thì skip
    if payment.Status == "success" && payment.VnpTransactionNo == params.TransactionNo {
        log.Info("IPN already processed", "txnRef", params.TxnRef)
        return nil
    }
    
    // 4. Start transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 5. Update payment record
    payment.VnpTransactionNo = params.TransactionNo
    payment.VnpBankCode = params.BankCode
    payment.VnpCardType = params.CardType
    payment.VnpResponseCode = params.ResponseCode
    payment.CallbackData = params.CallbackData // Store full callback
    
    if params.ResponseCode == "00" {
        // Payment success
        payment.Status = "success"
        payment.PaidAt = time.Now()
        
        // Update order
        order.PaymentStatus = "paid"
        order.Status = "confirmed"
        order.PaidAt = time.Now()
        
    } else {
        // Payment failed
        payment.Status = "failed"
        payment.FailedAt = time.Now()
        
        order.PaymentStatus = "failed"
        order.Status = "payment_failed"
    }
    
    err = s.paymentRepo.UpdateTx(ctx, tx, payment)
    if err != nil {
        return err
    }
    
    err = s.orderRepo.UpdateTx(ctx, tx, order)
    if err != nil {
        return err
    }
    
    // 6. Log to order_status_history
    s.orderRepo.AddStatusHistory(ctx, tx, &OrderStatusHistory{
        OrderID:   order.ID,
        OldStatus: "pending",
        NewStatus: order.Status,
        Note:      fmt.Sprintf("VNPay payment %s", payment.Status),
    })
    
    // 7. Commit transaction
    if err := tx.Commit(); err != nil {
        return err
    }
    
    // 8. Trigger background jobs (after commit)
    if payment.Status == "success" {
        // Send order confirmation email
        task := asynq.NewTask(jobs.TypeOrderConfirmation, &jobs.OrderConfirmationPayload{
            OrderID: order.ID,
        })
        s.queueClient.Enqueue(task, queue.QueueHigh)
        
        // Deduct inventory (already reserved, now confirm)
        // This happens in order service
    }
    
    return nil
}
```


#### 4.3 Idempotency

- Check `payment.status == "success"` và `vnp_transaction_no` match[^1]
- Nếu đã xử lý thì return success ngay, không update lại[^1]
- Log mọi IPN requests để audit[^1]


#### 4.4 Security

- **MUST validate signature** từ VNPay[^1]
- Không trust bất kỳ data nào nếu signature invalid[^1]
- Log suspicious requests[^1]


#### 4.5 VNPay IPN Response Format

**Success**:

```json
{
  "RspCode": "00",
  "Message": "Confirm Success"
}
```

**Error codes**:

- `97`: Invalid signature
- `99`: Unknown error


### Acceptance Criteria

- Validate signature chính xác[^1]
- Idempotent - xử lý nhiều lần không gây side effect[^1]
- Transaction safety - rollback nếu có lỗi[^1]
- Log tất cả IPN requests[^1]
- Trigger email confirmation khi success[^1]
- Response đúng format VNPay yêu cầu[^1]


### Dependencies

- P2-T001: VNPay SDK integration[^1]
- P2-T002: VNPay payment creation[^1]


### Effort

2 ngày[^1]

***

## 5. VNPay Return URL Handler (P2-T004)

### Mô tả

Xử lý redirect từ VNPay về website sau khi user thanh toán (browser-based).[^1]

### API Endpoint

`GET /v1/payments/vnpay/return`[^1]

### Flow

```
User pays at VNPay → VNPay redirects → /vnpay/return → Show result page
```


### Công việc cụ thể

#### 5.1 Return Handler

```go
func (h *VNPayHandler) HandleReturn(c *gin.Context) {
    // Parse query parameters
    params := c.Request.URL.Query()
    
    // 1. Validate signature
    if !h.paymentService.ValidateVNPaySignature(params) {
        c.Redirect(302, "/payment/failed?error=invalid_signature")
        return
    }
    
    // 2. Extract info
    txnRef := params.Get("vnp_TxnRef")
    responseCode := params.Get("vnp_ResponseCode")
    transactionNo := params.Get("vnp_TransactionNo")
    
    // 3. Get order info
    order, err := h.paymentService.GetOrderByNumber(c.Request.Context(), txnRef)
    if err != nil {
        c.Redirect(302, "/payment/failed?error=order_not_found")
        return
    }
    
    // 4. Redirect based on response code
    if responseCode == "00" {
        // Success - redirect to success page
        c.Redirect(302, fmt.Sprintf("/payment/success?order_id=%s&transaction_no=%s", order.ID, transactionNo))
    } else {
        // Failed - redirect to failed page with reason
        message := vnpay.GetResponseMessage(responseCode)
        c.Redirect(302, fmt.Sprintf("/payment/failed?order_id=%s&reason=%s", order.ID, url.QueryEscape(message)))
    }
}
```


#### 5.2 Redirect URLs

**Success**: `/payment/success?order_id=xxx&transaction_no=xxx`

**Failed**: `/payment/failed?order_id=xxx&reason=Insufficient+balance`

#### 5.3 Frontend Pages (out of scope, chỉ design API)

- Frontend sẽ handle render success/failed page[^1]
- Backend chỉ cần redirect với query params[^1]


#### 5.4 Note về IPN vs Return URL

- **IPN**: Server-to-server, reliable, dùng để update database[^1]
- **Return URL**: Browser redirect, không reliable (user có thể đóng browser), chỉ dùng để hiển thị UI[^1]
- **KHÔNG được** update order status trong Return URL handler[^1]
- Return URL chỉ validate signature và redirect[^1]


### Acceptance Criteria

- Validate signature trước khi redirect[^1]
- Redirect đúng page (success/failed)[^1]
- Pass order info qua query params[^1]
- KHÔNG update database trong handler này[^1]


### Dependencies

- P2-T001: VNPay SDK integration[^1]
- P2-T003: VNPay IPN (để đảm bảo order đã được update)[^1]


### Effort

1 ngày[^1]

***

## 6. Momo SDK Integration (P2-T005)

### Mô tả

Tích hợp Momo E-Wallet payment gateway.[^1]

### Momo Integration Requirements

- **Payment methods**: Momo Wallet, Momo QR Code, Momo ATM[^1]
- **Flow**: Redirect hoặc QR code[^1]
- **Security**: HMAC SHA256[^1]
- **Deep link support**: momo://app[^1]


### Công việc cụ thể

#### 6.1 Momo Configuration

Tạo file `internal/infrastructure/payment/momo/config.go`:[^1]

```go
package momo

type Config struct {
    PartnerCode string
    AccessKey   string
    SecretKey   string
    Endpoint    string // https://test-payment.momo.vn/v2/gateway/api/create
    ReturnURL   string
    NotifyURL   string // IPN
}

func LoadConfig() *Config {
    return &Config{
        PartnerCode: os.Getenv("MOMO_PARTNER_CODE"),
        AccessKey:   os.Getenv("MOMO_ACCESS_KEY"),
        SecretKey:   os.Getenv("MOMO_SECRET_KEY"),
        Endpoint:    "https://test-payment.momo.vn/v2/gateway/api/create",
        ReturnURL:   os.Getenv("APP_URL") + "/v1/payments/momo/return",
        NotifyURL:   os.Getenv("APP_URL") + "/webhooks/momo",
    }
}
```


#### 6.2 Momo Client

Tạo file `internal/infrastructure/payment/momo/client.go`:[^1]

```go
package momo

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    config *Config
}

func NewClient(cfg *Config) *Client {
    return &Client{config: cfg}
}

type PaymentRequest struct {
    OrderID   string
    Amount    int64
    OrderInfo string
    RequestID string // Unique request ID
    ExtraData string // Optional JSON data
}

type PaymentResponse struct {
    PartnerCode string `json:"partnerCode"`
    RequestID   string `json:"requestId"`
    OrderID     string `json:"orderId"`
    Amount      int64  `json:"amount"`
    ResponseTime int64 `json:"responseTime"`
    Message     string `json:"message"`
    ResultCode  int    `json:"resultCode"`
    PayURL      string `json:"payUrl"`
    Deeplink    string `json:"deeplink"`
    QRCodeURL   string `json:"qrCodeUrl"`
}

func (c *Client) CreatePayment(req *PaymentRequest) (*PaymentResponse, error) {
    // Build request
    payload := map[string]interface{}{
        "partnerCode": c.config.PartnerCode,
        "accessKey":   c.config.AccessKey,
        "requestId":   req.RequestID,
        "amount":      req.Amount,
        "orderId":     req.OrderID,
        "orderInfo":   req.OrderInfo,
        "returnUrl":   c.config.ReturnURL,
        "notifyUrl":   c.config.NotifyURL,
        "requestType": "captureWallet",
        "extraData":   req.ExtraData,
    }
    
    // Generate signature
    rawSignature := fmt.Sprintf("accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=captureWallet",
        c.config.AccessKey,
        req.Amount,
        req.ExtraData,
        c.config.NotifyURL,
        req.OrderID,
        req.OrderInfo,
        c.config.PartnerCode,
        c.config.ReturnURL,
        req.RequestID,
    )
    
    signature := c.generateSignature(rawSignature)
    payload["signature"] = signature
    
    // Send request
    jsonData, _ := json.Marshal(payload)
    resp, err := http.Post(c.config.Endpoint, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result PaymentResponse
    json.NewDecoder(resp.Body).Decode(&result)
    
    if result.ResultCode != 0 {
        return nil, fmt.Errorf("momo error: %s", result.Message)
    }
    
    return &result, nil
}

func (c *Client) generateSignature(rawData string) string {
    h := hmac.New(sha256.New, []byte(c.config.SecretKey))
    h.Write([]byte(rawData))
    return hex.EncodeToString(h.Sum(nil))
}

func (c *Client) ValidateSignature(data map[string]interface{}, receivedSignature string) bool {
    // Build raw signature string from callback data
    // Order matters! Follow Momo documentation
    rawSignature := fmt.Sprintf("accessKey=%s&amount=%v&extraData=%s&message=%s&orderId=%s&orderInfo=%s&orderType=%s&partnerCode=%s&payType=%s&requestId=%s&responseTime=%v&resultCode=%v&transId=%v",
        data["accessKey"],
        data["amount"],
        data["extraData"],
        data["message"],
        data["orderId"],
        data["orderInfo"],
        data["orderType"],
        data["partnerCode"],
        data["payType"],
        data["requestId"],
        data["responseTime"],
        data["resultCode"],
        data["transId"],
    )
    
    expectedSignature := c.generateSignature(rawSignature)
    return expectedSignature == receivedSignature
}
```


#### 6.3 Momo Response Codes

```go
var MomoResultCodes = map[int]string{
    0:    "Success",
    9000: "Transaction confirmed",
    1001: "Transaction failed",
    1002: "Transaction expired",
    1003: "Transaction cancelled by user",
    1004: "Transaction pending",
    1005: "Insufficient balance",
    1006: "Account locked",
    1007: "Payment limit exceeded",
    9999: "System error",
}
```


#### 6.4 Update Payments Table Migration

```sql
ALTER TABLE payments ADD COLUMN momo_trans_id TEXT;
ALTER TABLE payments ADD COLUMN momo_request_id TEXT;
ALTER TABLE payments ADD COLUMN momo_pay_url TEXT;
ALTER TABLE payments ADD COLUMN momo_deeplink TEXT;
ALTER TABLE payments ADD COLUMN momo_result_code INT;

CREATE INDEX idx_payments_momo_request ON payments(momo_request_id) WHERE momo_request_id IS NOT NULL;
```


### Acceptance Criteria

- Generate được Momo payment request với signature đúng[^1]
- Nhận được pay URL và deeplink từ Momo[^1]
- Validate signature từ Momo callback[^1]
- Test với Momo sandbox environment[^1]


### Dependencies

- P1-T025: Order creation API[^1]
- P2-T001: VNPay SDK (reference implementation)[^1]


### Effort

3 ngày[^1]

***

## 7. Momo Payment Flow (P2-T006)

### Mô tả

Implement full Momo payment flow bao gồm creation, IPN, return URL.[^1]

### API Endpoints

- `POST /v1/payments/momo/create`[^1]
- `POST /webhooks/momo` (IPN)[^1]
- `GET /v1/payments/momo/return` (Return URL)[^1]


### Công việc cụ thể

#### 7.1 Create Payment Handler

```go
func (h *MomoHandler) CreatePayment(c *gin.Context) {
    var req CreateMomoPaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, errors.NewValidationError(err.Error()))
        return
    }
    
    userID := c.GetString("user_id")
    
    paymentData, err := h.paymentService.CreateMomoPayment(c.Request.Context(), CreateMomoPaymentParams{
        UserID:  userID,
        OrderID: req.OrderID,
    })
    
    if err != nil {
        c.JSON(500, errors.NewInternalError(err.Error()))
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "payment_id": paymentData.PaymentID,
            "pay_url":    paymentData.PayURL,
            "deeplink":   paymentData.Deeplink,
            "qr_code":    paymentData.QRCodeURL,
        },
    })
}
```


#### 7.2 Momo Payment Service

```go
func (s *PaymentService) CreateMomoPayment(ctx context.Context, params CreateMomoPaymentParams) (*MomoPaymentData, error) {
    // 1. Validate order
    order, err := s.orderRepo.FindByID(ctx, params.OrderID)
    if err != nil {
        return nil, err
    }
    
    if order.UserID != params.UserID {
        return nil, fmt.Errorf("unauthorized")
    }
    
    // 2. Create payment record
    requestID := uuid.New().String()
    
    payment := &Payment{
        OrderID:        order.ID,
        PaymentMethod:  "momo",
        Amount:         order.Total,
        Status:         "pending",
        MomoRequestID:  requestID,
    }
    
    err = s.paymentRepo.Create(ctx, payment)
    if err != nil {
        return nil, err
    }
    
    // 3. Call Momo API
    momoResp, err := s.momoClient.CreatePayment(&momo.PaymentRequest{
        OrderID:   order.OrderNumber,
        Amount:    int64(order.Total),
        OrderInfo: fmt.Sprintf("Thanh toan don hang %s", order.OrderNumber),
        RequestID: requestID,
        ExtraData: fmt.Sprintf(`{"payment_id":"%s"}`, payment.ID),
    })
    
    if err != nil {
        payment.Status = "failed"
        s.paymentRepo.Update(ctx, payment)
        return nil, err
    }
    
    // 4. Update payment with Momo info
    payment.MomoPayURL = momoResp.PayURL
    payment.MomoDeeplink = momoResp.Deeplink
    s.paymentRepo.Update(ctx, payment)
    
    // 5. Schedule timeout job
    task := asynq.NewTask(jobs.TypePaymentTimeout, &jobs.PaymentTimeoutPayload{
        OrderID: order.ID,
    })
    s.queueClient.Enqueue(task, queue.QueueCritical, asynq.ProcessIn(15*time.Minute))
    
    return &MomoPaymentData{
        PaymentID: payment.ID,
        PayURL:    momoResp.PayURL,
        Deeplink:  momoResp.Deeplink,
        QRCodeURL: momoResp.QRCodeURL,
    }, nil
}
```


#### 7.3 Momo IPN Handler

```go
func (h *MomoHandler) HandleIPN(c *gin.Context) {
    var callback map[string]interface{}
    if err := c.ShouldBindJSON(&callback); err != nil {
        c.JSON(400, gin.H{"message": "Invalid request"})
        return
    }
    
    // 1. Validate signature
    signature, _ := callback["signature"].(string)
    if !h.paymentService.ValidateMomoSignature(callback, signature) {
        c.JSON(200, gin.H{
            "partnerCode": callback["partnerCode"],
            "requestId":   callback["requestId"],
            "resultCode":  97,
            "message":     "Invalid signature",
        })
        return
    }
    
    // 2. Process payment
    resultCode, _ := callback["resultCode"].(float64)
    err := h.paymentService.ProcessMomoIPN(c.Request.Context(), ProcessMomoIPNParams{
        RequestID:    callback["requestId"].(string),
        OrderID:      callback["orderId"].(string),
        TransID:      callback["transId"].(string),
        ResultCode:   int(resultCode),
        Message:      callback["message"].(string),
        CallbackData: callback,
    })
    
    if err != nil {
        log.Error("Momo IPN error", "error", err)
    }
    
    // 3. Return response
    c.JSON(200, gin.H{
        "partnerCode": callback["partnerCode"],
        "requestId":   callback["requestId"],
        "resultCode":  0,
        "message":     "Success",
    })
}
```


#### 7.4 Process Momo IPN Service

Logic tương tự VNPay IPN:

- Validate signature[^1]
- Idempotency check[^1]
- Update payment \& order trong transaction[^1]
- Trigger email job nếu success[^1]


#### 7.5 Momo Return URL Handler

```go
func (h *MomoHandler) HandleReturn(c *gin.Context) {
    // Parse query params
    resultCode := c.Query("resultCode")
    orderID := c.Query("orderId")
    transID := c.Query("transId")
    
    if resultCode == "0" || resultCode == "9000" {
        c.Redirect(302, fmt.Sprintf("/payment/success?order_id=%s&transaction_no=%s", orderID, transID))
    } else {
        message := momo.MomoResultCodes[resultCode]
        c.Redirect(302, fmt.Sprintf("/payment/failed?order_id=%s&reason=%s", orderID, url.QueryEscape(message)))
    }
}
```


### Acceptance Criteria

- User tạo được Momo payment và nhận deeplink[^1]
- IPN xử lý thành công với idempotency[^1]
- Return URL redirect đúng page[^1]
- Support cả QR code và deeplink[^1]


### Dependencies

- P2-T005: Momo SDK integration[^1]
- P2-T008: Asynq setup[^1]


### Effort

2 ngày[^1]

***

## 8. Payment Timeout Job (P2-T007)

### Mô tả

Background job tự động hủy order nếu user không thanh toán trong 15 phút.[^1]

### Business Logic

- Order với `payment_method = vnpay/momo` và `payment_status = pending`[^1]
- Sau 15 phút kể từ order created, nếu chưa paid → auto cancel[^1]
- Release inventory reserved[^1]
- Update order status = "cancelled"[^1]


### Công việc cụ thể

#### 8.1 Payment Timeout Job Handler

Tạo file `internal/domains/payment/jobs/payment_timeout.go`:[^1]

```go
package jobs

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/hibiken/asynq"
    "bookstore/internal/domains/order/service"
)

type PaymentTimeoutHandler struct {
    orderService *service.OrderService
}

func NewPaymentTimeoutHandler(os *service.OrderService) *PaymentTimeoutHandler {
    return &PaymentTimeoutHandler{orderService: os}
}

func (h *PaymentTimeoutHandler) HandlePaymentTimeout(ctx context.Context, task *asynq.Task) error {
    var payload PaymentTimeoutPayload
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return fmt.Errorf("unmarshal error: %w", err)
    }
    
    log.Info("Processing payment timeout", "order_id", payload.OrderID)
    
    // Call service to cancel order
    err := h.orderService.CancelOrderDueToTimeout(ctx, payload.OrderID)
    if err != nil {
        return err
    }
    
    log.Info("Order cancelled due to payment timeout", "order_id", payload.OrderID)
    return nil
}
```


#### 8.2 Order Service - Cancel Due to Timeout

```go
func (s *OrderService) CancelOrderDueToTimeout(ctx context.Context, orderID string) error {
    // 1. Get order
    order, err := s.orderRepo.FindByID(ctx, orderID)
    if err != nil {
        return err
    }
    
    // 2. Check if order is still pending payment
    if order.PaymentStatus != "pending" {
        log.Info("Order already processed, skip timeout cancellation", "order_id", orderID)
        return nil // Không cần cancel nếu đã paid
    }
    
    // 3. Check payment method
    if order.PaymentMethod == "cod" {
        log.Info("COD order, skip timeout cancellation", "order_id", orderID)
        return nil // COD không cần timeout
    }
    
    // 4. Check created time
    if time.Since(order.CreatedAt) < 15*time.Minute {
        log.Info("Order not yet timed out", "order_id", orderID)
        return nil
    }
    
    // 5. Start transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 6. Update order status
    order.Status = "cancelled"
    order.PaymentStatus = "failed"
    order.CancelledReason = "Payment timeout - no payment received within 15 minutes"
    order.CancelledAt = time.Now()
    
    err = s.orderRepo.UpdateTx(ctx, tx, order)
    if err != nil {
        return err
    }
    
    // 7. Release inventory reservation
    err = s.inventoryService.ReleaseReservationTx(ctx, tx, order.ID)
    if err != nil {
        return err
    }
    
    // 8. Log status history
    s.orderRepo.AddStatusHistory(ctx, tx, &OrderStatusHistory{
        OrderID:   order.ID,
        OldStatus: "pending",
        NewStatus: "cancelled",
        Note:      "Auto-cancelled: Payment timeout",
    })
    
    // 9. Commit
    if err := tx.Commit(); err != nil {
        return err
    }
    
    // 10. Send notification email (optional)
    task := asynq.NewTask(jobs.TypePaymentReminder, &jobs.PaymentReminderPayload{
        OrderID: order.ID,
    })
    s.queueClient.Enqueue(task, queue.QueueHigh)
    
    return nil
}
```


#### 8.3 Register Job Handler

Trong `cmd/worker/main.go`:

```go
mux := asynq.NewServeMux()

// Payment timeout handler
paymentTimeoutHandler := jobs.NewPaymentTimeoutHandler(orderService)
mux.HandleFunc(jobs.TypePaymentTimeout, paymentTimeoutHandler.HandlePaymentTimeout)
```


#### 8.4 Schedule Job

Job được schedule khi tạo payment (đã implement ở P2-T002, P2-T006):

```go
task := asynq.NewTask(jobs.TypePaymentTimeout, &jobs.PaymentTimeoutPayload{
    OrderID: order.ID,
})

// Schedule 15 minutes from now
s.queueClient.Enqueue(task, queue.QueueCritical, asynq.ProcessIn(15*time.Minute))
```


#### 8.5 Job Configuration

- **Priority**: Critical[^1]
- **Timeout**: 60s[^1]
- **Retry**: 5 times[^1]
- **Delay**: 15 minutes from order creation[^1]


### Acceptance Criteria

- Job chạy đúng sau 15 phút order created[^1]
- Chỉ cancel orders với payment_status = pending[^1]
- COD orders không bị cancel[^1]
- Inventory reserved được release[^1]
- Transaction safety - rollback nếu có lỗi[^1]
- Idempotent - không cancel nếu order đã paid[^1]


### Dependencies

- P2-T008: Asynq setup[^1]
- P2-T002: VNPay payment creation (schedule job)[^1]
- P2-T006: Momo payment flow (schedule job)[^1]


### Effort

1 ngày[^1]

***

## SUMMARY

### Total Effort Sprint 9-10

| Task ID | Task | Effort (days) |
| :-- | :-- | :-- |
| P2-T008 | Asynq setup + basic jobs | 2 |
| P2-T001 | VNPay SDK integration | 3 |
| P2-T002 | VNPay payment creation API | 2 |
| P2-T003 | VNPay IPN webhook handler | 2 |
| P2-T004 | VNPay return URL handler | 1 |
| P2-T005 | Momo SDK integration | 3 |
| P2-T006 | Momo payment flow | 2 |
| P2-T007 | Payment timeout job (Asynq) | 1 |
| **TOTAL** |  | **16 days** |

**Sprint duration**: 2 tuần (10 ngày làm việc)[^1]
**Team size**: 2 backend developers (có thể song song hóa tasks)[^1]

### Parallelization Strategy

**Week 1** (5 ngày):

- **Dev 1**: P2-T008 (Asynq setup) → P2-T001 (VNPay SDK) (2+3 = 5 days)[^1]
- **Dev 2**: P2-T005 (Momo SDK integration) (3 days) → Nghỉ/review code (2 days)[^1]

**Week 2** (5 ngày):

- **Dev 1**: P2-T002, P2-T003, P2-T004 (VNPay APIs) (2+2+1 = 5 days)[^1]
- **Dev 2**: P2-T006 (Momo flow) → P2-T007 (Payment timeout) (2+1 = 3 days) → Testing (2 days)[^1]


### Deliverables Checklist Sprint 9-10

- ✅ Asynq background job system hoạt động[^1]
- ✅ VNPay payment gateway integration hoàn chỉnh (create, IPN, return)[^1]
- ✅ Momo payment gateway integration hoàn chỉnh[^1]
- ✅ Payment timeout auto-cancel sau 15 phút[^1]
- ✅ Idempotent webhook handlers[^1]
- ✅ Transaction safety cho tất cả payment flows[^1]
- ✅ Support 3 payment methods: COD, VNPay, Momo[^1]
- ✅ Payment audit logs đầy đủ[^1]


### Key Technical Notes

**Security**:

- Luôn validate signature từ payment gateways[^1]
- Không trust client-side data[^1]
- Log tất cả payment webhooks để audit[^1]

**Reliability**:

- IPN handlers phải idempotent[^1]
- Sử dụng database transactions[^1]
- Retry logic cho failed jobs[^1]

**Performance**:

- Payment creation API: P95 < 500ms[^1]
- Webhook processing: < 2s[^1]
- Background jobs: Process within 5s[^1]

<div align="center">⁂</div>

[^1]: USER-REQUIREMENTS-DOCUMENT-URD-PHIEN-BAN-HOA.docx

