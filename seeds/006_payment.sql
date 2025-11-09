-- =====================================================
-- 15. PAYMENT_TRANSACTIONS (40+ transactions)
-- =====================================================
-- Match with orders created earlier

INSERT INTO payment_transactions (
    id, order_id, gateway, transaction_id, amount, currency,
    status, error_code, error_message,
    gateway_response, gateway_signature, payment_details,
    refund_amount, refund_reason, refunded_at,
    initiated_at, processing_at, completed_at, failed_at,
    created_at, updated_at
) VALUES

-- ========================================
-- SUCCESSFUL PAYMENTS (Delivered Orders 1-15)
-- ========================================

-- Order 1: VNPay successful
(
    'b0000000-0000-0000-0000-000000000001',
    '90000000-0000-0000-0000-000000000001',
    'vnpay', 'VNP20240910123456', 466500, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_TxnRef":"VNP20240910123456","vnp_BankCode":"NCB","vnp_CardType":"ATM","vnp_PayDate":"20240910100530","vnp_SecureHash":"abc123def456"}'::jsonb,
    'abc123def456signature',
    '{"bank_code":"NCB","bank_name":"NCB Bank","card_type":"ATM"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 months',
    NOW() - INTERVAL '2 months' + INTERVAL '30 seconds',
    NOW() - INTERVAL '2 months' + INTERVAL '1 minute',
    NULL,
    NOW() - INTERVAL '2 months', NOW() - INTERVAL '2 months'
),

-- Order 2: Momo successful
(
    'b0000000-0000-0000-0000-000000000002',
    '90000000-0000-0000-0000-000000000002',
    'momo', 'MOMO2566109922', 630000, 'VND',
    'success', NULL, NULL,
    '{"resultCode":0,"message":"Success","transId":"MOMO2566109922","amount":630000,"orderInfo":"Pay for order","payType":"qr","responseTime":1694334123456}'::jsonb,
    'momo_signature_xyz',
    '{"pay_type":"qr","order_type":"momo_wallet"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 months',
    NOW() - INTERVAL '2 months' + INTERVAL '20 seconds',
    NOW() - INTERVAL '2 months' + INTERVAL '45 seconds',
    NULL,
    NOW() - INTERVAL '2 months', NOW() - INTERVAL '2 months'
),

-- Order 3: VNPay successful
(
    'b0000000-0000-0000-0000-000000000003',
    '90000000-0000-0000-0000-000000000003',
    'vnpay', 'VNP20240915234567', 715000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_TxnRef":"VNP20240915234567","vnp_BankCode":"VCB","vnp_CardType":"ATM","vnp_PayDate":"20240915143022"}'::jsonb,
    'def789signature',
    '{"bank_code":"VCB","card_type":"ATM"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 months',
    NOW() - INTERVAL '2 months' + INTERVAL '25 seconds',
    NOW() - INTERVAL '2 months' + INTERVAL '50 seconds',
    NULL,
    NOW() - INTERVAL '2 months', NOW() - INTERVAL '2 months'
),

-- Order 4: COD (paid on delivery)
(
    'b0000000-0000-0000-0000-000000000004',
    '90000000-0000-0000-0000-000000000004',
    'cod', NULL, 355000, 'VND',
    'success', NULL, NULL,
    NULL, NULL,
    '{"paid_to":"Delivery staff","payment_method":"cash"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 months',
    NULL,
    NOW() - INTERVAL '1 month 12 days', -- Paid on delivery date
    NULL,
    NOW() - INTERVAL '2 months', NOW() - INTERVAL '1 month 12 days'
),

