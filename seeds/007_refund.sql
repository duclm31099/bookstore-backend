-- =====================================================
-- SEED DATA: REFUND REQUESTS
-- =====================================================

-- Disable triggers temporarily
SET session_replication_role = replica;

-- =====================================================
-- REFUND_REQUESTS (10 refund requests - various statuses)
-- =====================================================

INSERT INTO refund_requests (
    id, payment_transaction_id, order_id,
    requested_by, requested_amount, reason, proof_images,
    status,
    approved_by, approved_at, admin_notes,
    rejected_by, rejected_at, rejection_reason,
    gateway_refund_id, gateway_refund_response,
    requested_at, processing_at, completed_at, failed_at, updated_at
) VALUES

-- ========================================
-- COMPLETED REFUNDS (2 refunds)
-- ========================================

-- Refund 1: Order 39 - Completed refund (VNPay)
(
    'c0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000039',
    '90000000-0000-0000-0000-000000000039',
    '10000000-0000-0000-0000-000000000009',
    595000, 
    'Sản phẩm nhận được không đúng như mô tả trên website. Sách bị rách bìa và thiếu trang.',
    '["https://cdn.bookstore.com/refunds/proof_001_1.jpg", "https://cdn.bookstore.com/refunds/proof_001_2.jpg", "https://cdn.bookstore.com/refunds/proof_001_3.jpg"]'::jsonb,
    'completed',
    '00000000-0000-0000-0000-000000000001',
    NOW() - INTERVAL '25 days',
    'Refund approved - product damaged. Processing refund via VNPay.',
    NULL, NULL, NULL,
    'VNPREFUND2024101001',
    '{"vnp_ResponseCode":"00","vnp_TransactionType":"02","vnp_Message":"Refund successful","vnp_Amount":"59500000","vnp_TransactionNo":"VNPREFUND2024101001"}'::jsonb,
    NOW() - INTERVAL '1 month',
    NOW() - INTERVAL '24 days',
    NOW() - INTERVAL '20 days',
    NULL,
    NOW() - INTERVAL '20 days'
),

-- Refund 2: Order 40 - Completed refund (Momo)
(
    'c0000000-0000-0000-0000-000000000002',
    'b0000000-0000-0000-0000-000000000040',
    '90000000-0000-0000-0000-000000000040',
    '10000000-0000-0000-0000-000000000010',
    415000,
    'Sản phẩm bị lỗi in - nhiều trang bị nhòe và không đọc được. Yêu cầu hoàn tiền.',
    '["https://cdn.bookstore.com/refunds/proof_002_1.jpg", "https://cdn.bookstore.com/refunds/proof_002_2.jpg"]'::jsonb,
    'completed',
    '00000000-0000-0000-0000-000000000001',
    NOW() - INTERVAL '18 days',
    'Approved - printing defect confirmed. Refund via Momo.',
    NULL, NULL, NULL,
    'MOMOREFUND2024101502',
    '{"resultCode":0,"message":"Refund successful","transId":"MOMOREFUND2024101502","amount":415000}'::jsonb,
    NOW() - INTERVAL '3 weeks',
    NOW() - INTERVAL '17 days',
    NOW() - INTERVAL '2 weeks',
    NULL,
    NOW() - INTERVAL '2 weeks'
),

-- ========================================
-- PROCESSING REFUNDS (2 refunds)
-- ========================================

-- Refund 3: Processing
(
    'c0000000-0000-0000-0000-000000000003',
    'b0000000-0000-0000-0000-000000000011',
    '90000000-0000-0000-0000-000000000011',
    '10000000-0000-0000-0000-000000000011',
    542000,
    'Đặt nhầm sản phẩm. Muốn đổi sang sách khác nhưng hệ thống không hỗ trợ. Yêu cầu hoàn tiền để đặt lại.',
    '["https://cdn.bookstore.com/refunds/proof_003_1.jpg"]'::jsonb,
    'processing',
    '00000000-0000-0000-0000-000000000002',
    NOW() - INTERVAL '2 days',
    'Approved - customer ordered wrong product. Processing refund.',
    NULL, NULL, NULL,
    NULL, NULL,
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '1 day',
    NULL, NULL,
    NOW() - INTERVAL '1 day'
),

-- Refund 4: Processing
(
    'c0000000-0000-0000-0000-000000000004',
    'b0000000-0000-0000-0000-000000000008',
    '90000000-0000-0000-0000-000000000008',
    '10000000-0000-0000-0000-000000000008',
    450000,
    'Sản phẩm bị hư hại trong quá trình vận chuyển. Bao bì rách và sách bị ướt.',
    '["https://cdn.bookstore.com/refunds/proof_004_1.jpg", "https://cdn.bookstore.com/refunds/proof_004_2.jpg"]'::jsonb,
    'processing',
    '00000000-0000-0000-0000-000000000001',
    NOW() - INTERVAL '1 day',
    'Approved - shipping damage confirmed. Processing bank transfer refund.',
    NULL, NULL, NULL,
    NULL, NULL,
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '12 hours',
    NULL, NULL,
    NOW() - INTERVAL '12 hours'
),

-- ========================================
-- APPROVED REFUNDS (2 refunds)
-- ========================================

