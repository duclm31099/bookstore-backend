-- =====================================================
-- BOOKSTORE COMPLETE SEED DATA
-- Part 1: Master Data (Users, Categories, Authors, Publishers, Books)
-- =====================================================

-- Disable triggers temporarily for faster insert
SET session_replication_role = replica;

-- =====================================================
-- 1. USERS (32 users: 2 admins + 30 customers)
-- =====================================================
-- Password for all: "password123"
-- Bcrypt cost 12: $2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S

INSERT INTO users (id, email, password_hash, full_name, phone, role, is_active, points, is_verified, verification_token, verification_sent_at, last_login_at, created_at, updated_at) VALUES
-- Admins
('00000000-0000-0000-0000-000000000001', 'admin@bookstore.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Admin Nguyen Van A', '+84901000001', 'admin', true, 0, true, NULL, NULL, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '6 months', NOW()),
('00000000-0000-0000-0000-000000000002', 'manager@bookstore.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Manager Tran Thi B', '+84901000002', 'admin', true, 0, true, NULL, NULL, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '6 months', NOW()),
('10000000-0000-0000-0000-000000000001', 'user1@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Nguyen Van Anh', '+84901234001', 'user', true, 250, true, NULL, NULL, NOW() - INTERVAL '1 day', NOW() - INTERVAL '3 months', NOW()),
('10000000-0000-0000-0000-000000000002', 'user2@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Tran Thi Binh', '+84901234002', 'user', true, 180, true, NULL, NULL, NOW() - INTERVAL '2 days', NOW() - INTERVAL '3 months', NOW()),
('10000000-0000-0000-0000-000000000003', 'user3@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Le Van Cuong', '+84901234003', 'user', true, 320, true, NULL, NULL, NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 months', NOW()),
('10000000-0000-0000-0000-000000000004', 'user4@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Pham Thi Dung', '+84901234004', 'user', true, 145, true, NULL, NULL, NOW() - INTERVAL '5 days', NOW() - INTERVAL '2 months', NOW()),
('10000000-0000-0000-0000-000000000005', 'user5@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Hoang Van Em', '+84901234005', 'user', true, 420, true, NULL, NULL, NOW() - INTERVAL '7 days', NOW() - INTERVAL '2 months', NOW()),
('10000000-0000-0000-0000-000000000006', 'user6@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Vo Thi Phuong', '+84901234006', 'user', true, 95, true, NULL, NULL, NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 month', NOW()),
('10000000-0000-0000-0000-000000000007', 'user7@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Dang Van Giang', '+84901234007', 'user', true, 560, true, NULL, NULL, NOW() - INTERVAL '12 days', NOW() - INTERVAL '1 month', NOW()),
('10000000-0000-0000-0000-000000000008', 'user8@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Bui Thi Hoa', '+84901234008', 'user', true, 78, true, NULL, NULL, NOW() - INTERVAL '15 days', NOW() - INTERVAL '1 month', NOW()),
('10000000-0000-0000-0000-000000000009', 'user9@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Duong Van Kien', '+84901234009', 'user', true, 340, true, NULL, NULL, NOW() - INTERVAL '3 weeks', NOW() - INTERVAL '3 weeks', NOW()),
('10000000-0000-0000-0000-000000000010', 'user10@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Ngo Thi Lan', '+84901234010', 'user', true, 205, true, NULL, NULL, NOW() - INTERVAL '3 weeks', NOW() - INTERVAL '3 weeks', NOW()),
('10000000-0000-0000-0000-000000000011', 'user11@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Ly Van Minh', '+84901234011', 'user', true, 125, true, NULL, NULL, NOW() - INTERVAL '2 weeks', NOW() - INTERVAL '2 weeks', NOW()),
('10000000-0000-0000-0000-000000000012', 'user12@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Mac Thi Nga', '+84901234012', 'user', true, 385, true, NULL, NULL, NOW() - INTERVAL '2 weeks', NOW() - INTERVAL '2 weeks', NOW()),
('10000000-0000-0000-0000-000000000013', 'user13@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Ung Van Phong', '+84901234013', 'user', true, 445, true, NULL, NULL, NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days', NOW()),
('10000000-0000-0000-0000-000000000014', 'user14@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Dinh Thi Quyen', '+84901234014', 'user', true, 90, true, NULL, NULL, NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days', NOW()),
('10000000-0000-0000-0000-000000000015', 'user15@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'To Van Son', '+84901234015', 'user', true, 270, true, NULL, NULL, NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days', NOW()),
('10000000-0000-0000-0000-000000000016', 'user16@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Cao Thi Thao', '+84901234016', 'user', true, 156, true, NULL, NULL, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days', NOW()),
('10000000-0000-0000-0000-000000000017', 'user17@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Tang Van Tuan', '+84901234017', 'user', true, 512, true, NULL, NULL, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days', NOW()),
('10000000-0000-0000-0000-000000000018', 'user18@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Trinh Thi Van', '+84901234018', 'user', true, 67, true, NULL, NULL, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days', NOW()),
('10000000-0000-0000-0000-000000000019', 'user19@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Mai Van Hung', '+84901234019', 'user', true, 298, true, NULL, NULL, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days', NOW()),
('10000000-0000-0000-0000-000000000020', 'user20@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Phan Thi Xuan', '+84901234020', 'user', true, 178, true, NULL, NULL, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days', NOW()),
('10000000-0000-0000-0000-000000000021', 'user21@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Ha Van Yen', '+84901234021', 'user', true, 423, true, NULL, NULL, NOW() - INTERVAL '1 day', NOW() - INTERVAL '4 months', NOW()),
('10000000-0000-0000-0000-000000000022', 'user22@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Quach Thi Thu', '+84901234022', 'user', true, 234, true, NULL, NULL, NOW() - INTERVAL '1 day', NOW() - INTERVAL '3 months', NOW()),
('10000000-0000-0000-0000-000000000023', 'user23@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Luu Van Thanh', '+84901234023', 'user', true, 156, true, NULL, NULL, NOW() - INTERVAL '12 hours', NOW() - INTERVAL '2 months', NOW()),
('10000000-0000-0000-0000-000000000024', 'user24@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Khong Thi Uyen', '+84901234024', 'user', true, 89, true, NULL, NULL, NOW() - INTERVAL '6 hours', NOW() - INTERVAL '1 month', NOW()),
('10000000-0000-0000-0000-000000000025', 'user25@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Truong Van Vinh', '+84901234025', 'user', true, 367, true, NULL, NULL, NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2 weeks', NOW()),
('10000000-0000-0000-0000-000000000026', 'user26@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Thach Thi Hien', '+84901234026', 'user', true, 45, true, NULL, NULL, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 week', NOW()),
('10000000-0000-0000-0000-000000000027', 'user27@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Kieu Van Khoi', '+84901234027', 'user', true, 512, true, NULL, NULL, NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '3 days', NOW()),
('10000000-0000-0000-0000-000000000028', 'user28@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Vu Thi Lan', '+84901234028', 'user', true, 0, false, 'verify_token_abc123', NOW() - INTERVAL '1 hour', NULL, NOW() - INTERVAL '15 minutes', NOW()),
('10000000-0000-0000-0000-000000000029', 'user29@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Nghiem Van Nam', '+84901234029', 'user', false, 0, true, NULL, NULL, NOW() - INTERVAL '1 month', NOW() - INTERVAL '10 minutes', NOW()),
('10000000-0000-0000-0000-000000000030', 'user30@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIz7e6X8.S', 'Uyen Thi Mai', '+84901234030', 'user', true, 0, true, NULL, NULL, NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '5 minutes', NOW());



INSERT INTO addresses (
  id, user_id, address_type,recipient_name, phone, province, district, 
  ward, street, is_default, created_at, updated_at) VALUES
('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001','home', 'Nguyen Van A', '+84901234001', 'Ho Chi Minh', 'Quan 1', 'Phuong Ben Nghe', '123 Le Loi', true, NOW() - INTERVAL '3 months', NOW()),
('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002','office', 'Tran Thi B', '+84901234002', 'Ha Noi', 'Hoan Kiem', 'Phuong Hang Bai', '456 Ba Trieu', true, NOW() - INTERVAL '3 months', NOW()),
('20000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000003','other', 'Le Van C', '+84901234003', 'Da Nang', 'Hai Chau', 'Phuong Thuan Phuoc', '789 Tran Phu', true, NOW() - INTERVAL '2 months', NOW()),
('20000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000004','home', 'Pham Thi D', '+84901234004', 'Can Tho', 'Ninh Kieu', 'Phuong Xuan Khanh', '321 3 Thang 2', true, NOW() - INTERVAL '2 months', NOW()),
('20000000-0000-0000-0000-000000000005', '10000000-0000-0000-0000-000000000005','home', 'Hoang Van E', '+84901234005', 'Hai Phong', 'Le Chan', 'Phuong Tran Nguyen Han', '654 Dien Bien Phu', true, NOW() - INTERVAL '2 months', NOW()),
('20000000-0000-0000-0000-000000000006', '10000000-0000-0000-0000-000000000006','home', 'Vo Thi F', '+84901234006', 'Ho Chi Minh', 'Quan 3', 'Phuong 1', '987 Vo Van Tan', true, NOW() - INTERVAL '1 month', NOW()),
('20000000-0000-0000-0000-000000000007', '10000000-0000-0000-0000-000000000007','home', 'Dang Van G', '+84901234007', 'Ho Chi Minh', 'Quan 7', 'Phuong Tan Phu', '147 Nguyen Thi Thap', true, NOW() - INTERVAL '1 month', NOW()),
('20000000-0000-0000-0000-000000000008', '10000000-0000-0000-0000-000000000008','home', 'Bui Thi H', '+84901234008', 'Ha Noi', 'Dong Da', 'Phuong Kham Thien', '258 Tran Dai Nghia', true, NOW() - INTERVAL '1 month', NOW()),
('20000000-0000-0000-0000-000000000009', '10000000-0000-0000-0000-000000000009','home', 'Duong Van I', '+84901234009', 'Da Nang', 'Son Tra', 'Phuong Man Thai', '369 Hoang Sa', true, NOW() - INTERVAL '3 weeks', NOW()),
('20000000-0000-0000-0000-000000000010', '10000000-0000-0000-0000-000000000010','home', 'Ngo Thi K', '+84901234010', 'Can Tho', 'Cai Rang', 'Phuong Le Binh', '741 30 Thang 4', true, NOW() - INTERVAL '3 weeks', NOW()),
('20000000-0000-0000-0000-000000000011', '10000000-0000-0000-0000-000000000011','home', 'Ly Van L', '+84901234011', 'Hai Phong', 'Ngo Quyen', 'Phuong Cau Dat', '852 Minh Khai', true, NOW() - INTERVAL '2 weeks', NOW()),
('20000000-0000-0000-0000-000000000012', '10000000-0000-0000-0000-000000000012','home', 'Mac Thi M', '+84901234012', 'Ho Chi Minh', 'Tan Binh', 'Phuong 12', '963 Hoang Hoa Tham', true, NOW() - INTERVAL '2 weeks', NOW()),
('20000000-0000-0000-0000-000000000013', '10000000-0000-0000-0000-000000000013','home', 'Ung Van N', '+84901234013', 'Ho Chi Minh', 'Binh Thanh', 'Phuong 25', '159 Xo Viet Nghe Tinh', true, NOW() - INTERVAL '10 days', NOW()),
('20000000-0000-0000-0000-000000000014', '10000000-0000-0000-0000-000000000014','home', 'Dinh Thi O', '+84901234014', 'Ha Noi', 'Cau Giay', 'Phuong Dich Vong', '357 Nguyen Phong Sac', true, NOW() - INTERVAL '10 days', NOW()),
('20000000-0000-0000-0000-000000000015', '10000000-0000-0000-0000-000000000015','office', 'To Van P', '+84901234015', 'Da Nang', 'Thanh Khe', 'Phuong Tan Chinh', '753 Ong Ich Khiem', true, NOW() - INTERVAL '7 days', NOW()),
('20000000-0000-0000-0000-000000000016', '10000000-0000-0000-0000-000000000016', 'office','Cao Thi Q', '+84901234016', 'Can Tho', 'Binh Thuy', 'Phuong An Thoi', '951 Nguyen Van Cu', true, NOW() - INTERVAL '5 days', NOW()),
('20000000-0000-0000-0000-000000000017', '10000000-0000-0000-0000-000000000017','office', 'Tang Van R', '+84901234017', 'Hai Phong', 'Hong Bang', 'Phuong So Dau', '357 Luong Khanh Thien', true, NOW() - INTERVAL '5 days', NOW()),
('20000000-0000-0000-0000-000000000018', '10000000-0000-0000-0000-000000000018', 'office','Trinh Thi S', '+84901234018', 'Ho Chi Minh', 'Go Vap', 'Phuong 12', '159 Quang Trung', true, NOW() - INTERVAL '3 days', NOW()),
('20000000-0000-0000-0000-000000000019', '10000000-0000-0000-0000-000000000019','office', 'Mai Van T', '+84901234019', 'Ho Chi Minh', 'Thu Duc', 'Phuong Linh Trung', '753 Vo Van Ngan', true, NOW() - INTERVAL '3 days', NOW()),
('20000000-0000-0000-0000-000000000020', '10000000-0000-0000-0000-000000000020', 'office','Phan Thi U', '+84901234020', 'Ha Noi', 'Hai Ba Trung', 'Phuong Bui Thi Xuan', '951 Tran Khat Chan', true, NOW() - INTERVAL '2 days', NOW()),
('20000000-0000-0000-0000-000000000021', '10000000-0000-0000-0000-000000000021','office', 'Ha Van V', '+84901234021', 'Da Nang', 'Lien Chieu', 'Phuong Hoa Hiep Bac', '357 Ton That Dam', true, NOW() - INTERVAL '1 day', NOW()),
('20000000-0000-0000-0000-000000000022', '10000000-0000-0000-0000-000000000022','office', 'Quach Thi W', '+84901234022', 'Can Tho', 'O Mon', 'Phuong Chau Van Liem', '159 Vo Van Kiet', true, NOW() - INTERVAL '1 day', NOW()),
('20000000-0000-0000-0000-000000000023', '10000000-0000-0000-0000-000000000023','office', 'Luu Van X', '+84901234023', 'Ho Chi Minh', 'Phu Nhuan', 'Phuong 13', '753 Phan Dang Luu', true, NOW() - INTERVAL '12 hours', NOW()),
('20000000-0000-0000-0000-000000000024', '10000000-0000-0000-0000-000000000024','office', 'Khong Thi Y', '+84901234024', 'Ha Noi', 'Thanh Xuan', 'Phuong Nhan Chinh', '951 Nguyen Trai', true, NOW() - INTERVAL '6 hours', NOW()),
('20000000-0000-0000-0000-000000000025', '10000000-0000-0000-0000-000000000025','office', 'Truong Van Z', '+84901234025', 'Da Nang', 'Ngu Hanh Son', 'Phuong My An', '357 Nguyen Van Thoai', true, NOW() - INTERVAL '3 hours', NOW()),
('20000000-0000-0000-0000-000000000026', '10000000-0000-0000-0000-000000000026', 'office','Thach Thi AA', '+84901234026', 'Can Tho', 'Thot Not', 'Phuong Thoi Thuan', '159 Nguyen Huu Canh', true, NOW() - INTERVAL '1 hour', NOW()),
('20000000-0000-0000-0000-000000000027', '10000000-0000-0000-0000-000000000027', 'office','Kieu Van BB', '+84901234027', 'Hai Phong', 'Kien An', 'Phuong Quang Trung', '753 Le Duan', true, NOW() - INTERVAL '30 minutes', NOW()),
('20000000-0000-0000-0000-000000000028', '10000000-0000-0000-0000-000000000028', 'office','Vu Thi CC', '+84901234028', 'Ho Chi Minh', 'District 2', 'Phuong Thao Dien', '951 Nguyen Van Huong', true, NOW() - INTERVAL '15 minutes', NOW()),
('20000000-0000-0000-0000-000000000029', '10000000-0000-0000-0000-000000000029','other', 'Nghiem Van DD', '+84901234029', 'Ho Chi Minh', 'District 10', 'Phuong 14', '357 Ba Thang Hai', true, NOW() - INTERVAL '10 minutes', NOW()),
('20000000-0000-0000-0000-000000000030', '10000000-0000-0000-0000-000000000030', 'other','Uyen Thi EE', '+84901234030', 'Ha Noi', 'Ba Dinh', 'Phuong Giang Vo', '159 Lieu Giai', true, NOW() - INTERVAL '5 minutes', NOW());



INSERT INTO categories (id, name, slug, description, icon_url, parent_id, sort_order, is_active, created_at, updated_at) VALUES
('30000000-0000-0000-0000-000000000001', 'Văn học', 'van-hoc', 'Sách văn học trong và ngoài nước', 'https://cdn.bookstore.com/icons/literature.svg', NULL, 1, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000002', 'Kinh tế', 'kinh-te', 'Sách kinh tế, kinh doanh, khởi nghiệp', 'https://cdn.bookstore.com/icons/business.svg', NULL, 2, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000003', 'Kỹ năng sống', 'ky-nang-song', 'Sách kỹ năng mềm, phát triển bản thân', 'https://cdn.bookstore.com/icons/skills.svg', NULL, 3, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000004', 'Thiếu nhi', 'thieu-nhi', 'Sách dành cho trẻ em', 'https://cdn.bookstore.com/icons/kids.svg', NULL, 4, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000005', 'Công nghệ', 'cong-nghe', 'Sách lập trình, IT, AI', 'https://cdn.bookstore.com/icons/tech.svg', NULL, 5, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000006', 'Tiểu thuyết', 'tieu-thuyet', 'Tiểu thuyết Việt Nam và nước ngoài', NULL, '30000000-0000-0000-0000-000000000001', 1, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000007', 'Truyện ngắn', 'truyen-ngan', 'Tuyển tập truyện ngắn', NULL, '30000000-0000-0000-0000-000000000001', 2, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000008', 'Thơ', 'tho', 'Tập thơ các tác giả', NULL, '30000000-0000-0000-0000-000000000001', 3, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000009', 'Light Novel', 'light-novel', 'Tiểu thuyết nhẹ Nhật Bản', NULL, '30000000-0000-0000-0000-000000000001', 4, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000010', 'Khởi nghiệp', 'khoi-nghiep', 'Sách về khởi nghiệp kinh doanh', NULL, '30000000-0000-0000-0000-000000000002', 1, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000011', 'Marketing', 'marketing', 'Sách marketing, bán hàng', NULL, '30000000-0000-0000-0000-000000000002', 2, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000012', 'Quản trị', 'quan-tri', 'Sách quản trị doanh nghiệp', NULL, '30000000-0000-0000-0000-000000000002', 3, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000013', 'Tài chính', 'tai-chinh', 'Sách đầu tư, tài chính', NULL, '30000000-0000-0000-0000-000000000002', 4, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000014', 'Tư duy', 'tu-duy', 'Sách về tư duy logic, sáng tạo', NULL, '30000000-0000-0000-0000-000000000003', 1, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000015', 'Giao tiếp', 'giao-tiep', 'Kỹ năng giao tiếp xã hội', NULL, '30000000-0000-0000-0000-000000000003', 2, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000016', 'Sức khỏe', 'suc-khoe', 'Chăm sóc sức khỏe tinh thần', NULL, '30000000-0000-0000-0000-000000000003', 3, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000017', 'Truyện tranh', 'truyen-tranh', 'Truyện tranh cho bé', NULL, '30000000-0000-0000-0000-000000000004', 1, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000018', 'Sách tô màu', 'sach-to-mau', 'Sách tô màu phát triển trí tuệ', NULL, '30000000-0000-0000-0000-000000000004', 2, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000019', 'Truyện cổ tích', 'truyen-co-tich', 'Truyện cổ tích thế giới', NULL, '30000000-0000-0000-0000-000000000004', 3, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000020', 'Lập trình', 'lap-trinh', 'Sách học lập trình', NULL, '30000000-0000-0000-0000-000000000005', 1, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000021', 'AI & ML', 'ai-machine-learning', 'Sách về AI, Machine Learning', NULL, '30000000-0000-0000-0000-000000000005', 2, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000022', 'Web Development', 'web-development', 'Phát triển web', NULL, '30000000-0000-0000-0000-000000000005', 3, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000023', 'Mobile Dev', 'mobile-development', 'Phát triển ứng dụng di động', NULL, '30000000-0000-0000-0000-000000000005', 4, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000024', 'DevOps', 'devops', 'Sách về DevOps, CI/CD', NULL, '30000000-0000-0000-0000-000000000005', 5, true, NOW() - INTERVAL '1 year', NOW()),
('30000000-0000-0000-0000-000000000025', 'Data Science', 'data-science', 'Khoa học dữ liệu', NULL, '30000000-0000-0000-0000-000000000005', 6, true, NOW() - INTERVAL '1 year', NOW());


INSERT INTO publishers (
  id, name, slug, description, email, phone, address,
   is_active, created_at, updated_at) VALUES
('40000000-0000-0000-0000-000000000001', 'Nhà Xuất Bản Trẻ', 'nxb-tre', 'Nhà xuất bản văn học và thiếu nhi hàng đầu Việt Nam', 'contact@nxbtre.com.vn', '+842838434344', '161B Lý Chính Thắng, Phường 7, Quận 3, TP.HCM', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000002', 'Nhà Xuất Bản Kim Đồng', 'nxb-kim-dong', 'Chuyên xuất bản sách thiếu nhi', 'info@nxbkimdong.com.vn', '+842439434730', '55 Quang Trung, Hai Bà Trưng, Hà Nội', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000003', 'Nhà Xuất Bản Hội Nhà Văn', 'nxb-hoi-nha-van', 'Xuất bản các tác phẩm văn học nghệ thuật', 'nxbhoinhavan@gmail.com', '+842437163104', '65 Nguyễn Du, Hai Bà Trưng, Hà Nội', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000004', 'Nhà Xuất Bản Lao Động', 'nxb-lao-dong', 'Xuất bản sách kinh tế, kỹ năng sống', 'nxblaodong@hn.vnn.vn', '+842438517788', '175 Giảng Võ, Đống Đa, Hà Nội', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000005', 'Nhà Xuất Bản Thế Giới', 'nxb-the-gioi', 'Chuyên dịch và xuất bản sách nước ngoài', 'nxbthegioi@hn.vnn.vn', '+842438255623', '46 Trần Hưng Đạo, Hoàn Kiếm, Hà Nội', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000006', 'First News', 'first-news', 'Công ty phát hành sách hàng đầu', 'info@firstnews.com.vn', '+842838434323', '368/2 Cống Quỳnh, P. Phạm Ngũ Lão, Q.1, TP.HCM', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000007', 'Nhà Xuất Bản Dân Trí', 'nxb-dan-tri', 'Xuất bản sách giáo dục và kỹ năng sống', 'nxbdantri@gmail.com', '+842437264238', '9 Phạm Ngọc Thạch, Đống Đa, Hà Nội', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000008', 'Reilly Media', 'reilly-media', 'Leading tech publisher worldwide', 'info@oreilly.com', '+1-800-998-9938', '1005 Gravenstein Highway North, Sebastopol, CA', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000009', 'Packt Publishing', 'packt-publishing', 'Tech books and video courses', 'support@packtpub.com', '+44-121-265-6484', 'Livery Place, 35 Livery Street, Birmingham', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000010', 'Manning Publications', 'manning-publications', 'Computer science books', 'support@manning.com', '+1-800-626-6464', '20 Baldwin Road, PO Box 761, Shelter Island, NY', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000011', 'Penguin Random House', 'penguin-random-house', 'World largest trade publisher', 'info@penguinrandomhouse.com', '+1-212-782-9000', '1745 Broadway, New York, NY', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000012', 'HarperCollins', 'harpercollins', 'Leading publisher of fiction and non-fiction', 'info@harpercollins.com', '+1-212-207-7000', '195 Broadway, New York, NY', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000013', 'Bloomsbury Publishing', 'bloomsbury', 'Academic and literary publisher', 'info@bloomsbury.com', '+44-20-7631-5600', '50 Bedford Square, London', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000014', 'Wiley', 'wiley', 'Academic, scientific and technical publisher', 'info@wiley.com', '+1-201-748-6000', '111 River Street, Hoboken, NJ', true, NOW() - INTERVAL '2 years', NOW()),
('40000000-0000-0000-0000-000000000015', 'Springer', 'springer', 'Science, technology and medicine publisher', 'info@springer.com', '+49-6221-487-0', 'Tiergartenstrasse 17, Heidelberg, Germany', true, NOW() - INTERVAL '2 years', NOW());



INSERT INTO authors (id, name, slug, bio, photo_url, created_at, updated_at) VALUES
('50000000-0000-0000-0000-000000000001', 'Nguyễn Nhật Ánh', 'nguyen-nhat-anh', 'Nhà văn nổi tiếng với nhiều tác phẩm văn học thiếu nhi', 'https://cdn.bookstore.com/authors/nguyen-nhat-anh.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000002', 'Ngô Tất Tố', 'ngo-tat-to', 'Nhà văn hiện thực Việt Nam', 'https://cdn.bookstore.com/authors/ngo-tat-to.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000003', 'Tô Hoài', 'to-hoai', 'Nhà văn thiếu nhi xuất sắc', 'https://cdn.bookstore.com/authors/to-hoai.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000004', 'Nam Cao', 'nam-cao', 'Nhà văn hiện thực phê phán', 'https://cdn.bookstore.com/authors/nam-cao.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000005', 'Nguyễn Du', 'nguyen-du', 'Đại thi hào Việt Nam', 'https://cdn.bookstore.com/authors/nguyen-du.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000006', 'Haruki Murakami', 'haruki-murakami', 'Contemporary Japanese writer', 'https://cdn.bookstore.com/authors/haruki-murakami.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000007', 'J.K. Rowling', 'jk-rowling', 'Author of Harry Potter series', 'https://cdn.bookstore.com/authors/jk-rowling.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000008', 'Paulo Coelho', 'paulo-coelho', 'Brazilian lyricist and novelist', 'https://cdn.bookstore.com/authors/paulo-coelho.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000009', 'Dale Carnegie', 'dale-carnegie', 'American writer and lecturer', 'https://cdn.bookstore.com/authors/dale-carnegie.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000010', 'Robert Kiyosaki', 'robert-kiyosaki', 'American businessman and author', 'https://cdn.bookstore.com/authors/robert-kiyosaki.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000011', 'Malcolm Gladwell', 'malcolm-gladwell', 'Canadian journalist and author', 'https://cdn.bookstore.com/authors/malcolm-gladwell.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000012', 'Simon Sinek', 'simon-sinek', 'British-American author and speaker', 'https://cdn.bookstore.com/authors/simon-sinek.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000013', 'Robert C. Martin', 'robert-c-martin', 'Software engineer and author (Uncle Bob)', 'https://cdn.bookstore.com/authors/robert-c-martin.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000014', 'Andrew S. Tanenbaum', 'andrew-tanenbaum', 'Computer science professor and author', 'https://cdn.bookstore.com/authors/andrew-tanenbaum.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000015', 'Martin Fowler', 'martin-fowler', 'British software developer and author', 'https://cdn.bookstore.com/authors/martin-fowler.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000016', 'Eric Ries', 'eric-ries', 'Entrepreneur and author of The Lean Startup', 'https://cdn.bookstore.com/authors/eric-ries.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000017', 'Seth Godin', 'seth-godin', 'American author and former dot com executive', 'https://cdn.bookstore.com/authors/seth-godin.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000018', 'Stephen R. Covey', 'stephen-covey', 'American educator and author', 'https://cdn.bookstore.com/authors/stephen-covey.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000019', 'Daniel Kahneman', 'daniel-kahneman', 'Psychologist and economist', 'https://cdn.bookstore.com/authors/daniel-kahneman.jpg', NOW() - INTERVAL '2 years', NOW()),
('50000000-0000-0000-0000-000000000020', 'Yuval Noah Harari', 'yuval-noah-harari', 'Israeli historian and author', 'https://cdn.bookstore.com/authors/yuval-noah-harari.jpg', NOW() - INTERVAL '2 years', NOW());