-- Order 5: VNPay successful (with promotion)
(
    'b0000000-0000-0000-0000-000000000005',
    '90000000-0000-0000-0000-000000000005',
    'vnpay', 'VNP20241001345678', 920000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_TxnRef":"VNP20241001345678","vnp_BankCode":"TCB","vnp_CardType":"ATM","vnp_PayDate":"20241001093045"}'::jsonb,
    'tcb_signature_123',
    '{"bank_code":"TCB","card_type":"ATM"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '1 month',
    NOW() - INTERVAL '1 month' + INTERVAL '15 seconds',
    NOW() - INTERVAL '1 month' + INTERVAL '40 seconds',
    NULL,
    NOW() - INTERVAL '1 month', NOW() - INTERVAL '1 month'
),

-- Order 6: Momo successful
(
    'b0000000-0000-0000-0000-000000000006',
    '90000000-0000-0000-0000-000000000006',
    'momo', 'MOMO2566109924', 478000, 'VND',
    'success', NULL, NULL,
    '{"resultCode":0,"message":"Success","transId":"MOMO2566109924","amount":478000,"payType":"qr"}'::jsonb,
    'momo_sig_456',
    '{"pay_type":"qr"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '1 month',
    NOW() - INTERVAL '1 month' + INTERVAL '18 seconds',
    NOW() - INTERVAL '1 month' + INTERVAL '35 seconds',
    NULL,
    NOW() - INTERVAL '1 month', NOW() - INTERVAL '1 month'
),

-- Order 7: VNPay successful
(
    'b0000000-0000-0000-0000-000000000007',
    '90000000-0000-0000-0000-000000000007',
    'vnpay', 'VNP20241005456789', 905000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_TxnRef":"VNP20241005456789","vnp_BankCode":"MB","vnp_CardType":"ATM"}'::jsonb,
    'mb_signature',
    '{"bank_code":"MB","card_type":"ATM"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '1 month',
    NOW() - INTERVAL '1 month' + INTERVAL '20 seconds',
    NOW() - INTERVAL '1 month' + INTERVAL '42 seconds',
    NULL,
    NOW() - INTERVAL '1 month', NOW() - INTERVAL '1 month'
),

-- Order 8: Bank Transfer successful
(
    'b0000000-0000-0000-0000-000000000008',
    '90000000-0000-0000-0000-000000000008',
    'bank_transfer', 'BANK20241010001', 450000, 'VND',
    'success', NULL, NULL,
    '{"status":"completed","bank":"Vietcombank","reference":"BANK20241010001"}'::jsonb,
    NULL,
    '{"bank":"Vietcombank","account_number":"0123456789","transfer_time":"2024-10-10 08:30:15"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '3 weeks',
    NOW() - INTERVAL '3 weeks' + INTERVAL '2 hours', -- Manual verification delay
    NOW() - INTERVAL '3 weeks' + INTERVAL '2 hours 15 minutes',
    NULL,
    NOW() - INTERVAL '3 weeks', NOW() - INTERVAL '3 weeks'
),

-- Continue with Orders 9-15 (abbreviated for brevity)
(
    'b0000000-0000-0000-0000-000000000009',
    '90000000-0000-0000-0000-000000000009',
    'vnpay', 'VNP20241012567890', 1025000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"BIDV"}'::jsonb,
    'bidv_sig',
    '{"bank_code":"BIDV","card_type":"ATM"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '3 weeks',
    NOW() - INTERVAL '3 weeks' + INTERVAL '22 seconds',
    NOW() - INTERVAL '3 weeks' + INTERVAL '47 seconds',
    NULL,
    NOW() - INTERVAL '3 weeks', NOW() - INTERVAL '3 weeks'
),

(
    'b0000000-0000-0000-0000-000000000010',
    '90000000-0000-0000-0000-000000000010',
    'cod', NULL, 205000, 'VND',
    'success', NULL, NULL,
    NULL, NULL,
    '{"paid_to":"Delivery staff","payment_method":"cash"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 weeks',
    NULL,
    NOW() - INTERVAL '7 days',
    NULL,
    NOW() - INTERVAL '2 weeks', NOW() - INTERVAL '7 days'
),

