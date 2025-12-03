-- ============================================================================
-- NOTIFICATION DOMAIN - COMPLETE SEED DATA (IDEMPOTENT, UUID CHUẨN)
-- User ID: d1d1ecf4-e892-443d-bb06-94e6d2a87342
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. NOTIFICATION_TEMPLATES (UPSERT)
-- ============================================================================

INSERT INTO notification_templates (
    id, code, name, description, category,
    email_subject, email_body_html, email_body_text, sms_body,
    push_title, push_body,
    in_app_title, in_app_body, in_app_action_url,
    required_variables, language,
    default_channels, default_priority, expires_after_hours,
    version, is_active,
    created_by, updated_by,
    created_at, updated_at
) VALUES
('a1000000-0000-0000-0000-000000000001', 'promotion_removed', 'Promotion Removed from Cart', 'Notification when promotion is automatically removed from cart', 'transactional',
 'Mã giảm giá không còn khả dụng', '<p>Xin chào,</p><p>Mã giảm giá <strong>{{promo_code}}</strong> {{reason}} và đã được tự động xóa khỏi giỏ hàng của bạn.</p>', 'Mã giảm giá {{promo_code}} {{reason}} và đã được xóa khỏi giỏ hàng của bạn.',
 'Mã {{promo_code}} {{reason}}. Vui lòng kiểm tra giỏ hàng.', 'Mã giảm giá đã được xóa', 'Mã {{promo_code}} {{reason}}',
 'Mã giảm giá đã được xóa', 'Mã giảm giá "{{promo_code}}" {{reason}} và đã được xóa khỏi giỏ hàng của bạn.', '/cart',
 ARRAY['promo_code','reason','removed_at'], 'vi', ARRAY['in_app', 'email'],
 2, 720, 1, true,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'd1d1ecf4-e892-443d-bb06-94e6d2a87342',
 NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days'),

('a1000000-0000-0000-0000-000000000002', 'order_created', 'Order Confirmation', 'Notification when order is successfully created', 'transactional',
 'Đơn hàng #{{order_id}} đã được xác nhận', '<p>Cảm ơn bạn đã đặt hàng!</p><p>Đơn hàng <strong>#{{order_id}}</strong> của bạn đã được xác nhận.</p>', 'Đơn hàng #{{order_id}} đã được xác nhận.', 
 'Đơn hàng #{{order_id}} đã xác nhận.', 'Đơn hàng đã được xác nhận', 'Đơn hàng #{{order_id}} đang được xử lý',
 'Đơn hàng #{{order_id}} đã được xác nhận', 'Đơn hàng của bạn đã được xác nhận và đang được chuẩn bị.', '/orders/{{order_id}}',
 ARRAY['order_id','total_amount'], 'vi', ARRAY['in_app', 'email', 'push'],
 3, NULL, 1, true,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'd1d1ecf4-e892-443d-bb06-94e6d2a87342',
 NOW() - INTERVAL '25 days', NOW() - INTERVAL '25 days'),

('a1000000-0000-0000-0000-000000000003', 'order_delivered', 'Order Delivered Successfully', 'Notification when order is delivered', 'transactional',
 'Đơn hàng #{{order_id}} đã được giao', '<p>Đơn hàng <strong>#{{order_id}}</strong> đã được giao thành công!</p>', 'Đơn hàng #{{order_id}} đã được giao thành công!',
 'Đơn hàng #{{order_id}} đã giao thành công!', 'Đơn hàng đã được giao', 'Đơn hàng #{{order_id}} đã giao thành công',
 'Đơn hàng #{{order_id}} đã được giao', 'Đơn hàng của bạn đã được giao thành công.', '/orders/{{order_id}}',
 ARRAY['order_id','delivered_at'], 'vi', ARRAY['in_app', 'email', 'push'],
 3, 168, 1, true,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'd1d1ecf4-e892-443d-bb06-94e6d2a87342',
 NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days'),

('a1000000-0000-0000-0000-000000000004', 'new_promotion', 'New Promotion Available', 'Marketing notification for new promotions', 'marketing',
 'Khuyến mãi {{discount}}% cho đơn hàng tiếp theo!', '<p>🎉 Khuyến mãi đặc biệt!</p><p>Sử dụng mã <strong>{{promo_code}}</strong> để được giảm <strong>{{discount}}%</strong>.</p>', 'Mã {{promo_code}}: Giảm {{discount}}%.',
 'Mã {{promo_code}}: Giảm {{discount}}%', 'Khuyến mãi {{discount}}%', 'Sử dụng mã {{promo_code}}',
 'Khuyến mãi {{discount}}%', 'Sử dụng mã {{promo_code}} để được giảm {{discount}}%.', '/promotions',
 ARRAY['promo_code','discount','expires_at'], 'vi', ARRAY['in_app', 'email'],
 1, 720, 1, true,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'd1d1ecf4-e892-443d-bb06-94e6d2a87342',
 NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days'),

('a1000000-0000-0000-0000-000000000005', 'system_maintenance', 'System Maintenance Notice', 'Notification for system maintenance', 'system',
 'Thông báo bảo trì hệ thống', '<p>Hệ thống sẽ bảo trì vào lúc <strong>{{maintenance_time}}</strong>.</p>', 'Hệ thống bảo trì lúc {{maintenance_time}}.',
 'Bảo trì hệ thống: {{maintenance_time}}', 'Thông báo bảo trì', 'Hệ thống bảo trì {{maintenance_time}}',
 'Thông báo bảo trì hệ thống', 'Hệ thống sẽ bảo trì vào {{maintenance_time}}.', NULL,
 ARRAY['maintenance_time','duration'], 'vi', ARRAY['in_app', 'email', 'push'],
 3, 24, 1, false,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'd1d1ecf4-e892-443d-bb06-94e6d2a87342',
 NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = EXCLUDED.updated_at;

-- ============================================================================
-- 2. NOTIFICATION_PREFERENCES (UPSERT)
-- ============================================================================

INSERT INTO notification_preferences (
    user_id, preferences, do_not_disturb, quiet_hours_start,
    quiet_hours_end, created_at, updated_at
) VALUES (
    'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid,
    '{
        "order_created": {"in_app": true, "email": true, "push": true, "sms": false},
        "order_delivered": {"in_app": true, "email": true, "push": true, "sms": false},
        "promotion_removed": {"in_app": true, "email": false, "push": false, "sms": false},
        "new_promotion": {"in_app": true, "email": false, "push": false, "sms": false},
        "system_maintenance": {"in_app": true, "email": true, "push": true, "sms": false}
    }'::jsonb,
    false, '22:00:00'::time, '07:00:00'::time,
    NOW() - INTERVAL '35 days', NOW() - INTERVAL '5 days'
)
ON CONFLICT (user_id)
DO UPDATE SET preferences = EXCLUDED.preferences,
    do_not_disturb = EXCLUDED.do_not_disturb,
    quiet_hours_start = EXCLUDED.quiet_hours_start,
    quiet_hours_end = EXCLUDED.quiet_hours_end,
    updated_at = EXCLUDED.updated_at;

-- ============================================================================
-- 3. NOTIFICATIONS (INSERT, UUID hợp lệ)
-- ============================================================================

INSERT INTO notifications (
    id, user_id, type, title, message, data,
    channels, delivery_status,
    is_read, read_at, is_sent, sent_at,
    priority, reference_type, reference_id,
    template_code, template_version, template_data,
    idempotency_key, expires_at,
    created_at, updated_at
) VALUES
-- chỉ 3 bản ghi mẫu cho ngắn, bạn có thể bổ sung thêm bản ghi theo mẫu này:

('a2000000-0000-0000-0000-000000000001'::uuid, 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid, 'order_status', 'Đơn hàng #ORD-1001 đã được xác nhận', 'Đơn hàng của bạn đã được xác nhận và đang được chuẩn bị.', '{"order_id": "ORD-1001", "total_amount": "450000"}'::jsonb, ARRAY['in_app', 'email', 'push'], '{"in_app": "delivered", "email": "sent", "push": "delivered"}'::jsonb, false, NULL, true, NOW() - INTERVAL '2 hours', 3, 'order', 'aaaaaaaa-0000-0000-0000-000000000001'::uuid, 'order_created', 1, '{"order_id": "ORD-1001", "total_amount": "450000"}'::jsonb, 'notif-d1d1ecf4-order-ORD-1001', NULL, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'),

('a2000000-0000-0000-0000-000000000002'::uuid, 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid, 'promotion_removed', 'Mã giảm giá đã hết hạn', 'Mã giảm giá "SUMMER20" đã hết hạn và được xóa khỏi giỏ hàng của bạn.', '{"promo_code": "SUMMER20", "reason": "đã hết hạn", "removed_at": "2025-12-01T14:00:00Z"}'::jsonb, ARRAY['in_app', 'email'], '{"in_app": "delivered", "email": "sent"}'::jsonb, false, NULL, true, NOW() - INTERVAL '5 hours', 2, 'cart', 'bbbbbbbb-0000-0000-0000-000000000001'::uuid, 'promotion_removed', 1, '{"promo_code": "SUMMER20", "reason": "đã hết hạn", "removed_at": "2025-12-01T14:00:00Z"}'::jsonb, 'notif-d1d1ecf4-cart-promo-SUMMER20', NOW() + INTERVAL '30 days', NOW() - INTERVAL '5 hours', NOW() - INTERVAL '5 hours'),

('a2000000-0000-0000-0000-000000000003'::uuid, 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid, 'promotion', 'Khuyến mãi 15% cho đơn hàng tiếp theo', 'Sử dụng mã "NEW15" để được giảm 15% cho đơn hàng tiếp theo.', '{"promo_code": "NEW15", "discount": "15", "expires_at": "2025-12-15"}'::jsonb, ARRAY['in_app'], '{"in_app": "delivered"}'::jsonb, false, NULL, true, NOW() - INTERVAL '1 day', 1, NULL, NULL, 'new_promotion', 1, '{"promo_code": "NEW15", "discount": "15", "expires_at": "2025-12-15"}'::jsonb, 'notif-d1d1ecf4-promo-NEW15', NOW() + INTERVAL '14 days', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 4. NOTIFICATION_DELIVERY_LOGS (UUID hợp lệ, KHÔNG có updated_at)
-- ============================================================================

-- ============================================================================
-- 4. NOTIFICATION_DELIVERY_LOGS (UUID hợp lệ, có recipient cho mọi channel)
-- ============================================================================

INSERT INTO notification_delivery_logs (
    id, notification_id, channel, attempt_number, status,
    recipient, provider, provider_message_id, provider_response,
    error_code, error_message,
    queued_at, processing_at, sent_at, delivered_at, failed_at,
    retry_after, max_retries, estimated_cost, created_at
) VALUES
-- Log 1: in_app channel (recipient = user_id)
('b2000000-0000-0000-0000-000000000001'::uuid, 'a2000000-0000-0000-0000-000000000001'::uuid, 'in_app', 1, 'delivered',
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'internal', NULL, '{"status": "ok"}', NULL, NULL, 
 NOW() - INTERVAL '2 hours 5 minutes', NOW() - INTERVAL '2 hours 4 minutes', 
 NOW() - INTERVAL '2 hours 3 minutes', NOW() - INTERVAL '2 hours 3 minutes', NULL, 
 NULL, 3, 0, NOW() - INTERVAL '2 hours 5 minutes'),

-- Log 2: email channel (recipient = email)
('b2000000-0000-0000-0000-000000000002'::uuid, 'a2000000-0000-0000-0000-000000000001'::uuid, 'email', 1, 'sent',
 'user@example.com', 'smtp', 'smtp-msg-001', '{"status": "sent"}', NULL, NULL, 
 NOW() - INTERVAL '2 hours 5 minutes', NOW() - INTERVAL '2 hours 4 minutes', 
 NOW() - INTERVAL '2 hours 3 minutes', NULL, NULL, 
 NULL, 3, 100, NOW() - INTERVAL '2 hours 5 minutes'),

-- Log 3: push channel (recipient = device_token)
('b2000000-0000-0000-0000-000000000003'::uuid, 'a2000000-0000-0000-0000-000000000001'::uuid, 'push', 1, 'delivered',
 'device-token-123abc', 'fcm', 'fcm-msg-001', '{"status": "ok"}', NULL, NULL, 
 NOW() - INTERVAL '2 hours 5 minutes', NOW() - INTERVAL '2 hours 4 minutes', 
 NOW() - INTERVAL '2 hours 3 minutes', NOW() - INTERVAL '2 hours 2 minutes', NULL, 
 NULL, 3, 50, NOW() - INTERVAL '2 hours 5 minutes'),

-- Log 4: in_app cho notification 2
('b2000000-0000-0000-0000-000000000004'::uuid, 'a2000000-0000-0000-0000-000000000002'::uuid, 'in_app', 1, 'delivered',
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'internal', NULL, '{}', NULL, NULL, 
 NOW() - INTERVAL '5 hours', NOW() - INTERVAL '5 hours', 
 NOW() - INTERVAL '5 hours', NOW() - INTERVAL '5 hours', NULL, 
 NULL, 3, 0, NOW() - INTERVAL '5 hours'),

-- Log 5: email cho notification 2
('b2000000-0000-0000-0000-000000000005'::uuid, 'a2000000-0000-0000-0000-000000000002'::uuid, 'email', 1, 'sent',
 'user@example.com', 'smtp', 'smtp-msg-002', '{}', NULL, NULL, 
 NOW() - INTERVAL '5 hours', NOW() - INTERVAL '5 hours', 
 NOW() - INTERVAL '5 hours', NULL, NULL, 
 NULL, 3, 100, NOW() - INTERVAL '5 hours'),

-- Log 6: in_app cho notification 3
('b2000000-0000-0000-0000-000000000006'::uuid, 'a2000000-0000-0000-0000-000000000003'::uuid, 'in_app', 1, 'delivered',
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342', 'internal', NULL, '{}', NULL, NULL, 
 NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', 
 NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NULL, 
 NULL, 3, 0, NOW() - INTERVAL '1 day'),

-- Log 7: Failed SMS example (có recipient)
('b2000000-0000-0000-0000-000000000007'::uuid, 'a2000000-0000-0000-0000-000000000003'::uuid, 'sms', 1, 'failed',
 '+84901234567', 'twilio', NULL, '{"error": "Provider not configured"}', 
 'PROVIDER_NOT_CONFIGURED', 'SMS provider not configured', 
 NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NULL, NULL, 
 NOW() - INTERVAL '1 day', NULL, 0, 0, NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;


-- ============================================================================
-- 5. NOTIFICATION_CAMPAIGNS (UUID hợp lệ)
-- ============================================================================

-- INSERT INTO notification_campaigns (
--     id, name, description, template_code,
--     target_type, target_segment, target_user_ids,
--     channels, template_data, scheduled_at, started_at, completed_at, cancelled_at,
--     status, batch_size, batch_delay_seconds,
--     total_recipients, processed_count, sent_count, delivered_count, failed_count,
--     created_by, created_at, updated_at
-- ) VALUES
-- ('c3000000-0000-0000-0000-000000000001'::uuid, 'Tết 2025 Promotion Campaign', 'Gửi thông báo khuyến mãi Tết cho tất cả user', 'new_promotion',
--  'all_users', NULL, NULL, ARRAY['in_app', 'email'],
--  '{"promo_code":"TET2025","discount":"25","expires_at":"2025-02-15"}'::jsonb,
--  NULL, NULL, NULL, NULL, 'draft', 1000, 5, NULL, 0, 0, 0, 0,
--  'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid,
--  NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days')
-- ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 6. NOTIFICATION_RATE_LIMITS (UPSERT)
-- ============================================================================

-- ============================================================================
-- 5. NOTIFICATION_CAMPAIGNS (UUID hợp lệ, KHÔNG có updated_by)
-- ============================================================================

INSERT INTO notification_campaigns (
    id, name, description, template_code,
    target_type, target_segment, target_user_ids,
    channels, template_data, 
    scheduled_at, started_at, completed_at, cancelled_at,
    status, batch_size, batch_delay_seconds,
    total_recipients, processed_count, sent_count, delivered_count, failed_count,
    created_by, created_at, updated_at
) VALUES
-- Campaign 1: Draft
('c3000000-0000-0000-0000-000000000001'::uuid, 
 'Tết 2025 Promotion Campaign', 
 'Gửi thông báo khuyến mãi Tết cho tất cả user', 
 'new_promotion',
 'all_users', NULL, NULL, 
 ARRAY['in_app', 'email'],
 '{"promo_code":"TET2025","discount":"25","expires_at":"2025-02-15"}'::jsonb,
 NULL, NULL, NULL, NULL, 
 'draft', 1000, 5, 
 NULL, 0, 0, 0, 0,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid,
 NOW() - INTERVAL '2 days', 
 NOW() - INTERVAL '2 days'),

-- Campaign 2: Scheduled
('c3000000-0000-0000-0000-000000000002'::uuid,
 'December Flash Sale',
 'Flash sale cuối tháng 12',
 'new_promotion',
 'segment', 'active_users', NULL,
 ARRAY['in_app', 'email', 'push'],
 '{"promo_code":"FLASH12","discount":"30"}'::jsonb,
 NOW() + INTERVAL '1 day', NULL, NULL, NULL,
 'scheduled', 500, 3,
 NULL, 0, 0, 0, 0,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid,
 NOW() - INTERVAL '1 day',
 NOW() - INTERVAL '1 day'),

-- Campaign 3: Running
('c3000000-0000-0000-0000-000000000003'::uuid,
 'System Maintenance Notice',
 'Thông báo bảo trì hệ thống',
 'system_maintenance',
 'all_users', NULL, NULL,
 ARRAY['in_app', 'email', 'push'],
 '{"maintenance_time":"2025-12-05T23:00:00Z","duration":"3 giờ"}'::jsonb,
 NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '25 minutes', NULL, NULL,
 'running', 1000, 5,
 5000, 2500, 2400, 2200, 100,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid,
 NOW() - INTERVAL '2 hours',
 NOW() - INTERVAL '5 minutes'),

-- Campaign 4: Completed
('c3000000-0000-0000-0000-000000000004'::uuid,
 'November Mega Sale',
 'Khuyến mãi lớn tháng 11 đã hoàn thành',
 'new_promotion',
 'all_users', NULL, NULL,
 ARRAY['in_app', 'email'],
 '{"promo_code":"MEGA11","discount":"40"}'::jsonb,
 NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days', NOW() - INTERVAL '9 days', NULL,
 'completed', 1000, 5,
 8500, 8500, 8400, 8100, 400,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid,
 NOW() - INTERVAL '11 days',
 NOW() - INTERVAL '9 days'),

-- Campaign 5: Cancelled
('c3000000-0000-0000-0000-000000000005'::uuid,
 'Black Friday 2024 (Cancelled)',
 'Campaign bị hủy vì lý do nội bộ',
 'new_promotion',
 'segment', 'vip_users', NULL,
 ARRAY['in_app', 'email', 'push'],
 '{"promo_code":"BF2024","discount":"50"}'::jsonb,
 NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days', NULL, NOW() - INTERVAL '19 days',
 'cancelled', 500, 3,
 2000, 450, 430, 400, 20,
 'd1d1ecf4-e892-443d-bb06-94e6d2a87342'::uuid,
 NOW() - INTERVAL '22 days',
 NOW() - INTERVAL '19 days')
ON CONFLICT (id) DO NOTHING;


COMMIT;
