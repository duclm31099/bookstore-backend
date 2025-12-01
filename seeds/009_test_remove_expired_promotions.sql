-- =====================================================
-- TEST DATA: Remove Expired Promotions Job
-- Purpose: Test scenarios for auto-remove expired promotions
-- =====================================================

-- Update user login status for testing
UPDATE users SET last_login_at = NOW() - INTERVAL '1 day'
WHERE id = '10000000-0000-0000-0000-000000000001';

UPDATE users SET last_login_at = NOW() - INTERVAL '3 days'
WHERE id = '10000000-0000-0000-0000-000000000003';

UPDATE users SET last_login_at = NOW() - INTERVAL '7 days'
WHERE id = '10000000-0000-0000-0000-000000000005';

UPDATE users SET last_login_at = NOW() - INTERVAL '10 days'
WHERE id = '10000000-0000-0000-0000-000000000006';

UPDATE users SET last_login_at = NOW() - INTERVAL '15 days'
WHERE id = '10000000-0000-0000-0000-000000000008';

-- Verify required users exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users WHERE id = '10000000-0000-0000-0000-000000000001') THEN
        RAISE EXCEPTION 'User 10000000-0000-0000-0000-000000000001 does not exist. Run seed-1 first.';
    END IF;
END $$;

-- SCENARIO 1: Active User with Expired Promotion
INSERT INTO carts (id, user_id, promo_code, discount, total, subtotal, items_count, created_at, updated_at, expires_at)
VALUES (
    'c0000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    'TET2025',
    100000,
    400000,
    500000,
    2,
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '1 day',
    NOW() + INTERVAL '30 days'
)
ON CONFLICT (id) DO UPDATE SET 
    updated_at = NOW() - INTERVAL '1 day',
    created_at = NOW() - INTERVAL '2 days';

-- SCENARIO 2: Inactive User with Expired Promotion
INSERT INTO carts (id, user_id, promo_code, discount, total, subtotal, items_count, promo_metadata, created_at, updated_at, expires_at)
VALUES (
    'c0000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000006',
    'SUMMER23',
    75000,
    225000,
    300000,
    1,
    jsonb_build_object('last_checked_at', to_char(NOW() - INTERVAL '25 hours', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')),
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '2 days',
    NOW() + INTERVAL '30 days'
)
ON CONFLICT (id) DO UPDATE SET 
    promo_metadata = jsonb_build_object('last_checked_at', to_char(NOW() - INTERVAL '25 hours', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')),
    updated_at = NOW() - INTERVAL '2 days';

-- SCENARIO 3: Active User with Disabled Promotion
INSERT INTO carts (id, user_id, promo_code, discount, total, subtotal, items_count, created_at, updated_at, expires_at)
VALUES (
    'c0000000-0000-0000-0000-000000000003',
    '10000000-0000-0000-0000-000000000003',
    'BLACKFRI',
    250000,
    750000,
    1000000,
    3,
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '6 hours',
    NOW() + INTERVAL '30 days'
)
ON CONFLICT (id) DO UPDATE SET 
    updated_at = NOW() - INTERVAL '6 hours',
    created_at = NOW() - INTERVAL '1 day';

-- SCENARIO 4: Active User with Valid Promotion
INSERT INTO carts (id, user_id, promo_code, discount, total, subtotal, items_count, created_at, updated_at, expires_at)
VALUES (
    'c0000000-0000-0000-0000-000000000004',
    '10000000-0000-0000-0000-000000000005',
    'WELCOME10',
    50000,
    450000,
    500000,
    2,
    NOW() - INTERVAL '3 hours',
    NOW() - INTERVAL '1 hour',
    NOW() + INTERVAL '30 days'
)
ON CONFLICT (id) DO UPDATE SET 
    updated_at = NOW() - INTERVAL '1 hour',
    created_at = NOW() - INTERVAL '3 hours';

-- SCENARIO 5: Inactive User - Recently Checked
INSERT INTO carts (id, user_id, promo_code, discount, total, subtotal, items_count, promo_metadata, created_at, updated_at, expires_at)
VALUES (
    'c0000000-0000-0000-0000-000000000005',
    '10000000-0000-0000-0000-000000000008',
    'XMAS2024',
    150000,
    350000,
    500000,
    2,
    jsonb_build_object('last_checked_at', to_char(NOW() - INTERVAL '12 hours', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')),
    NOW() - INTERVAL '7 days',
    NOW() - INTERVAL '3 days',
    NOW() + INTERVAL '30 days'
)
ON CONFLICT (id) DO UPDATE SET 
    promo_metadata = jsonb_build_object('last_checked_at', to_char(NOW() - INTERVAL '12 hours', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')),
    updated_at = NOW() - INTERVAL '3 days';