-- Orders 11-15: Quick batch insert
(
    'b0000000-0000-0000-0000-000000000011',
    '90000000-0000-0000-0000-000000000011',
    'vnpay', 'VNP20241022678901', 542000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"VIB"}'::jsonb, 'vib_sig',
    '{"bank_code":"VIB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 weeks', NOW() - INTERVAL '2 weeks' + INTERVAL '20 seconds',
    NOW() - INTERVAL '2 weeks' + INTERVAL '45 seconds', NULL,
    NOW() - INTERVAL '2 weeks', NOW() - INTERVAL '2 weeks'
),

(
    'b0000000-0000-0000-0000-000000000012',
    '90000000-0000-0000-0000-000000000012',
    'momo', 'MOMO2566109926', 415000, 'VND',
    'success', NULL, NULL,
    '{"resultCode":0,"transId":"MOMO2566109926"}'::jsonb, 'momo_sig',
    '{"pay_type":"app"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days' + INTERVAL '15 seconds',
    NOW() - INTERVAL '10 days' + INTERVAL '35 seconds', NULL,
    NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days'
),

(
    'b0000000-0000-0000-0000-000000000013',
    '90000000-0000-0000-0000-000000000013',
    'vnpay', 'VNP20241028789012', 1188000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"AGRI"}'::jsonb, 'agri_sig',
    '{"bank_code":"AGRI"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days' + INTERVAL '18 seconds',
    NOW() - INTERVAL '10 days' + INTERVAL '40 seconds', NULL,
    NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days'
),

(
    'b0000000-0000-0000-0000-000000000014',
    '90000000-0000-0000-0000-000000000014',
    'vnpay', 'VNP20241030890123', 315000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"SHB"}'::jsonb, 'shb_sig',
    '{"bank_code":"SHB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days' + INTERVAL '25 seconds',
    NOW() - INTERVAL '7 days' + INTERVAL '50 seconds', NULL,
    NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'
),

(
    'b0000000-0000-0000-0000-000000000015',
    '90000000-0000-0000-0000-000000000015',
    'cod', NULL, 555000, 'VND',
    'success', NULL, NULL,
    NULL, NULL,
    '{"paid_to":"Delivery staff"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '5 days', NULL,
    NOW() - INTERVAL '1 day', NULL,
    NOW() - INTERVAL '5 days', NOW() - INTERVAL '1 day'
),

-- ========================================
-- SHIPPING ORDERS (16-20) - Paid
-- ========================================

(
    'b0000000-0000-0000-0000-000000000016',
    '90000000-0000-0000-0000-000000000016',
    'vnpay', 'VNP20241104901234', 780000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"NCB"}'::jsonb, 'ncb_sig2',
    '{"bank_code":"NCB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days' + INTERVAL '20 seconds',
    NOW() - INTERVAL '2 days' + INTERVAL '44 seconds', NULL,
    NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'
),

(
    'b0000000-0000-0000-0000-000000000017',
    '90000000-0000-0000-0000-000000000017',
    'momo', 'MOMO2566109928', 366000, 'VND',
    'success', NULL, NULL,
    '{"resultCode":0,"transId":"MOMO2566109928"}'::jsonb, 'momo_sig7',
    '{"pay_type":"qr"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days' + INTERVAL '17 seconds',
    NOW() - INTERVAL '2 days' + INTERVAL '38 seconds', NULL,
    NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'
),

(
    'b0000000-0000-0000-0000-000000000018',
    '90000000-0000-0000-0000-000000000018',
    'vnpay', 'VNP20241105012345', 925000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"VCB"}'::jsonb, 'vcb_sig2',
    '{"bank_code":"VCB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '23 seconds',
    NOW() - INTERVAL '1 day' + INTERVAL '49 seconds', NULL,
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'
),

-- Order 19: COD (not yet paid - shipping)
(
    'b0000000-0000-0000-0000-000000000019',
    '90000000-0000-0000-0000-000000000019',
    'cod', NULL, 195000, 'VND',
    'pending', NULL, NULL,
    NULL, NULL,
    '{"payment_method":"cash_on_delivery"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '1 day', NULL, NULL, NULL,
    NOW() - INTERVAL '1 day', NOW()
),

