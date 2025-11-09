-- =====================================================
-- 9. PROMOTIONS (20 promotions - active, expired, various types)
-- =====================================================

INSERT INTO promotions (
    id, code, name, description,
    discount_type, discount_value, max_discount_amount,
    min_order_amount, applicable_category_ids, first_order_only,
    max_uses, max_uses_per_user, current_uses,
    starts_at, expires_at, is_active,
    created_at, updated_at
) VALUES

-- ========================================
-- ACTIVE PROMOTIONS (12 active)
-- ========================================

-- 1. Welcome promotion (first order only)
(
    '80000000-0000-0000-0000-000000000001',
    'WELCOME10',
    'Chào mừng khách hàng mới',
    'Giảm 10% cho đơn hàng đầu tiên, tối đa 50k',
    'percentage', 10.00, 50000,
    100000, NULL, true,  -- first_order_only = true
    1000, 1, 234,
    NOW() - INTERVAL '2 months', NOW() + INTERVAL '2 months', true,
    NOW() - INTERVAL '2 months', NOW()
),

-- 2. Book lovers (category specific - Văn học)
(
    '80000000-0000-0000-0000-000000000002',
    'BOOK20',
    'Yêu sách - Giảm 20%',
    'Giảm 20% cho sách văn học, tối đa 100k',
    'percentage', 20.00, 100000,
    500000, 
    ARRAY['30000000-0000-0000-0000-000000000001'::uuid], -- Văn học category
    false,
    500, 1, 145,
    NOW() - INTERVAL '1 month', NOW() + INTERVAL '3 months', true,
    NOW() - INTERVAL '1 month', NOW()
),

-- 3. Free shipping
(
    '80000000-0000-0000-0000-000000000003',
    'FREESHIP',
    'Miễn phí vận chuyển',
    'Miễn phí ship cho đơn từ 200k',
    'fixed', 30000, 30000,
    200000, NULL, false,
    5000, 2, 1234,
    NOW() - INTERVAL '1 month', NOW() + INTERVAL '2 months', true,
    NOW() - INTERVAL '1 month', NOW()
),

-- 4. Tech books discount
(
    '80000000-0000-0000-0000-000000000004',
    'TECH50',
    'Sách công nghệ giảm 50k',
    'Giảm 50k cho sách công nghệ từ 300k',
    'fixed', 50000, 50000,
    300000,
    ARRAY['30000000-0000-0000-0000-000000000005'::uuid], -- Công nghệ category
    false,
    300, 1, 89,
    NOW() - INTERVAL '2 weeks', NOW() + INTERVAL '6 weeks', true,
    NOW() - INTERVAL '2 weeks', NOW()
),

-- 5. VIP customer (unlimited uses)
(
    '80000000-0000-0000-0000-000000000005',
    'VIP15',
    'Khách hàng VIP',
    'Giảm 15% không giới hạn số lần, tối đa 150k',
    'percentage', 15.00, 150000,
    0, NULL, false,
    NULL, NULL, 567,  -- max_uses = NULL (unlimited)
    NOW() - INTERVAL '3 months', NOW() + INTERVAL '6 months', true,
    NOW() - INTERVAL '3 months', NOW()
),

-- 6. Big spender
(
    '80000000-0000-0000-0000-000000000006',
    'SAVE100K',
    'Tiết kiệm 100k',
    'Giảm 100k cho đơn từ 1 triệu',
    'fixed', 100000, 100000,
    1000000, NULL, false,
    200, 1, 67,
    NOW() - INTERVAL '1 week', NOW() + INTERVAL '5 weeks', true,
    NOW() - INTERVAL '1 week', NOW()
),

-- 7. Mega sale
(
    '80000000-0000-0000-0000-000000000007',
    'MEGA30',
    'Mega Sale 30%',
    'Giảm 30% tối đa 200k cho đơn từ 800k',
    'percentage', 30.00, 200000,
    800000, NULL, false,
    100, 1, 45,
    NOW() - INTERVAL '3 days', NOW() + INTERVAL '11 days', true,
    NOW() - INTERVAL '3 days', NOW()
),

-- 8. Weekend sale
(
    '80000000-0000-0000-0000-000000000008',
    'WEEKEND25',
    'Cuối tuần giảm 25%',
    'Sale cuối tuần từ 400k',
    'percentage', 25.00, 120000,
    400000, NULL, false,
    500, 2, 234,
    NOW() - INTERVAL '2 days', NOW() + INTERVAL '5 days', true,
    NOW() - INTERVAL '2 days', NOW()
),

-- 9. Kids books (category specific)
(
    '80000000-0000-0000-0000-000000000009',
    'KIDS30',
    'Sách thiếu nhi giảm 30%',
    'Giảm 30% sách thiếu nhi',
    'percentage', 30.00, 80000,
    150000,
    ARRAY['30000000-0000-0000-0000-000000000004'::uuid], -- Thiếu nhi category
    false,
    400, 1, 156,
    NOW() - INTERVAL '10 days', NOW() + INTERVAL '20 days', true,
    NOW() - INTERVAL '10 days', NOW()
),

