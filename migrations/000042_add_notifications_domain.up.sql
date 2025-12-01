-- ================================================
-- ENHANCEMENT 1: NOTIFICATION TEMPLATES
-- ================================================

CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Template identification
    code VARCHAR(100) UNIQUE NOT NULL,  -- order_confirmed, payment_failed, promo_expired
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,  -- transactional, marketing, system
    
    -- Channel-specific templates
    -- WHY SEPARATE? Each channel needs different format
    email_subject TEXT,
    email_body_html TEXT,  -- HTML template with {{placeholders}}
    email_body_text TEXT,  -- Plain text fallback
    
    sms_body TEXT,  -- SMS template (160 chars limit)
    push_title TEXT,
    push_body TEXT,
    
    in_app_title TEXT,
    in_app_body TEXT,
    in_app_action_url TEXT,  -- Deep link for app
    
    -- Template variables
    -- WHY? Document what variables this template expects
    -- Example: ["order_number", "total_amount", "delivery_date"]
    required_variables TEXT[],
    
    -- Multi-language support
    language VARCHAR(5) DEFAULT 'vi',
    
    -- Default settings
    default_channels TEXT[] DEFAULT '{in_app}',
    default_priority INT DEFAULT 2,
    expires_after_hours INT,  -- Auto-expire after X hours
    
    -- Versioning
    -- WHY? Track template changes over time
    version INT DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Audit
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_templates_code ON notification_templates(code, is_active);
CREATE INDEX idx_templates_category ON notification_templates(category, is_active);

-- ================================================
-- ENHANCEMENT 2: NOTIFICATION DELIVERY LOGS
-- ================================================

CREATE TABLE IF NOT EXISTS notification_delivery_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Reference to notification
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    
    -- Delivery attempt details
    channel VARCHAR(20) NOT NULL,  -- email, sms, push
    attempt_number INT DEFAULT 1,
    
    -- Status tracking
    status VARCHAR(50) NOT NULL,  -- queued, processing, sent, delivered, failed, bounced, opened, clicked
    
    -- Recipient info
    recipient VARCHAR(255) NOT NULL,  -- email address / phone number / device token
    
    -- Provider details
    provider VARCHAR(50),  -- aws_ses, twilio, fcm, apns
    provider_message_id VARCHAR(255),  -- External tracking ID
    provider_response JSONB,  -- Full provider response
    
    -- Error tracking
    error_code VARCHAR(50),
    error_message TEXT,
    
    -- Performance metrics
    queued_at TIMESTAMPTZ,
    processing_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    
    -- Retry metadata
    retry_after TIMESTAMPTZ,
    max_retries INT DEFAULT 3,
    
    -- Cost tracking (optional)
    estimated_cost DECIMAL(10,4),  -- Track SMS/email costs
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_delivery_logs_notification ON notification_delivery_logs(notification_id);
CREATE INDEX idx_delivery_logs_status ON notification_delivery_logs(channel, status);
CREATE INDEX idx_delivery_logs_failed ON notification_delivery_logs(status, retry_after) 
    WHERE status = 'failed' AND retry_after IS NOT NULL;

-- ================================================
-- ENHANCEMENT 3: NOTIFICATION CAMPAIGNS
-- ================================================

CREATE TABLE IF NOT EXISTS notification_campaigns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Campaign info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_code VARCHAR(100) REFERENCES notification_templates(code),
    
    -- Targeting
    target_type VARCHAR(50) NOT NULL,  -- all_users, segment, specific_users
    target_segment VARCHAR(50),  -- vip_users, inactive_users, new_users
    target_user_ids UUID[],  -- For specific users
    
    -- Targeting filters (JSONB for flexibility)
    -- Example: {"last_order_days_ago": ">30", "total_spent": ">1000000"}
    target_filters JSONB,
    
    -- Scheduling
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    
    -- Status
    status VARCHAR(50) DEFAULT 'draft',  -- draft, scheduled, running, paused, completed, cancelled
    
    -- Batch processing
    batch_size INT DEFAULT 1000,  -- Process N users per batch
    batch_delay_seconds INT DEFAULT 5,  -- Delay between batches
    
    -- Progress tracking
    total_recipients INT,
    processed_count INT DEFAULT 0,
    sent_count INT DEFAULT 0,
    delivered_count INT DEFAULT 0,
    failed_count INT DEFAULT 0,
    
    -- Template data (same for all recipients)
    template_data JSONB,
    
    -- Channels to use
    channels TEXT[],
    
    -- Audit
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_campaigns_status ON notification_campaigns(status, scheduled_at);
CREATE INDEX idx_campaigns_created_by ON notification_campaigns(created_by);

