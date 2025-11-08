-- Migration: 20251108_006_add_retry_count_to_payments.up.sql
-- Description: Add retry_count field to payment_transactions
-- Author: System
-- Date: 2025-11-08

-- =====================================================
-- ADD RETRY TRACKING
-- =====================================================
ALTER TABLE payment_transactions 
ADD COLUMN retry_count INT NOT NULL DEFAULT 0 CHECK (retry_count >= 0);

-- =====================================================
-- INDEX FOR RETRY LIMIT CHECK
-- =====================================================
CREATE INDEX idx_payment_transactions_retry 
ON payment_transactions(order_id, retry_count) 
WHERE status IN ('failed', 'cancelled');

-- =====================================================
-- HELPER FUNCTION: Check retry limit
-- =====================================================
CREATE OR REPLACE FUNCTION can_retry_payment(p_order_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_retry_count INT;
BEGIN
    SELECT COUNT(*)
    INTO v_retry_count
    FROM payment_transactions
    WHERE order_id = p_order_id
    AND status IN ('failed', 'cancelled');
    
    RETURN v_retry_count < 3; -- Max 3 attempts
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENT
-- =====================================================
COMMENT ON COLUMN payment_transactions.retry_count IS 'Number of payment retry attempts for this order (max 3)';