(
    'b0000000-0000-0000-0000-000000000020',
    '90000000-0000-0000-0000-000000000020',
    'vnpay', 'VNP20241106123456', 988000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"TCB"}'::jsonb, 'tcb_sig2',
    '{"bank_code":"TCB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '18 hours', NOW() - INTERVAL '18 hours' + INTERVAL '19 seconds',
    NOW() - INTERVAL '18 hours' + INTERVAL '41 seconds', NULL,
    NOW() - INTERVAL '18 hours', NOW() - INTERVAL '18 hours'
),

-- ========================================
-- PROCESSING ORDERS (21-25) - All paid
-- ========================================

(
    'b0000000-0000-0000-0000-000000000021',
    '90000000-0000-0000-0000-000000000021',
    'vnpay', 'VNP20241107234567', 375000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"MB"}'::jsonb, 'mb_sig2',
    '{"bank_code":"MB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '12 hours', NOW() - INTERVAL '12 hours' + INTERVAL '21 seconds',
    NOW() - INTERVAL '12 hours' + INTERVAL '46 seconds', NULL,
    NOW() - INTERVAL '12 hours', NOW() - INTERVAL '12 hours'
),

(
    'b0000000-0000-0000-0000-000000000022',
    '90000000-0000-0000-0000-000000000022',
    'momo', 'MOMO2566109930', 658000, 'VND',
    'success', NULL, NULL,
    '{"resultCode":0,"transId":"MOMO2566109930"}'::jsonb, 'momo_sig8',
    '{"pay_type":"app"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '8 hours', NOW() - INTERVAL '8 hours' + INTERVAL '16 seconds',
    NOW() - INTERVAL '8 hours' + INTERVAL '37 seconds', NULL,
    NOW() - INTERVAL '8 hours', NOW() - INTERVAL '8 hours'
),

(
    'b0000000-0000-0000-0000-000000000023',
    '90000000-0000-0000-0000-000000000023',
    'vnpay', 'VNP20241108345678', 655000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"BIDV"}'::jsonb, 'bidv_sig2',
    '{"bank_code":"BIDV"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '6 hours', NOW() - INTERVAL '6 hours' + INTERVAL '24 seconds',
    NOW() - INTERVAL '6 hours' + INTERVAL '48 seconds', NULL,
    NOW() - INTERVAL '6 hours', NOW() - INTERVAL '6 hours'
),

(
    'b0000000-0000-0000-0000-000000000024',
    '90000000-0000-0000-0000-000000000024',
    'bank_transfer', 'BANK20241108002', 515000, 'VND',
    'success', NULL, NULL,
    '{"status":"completed","bank":"ACB","reference":"BANK20241108002"}'::jsonb, NULL,
    '{"bank":"ACB","account_number":"9876543210"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours' + INTERVAL '1 hour',
    NOW() - INTERVAL '4 hours' + INTERVAL '1 hour 10 minutes', NULL,
    NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours'
),

(
    'b0000000-0000-0000-0000-000000000025',
    '90000000-0000-0000-0000-000000000025',
    'vnpay', 'VNP20241109456789', 1025000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"VIB"}'::jsonb, 'vib_sig2',
    '{"bank_code":"VIB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours' + INTERVAL '22 seconds',
    NOW() - INTERVAL '2 hours' + INTERVAL '47 seconds', NULL,
    NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'
),

-- ========================================
-- CONFIRMED ORDERS (26-30) - All paid
-- ========================================

(
    'b0000000-0000-0000-0000-000000000026',
    '90000000-0000-0000-0000-000000000026',
    'vnpay', 'VNP20241109567890', 305000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"AGRI"}'::jsonb, 'agri_sig2',
    '{"bank_code":"AGRI"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour' + INTERVAL '18 seconds',
    NOW() - INTERVAL '1 hour' + INTERVAL '39 seconds', NULL,
    NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour'
),