-- ================================================
-- ENHANCEMENT 4: RATE LIMITING
-- ================================================

CREATE TABLE IF NOT EXISTS notification_rate_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Scope
    scope VARCHAR(50) NOT NULL,  -- global, user, notification_type
    scope_id VARCHAR(255),  -- user_id or notification type
    
    -- Limits
    max_notifications INT NOT NULL,
    window_minutes INT NOT NULL,  -- Time window
    
    -- Current usage (reset periodically by background job)
    current_count INT DEFAULT 0,
    window_start TIMESTAMPTZ DEFAULT NOW(),
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(scope, scope_id, window_minutes)
);

CREATE INDEX idx_rate_limits_scope ON notification_rate_limits(scope, scope_id);

-- ================================================
-- ENHANCEMENT 5: UPDATE NOTIFICATIONS TABLE
-- ================================================

-- Add template reference
ALTER TABLE notifications 
ADD COLUMN template_code VARCHAR(100) REFERENCES notification_templates(code),
ADD COLUMN template_version INT,
ADD COLUMN template_data JSONB;  -- Variables used to render template

COMMENT ON COLUMN notifications.template_code IS 
'Reference to notification template used (if any)';

COMMENT ON COLUMN notifications.template_data IS 
'Variables passed to template: {"order_number": "ORD-123", "total": 500000}';

-- ================================================
-- ENHANCEMENT 6: FUNCTIONS FOR COMMON OPERATIONS
-- ================================================

-- Function: Get unread count per user
CREATE OR REPLACE FUNCTION get_unread_notification_count(p_user_id UUID)
RETURNS INT AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)
        FROM notifications
        WHERE user_id = p_user_id 
        AND is_read = FALSE
        AND (expires_at IS NULL OR expires_at > NOW())
    );
END;
$$ LANGUAGE plpgsql;

-- Function: Mark all as read
CREATE OR REPLACE FUNCTION mark_all_notifications_read(p_user_id UUID)
RETURNS INT AS $$
DECLARE
    updated_count INT;
BEGIN
    UPDATE notifications
    SET is_read = TRUE, read_at = NOW()
    WHERE user_id = p_user_id AND is_read = FALSE;
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Function: Check rate limit
CREATE OR REPLACE FUNCTION check_rate_limit(
    p_scope VARCHAR,
    p_scope_id VARCHAR,
    p_max_notifications INT,
    p_window_minutes INT
) RETURNS BOOLEAN AS $$
DECLARE
    current_usage INT;
    limit_start TIMESTAMPTZ;
BEGIN
    -- Get or create rate limit record
    INSERT INTO notification_rate_limits (scope, scope_id, max_notifications, window_minutes)
    VALUES (p_scope, p_scope_id, p_max_notifications, p_window_minutes)
    ON CONFLICT (scope, scope_id, window_minutes) 
    DO UPDATE SET updated_at = NOW()
    RETURNING current_count, window_start INTO current_usage, limit_start;
    
    -- Reset if window expired
    IF limit_start + (p_window_minutes || ' minutes')::INTERVAL < NOW() THEN
        UPDATE notification_rate_limits
        SET current_count = 0, window_start = NOW()
        WHERE scope = p_scope AND scope_id = p_scope_id AND window_minutes = p_window_minutes;
        RETURN TRUE;
    END IF;
    
    -- Check if under limit
    RETURN current_usage < p_max_notifications;
END;
$$ LANGUAGE plpgsql;

-- Function: Increment rate limit counter
CREATE OR REPLACE FUNCTION increment_rate_limit(
    p_scope VARCHAR,
    p_scope_id VARCHAR,
    p_window_minutes INT
) RETURNS VOID AS $$
BEGIN
    UPDATE notification_rate_limits
    SET current_count = current_count + 1
    WHERE scope = p_scope AND scope_id = p_scope_id AND window_minutes = p_window_minutes;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- ENHANCEMENT 7: VIEWS FOR COMMON QUERIES
