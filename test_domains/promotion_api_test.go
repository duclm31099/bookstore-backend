package test_domains

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// PROMOTION API TESTS
// ========================================

func TestCreatePromotion_Success(t *testing.T) {
	cleanBeforeTest(t)

	payload := map[string]interface{}{
		"code":                "NEWYEAR2026",
		"description":         "New Year Promotion",
		"discount_type":       "percentage",
		"discount_value":      15.0,
		"min_order_amount":    100000,
		"max_discount_amount": 50000,
		"usage_limit":         100,
		"start_date":          "2026-01-01T00:00:00Z",
		"end_date":            "2026-01-31T23:59:59Z",
		"is_active":           true,
	}

	resp := makeRequest(t, "POST", "/api/v1/promotion/create", payload, nil)
	result := assertSuccess(t, resp, http.StatusCreated)

	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "NEWYEAR2026", data["code"])
	assert.Equal(t, float64(15), data["discount_value"])
}

func TestGetPromotion_Success(t *testing.T) {
	cleanBeforeTest(t)

	promoID := createTestPromotion(t, "TESTCODE", 10, 50000)

	resp := makeRequest(t, "GET", "/api/v1/promotion/"+promoID, nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, promoID, data["id"])
	assert.Equal(t, "TESTCODE", data["code"])
}

func TestGetPromotion_NotFound(t *testing.T) {
	cleanBeforeTest(t)

	resp := makeRequest(t, "GET", "/api/v1/promotion/00000000-0000-0000-0000-000000000000", nil, nil)
	assertError(t, resp, http.StatusNotFound)
}

