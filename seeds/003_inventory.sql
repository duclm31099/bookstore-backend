INSERT INTO warehouses (
  id, name, code, address, province, latitude, longitude, is_active, version, created_at, updated_at) VALUES
('70000000-0000-0000-0000-000000000001', 'Kho Trung Tâm TP.HCM', 'WH-HCM-01', '123 Đường D2, Phường 25, Quận Bình Thạnh', 'Ho Chi Minh', 10.807440, 106.691320, true, 1, NOW() - INTERVAL '1 year', NOW()),
('70000000-0000-0000-0000-000000000002', 'Kho Miền Bắc Hà Nội', 'WH-HN-01', '456 Đường Giải Phóng, Phường Hoàng Liệt, Quận Hoàng Mai', 'Ha Noi', 20.980850, 105.836210, true, 1, NOW() - INTERVAL '1 year', NOW()),
('70000000-0000-0000-0000-000000000003', 'Kho Miền Trung Đà Nẵng', 'WH-DN-01', '789 Đường Điện Biên Phủ, Phường Chính Gián, Quận Thanh Khê', 'Da Nang', 16.062110, 108.211380, true, 1, NOW() - INTERVAL '1 year', NOW());

-- =====================================================
-- 8. WAREHOUSE_INVENTORY (Stock for all books in all warehouses)
-- =====================================================

-- Warehouse HCM (largest stock)
INSERT INTO warehouse_inventory (
  warehouse_id, book_id, quantity, reserved,
   alert_threshold, last_restocked_at, updated_at, updated_by) VALUES
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', 150, 5, 10, NOW() - INTERVAL '1 month', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000002', 200, 8, 10, NOW() - INTERVAL '1 month', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', 80, 2, 10, NOW() - INTERVAL '2 months', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000004', 120, 3, 10, NOW() - INTERVAL '1 month', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000005', 60, 1, 10, NOW() - INTERVAL '3 months', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000006', 100, 4, 10, NOW() - INTERVAL '1 month', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000007', 90, 3, 10, NOW() - INTERVAL '2 months', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000008', 180, 6, 10, NOW() - INTERVAL '1 month', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000009', 250, 10, 10, NOW() - INTERVAL '2 weeks', NOW(), '00000000-0000-0000-0000-000000000001'),
('70000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000010', 70, 2, 10, NOW() - INTERVAL '3 months', NOW(), '00000000-0000-0000-0000-000000000001');

-- Add stock for more books (11-60) in HCM warehouse
INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity, reserved, alert_threshold, last_restocked_at, updated_at, updated_by)
SELECT 
    '70000000-0000-0000-0000-000000000001',
    id,
    FLOOR(50 + RANDOM() * 200)::INT, -- Random stock 50-250
    FLOOR(RANDOM() * 10)::INT,       -- Random reserved 0-10
    10,
    NOW() - (FLOOR(RANDOM() * 90) || ' days')::INTERVAL,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
FROM books
WHERE id >= '60000000-0000-0000-0000-000000000011';

-- Warehouse Hanoi (medium stock - 60% of HCM)
INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity, reserved, alert_threshold, last_restocked_at, updated_at, updated_by)
SELECT 
    '70000000-0000-0000-0000-000000000002',
    id,
    FLOOR(30 + RANDOM() * 120)::INT, -- Random stock 30-150
    FLOOR(RANDOM() * 5)::INT,
    10,
    NOW() - (FLOOR(RANDOM() * 90) || ' days')::INTERVAL,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
FROM books;

-- Warehouse Da Nang (smaller stock - 40% of HCM)
INSERT INTO warehouse_inventory (warehouse_id, book_id, quantity, reserved, alert_threshold, last_restocked_at, updated_at, updated_by)
SELECT 
    '70000000-0000-0000-0000-000000000003',
    id,
    FLOOR(20 + RANDOM() * 80)::INT, -- Random stock 20-100
    FLOOR(RANDOM() * 3)::INT,
    10,
    NOW() - (FLOOR(RANDOM() * 90) || ' days')::INTERVAL,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
FROM books;