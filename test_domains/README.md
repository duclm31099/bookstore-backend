# Domain Test Suite

Comprehensive test suite for all bookstore backend domains including Order, Inventory, Cart, Payment, Book, and Promotion.

## Test Structure

```
test_domains/
├── setup_test.go              # Test infrastructure and helpers
├── order/
│   ├── order_api_test.go      # Order API tests
│   └── order_concurrency_test.go  # Order concurrency tests
├── inventory/
│   ├── inventory_api_test.go  # Inventory API tests
│   └── inventory_concurrency_test.go  # Inventory concurrency tests
├── cart/
│   ├── cart_api_test.go       # Cart API tests
│   └── cart_concurrency_test.go  # Cart concurrency tests
├── payment/
│   ├── payment_api_test.go    # Payment API tests
│   └── payment_webhook_test.go  # Payment webhook tests
├── book/
│   └── book_api_test.go       # Book API tests
└── promotion/
    └── promotion_api_test.go  # Promotion API tests
```

## Prerequisites

1. **PostgreSQL Database**: Ensure PostgreSQL is running
2. **Environment Variables**: Set up `.env` or `.env.test` file
3. **Dependencies**: Run `go mod download`

## Running Tests

### Run All Tests

```bash
cd test_domains
go test ./... -v
```

### Run Specific Domain Tests

**Order tests:**

```bash
go test ./order/... -v
```

**Inventory tests:**

```bash
go test ./inventory/... -v
```

**Cart tests:**

```bash
go test ./cart/... -v
```

**Payment tests:**

```bash
go test ./payment/... -v
```

**Book tests:**

```bash
go test ./book/... -v
```

**Promotion tests:**

```bash
go test ./promotion/... -v
```

### Run Concurrency Tests Only

```bash
go test ./... -v -run Concurrent
```

### Run with Race Detector

```bash
go test ./... -v -race
```

This is **highly recommended** for concurrency tests to detect race conditions.

### Generate Coverage Report

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

Then open `coverage.html` in your browser.

## Test Categories

### 1. API Tests

Test basic CRUD operations and business logic:

- ✅ Success scenarios
- ✅ Validation errors
- ✅ Not found errors
- ✅ Authorization errors
- ✅ Edge cases

### 2. Concurrency Tests

Test thread-safe operations:

- ✅ Concurrent order creation
- ✅ Concurrent stock reservation
- ✅ Optimistic locking
- ✅ Race condition prevention
- ✅ No overselling verification

### 3. Integration Tests

Test cross-domain interactions:

- ✅ Order → Inventory (stock reservation)
- ✅ Cart → Order (checkout flow)
- ✅ Order → Payment (payment creation)
- ✅ Payment → Order (status updates)

## Test Helpers

The `setup_test.go` file provides comprehensive test utilities:

### Database Helpers

- `cleanDatabase()` - Truncate all tables
- `cleanBeforeTest(t)` - Clean before each test

### HTTP Helpers

- `makeRequest(t, method, path, body, headers)` - Make HTTP request
- `assertSuccess(t, resp, status)` - Assert successful response
- `assertError(t, resp, status)` - Assert error response

### Test Data Helpers

- `createVerifiedUser(t, email, password)` - Create verified user
- `loginUser(t, email, password)` - Login and get token
- `createTestCategory(t, name)` - Create test category
- `createTestAuthor(t, name)` - Create test author
- `createTestPublisher(t, name)` - Create test publisher
- `createTestBook(t, title, price, ...)` - Create test book
- `createTestWarehouse(t, name, lat, lng)` - Create test warehouse
- `createTestInventory(t, warehouseID, bookID, quantity)` - Create inventory
- `createTestPromotion(t, code, discount, minAmount)` - Create promotion
- `createTestAddress(t, userID, isDefault)` - Create address

### Concurrency Helpers

- `runConcurrent(t, count, fn)` - Run functions concurrently
- `runConcurrentWithResults(t, count, fn)` - Run and collect results
- `waitForCondition(t, timeout, condition)` - Wait for condition

## Test Coverage Goals

| Domain    | Target Coverage |
| --------- | --------------- |
| Order     | ≥ 80%           |
| Inventory | ≥ 80%           |
| Cart      | ≥ 75%           |
| Payment   | ≥ 70%           |
| Book      | ≥ 70%           |
| Promotion | ≥ 70%           |

## Key Test Scenarios

### Order Domain

- ✅ Create order from cart
- ✅ Insufficient stock handling
- ✅ Empty cart validation
- ✅ Order cancellation
- ✅ Version conflict (optimistic locking)
- ✅ Concurrent order creation
- ✅ Concurrent cancellation

### Inventory Domain

- ✅ CRUD operations
- ✅ Stock reservation
- ✅ Stock release
- ✅ Complete sale
- ✅ Warehouse selection
- ✅ Concurrent reservations
- ✅ Overselling prevention
- ✅ Race condition detection

### Cart Domain

- ✅ Add/update/remove items
- ✅ Cart validation
- ✅ Promotion application
- ✅ Checkout flow
- ✅ Concurrent updates

### Payment Domain

- ✅ Payment creation
- ✅ Payment status tracking
- ✅ Webhook handling (VNPay, Momo)
- ✅ Refund workflows
- ✅ Concurrent processing

## Troubleshooting

### Database Connection Issues

Ensure PostgreSQL is running and environment variables are set:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=bookstore_test
```

### Test Failures

1. Check database is clean before tests
2. Verify all migrations are applied
3. Check for port conflicts
4. Review test logs for specific errors

### Race Conditions

Always run concurrency tests with race detector:

```bash
go test ./... -race -run Concurrent
```

## Best Practices

1. **Clean State**: Always call `cleanBeforeTest(t)` at the start of each test
2. **Isolation**: Tests should not depend on each other
3. **Descriptive Names**: Use clear test names like `TestCreateOrder_InsufficientStock`
4. **Assertions**: Use `require` for critical checks, `assert` for non-critical
5. **Concurrency**: Test with realistic concurrency levels (5-20 goroutines)

## Contributing

When adding new tests:

1. Follow existing test structure
2. Add test helpers to `setup_test.go` if reusable
3. Include both success and error scenarios
4. Add concurrency tests for critical operations
5. Update this README with new test scenarios
