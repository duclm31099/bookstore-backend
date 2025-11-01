-- ================================================
-- PAYMENT TRANSACTIONS TABLE
-- Track all payment gateway interactions
-- ================================================
CREATE TABLE IF NOT EXISTS payment_transactions (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Gateway information
    gateway TEXT NOT NULL CHECK (gateway IN ('cod', 'vnpay', 'momo', 'bank_transfer')),
    transaction_id TEXT, -- Gateway's unique transaction ID (VNPay txnRef, Momo transId)
    
    -- Amount
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL DEFAULT 'VND',
    
    -- Status tracking
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'processing', 'success', 'failed', 'refunded', 'cancelled')
    ),
    
    -- Error handling
    error_code TEXT,
    error_message TEXT,
    
    -- Gateway response (raw webhook data)
    gateway_response JSONB,
    gateway_signature TEXT, -- For signature verification
    
    -- Payment method details (for COD/Bank Transfer)
    payment_details JSONB, -- {bank_code, card_type, etc.}
    
    -- Refund tracking
    refund_amount NUMERIC(12,2) DEFAULT 0 CHECK (refund_amount >= 0 AND refund_amount <= amount),
    refund_reason TEXT,
    refunded_at TIMESTAMPTZ,
    
    -- Timestamps
    initiated_at TIMESTAMPTZ DEFAULT NOW(),
    processing_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CHECK (
        (status = 'success' AND completed_at IS NOT NULL) OR
        (status = 'failed' AND failed_at IS NOT NULL) OR
        (status IN ('pending', 'processing', 'cancelled'))
    )
);

-- ================================================
-- PAYMENT WEBHOOK LOGS TABLE
-- Log all webhook attempts (for debugging & idempotency)
-- ================================================
CREATE TABLE IF NOT EXISTS payment_webhook_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Transaction reference
    payment_transaction_id UUID REFERENCES payment_transactions(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Webhook details
    gateway TEXT NOT NULL,
    webhook_event TEXT, -- 'payment.success', 'payment.failed', etc.
    
    -- Request data
    headers JSONB,
    body JSONB,
    signature TEXT,
    
    -- Processing result
    is_valid BOOLEAN, -- Signature validation result
    is_processed BOOLEAN DEFAULT false,
    processing_error TEXT,
    
    -- Timestamp
    received_at TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================
-- INDEXES
-- ================================================

-- Payment Transactions
CREATE INDEX idx_payment_transactions_order ON payment_transactions(order_id, created_at DESC);
CREATE INDEX idx_payment_transactions_status ON payment_transactions(status, created_at DESC);
CREATE INDEX idx_payment_transactions_gateway ON payment_transactions(gateway, status);

-- Unique constraint: one transaction ID per gateway
CREATE UNIQUE INDEX idx_payment_transactions_gateway_txn_id 
ON payment_transactions(gateway, transaction_id) 
WHERE transaction_id IS NOT NULL;

-- Find pending/processing payments (for retry jobs)
CREATE INDEX idx_payment_transactions_pending 
ON payment_transactions(status, initiated_at) 
WHERE status IN ('pending', 'processing');

-- Webhook Logs
CREATE INDEX idx_payment_webhook_logs_transaction ON payment_webhook_logs(payment_transaction_id);
CREATE INDEX idx_payment_webhook_logs_order ON payment_webhook_logs(order_id);
CREATE INDEX idx_payment_webhook_logs_received ON payment_webhook_logs(received_at DESC);

-- Idempotency: prevent duplicate webhook processing
CREATE UNIQUE INDEX idx_payment_webhook_logs_idempotency 
ON payment_webhook_logs(gateway, webhook_event, (body->>'transaction_id'))
WHERE is_processed = true;

-- ================================================
-- TRIGGERS
-- ================================================

-- Auto-update updated_at
CREATE TRIGGER update_payment_transactions_updated_at
BEFORE UPDATE ON payment_transactions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Auto-sync order payment status
CREATE OR REPLACE FUNCTION sync_order_payment_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Update order when payment succeeds
    IF NEW.status = 'success' AND (OLD.status IS NULL OR OLD.status != 'success') THEN
        UPDATE orders
        SET payment_status = 'paid',
            paid_at = NEW.completed_at,
            status = CASE 
                WHEN status = 'pending' THEN 'confirmed'
                ELSE status
            END
        WHERE id = NEW.order_id;
    END IF;
    
    -- Update order when payment fails
    IF NEW.status = 'failed' AND (OLD.status IS NULL OR OLD.status != 'failed') THEN
        UPDATE orders
        SET payment_status = 'failed'
        WHERE id = NEW.order_id;
    END IF;
    
    -- Update order when refunded
    IF NEW.status = 'refunded' AND (OLD.status IS NULL OR OLD.status != 'refunded') THEN
        UPDATE orders
        SET payment_status = 'refunded',
            status = 'cancelled'
        WHERE id = NEW.order_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_order_payment_status
AFTER UPDATE OF status ON payment_transactions
FOR EACH ROW
EXECUTE FUNCTION sync_order_payment_status();

-- ================================================
-- HELPER FUNCTIONS
-- ================================================

-- Check if payment is still pending (for retry jobs)
CREATE OR REPLACE FUNCTION is_payment_expired(p_transaction_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM payment_transactions
        WHERE id = p_transaction_id
          AND status IN ('pending', 'processing')
          AND initiated_at < NOW() - INTERVAL '15 minutes'
    );
END;
$$ LANGUAGE plpgsql;

-- Get latest transaction for order
CREATE OR REPLACE FUNCTION get_latest_payment_transaction(p_order_id UUID)
RETURNS UUID AS $$
BEGIN
    RETURN (
        SELECT id
        FROM payment_transactions
        WHERE order_id = p_order_id
        ORDER BY created_at DESC
        LIMIT 1
    );
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE payment_transactions IS 'Payment gateway transactions (VNPay, Momo, Bank Transfer, COD)';
COMMENT ON COLUMN payment_transactions.transaction_id IS 'Gateway unique transaction ID (vnp_TxnRef, momo_transId)';
COMMENT ON COLUMN payment_transactions.gateway_response IS 'Raw webhook JSON for debugging';
COMMENT ON COLUMN payment_transactions.gateway_signature IS 'Signature from gateway (for verification)';

COMMENT ON TABLE payment_webhook_logs IS 'Webhook request logs for debugging and idempotency';
COMMENT ON COLUMN payment_webhook_logs.is_processed IS 'Prevent duplicate processing';