(
    'b0000000-0000-0000-0000-000000000027',
    '90000000-0000-0000-0000-000000000027',
    'momo', 'MOMO2566109932', 482000, 'VND',
    'success', NULL, NULL,
    '{"resultCode":0,"transId":"MOMO2566109932"}'::jsonb, 'momo_sig9',
    '{"pay_type":"qr"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '45 minutes', NOW() - INTERVAL '45 minutes' + INTERVAL '19 seconds',
    NOW() - INTERVAL '45 minutes' + INTERVAL '42 seconds', NULL,
    NOW() - INTERVAL '45 minutes', NOW() - INTERVAL '45 minutes'
),

(
    'b0000000-0000-0000-0000-000000000028',
    '90000000-0000-0000-0000-000000000028',
    'vnpay', 'VNP20241109678901', 1108000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"SHB"}'::jsonb, 'shb_sig2',
    '{"bank_code":"SHB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes' + INTERVAL '20 seconds',
    NOW() - INTERVAL '30 minutes' + INTERVAL '43 seconds', NULL,
    NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes'
),

-- Order 29: COD pending
(
    'b0000000-0000-0000-0000-000000000029',
    '90000000-0000-0000-0000-000000000029',
    'cod', NULL, 225000, 'VND',
    'pending', NULL, NULL,
    NULL, NULL,
    '{"payment_method":"cash_on_delivery"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '20 minutes', NULL, NULL, NULL,
    NOW() - INTERVAL '20 minutes', NOW()
),

(
    'b0000000-0000-0000-0000-000000000030',
    '90000000-0000-0000-0000-000000000030',
    'vnpay', 'VNP20241109789012', 875000, 'VND',
    'success', NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"NCB"}'::jsonb, 'ncb_sig3',
    '{"bank_code":"NCB"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '10 minutes' + INTERVAL '17 seconds',
    NOW() - INTERVAL '10 minutes' + INTERVAL '38 seconds', NULL,
    NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '10 minutes'
),

-- ========================================
-- PENDING ORDERS (31-35) - Waiting payment
-- ========================================

(
    'b0000000-0000-0000-0000-000000000031',
    '90000000-0000-0000-0000-000000000031',
    'vnpay', NULL, 455000, 'VND',
    'pending', NULL, NULL,
    NULL, NULL, NULL,
    0, NULL, NULL,
    NOW() - INTERVAL '15 minutes', NULL, NULL, NULL,
    NOW() - INTERVAL '15 minutes', NOW()
),

(
    'b0000000-0000-0000-0000-000000000032',
    '90000000-0000-0000-0000-000000000032',
    'momo', NULL, 715000, 'VND',
    'pending', NULL, NULL,
    NULL, NULL, NULL,
    0, NULL, NULL,
    NOW() - INTERVAL '10 minutes', NULL, NULL, NULL,
    NOW() - INTERVAL '10 minutes', NOW()
),

(
    'b0000000-0000-0000-0000-000000000033',
    '90000000-0000-0000-0000-000000000033',
    'vnpay', NULL, 325000, 'VND',
    'pending', NULL, NULL,
    NULL, NULL, NULL,
    0, NULL, NULL,
    NOW() - INTERVAL '8 minutes', NULL, NULL, NULL,
    NOW() - INTERVAL '8 minutes', NOW()
),

(
    'b0000000-0000-0000-0000-000000000034',
    '90000000-0000-0000-0000-000000000034',
    'cod', NULL, 185000, 'VND',
    'pending', NULL, NULL,
    NULL, NULL,
    '{"payment_method":"cash_on_delivery"}'::jsonb,
    0, NULL, NULL,
    NOW() - INTERVAL '5 minutes', NULL, NULL, NULL,
    NOW() - INTERVAL '5 minutes', NOW()
),