-- ================================================

-- View: Active notifications per user
CREATE OR REPLACE VIEW v_active_notifications AS
SELECT 
    n.*,
    t.name as template_name,
    t.category as notification_category
FROM notifications n
LEFT JOIN notification_templates t ON n.template_code = t.code
WHERE n.is_read = FALSE
AND (n.expires_at IS NULL OR n.expires_at > NOW());

-- View: Notification delivery metrics
CREATE OR REPLACE VIEW v_notification_metrics AS
SELECT 
    n.type,
    n.created_at::DATE as date,
    COUNT(*) as total_notifications,
    COUNT(*) FILTER (WHERE n.is_sent = TRUE) as sent_count,
    COUNT(*) FILTER (WHERE n.is_read = TRUE) as read_count,
    COUNT(DISTINCT n.user_id) as unique_users,
    AVG(EXTRACT(EPOCH FROM (n.read_at - n.created_at))) FILTER (WHERE n.read_at IS NOT NULL) as avg_time_to_read_seconds
FROM notifications n
GROUP BY n.type, n.created_at::DATE;

-- View: Campaign performance
CREATE OR REPLACE VIEW v_campaign_performance AS
SELECT 
    c.id,
    c.name,
    c.status,
    c.total_recipients,
    c.sent_count,
    c.delivered_count,
    c.failed_count,
    CASE 
        WHEN c.total_recipients > 0 
        THEN ROUND((c.delivered_count::DECIMAL / c.total_recipients) * 100, 2)
        ELSE 0 
    END as delivery_rate_percent,
    c.started_at,
    c.completed_at,
    EXTRACT(EPOCH FROM (c.completed_at - c.started_at)) / 60 as duration_minutes
FROM notification_campaigns c
WHERE c.status IN ('completed', 'running');

-- ================================================
-- SEED DEFAULT TEMPLATES
-- ================================================

INSERT INTO notification_templates (code, name, category, email_subject, email_body_html, in_app_title, in_app_body, required_variables, default_channels, default_priority)
VALUES 
(
    'order_confirmed',
    'Order Confirmation',
    'transactional',
    'Đơn hàng {{order_number}} đã được xác nhận',
    '<h2>Cảm ơn bạn đã đặt hàng!</h2><p>Đơn hàng <strong>{{order_number}}</strong> của bạn đã được xác nhận.</p><p>Tổng tiền: <strong>{{total_amount}} VNĐ</strong></p>',
    'Đơn hàng đã xác nhận',
    'Đơn hàng {{order_number}} đã được xác nhận. Tổng tiền: {{total_amount}} VNĐ',
    ARRAY['order_number', 'total_amount'],
    ARRAY['in_app', 'email'],
    3
),
(
    'promotion_removed',
    'Promotion Expired',
    'transactional',
    'Mã giảm giá {{promo_code}} đã hết hạn',
    '<p>Mã giảm giá <strong>{{promo_code}}</strong> trong giỏ hàng của bạn đã hết hạn và đã được gỡ bỏ.</p>',
    'Mã giảm giá hết hạn',
    'Mã {{promo_code}} đã hết hạn và bị gỡ khỏi giỏ hàng',
    ARRAY['promo_code'],
    ARRAY['in_app'],
    2
),
(
    'payment_success',
    'Payment Successful',
    'transactional',
    'Thanh toán thành công cho đơn hàng {{order_number}}',
    '<h2>Thanh toán thành công!</h2><p>Chúng tôi đã nhận được thanh toán <strong>{{amount}} VNĐ</strong> cho đơn hàng {{order_number}}.</p>',
    'Thanh toán thành công',
    'Đã thanh toán {{amount}} VNĐ cho đơn {{order_number}}',
    ARRAY['order_number', 'amount'],
    ARRAY['in_app', 'email'],
    3
);

-- ================================================
-- COMMENTS
-- ================================================

COMMENT ON TABLE notification_templates IS 
'Reusable notification templates supporting multi-channel and multi-language';

COMMENT ON TABLE notification_delivery_logs IS 
'Detailed delivery tracking per channel with provider response and retry logic';

COMMENT ON TABLE notification_campaigns IS 
'Batch notification campaigns for marketing and announcements';

COMMENT ON TABLE notification_rate_limits IS 
'Rate limiting to prevent notification spam';
