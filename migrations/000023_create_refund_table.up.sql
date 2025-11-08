-- Migration: 20251108_005_create_refund_requests.up.sql
-- Description: Create refund_requests table for refund approval workflow
-- Author: System
-- Date: 2025-11-08

-- =====================================================
-- REFUND REQUESTS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS refund_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- References
    payment_transaction_id UUID NOT NULL REFERENCES payment_transactions(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Request details
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_amount NUMERIC(12,2) NOT NULL CHECK (requested_amount > 0),
    reason TEXT NOT NULL,
    proof_images JSONB, -- Array of image URLs as proof
    
    -- Status workflow
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'approved', 'rejected', 'processing', 'completed', 'failed')
    ),
    
    -- Approval details
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    admin_notes TEXT,
    
    -- Rejection details
    rejected_by UUID REFERENCES users(id),
    rejected_at TIMESTAMPTZ,
    rejection_reason TEXT,
    
    -- Gateway refund tracking
    gateway_refund_id TEXT, -- VNPay/Momo refund transaction ID
    gateway_refund_response JSONB,
    
    -- Timestamps
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CHECK (
        (status = 'approved' AND approved_by IS NOT NULL AND approved_at IS NOT NULL) OR
        (status = 'rejected' AND rejected_by IS NOT NULL AND rejected_at IS NOT NULL) OR
        (status IN ('pending', 'processing', 'completed', 'failed'))
    )
);

-- =====================================================
-- INDEXES
-- =====================================================
CREATE INDEX idx_refund_requests_payment ON refund_requests(payment_transaction_id);
CREATE INDEX idx_refund_requests_order ON refund_requests(order_id);
CREATE INDEX idx_refund_requests_user ON refund_requests(requested_by, requested_at DESC);
CREATE INDEX idx_refund_requests_status ON refund_requests(status, requested_at DESC);

-- Find pending refund requests for admin dashboard
CREATE INDEX idx_refund_requests_pending 
ON refund_requests(status, requested_at DESC) 
WHERE status = 'pending';

-- Prevent duplicate refund requests for same payment
CREATE UNIQUE INDEX idx_refund_requests_active 
ON refund_requests(payment_transaction_id) 
WHERE status IN ('pending', 'approved', 'processing');

-- =====================================================
-- TRIGGERS
-- =====================================================

-- Auto-update updated_at
CREATE TRIGGER update_refund_requests_updated_at
BEFORE UPDATE ON refund_requests
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Sync payment transaction refund status
CREATE OR REPLACE FUNCTION sync_payment_refund_status()
RETURNS TRIGGER AS $$
BEGIN
    -- When refund approved, update payment_transactions
    IF NEW.status = 'approved' AND (OLD.status IS NULL OR OLD.status != 'approved') THEN
        UPDATE payment_transactions
        SET refund_amount = refund_amount + NEW.requested_amount,
            refund_reason = NEW.reason,
            updated_at = NOW()
        WHERE id = NEW.payment_transaction_id;
    END IF;
    
    -- When refund completed, mark payment as refunded
    IF NEW.status = 'completed' AND (OLD.status IS NULL OR OLD.status != 'completed') THEN
        UPDATE payment_transactions
        SET status = 'refunded',
            refunded_at = NEW.completed_at,
            updated_at = NOW()
        WHERE id = NEW.payment_transaction_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_payment_refund_status
AFTER UPDATE OF status ON refund_requests
FOR EACH ROW
EXECUTE FUNCTION sync_payment_refund_status();

-- =====================================================
-- HELPER FUNCTIONS
-- =====================================================

-- Check if payment has pending refund request
CREATE OR REPLACE FUNCTION has_pending_refund_request(p_payment_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM refund_requests
        WHERE payment_transaction_id = p_payment_id
        AND status IN ('pending', 'approved', 'processing')
    );
END;
$$ LANGUAGE plpgsql;

-- Get refund request stats for admin dashboard
CREATE OR REPLACE FUNCTION get_refund_request_stats()
RETURNS TABLE (
    pending_count BIGINT,
    approved_count BIGINT,
    rejected_count BIGINT,
    total_pending_amount NUMERIC,
    total_approved_amount NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
        COUNT(*) FILTER (WHERE status = 'approved') as approved_count,
        COUNT(*) FILTER (WHERE status = 'rejected') as rejected_count,
        COALESCE(SUM(requested_amount) FILTER (WHERE status = 'pending'), 0) as total_pending_amount,
        COALESCE(SUM(requested_amount) FILTER (WHERE status = 'approved'), 0) as total_approved_amount
    FROM refund_requests
    WHERE requested_at > NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENTS
-- =====================================================
COMMENT ON TABLE refund_requests IS 'Refund approval workflow - user request, admin approve/reject';
COMMENT ON COLUMN refund_requests.proof_images IS 'JSON array of image URLs as proof for refund request';
COMMENT ON COLUMN refund_requests.gateway_refund_id IS 'Gateway refund transaction ID (VNPay/Momo)';
COMMENT ON COLUMN refund_requests.status IS 'pending → approved/rejected → processing → completed/failed';

-- =====================================================
-- SEED DATA (Optional - for testing)
-- =====================================================
-- INSERT INTO refund_requests (id, payment_transaction_id, order_id, requested_by, requested_amount, reason)
-- VALUES (...);