(
    'b0000000-0000-0000-0000-000000000035',
    '90000000-0000-0000-0000-000000000035',
    'vnpay', NULL, 900000, 'VND',
    'pending', NULL, NULL,
    NULL, NULL, NULL,
    0, NULL, NULL,
    NOW() - INTERVAL '3 minutes', NULL, NULL, NULL,
    NOW() - INTERVAL '3 minutes', NOW()
),

-- ========================================
-- CANCELLED/FAILED ORDERS (36-38)
-- ========================================

-- Order 36: Payment failed - user cancelled
(
    'b0000000-0000-0000-0000-000000000036',
    '90000000-0000-0000-0000-000000000036',
    'vnpay', 'VNP20241028890123', 515000, 'VND',
    'failed', 'PAY_USER_CANCELLED', 'User cancelled payment',
    '{"vnp_ResponseCode":"24","vnp_Message":"User cancelled"}'::jsonb, 'cancelled_sig',
    NULL,
    0, NULL, NULL,
    NOW() - INTERVAL '1 week', NOW() - INTERVAL '1 week' + INTERVAL '10 seconds',
    NULL,
    NOW() - INTERVAL '1 week' + INTERVAL '15 seconds',
    NOW() - INTERVAL '1 week', NOW() - INTERVAL '6 days'
),

-- Order 37: Payment failed - insufficient balance
(
    'b0000000-0000-0000-0000-000000000037',
    '90000000-0000-0000-0000-000000000037',
    'momo', 'MOMO2566109940', 755000, 'VND',
    'failed', 'INSUFFICIENT_BALANCE', 'Insufficient balance',
    '{"resultCode":1002,"message":"Insufficient balance"}'::jsonb, 'failed_sig',
    NULL,
    0, NULL, NULL,
    NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days' + INTERVAL '8 seconds',
    NULL,
    NOW() - INTERVAL '5 days' + INTERVAL '12 seconds',
    NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days'
),

-- Order 38: No payment attempt (cancelled before payment)
(
    'b0000000-0000-0000-0000-000000000038',
    '90000000-0000-0000-0000-000000000038',
    'vnpay', NULL, 375000, 'VND',
    'cancelled', 'ORDER_CANCELLED', 'Order cancelled - out of stock',
    NULL, NULL, NULL,
    0, NULL, NULL,
    NOW() - INTERVAL '3 days', NULL, NULL, NULL,
    NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days'
),

-- ========================================
-- RETURNED ORDERS (39-40) - Refunded
-- ========================================

-- Order 39: Successful payment → Later refunded
(
    'b0000000-0000-0000-0000-000000000039',
    '90000000-0000-0000-0000-000000000039',
    'vnpay', 'VNP20241010901234', 595000, 'VND',
    'success',  -- ✅ Đổi từ 'refunded' thành 'success'
    NULL, NULL,
    '{"vnp_ResponseCode":"00","vnp_BankCode":"VCB"}'::jsonb, 'refund_sig1',
    '{"bank_code":"VCB"}'::jsonb,
    595000, 'Sản phẩm không đúng mô tả',
    NOW() - INTERVAL '20 days',  -- refunded_at
    NOW() - INTERVAL '1 month',  -- initiated_at
    NOW() - INTERVAL '1 month' + INTERVAL '25 seconds',  -- processing_at
    NOW() - INTERVAL '1 month' + INTERVAL '50 seconds', -- completed_at
    NULL,  -- failed_at
    NOW() - INTERVAL '1 month', 
    NOW() - INTERVAL '20 days'  -- updated_at (when refunded)
),