-- 10. Flash sale
(
    '80000000-0000-0000-0000-000000000010',
    'FLASH40',
    'Flash Sale 40%',
    'Giảm 40% trong 24h, tối đa 150k',
    'percentage', 40.00, 150000,
    500000, NULL, false,
    50, 1, 34,
    NOW() - INTERVAL '12 hours', NOW() + INTERVAL '12 hours', true,
    NOW() - INTERVAL '12 hours', NOW()
),

-- 11. Economics books
(
    '80000000-0000-0000-0000-000000000011',
    'ECON15',
    'Sách kinh tế giảm 15%',
    'Giảm 15% cho sách kinh tế',
    'percentage', 15.00, 70000,
    200000,
    ARRAY['30000000-0000-0000-0000-000000000002'::uuid], -- Kinh tế category
    false,
    300, 1, 89,
    NOW() - INTERVAL '5 days', NOW() + INTERVAL '25 days', true,
    NOW() - INTERVAL '5 days', NOW()
),

-- 12. Loyal customer reward
(
    '80000000-0000-0000-0000-000000000012',
    'LOYAL50',
    'Thưởng khách hàng thân thiết',
    'Giảm 50k cho khách hàng cũ',
    'fixed', 50000, 50000,
    300000, NULL, false,
    1000, 3, 456,
    NOW() - INTERVAL '1 month', NOW() + INTERVAL '2 months', true,
    NOW() - INTERVAL '1 month', NOW()
),

-- ========================================
-- EXPIRED/INACTIVE PROMOTIONS (8 expired)
-- ========================================

-- 13. Tet 2025 (expired)
(
    '80000000-0000-0000-0000-000000000013',
    'TET2025',
    'Tết Nguyên Đán 2025',
    'Giảm 40% dịp Tết, tối đa 300k',
    'percentage', 40.00, 300000,
    500000, NULL, false,
    1000, 1, 856,
    NOW() - INTERVAL '3 months', NOW() - INTERVAL '2 months', false,
    NOW() - INTERVAL '3 months', NOW()
),

-- 14. Summer sale (expired)
(
    '80000000-0000-0000-0000-000000000014',
    'SUMMER23',
    'Hè 2023',
    'Sale hè giảm 35%',
    'percentage', 35.00, 150000,
    300000, NULL, false,
    500, 1, 423,
    NOW() - INTERVAL '4 months', NOW() - INTERVAL '3 months', false,
    NOW() - INTERVAL '4 months', NOW()
),

-- 15. Back to school (expired)
(
    '80000000-0000-0000-0000-000000000015',
    'BACK2SCHOOL',
    'Khai trường 2024',
    'Giảm 20% sách giáo khoa',
    'percentage', 20.00, 80000,
    200000, NULL, false,
    800, 1, 678,
    NOW() - INTERVAL '5 months', NOW() - INTERVAL '4 months', false,
    NOW() - INTERVAL '5 months', NOW()
),

-- 16. Black Friday (expired, reached max uses)
(
    '80000000-0000-0000-0000-000000000016',
    'BLACKFRI',
    'Black Friday Sale',
    'Sale khủng Black Friday 50%',
    'percentage', 50.00, 500000,
    1000000, NULL, false,
    200, 1, 200,  -- current_uses = max_uses (sold out!)
    NOW() - INTERVAL '6 months', NOW() - INTERVAL '5 months', false,
    NOW() - INTERVAL '6 months', NOW()
),

-- 17. Christmas 2024 (expired)
(
    '80000000-0000-0000-0000-000000000017',
    'XMAS2024',
    'Giáng Sinh 2024',
    'Merry Christmas - Giảm 30%',
    'percentage', 30.00, 200000,
    500000, NULL, false,
    600, 1, 534,
    NOW() - INTERVAL '7 months', NOW() - INTERVAL '6 months', false,
    NOW() - INTERVAL '7 months', NOW()
),

-- 18. New Year 2025 (expired)
(
    '80000000-0000-0000-0000-000000000018',
    'NEWYEAR25',
    'Năm Mới 2025',
    'Chúc mừng năm mới - Giảm 200k',
    'fixed', 200000, 200000,
    2000000, NULL, false,
    100, 1, 87,
    NOW() - INTERVAL '8 months', NOW() - INTERVAL '7 months', false,
    NOW() - INTERVAL '8 months', NOW()
),

-- 19. Student discount (expired - academic year ended)
(
    '80000000-0000-0000-0000-000000000019',
    'STUDENT10',
    'Học sinh sinh viên',
    'Giảm 10% cho HSSV',
    'percentage', 10.00, 50000,
    0, NULL, false,
    5000, 2, 3456,
    NOW() - INTERVAL '9 months', NOW() - INTERVAL '1 month', false,
    NOW() - INTERVAL '9 months', NOW()
),

-- 20. Women's Day (expired)
(
    '80000000-0000-0000-0000-000000000020',
    'WOMEN8',
    'Quốc tế Phụ nữ 8/3',
    'Giảm 20% nhân ngày 8/3',
    'percentage', 20.00, 100000,
    300000, NULL, false,
    800, 1, 689,
    NOW() - INTERVAL '8 months', NOW() - INTERVAL '7 months 20 days', false,
    NOW() - INTERVAL '8 months', NOW()
);

-- =====================================================
-- 10. PROMOTION_USAGE (Sample usage tracking)
-- =====================================================
-- This will be populated when we create orders with promotions
-- Skipping for now - will be linked in PART 3 with orders

