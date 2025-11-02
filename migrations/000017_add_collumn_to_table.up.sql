-- ================================================
-- Migration: Add Version Control for Conflict Prevention
-- Purpose: Optimistic locking to prevent race conditions
-- Created: 2025-11-02
-- Description: Add version column to critical tables
-- ================================================

-- ================================================
-- TIER 1: CRITICAL (MUST HAVE)
-- ================================================

-- 1. INVENTORIES - Stock Management (HIGHEST PRIORITY)
-- ================================================
-- Risk: Over-selling, concurrent stock updates
-- Solution: Lock row, check version before deduct

ALTER TABLE inventories 
ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;

-- Update existing rows to version 1 (so they can be updated)
UPDATE inventories 
SET version = 1 
WHERE version = 0;

-- Create index for fast version checks
CREATE INDEX IF NOT EXISTS idx_inventories_version 
ON inventories(version);

-- Add comment
COMMENT ON COLUMN inventories.version IS 
'CRITICAL: Optimistic locking version. Prevents race conditions on stock updates. Increment on every UPDATE. Used in pessimistic locking (FOR UPDATE).';

-- ================================================
-- 2. ORDERS - Transaction Management (HIGHEST PRIORITY)
-- ================================================
-- Risk: Double charge, concurrent status changes
-- Solution: Check version before status update

ALTER TABLE orders 
ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;

-- Update existing rows to version 1
UPDATE orders 
SET version = 1 
WHERE version = 0;

-- Create index for fast version checks
CREATE INDEX IF NOT EXISTS idx_orders_version 
ON orders(version);

-- Add comment
COMMENT ON COLUMN orders.version IS 
'CRITICAL: Optimistic locking version. Prevents concurrent status changes and double-charge. Increment on every UPDATE. Handle webhook idempotency with payment_id.';

-- ================================================
-- 3. CARTS - Shopping Cart (HIGHEST PRIORITY)
-- ================================================
-- Risk: Concurrent item adds/removes, checkout race
-- Solution: Lock cart during checkout, check version

ALTER TABLE carts 
ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;

-- Update existing rows to version 1
UPDATE carts 
SET version = 1 
WHERE version = 0;

-- Create index for fast version checks
CREATE INDEX IF NOT EXISTS idx_carts_version 
ON carts(version);

-- Add comment
COMMENT ON COLUMN carts.version IS 
'CRITICAL: Optimistic locking version. Prevents concurrent cart modifications. Increment on every UPDATE. Lock during checkout to prevent concurrent checkouts.';

-- ================================================
-- 4. CART_ITEMS - Cart Items (HIGHEST PRIORITY)
-- ================================================
-- Risk: Concurrent quantity changes, item removal race
-- Solution: Check version before quantity update

ALTER TABLE cart_items 
ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;

-- Update existing rows to version 1
UPDATE cart_items 
SET version = 1 
WHERE version = 0;

-- Create index for fast version checks
CREATE INDEX IF NOT EXISTS idx_cart_items_version 
ON cart_items(version);

-- Add comment
COMMENT ON COLUMN cart_items.version IS 
'Optimistic locking version. Prevents concurrent quantity/item changes. Increment on every UPDATE.';

-- ================================================
-- TIER 2: HIGH PRIORITY (SHOULD HAVE)
-- ================================================

-- 5. BOOKS - Book Metadata
-- ================================================
-- Risk: Concurrent price updates, metadata races
-- Solution: Check version before update

ALTER TABLE books 
ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;

-- Update existing rows to version 1
UPDATE books 
SET version = 1 
WHERE version = 0;

-- Create index for fast version checks
CREATE INDEX IF NOT EXISTS idx_books_version 
ON books(version);

-- Add comment
COMMENT ON COLUMN books.version IS 
'Optimistic locking version. Prevents concurrent price/metadata updates. Increment on every UPDATE.';

-- ================================================
-- 6. PROMOTIONS - Promotion Management
-- ================================================
-- Risk: Concurrent budget updates, discount races
-- Solution: Check version before update

ALTER TABLE promotions 
ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;

-- Update existing rows to version 1
UPDATE promotions 
SET version = 1 
WHERE version = 0;

-- Create index for fast version checks
CREATE INDEX IF NOT EXISTS idx_promotions_version 
ON promotions(version);

-- Add comment
COMMENT ON COLUMN promotions.version IS 
'Optimistic locking version. Prevents concurrent promotion edits and budget races. Increment on every UPDATE.';

-- ================================================
-- 7. PROMOTION_USAGE - Usage Counter (CRITICAL)
-- ================================================
-- Risk: Counter increment races, limit bypass
-- Solution: Lock on counter update, check version

ALTER TABLE promotion_usage 
ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 0;

-- Update existing rows to version 1
UPDATE promotion_usage 
SET version = 1 
WHERE version = 0;

-- Create index for fast version checks
CREATE INDEX IF NOT EXISTS idx_promotion_usage_version 
ON promotion_usage(version);

-- Add comment
COMMENT ON COLUMN promotion_usage.version IS 
'CRITICAL: Optimistic locking version. Prevents concurrent usage increments and limit bypass. Increment on UPDATE. Use pessimistic lock (FOR UPDATE) on counter updates.';