-- Order 40: Successful payment → Later refunded
(
    'b0000000-0000-0000-0000-000000000040',
    '90000000-0000-0000-0000-000000000040',
    'momo', 'MOMO2566109934', 415000, 'VND',
    'success',  -- ✅ Đổi từ 'refunded' thành 'success'
    NULL, NULL,
    '{"resultCode":0,"transId":"MOMO2566109934"}'::jsonb, 'refund_sig2',
    '{"pay_type":"qr"}'::jsonb,
    415000, 'Sản phẩm bị lỗi',
    NOW() - INTERVAL '2 weeks',  -- refunded_at
    NOW() - INTERVAL '3 weeks',  -- initiated_at
    NOW() - INTERVAL '3 weeks' + INTERVAL '18 seconds',  -- processing_at
    NOW() - INTERVAL '3 weeks' + INTERVAL '40 seconds',  -- completed_at
    NULL,  -- failed_at
    NOW() - INTERVAL '3 weeks', 
    NOW() - INTERVAL '2 weeks'  -- updated_at
);


-- =====================================================
-- 16. PAYMENT_WEBHOOK_LOGS (Sample webhook logs)
-- =====================================================

-- =====================================================
-- PAYMENT_WEBHOOK_LOGS (Sample webhook logs)
-- =====================================================

INSERT INTO payment_webhook_logs (
    id, payment_transaction_id, order_id, 
    gateway, webhook_event,
    headers, body, signature,
    is_valid, is_processed, processing_error,
    received_at
) VALUES
(
    gen_random_uuid(),
    'b0000000-0000-0000-0000-000000000001',
    '90000000-0000-0000-0000-000000000001',
    'vnpay', 'payment.success',
    '{"content-type":"application/json","user-agent":"VNPay-Webhook/1.0"}'::jsonb,
    '{"vnp_ResponseCode":"00","vnp_TxnRef":"VNP20240910123456","vnp_Amount":"46650000","vnp_BankCode":"NCB","vnp_TransactionNo":"14123456","vnp_SecureHash":"abc123def456"}'::jsonb,
    'abc123def456signature',
    true, true, NULL,
    NOW() - INTERVAL '2 months' + INTERVAL '1 minute'
),

(
    gen_random_uuid(),
    'b0000000-0000-0000-0000-000000000002',
    '90000000-0000-0000-0000-000000000002',
    'momo', 'payment.success',
    '{"content-type":"application/json","user-agent":"MoMo-Webhook/1.0"}'::jsonb,
    '{"resultCode":0,"message":"Success","transId":"MOMO2566109922","amount":630000,"orderInfo":"Pay for order","signature":"momo_signature_xyz"}'::jsonb,
    'momo_signature_xyz',
    true, true, NULL,
    NOW() - INTERVAL '2 months' + INTERVAL '45 seconds'
),

(
    gen_random_uuid(),
    NULL,
    '90000000-0000-0000-0000-000000000036',
    'vnpay', 'payment.failed',
    '{"content-type":"application/json"}'::jsonb,
    '{"vnp_ResponseCode":"24","vnp_TxnRef":"VNP20241028890123","vnp_SecureHash":"invalid_signature"}'::jsonb,
    'invalid_signature',
    false, false, 'Signature validation failed',
    NOW() - INTERVAL '1 week' + INTERVAL '15 seconds'
),
(
    gen_random_uuid(),
    'b0000000-0000-0000-0000-000000000003',
    '90000000-0000-0000-0000-000000000003',
    'vnpay', 'payment.success',
    '{"content-type":"application/json"}'::jsonb,
    '{"vnp_ResponseCode":"00","vnp_TxnRef":"VNP20240915234567"}'::jsonb,
    'def789signature',
    true, true, NULL,
    NOW() - INTERVAL '2 months' + INTERVAL '1 minute'
),

(
    gen_random_uuid(),
    'b0000000-0000-0000-0000-000000000039',
    '90000000-0000-0000-0000-000000000039',
    'vnpay', 'refund.success',
    '{"content-type":"application/json"}'::jsonb,
    '{"vnp_ResponseCode":"00","vnp_TxnRef":"VNP20241010901234","vnp_TransactionType":"02","vnp_Amount":"59500000"}'::jsonb,
    'refund_sig1',
    true, true, NULL,
    NOW() - INTERVAL '20 days'
);