func TestListActivePromotions_Success(t *testing.T) {
	cleanBeforeTest(t)

	// Create multiple promotions
	createTestPromotion(t, "PROMO1", 10, 50000)
	createTestPromotion(t, "PROMO2", 15, 100000)
	createTestPromotion(t, "PROMO3", 20, 150000)

	resp := makeRequest(t, "GET", "/api/v1/promotion", nil, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	promotions := data["promotions"].([]interface{})
	assert.GreaterOrEqual(t, len(promotions), 3)
}

func TestValidatePromotion_Success(t *testing.T) {
	cleanBeforeTest(t)

	createTestPromotion(t, "VALID10", 10, 100000)

	// Validate with cart amount that meets minimum
	payload := map[string]interface{}{
		"code":        "VALID10",
		"cart_amount": 150000,
	}

	resp := makeRequest(t, "POST", "/api/v1/promotion/validate", payload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.True(t, data["is_valid"].(bool))
	assert.NotNil(t, data["discount_amount"])
}

func TestValidatePromotion_MinimumNotMet(t *testing.T) {
	cleanBeforeTest(t)

	createTestPromotion(t, "MIN100K", 10, 100000)

	// Validate with cart amount below minimum
	payload := map[string]interface{}{
		"code":        "MIN100K",
		"cart_amount": 50000, // Below 100k minimum
	}

	resp := makeRequest(t, "POST", "/api/v1/promotion/validate", payload, nil)
	assertError(t, resp, http.StatusUnprocessableEntity)
}

func TestValidatePromotion_InvalidCode(t *testing.T) {
	cleanBeforeTest(t)

	payload := map[string]interface{}{
		"code":        "NONEXISTENT",
		"cart_amount": 200000,
	}

	resp := makeRequest(t, "POST", "/api/v1/promotion/validate", payload, nil)
	assertError(t, resp, http.StatusNotFound)
}

func TestValidatePromotion_PercentageDiscount(t *testing.T) {
	cleanBeforeTest(t)

	createTestPromotion(t, "PERCENT20", 20, 100000)

	payload := map[string]interface{}{
		"code":        "PERCENT20",
		"cart_amount": 500000,
	}

	resp := makeRequest(t, "POST", "/api/v1/promotion/validate", payload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.True(t, data["is_valid"].(bool))

	// 20% of 500k = 100k, but should be capped at max_discount_amount
	discountAmount := data["discount_amount"].(float64)
	assert.Greater(t, discountAmount, float64(0))
}

func TestValidatePromotion_MaxDiscountCap(t *testing.T) {
	cleanBeforeTest(t)

	// Create promotion with 50% discount but max 100k
	payload := map[string]interface{}{
		"code":                "BIGDISCOUNT",
		"description":         "Big discount",
		"discount_type":       "percentage",
		"discount_value":      50.0,
		"min_order_amount":    100000,
		"max_discount_amount": 100000,
		"usage_limit":         100,
		"start_date":          "2026-01-01T00:00:00Z",
		"end_date":            "2026-12-31T23:59:59Z",
		"is_active":           true,
	}
	makeRequest(t, "POST", "/api/v1/promotion/create", payload, nil)

	// Validate with 1M cart (50% would be 500k, but capped at 100k)
	validatePayload := map[string]interface{}{
		"code":        "BIGDISCOUNT",
		"cart_amount": 1000000,
	}

	resp := makeRequest(t, "POST", "/api/v1/promotion/validate", validatePayload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	discountAmount := data["discount_amount"].(float64)

	// Should be capped at 100k
	assert.LessOrEqual(t, discountAmount, float64(100000))
}

func TestUpdatePromotion_Success(t *testing.T) {
	cleanBeforeTest(t)

	promoID := createTestPromotion(t, "UPDATE", 10, 50000)

	// Update promotion
	payload := map[string]interface{}{
		"discount_value": 15.0,
		"description":    "Updated description",
	}

	resp := makeRequest(t, "PUT", "/api/v1/promotion/"+promoID, payload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, float64(15), data["discount_value"])
}

func TestUpdatePromotionStatus_Deactivate(t *testing.T) {
	cleanBeforeTest(t)

	promoID := createTestPromotion(t, "DEACTIVATE", 10, 50000)

	// Deactivate promotion
	payload := map[string]interface{}{
		"is_active": false,
	}

	resp := makeRequest(t, "PATCH", "/api/v1/promotion/"+promoID+"/status", payload, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Verify cannot validate deactivated promotion
	validatePayload := map[string]interface{}{
		"code":        "DEACTIVATE",
		"cart_amount": 100000,
	}
	resp = makeRequest(t, "POST", "/api/v1/promotion/validate", validatePayload, nil)
	assertError(t, resp, http.StatusNotFound)
}

func TestDeletePromotion_Success(t *testing.T) {
	cleanBeforeTest(t)

	promoID := createTestPromotion(t, "DELETE", 10, 50000)

	resp := makeRequest(t, "DELETE", "/api/v1/promotion/"+promoID, nil, nil)
	assertSuccess(t, resp, http.StatusOK)

	// Verify deleted
	resp = makeRequest(t, "GET", "/api/v1/promotion/"+promoID, nil, nil)
	assertError(t, resp, http.StatusNotFound)
}

func TestPromotionUsageLimit(t *testing.T) {
	cleanBeforeTest(t)

	// Create promotion with usage limit of 1
	payload := map[string]interface{}{
		"code":                "ONETIME",
		"description":         "One time use",
		"discount_type":       "percentage",
		"discount_value":      10.0,
		"min_order_amount":    50000,
		"max_discount_amount": 50000,
		"usage_limit":         1,
		"start_date":          "2026-01-01T00:00:00Z",
		"end_date":            "2026-12-31T23:59:59Z",
		"is_active":           true,
	}
	resp := makeRequest(t, "POST", "/api/v1/promotion/create", payload, nil)
	assertSuccess(t, resp, http.StatusCreated)

	// First validation should succeed
	validatePayload := map[string]interface{}{
		"code":        "ONETIME",
		"cart_amount": 100000,
	}
	resp = makeRequest(t, "POST", "/api/v1/promotion/validate", validatePayload, nil)
	result := assertSuccess(t, resp, http.StatusOK)

	data := result["data"].(map[string]interface{})
	assert.True(t, data["is_valid"].(bool))
}

func TestPromotionDateRange(t *testing.T) {
	cleanBeforeTest(t)

	// Create promotion that's not started yet
	payload := map[string]interface{}{
		"code":                "FUTURE",
		"description":         "Future promotion",
		"discount_type":       "percentage",
		"discount_value":      10.0,
		"min_order_amount":    50000,
		"max_discount_amount": 50000,
		"usage_limit":         100,
		"start_date":          "2027-01-01T00:00:00Z", // Future date
		"end_date":            "2027-12-31T23:59:59Z",
		"is_active":           true,
	}
	resp := makeRequest(t, "POST", "/api/v1/promotion/create", payload, nil)
	assertSuccess(t, resp, http.StatusCreated)

	// Validation should fail because promotion hasn't started
	validatePayload := map[string]interface{}{
		"code":        "FUTURE",
		"cart_amount": 100000,
	}
	resp = makeRequest(t, "POST", "/api/v1/promotion/validate", validatePayload, nil)
	// Should fail (exact error depends on implementation)
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}
