-- migrations/000xxx_create_checkout_idempotency.up.sql

CREATE TABLE checkout_idempotency (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    user_id UUID NOT NULL,
    cart_id UUID NULL,
    order_id UUID NULL,
    status VARCHAR(20) NOT NULL,  -- processing, completed, failed
    request_payload JSONB NOT NULL,
    response_data JSONB NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP NULL,
    expires_at TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE SET NULL,
    CONSTRAINT chk_status CHECK (status IN ('processing', 'completed', 'failed'))
);

CREATE INDEX idx_idempotency_user_created ON checkout_idempotency(user_id, created_at DESC);
CREATE INDEX idx_idempotency_expires ON checkout_idempotency(expires_at) WHERE status = 'processing';

-- Cleanup job index
CREATE INDEX idx_idempotency_cleanup ON checkout_idempotency(expires_at, status);
