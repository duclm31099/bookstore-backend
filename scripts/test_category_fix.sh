#!/bin/bash

# ============================================================
# CATEGORY API COMPREHENSIVE TEST SUITE
# ============================================================
# Tests all 16 endpoints with edge cases
# Run: chmod +x test_category_complete.sh && ./test_category_complete.sh

BASE_URL="http://localhost:8080/api/v1/categories"
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================
# HELPER FUNCTIONS
# ============================================================

print_header() {
    echo ""
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}"
}

print_test() {
    echo ""
    echo -e "${YELLOW}TEST $1: $2${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

print_pass() {
    echo -e "${GREEN}✅ PASS: $1${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
}

print_fail() {
    echo -e "${RED}❌ FAIL: $1${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

extract_id() {
    local json="$1"
    echo "$json" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

extract_field() {
    local json="$1"
    local field="$2"
    echo "$json" | grep -o "\"$field\":[^,}]*" | cut -d':' -f2 | tr -d ' ",'
}

check_success() {
    local response="$1"
    if echo "$response" | grep -q '"success":true'; then
        return 0
    else
        return 1
    fi
}

# ============================================================
# SETUP
# ============================================================

print_header "CATEGORY API TEST SUITE"
echo "Base URL: $BASE_URL"
echo "Started at: $(date)"

# Cleanup old test data
print_info "Cleaning up old test data..."
curl -s -X DELETE "$BASE_URL/bulk" \
  -H "Content-Type: application/json" \
  -d '{"category_ids": []}' > /dev/null 2>&1

sleep 1

# ============================================================
# TEST GROUP 1: CREATE OPERATIONS
# ============================================================

print_header "GROUP 1: CREATE OPERATIONS (5 tests)"

# Test 1: Create root category
print_test "1.1" "Create root category"
ROOT_NAME="Test Root $(date +%s)"
ROOT_RESP=$(curl -s -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$ROOT_NAME\", \"description\": \"Test root category\"}")

if check_success "$ROOT_RESP"; then
    ROOT_ID=$(extract_id "$ROOT_RESP")
    ROOT_LEVEL=$(extract_field "$ROOT_RESP" "level")
    if [ "$ROOT_LEVEL" = "1" ]; then
        print_pass "Root created with level=1, ID=$ROOT_ID"
    else
        print_fail "Root level should be 1, got $ROOT_LEVEL"
    fi
else
    print_fail "Failed to create root category"
    echo "$ROOT_RESP"
    exit 1
fi

sleep 0.5

# Test 2: Create child category
print_test "1.2" "Create child category"
CHILD_NAME="Test Child $(date +%s)"
CHILD_RESP=$(curl -s -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$CHILD_NAME\", \"parent_id\": \"$ROOT_ID\", \"description\": \"Test child category\"}")

if check_success "$CHILD_RESP"; then
    CHILD_ID=$(extract_id "$CHILD_RESP")
    CHILD_LEVEL=$(extract_field "$CHILD_RESP" "level")
    if [ "$CHILD_LEVEL" = "2" ]; then
        print_pass "Child created with level=2, ID=$CHILD_ID"
    else
        print_fail "Child level should be 2, got $CHILD_LEVEL"
    fi
else
    print_fail "Failed to create child category"
fi

sleep 0.5

# Test 3: Create grandchild category
print_test "1.3" "Create grandchild category"
GRAND_NAME="Test Grandchild $(date +%s)"
GRAND_RESP=$(curl -s -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$GRAND_NAME\", \"parent_id\": \"$CHILD_ID\"}")

if check_success "$GRAND_RESP"; then
    GRAND_ID=$(extract_id "$GRAND_RESP")
    GRAND_LEVEL=$(extract_field "$GRAND_RESP" "level")
    if [ "$GRAND_LEVEL" = "3" ]; then
        print_pass "Grandchild created with level=3, ID=$GRAND_ID"
    else
        print_fail "Grandchild level should be 3, got $GRAND_LEVEL"
    fi
else
    print_fail "Failed to create grandchild category"
fi

sleep 0.5

# Test 4: Try create level 4 (should fail)
print_test "1.4" "Try create level 4 (should fail)"
LEVEL4_RESP=$(curl -s -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"Level 4\", \"parent_id\": \"$GRAND_ID\"}")

if echo "$LEVEL4_RESP" | grep -qi "max depth\|exceeds maximum"; then
    print_pass "Level 4 correctly rejected"
else
    print_fail "Level 4 should be rejected"
fi

sleep 0.5

# Test 5: Try create duplicate slug (should fail)
print_test "1.5" "Try create duplicate slug (should fail)"
DUP_RESP=$(curl -s -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$ROOT_NAME\"}")

if echo "$DUP_RESP" | grep -qi "slug already exists\|duplicate"; then
    print_pass "Duplicate slug correctly rejected"
else
    print_fail "Duplicate slug should be rejected"
fi

sleep 0.5

# ============================================================
# TEST GROUP 2: READ OPERATIONS
# ============================================================

print_header "GROUP 2: READ OPERATIONS (6 tests)"

# Test 6: Get by ID
print_test "2.1" "Get category by ID"
GET_ID_RESP=$(curl -s "$BASE_URL/$GRAND_ID")

if check_success "$GET_ID_RESP"; then
    GET_LEVEL=$(extract_field "$GET_ID_RESP" "level")
    if [ "$GET_LEVEL" = "3" ]; then
        print_pass "GetByID returned correct level=3"
    else
        print_fail "GetByID level should be 3, got $GET_LEVEL"
    fi
else
    print_fail "GetByID failed"
fi

sleep 0.5

# Test 7: Get by slug
print_test "2.2" "Get category by slug"
ROOT_SLUG=$(extract_field "$ROOT_RESP" "slug")
GET_SLUG_RESP=$(curl -s "$BASE_URL/by-slug/$ROOT_SLUG")

if check_success "$GET_SLUG_RESP"; then
    print_pass "GetBySlug successful"
else
    print_fail "GetBySlug failed"
fi

sleep 0.5

# Test 8: Get all categories
print_test "2.3" "Get all categories"
GET_ALL_RESP=$(curl -s "$BASE_URL?limit=10&offset=0")

if check_success "$GET_ALL_RESP"; then
    COUNT=$(echo "$GET_ALL_RESP" | grep -o '"name"' | wc -l)
    if [ "$COUNT" -ge "3" ]; then
        print_pass "GetAll returned $COUNT categories"
    else
        print_fail "GetAll should return at least 3 categories, got $COUNT"
    fi
else
    print_fail "GetAll failed"
fi

sleep 0.5

# Test 9: Get tree
print_test "2.4" "Get category tree"
GET_TREE_RESP=$(curl -s "$BASE_URL/tree")

if check_success "$GET_TREE_RESP"; then
    if echo "$GET_TREE_RESP" | grep -q '"level":3'; then
        print_pass "GetTree contains level 3"
    else
        print_fail "GetTree should contain level 3"
    fi
else
    print_fail "GetTree failed"
fi

sleep 0.5

# Test 10: Get breadcrumb
print_test "2.5" "Get category breadcrumb"
BREADCRUMB_RESP=$(curl -s "$BASE_URL/$GRAND_ID/breadcrumb")

if check_success "$BREADCRUMB_RESP"; then
    if echo "$BREADCRUMB_RESP" | grep -q "current_path"; then
        print_pass "GetBreadcrumb returned breadcrumb path"
    else
        print_fail "GetBreadcrumb missing current_path"
    fi
else
    print_fail "GetBreadcrumb failed"
fi

sleep 0.5

# Test 11: Get invalid ID (should fail)
print_test "2.6" "Get invalid ID (should fail)"
INVALID_RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/00000000-0000-0000-0000-000000000000")

HTTP_CODE=$(echo "$INVALID_RESP" | tail -n 1)
BODY=$(echo "$INVALID_RESP" | head -n -1)

if [ "$HTTP_CODE" = "404" ]; then
    print_pass "Invalid ID returned 404 Not Found"
else
    print_fail "Invalid ID should return 404, got $HTTP_CODE"
fi

sleep 0.5


# ============================================================
# TEST GROUP 3: UPDATE OPERATIONS
# ============================================================

print_header "GROUP 3: UPDATE OPERATIONS (4 tests)"

# Test 12: Update category
print_test "3.1" "Update category"
UPDATE_NAME="Updated Name $(date +%s)"
UPDATE_RESP=$(curl -s -X PUT "$BASE_URL/$CHILD_ID" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$UPDATE_NAME\", \"description\": \"Updated description\"}")

if check_success "$UPDATE_RESP"; then
    UPDATE_LEVEL=$(extract_field "$UPDATE_RESP" "level")
    if [ "$UPDATE_LEVEL" = "2" ]; then
        print_pass "Update successful, level preserved=2"
    else
        print_fail "Update level should be 2, got $UPDATE_LEVEL"
    fi
else
    print_fail "Update failed"
fi

sleep 0.5

# Test 13: Move to parent
print_test "3.2" "Move category to root"
MOVE_RESP=$(curl -s -X PATCH "$BASE_URL/$GRAND_ID/parent" \
  -H "Content-Type: application/json" \
  -d '{"parent_id": null}')

if check_success "$MOVE_RESP"; then
    MOVE_LEVEL=$(extract_field "$MOVE_RESP" "level")
    if [ "$MOVE_LEVEL" = "1" ]; then
        print_pass "Move to root successful, level=1"
    else
        print_fail "Move level should be 1, got $MOVE_LEVEL"
    fi
else
    print_fail "Move failed"
fi

sleep 0.5

# Test 14: Try circular reference (should fail)
print_test "3.3" "Try circular reference (should fail)"
CIRCULAR_RESP=$(curl -s -X PATCH "$BASE_URL/$ROOT_ID/parent" \
  -H "Content-Type: application/json" \
  -d "{\"parent_id\": \"$CHILD_ID\"}")

if echo "$CIRCULAR_RESP" | grep -qi "circular"; then
    print_pass "Circular reference correctly prevented"
else
    print_fail "Circular reference should be prevented"
fi

sleep 0.5

# Test 15: Deactivate category
print_test "3.4" "Deactivate category"
DEACT_RESP=$(curl -s -X POST "$BASE_URL/$CHILD_ID/deactivate")

if check_success "$DEACT_RESP"; then
    IS_ACTIVE=$(extract_field "$DEACT_RESP" "is_active")
    if [ "$IS_ACTIVE" = "false" ]; then
        print_pass "Deactivate successful"
    else
        print_fail "Category should be inactive"
    fi
else
    print_fail "Deactivate failed"
fi

sleep 0.5

# ============================================================
# TEST GROUP 4: ACTIVATE/DEACTIVATE
# ============================================================

print_header "GROUP 4: ACTIVATE/DEACTIVATE (2 tests)"

# Test 16: Activate category
print_test "4.1" "Activate category"
ACT_RESP=$(curl -s -X POST "$BASE_URL/$CHILD_ID/activate")

if check_success "$ACT_RESP"; then
    IS_ACTIVE=$(extract_field "$ACT_RESP" "is_active")
    if [ "$IS_ACTIVE" = "true" ]; then
        print_pass "Activate successful"
    else
        print_fail "Category should be active"
    fi
else
    print_fail "Activate failed"
fi

sleep 0.5

# Test 17: Try activate child with inactive parent (should fail)
print_test "4.2" "Try activate child when parent inactive (should fail)"
# Deactivate parent first
curl -s -X POST "$BASE_URL/$ROOT_ID/deactivate" > /dev/null 2>&1
sleep 0.3

# Try activate child
ACT_CHILD_RESP=$(curl -s -X POST "$BASE_URL/$CHILD_ID/activate")

if echo "$ACT_CHILD_RESP" | grep -qi "parent.*inactive"; then
    print_pass "Cannot activate child with inactive parent"
else
    print_fail "Should prevent activating child with inactive parent"
fi

# Reactivate parent for cleanup
curl -s -X POST "$BASE_URL/$ROOT_ID/activate" > /dev/null 2>&1
sleep 0.5

# ============================================================
# TEST GROUP 5: BULK OPERATIONS
# ============================================================

print_header "GROUP 5: BULK OPERATIONS (3 tests)"

# Create additional categories for bulk test
BULK_ID1=$(extract_id "$(curl -s -X POST "$BASE_URL" -H "Content-Type: application/json" -d '{"name": "Bulk Test 1"}')")
BULK_ID2=$(extract_id "$(curl -s -X POST "$BASE_URL" -H "Content-Type: application/json" -d '{"name": "Bulk Test 2"}')")
BULK_ID3=$(extract_id "$(curl -s -X POST "$BASE_URL" -H "Content-Type: application/json" -d '{"name": "Bulk Test 3"}')")
sleep 0.5

# Test 18: Bulk activate
print_test "5.1" "Bulk activate categories"
BULK_ACT_RESP=$(curl -s -X POST "$BASE_URL/bulk/activate" \
  -H "Content-Type: application/json" \
  -d "{\"category_ids\": [\"$BULK_ID1\", \"$BULK_ID2\", \"$BULK_ID3\"]}")

# Extract from data.success
SUCCESS=$(echo "$BULK_ACT_RESP" | grep -o '"success":[0-9]*' | tail -1 | cut -d':' -f2)
if [ "$SUCCESS" = "3" ]; then
    print_pass "Bulk activate successful (3 categories)"
else
    print_fail "Bulk activate should affect 3, got $SUCCESS"
fi


# Test 19: Bulk deactivate
print_test "5.2" "Bulk deactivate categories"
BULK_DEACT_RESP=$(curl -s -X POST "$BASE_URL/bulk/deactivate" \
  -H "Content-Type: application/json" \
  -d "{\"category_ids\": [\"$BULK_ID1\", \"$BULK_ID2\"]}")

if check_success "$BULK_DEACT_RESP"; then
    print_pass "Bulk deactivate successful"
else
    print_fail "Bulk deactivate failed"
fi

sleep 0.5

# Test 20: Bulk delete
print_test "5.3" "Bulk delete categories"
BULK_DEL_RESP=$(curl -s -X DELETE "$BASE_URL/bulk" \
  -H "Content-Type: application/json" \
  -d "{\"category_ids\": [\"$BULK_ID1\", \"$BULK_ID2\", \"$BULK_ID3\"]}")

if check_success "$BULK_DEL_RESP"; then
    print_pass "Bulk delete successful"
else
    print_fail "Bulk delete failed"
fi

sleep 0.5

# ============================================================
# TEST GROUP 6: DELETE OPERATIONS
# ============================================================

print_header "GROUP 6: DELETE OPERATIONS (2 tests)"

# Test 21: Try delete category with children (should fail)
print_test "6.1" "Try delete category with children (should fail)"
DEL_PARENT_RESP=$(curl -s -X DELETE "$BASE_URL/$ROOT_ID")

if echo "$DEL_PARENT_RESP" | grep -qi "children\|has"; then
    print_pass "Cannot delete category with children"
else
    print_fail "Should prevent deleting category with children"
fi

sleep 0.5

# Test 22: Delete leaf category
print_test "6.2" "Delete leaf category"
DEL_LEAF_RESP=$(curl -s -X DELETE "$BASE_URL/$GRAND_ID")

if check_success "$DEL_LEAF_RESP"; then
    print_pass "Delete leaf category successful"
else
    print_fail "Delete leaf category failed"
fi

sleep 0.5

# ============================================================
# TEST GROUP 7: BOOK-RELATED (2 tests)
# ============================================================

print_header "GROUP 7: BOOK-RELATED OPERATIONS (2 tests)"

# Test 23: Get books in category
print_test "7.1" "Get books in category"
BOOKS_RESP=$(curl -s "$BASE_URL/$ROOT_ID/books?limit=10&offset=0")

if check_success "$BOOKS_RESP"; then
    print_pass "Get books in category successful"
else
    print_fail "Get books in category failed"
fi

sleep 0.5

# Test 24: Get category book count
print_test "7.2" "Get category book count"
COUNT_RESP=$(curl -s "$BASE_URL/$ROOT_ID/book-count")

if check_success "$COUNT_RESP"; then
    BOOK_COUNT=$(extract_field "$COUNT_RESP" "book_count")
    print_pass "Get book count successful (count=$BOOK_COUNT)"
else
    print_fail "Get book count failed"
fi

sleep 0.5

# ============================================================
# CLEANUP
# ============================================================

print_header "CLEANUP"
print_info "Deleting test categories..."

# Delete in reverse order (children first)
curl -s -X DELETE "$BASE_URL/$CHILD_ID" > /dev/null 2>&1
sleep 0.3
curl -s -X DELETE "$BASE_URL/$ROOT_ID" > /dev/null 2>&1
sleep 0.3

print_info "Cleanup completed"

# ============================================================
# SUMMARY
# ============================================================

print_header "TEST SUMMARY"
echo ""
echo "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED! 🎉${NC}"
    exit 0
else
    echo -e "${RED}❌ SOME TESTS FAILED${NC}"
    exit 1
fi