-- Refund 5: Approved
(
    'c0000000-0000-0000-0000-000000000005',
    'b0000000-0000-0000-0000-000000000012',
    '90000000-0000-0000-0000-000000000012',
    '10000000-0000-0000-0000-000000000012',
    415000,
    'Sách giao muộn quá 15 ngày so với thời gian cam kết. Không còn nhu cầu sử dụng.',
    '["https://cdn.bookstore.com/refunds/proof_005_1.jpg"]'::jsonb,
    'approved',
    '00000000-0000-0000-0000-000000000001',
    NOW() - INTERVAL '6 hours',
    'Approved - late delivery confirmed. Will process refund within 24h.',
    NULL, NULL, NULL,
    NULL, NULL,
    NOW() - INTERVAL '2 days',
    NULL, NULL, NULL,
    NOW() - INTERVAL '6 hours'
),

-- Refund 6: Approved
(
    'c0000000-0000-0000-0000-000000000006',
    'b0000000-0000-0000-0000-000000000013',
    '90000000-0000-0000-0000-000000000013',
    '10000000-0000-0000-0000-000000000013',
    1188000,
    'Mua nhầm số lượng nhiều quá. Chỉ cần 2 cuốn nhưng đặt 4 cuốn. Yêu cầu hoàn tiền cho 2 cuốn dư.',
    '["https://cdn.bookstore.com/refunds/proof_006_1.jpg"]'::jsonb,
    'approved',
    '00000000-0000-0000-0000-000000000002',
    NOW() - INTERVAL '3 hours',
    'Approved - partial refund for excess quantity. Customer will return books.',
    NULL, NULL, NULL,
    NULL, NULL,
    NOW() - INTERVAL '1 day',
    NULL, NULL, NULL,
    NOW() - INTERVAL '3 hours'
),

-- ========================================
-- PENDING REFUNDS (2 refunds)
-- ========================================

-- Refund 7: Pending
(
    'c0000000-0000-0000-0000-000000000007',
    'b0000000-0000-0000-0000-000000000014',
    '90000000-0000-0000-0000-000000000014',
    '10000000-0000-0000-0000-000000000014',
    315000,
    'Sản phẩm không như mong đợi. Chất lượng giấy in kém và màu sắc không giống hình trên web.',
    '["https://cdn.bookstore.com/refunds/proof_007_1.jpg", "https://cdn.bookstore.com/refunds/proof_007_2.jpg"]'::jsonb,
    'pending',
    NULL, NULL, NULL,
    NULL, NULL, NULL,
    NULL, NULL,
    NOW() - INTERVAL '8 hours',
    NULL, NULL, NULL,
    NOW() - INTERVAL '8 hours'
),

-- Refund 8: Pending
(
    'c0000000-0000-0000-0000-000000000008',
    'b0000000-0000-0000-0000-000000000015',
    '90000000-0000-0000-0000-000000000015',
    '10000000-0000-0000-0000-000000000015',
    555000,
    'Nhận được sách bị lỗi - bìa bị cong và có vết xước. Yêu cầu đổi hoặc hoàn tiền.',
    '["https://cdn.bookstore.com/refunds/proof_008_1.jpg", "https://cdn.bookstore.com/refunds/proof_008_2.jpg", "https://cdn.bookstore.com/refunds/proof_008_3.jpg"]'::jsonb,
    'pending',
    NULL, NULL, NULL,
    NULL, NULL, NULL,
    NULL, NULL,
    NOW() - INTERVAL '2 hours',
    NULL, NULL, NULL,
    NOW() - INTERVAL '2 hours'
),

-- ========================================
-- REJECTED REFUNDS (2 refunds)
-- ========================================

-- Refund 9: Rejected
(
    'c0000000-0000-0000-0000-000000000009',
    'b0000000-0000-0000-0000-000000000006',
    '90000000-0000-0000-0000-000000000006',
    '10000000-0000-0000-0000-000000000006',
    478000,
    'Sản phẩm không đúng chất lượng.',
    NULL,
    'rejected',
    NULL, NULL, NULL,
    '00000000-0000-0000-0000-000000000001',
    NOW() - INTERVAL '10 days',
    'Rejected - insufficient evidence. Customer did not provide photos or specific details about the issue. Product was delivered in good condition according to delivery logs.',
    NULL, NULL,
    NOW() - INTERVAL '2 weeks',
    NULL, NULL, NULL,
    NOW() - INTERVAL '10 days'
),

-- Refund 10: Rejected
(
    'c0000000-0000-0000-0000-000000000010',
    'b0000000-0000-0000-0000-000000000007',
    '90000000-0000-0000-0000-000000000007',
    '10000000-0000-0000-0000-000000000007',
    905000,
    'Đã sử dụng sản phẩm được 1 tháng nhưng không hài lòng. Muốn trả lại.',
    '["https://cdn.bookstore.com/refunds/proof_010_1.jpg"]'::jsonb,
    'rejected',
    NULL, NULL, NULL,
    '00000000-0000-0000-0000-000000000002',
    NOW() - INTERVAL '1 week',
    'Rejected - refund request submitted 45 days after delivery, exceeding our 30-day return policy. Product shows signs of use.',
    NULL, NULL,
    NOW() - INTERVAL '10 days',
    NULL, NULL, NULL,
    NOW() - INTERVAL '1 week'
);

-- Re-enable triggers
SET session_replication_role = DEFAULT;

-- Verification
SELECT 'Refund requests created: ' || COUNT(*)::text FROM refund_requests;
SELECT 
    status,
    COUNT(*) as count,
    SUM(requested_amount) as total_amount
FROM refund_requests
GROUP BY status
ORDER BY 
    CASE status
        WHEN 'pending' THEN 1
        WHEN 'approved' THEN 2
        WHEN 'processing' THEN 3
        WHEN 'completed' THEN 4
        WHEN 'rejected' THEN 5
        WHEN 'failed' THEN 6
    END;
