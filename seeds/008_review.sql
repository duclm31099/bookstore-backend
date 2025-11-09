-- =====================================================
-- 18. REVIEWS (30+ reviews for delivered orders)
-- =====================================================
-- Rules:
-- - Only users who received books can review (status = 'delivered')
-- - One review per user per book
-- - Auto-approve some, keep some pending moderation

INSERT INTO reviews (
    id, user_id, book_id, order_id,
    rating, title, content, images,
    is_verified_purchase, is_approved, is_featured, admin_note,
    created_at, updated_at
) VALUES

-- ========================================
-- 5-STAR REVIEWS (Approved & Featured)
-- ========================================

-- Review 1: Mắt Biếc - User 1
(
    'd0000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    '60000000-0000-0000-0000-000000000001', -- Mắt Biếc
    '90000000-0000-0000-0000-000000000001',
    5,
    'Tuyệt phẩm văn học Việt Nam!',
    'Cuốn sách hay nhất tôi từng đọc của Nguyễn Nhật Ánh. Câu chuyện cảm động, lối viết giản dị nhưng sâu sắc. Đọc xong khóc nức nở. Rất recommend!',
    ARRAY['https://cdn.bookstore.com/reviews/review_001_1.jpg', 'https://cdn.bookstore.com/reviews/review_001_2.jpg'],
    true, true, true, 'Excellent detailed review',
    NOW() - INTERVAL '1 month 18 days', NOW() - INTERVAL '1 month 18 days'
),

-- Review 2: The Alchemist - User 1
(
    'd0000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000001',
    '60000000-0000-0000-0000-000000000008', -- The Alchemist
    '90000000-0000-0000-0000-000000000001',
    5,
    'Life-changing book!',
    'This book changed my perspective on life and dreams. Paulo Coelho is a genius. Every sentence is meaningful. A must-read for everyone!',
    ARRAY['https://cdn.bookstore.com/reviews/review_002_1.jpg'],
    true, true, true, 'Great review, featured',
    NOW() - INTERVAL '1 month 17 days', NOW() - INTERVAL '1 month 17 days'
),

-- Review 3: Rich Dad Poor Dad - User 2
(
    'd0000000-0000-0000-0000-000000000003',
    '10000000-0000-0000-0000-000000000002',
    '60000000-0000-0000-0000-000000000011', -- Rich Dad Poor Dad
    '90000000-0000-0000-0000-000000000002',
    5,
    'Kinh thánh về tài chính',
    'Cuốn sách mở mang tư duy về tiền bạc và đầu tư. Rất nhiều bài học quý giá. Đọc rồi mới hiểu tại sao người giàu ngày càng giàu. Chất lượng in ấn tốt, ship nhanh.',
    ARRAY['https://cdn.bookstore.com/reviews/review_003_1.jpg'],
    true, true, true, NULL,
    NOW() - INTERVAL '1 month 16 days', NOW() - INTERVAL '1 month 16 days'
),

-- Review 4: How to Win Friends - User 2
(
    'd0000000-0000-0000-0000-000000000004',
    '10000000-0000-0000-0000-000000000002',
    '60000000-0000-0000-0000-000000000021', -- How to Win Friends
    '90000000-0000-0000-0000-000000000002',
    5,
    'Sách hay về kỹ năng giao tiếp',
    'Dale Carnegie viết rất hay và dễ hiểu. Áp dụng được ngay vào cuộc sống. Sau khi đọc, mình thấy cải thiện rõ rệt trong giao tiếp với mọi người.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '1 month 15 days', NOW() - INTERVAL '1 month 15 days'
),

-- Review 5: Atomic Habits - User 5
(
    'd0000000-0000-0000-0000-000000000005',
    '10000000-0000-0000-0000-000000000005',
    '60000000-0000-0000-0000-000000000023', -- Atomic Habits
    '90000000-0000-0000-0000-000000000005',
    5,
    'Best book on habit formation!',
    'Clear, actionable advice on building good habits. The 1% improvement concept is brilliant. I have already started implementing the strategies and seeing results.',
    ARRAY['https://cdn.bookstore.com/reviews/review_005_1.jpg', 'https://cdn.bookstore.com/reviews/review_005_2.jpg'],
    true, true, true, 'Featured - excellent insights',
    NOW() - INTERVAL '18 days', NOW() - INTERVAL '18 days'
),

-- ========================================
-- 4-STAR REVIEWS (Approved)
-- ========================================

-- Review 6: Clean Code - User 3
(
    'd0000000-0000-0000-0000-000000000006',
    '10000000-0000-0000-0000-000000000003',
    '60000000-0000-0000-0000-000000000041', -- Clean Code
    '90000000-0000-0000-0000-000000000003',
    4,
    'Must-read for developers',
    'Excellent book on writing clean, maintainable code. Some examples are in Java but principles apply to all languages. A bit lengthy but worth it.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '1 month 13 days', NOW() - INTERVAL '1 month 13 days'
),

-- Review 7: Hands-On ML - User 3
(
    'd0000000-0000-0000-0000-000000000007',
    '10000000-0000-0000-0000-000000000003',
    '60000000-0000-0000-0000-000000000049', -- Hands-On ML
    '90000000-0000-0000-0000-000000000003',
    4,
    'Great practical ML guide',
    'Very hands-on approach with lots of code examples. Perfect for beginners to intermediate level. Could use more theory but overall excellent resource.',
    ARRAY['https://cdn.bookstore.com/reviews/review_007_1.jpg'],
    true, true, false, NULL,
    NOW() - INTERVAL '1 month 12 days', NOW() - INTERVAL '1 month 12 days'
),

-- Review 8: Doraemon volumes - User 4
(
    'd0000000-0000-0000-0000-000000000008',
    '10000000-0000-0000-0000-000000000004',
    '60000000-0000-0000-0000-000000000031', -- Doraemon 1
    '90000000-0000-0000-0000-000000000004',
    4,
    'Con thích lắm!',
    'Mua cho con đọc, bé rất thích. Truyện hay, hình vẽ đẹp. Giá hơi cao một chút nhưng chất lượng tốt nên chấp nhận được.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '1 month 10 days', NOW() - INTERVAL '1 month 10 days'
),

-- Review 9: Dế Mèn - User 8
(
    'd0000000-0000-0000-0000-000000000009',
    '10000000-0000-0000-0000-000000000008',
    '60000000-0000-0000-0000-000000000004', -- Dế Mèn
    '90000000-0000-0000-0000-000000000008',
    4,
    'Truyện thiếu nhi kinh điển',
    'Mua để đọc lại tuổi thơ. Vẫn hay như xưa. Bìa cứng đẹp, giấy in tốt. Giao hàng nhanh.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '11 days', NOW() - INTERVAL '11 days'
),

-- Review 10: Code Complete - User 9
(
    'd0000000-0000-0000-0000-000000000010',
    '10000000-0000-0000-0000-000000000009',
    '60000000-0000-0000-0000-000000000045', -- Code Complete
    '90000000-0000-0000-0000-000000000009',
    4,
    'Comprehensive software construction guide',
    'Very detailed and comprehensive. Covers everything from design to implementation. A bit outdated in some parts but core principles remain solid.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '8 days', NOW() - INTERVAL '8 days'
),

-- ========================================
-- 3-STAR REVIEWS (Approved - Mixed feedback)
-- ========================================

-- Review 11: Purple Cow - User 6
(
    'd0000000-0000-0000-0000-000000000011',
    '10000000-0000-0000-0000-000000000006',
    '60000000-0000-0000-0000-000000000014', -- Purple Cow
    '90000000-0000-0000-0000-000000000006',
    3,
    'Interesting but repetitive',
    'Good marketing concepts but the book repeats the same ideas multiple times. Could have been a long article instead. Still worth reading once.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '16 days', NOW() - INTERVAL '16 days'
),

-- Review 12: JS Good Parts - User 7
(
    'd0000000-0000-0000-0000-000000000012',
    '10000000-0000-0000-0000-000000000007',
    '60000000-0000-0000-0000-000000000052', -- JS Good Parts
    '90000000-0000-0000-0000-000000000007',
    3,
    'Outdated but still useful',
    'Book is quite old now (pre-ES6). Many parts are outdated. However, the core JavaScript insights are still valuable. Read with modern JS knowledge.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '13 days', NOW() - INTERVAL '13 days'
),

-- Review 13: Tắt Đèn - User 10
(
    'd0000000-0000-0000-0000-000000000013',
    '10000000-0000-0000-0000-000000000010',
    '60000000-0000-0000-0000-000000000003', -- Tắt Đèn
    '90000000-0000-0000-0000-000000000010',
    3,
    'Hay nhưng hơi buồn',
    'Tác phẩm kinh điển Việt Nam. Nội dung sâu sắc nhưng đọc hơi nặng nề và buồn. Không phù hợp để giải trí.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'
),

-- ========================================
-- 2-STAR REVIEWS (Approved - Negative feedback)
-- ========================================

-- Review 14: Start with Why - User 11
(
    'd0000000-0000-0000-0000-000000000014',
    '10000000-0000-0000-0000-000000000011',
    '60000000-0000-0000-0000-000000000015', -- Start with Why
    '90000000-0000-0000-0000-000000000011',
    2,
    'Over-hyped and repetitive',
    'Everyone recommended this book but I found it disappointing. The main concept is explained in first chapter, rest is just filler and repetition. Not worth the hype.',
    NULL,
    true, true, false, 'Honest feedback - approved',
    NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'
),

-- Review 15: Norwegian Wood - User 12
(
    'd0000000-0000-0000-0000-000000000015',
    '10000000-0000-0000-0000-000000000012',
    '60000000-0000-0000-0000-000000000006', -- Norwegian Wood
    '90000000-0000-0000-0000-000000000012',
    2,
    'Not my cup of tea',
    'Story is quite depressing and slow-paced. Characters are not very likable. Maybe I do not understand Murakami style. Translation quality is okay.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'
),

-- ========================================
-- PENDING REVIEWS (Not yet approved - need moderation)
-- ========================================

-- Review 16: Pending moderation - good review
(
    'd0000000-0000-0000-0000-000000000016',
    '10000000-0000-0000-0000-000000000013',
    '60000000-0000-0000-0000-000000000046', -- Intro to Algorithms
    '90000000-0000-0000-0000-000000000013',
    5,
    'The best algorithms book',
    'Comprehensive coverage of algorithms and data structures. Clear explanations with mathematical proofs. A must-have for CS students.',
    NULL,
    true, false, false, NULL,
    NOW() - INTERVAL '12 hours', NOW() - INTERVAL '12 hours'
),

-- Review 17: Pending moderation - spam/short
(
    'd0000000-0000-0000-0000-000000000017',
    '10000000-0000-0000-0000-000000000014',
    '60000000-0000-0000-0000-000000000053', -- You Dont Know JS
    '90000000-0000-0000-0000-000000000014',
    5,
    'Good',
    'Nice book',
    NULL,
    true, false, false, NULL,
    NOW() - INTERVAL '8 hours', NOW() - INTERVAL '8 hours'
),

-- Review 18: Pending - negative but valid
(
    'd0000000-0000-0000-0000-000000000018',
    '10000000-0000-0000-0000-000000000015',
    '60000000-0000-0000-0000-000000000017', -- 4-Hour Workweek
    '90000000-0000-0000-0000-000000000015',
    2,
    'Unrealistic advice',
    'Most advice in this book is not applicable to regular people. Only works if you already have money or connections. Very disappointed.',
    NULL,
    true, false, false, NULL,
    NOW() - INTERVAL '6 hours', NOW() - INTERVAL '6 hours'
),

-- ========================================
-- MORE APPROVED REVIEWS (Various ratings)
-- ========================================

-- Review 19: Sapiens - 5 stars
(
    'd0000000-0000-0000-0000-000000000019',
    '10000000-0000-0000-0000-000000000005',
    '60000000-0000-0000-0000-000000000025', -- Sapiens
    '90000000-0000-0000-0000-000000000005',
    5,
    'Mind-blowing history book!',
    'Yuval Noah Harari presents human history in a fascinating way. Makes you think about society, religion, and our future. Highly recommend!',
    ARRAY['https://cdn.bookstore.com/reviews/review_019_1.jpg'],
    true, true, true, 'Excellent detailed review',
    NOW() - INTERVAL '17 days', NOW() - INTERVAL '17 days'
),

-- Review 20: 7 Habits - 5 stars
(
    'd0000000-0000-0000-0000-000000000020',
    '10000000-0000-0000-0000-000000000005',
    '60000000-0000-0000-0000-000000000022', -- 7 Habits
    '90000000-0000-0000-0000-000000000005',
    5,
    'Timeless classic',
    'This book has changed my life. The 7 habits are simple but powerful. Everyone should read this at least once.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '16 days', NOW() - INTERVAL '16 days'
),

-- Review 21: Thinking Fast Slow - 5 stars
(
    'd0000000-0000-0000-0000-000000000021',
    '10000000-0000-0000-0000-000000000005',
    '60000000-0000-0000-0000-000000000019', -- Thinking Fast Slow
    '90000000-0000-0000-0000-000000000005',
    5,
    'Psychology masterpiece',
    'Kahneman explains how our mind works in decision making. Fascinating research and insights. Dense read but absolutely worth it.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days'
),

-- Review 22: Pragmatic Programmer - 4 stars
(
    'd0000000-0000-0000-0000-000000000022',
    '10000000-0000-0000-0000-000000000007',
    '60000000-0000-0000-0000-000000000042', -- Pragmatic Programmer
    '90000000-0000-0000-0000-000000000007',
    4,
    'Great programming wisdom',
    'Full of practical advice for software developers. Some tips feel dated but core principles are timeless. Good complement to Clean Code.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '12 days', NOW() - INTERVAL '12 days'
),

-- Review 23: Hoa Vàng Cỏ Xanh - 5 stars
(
    'd0000000-0000-0000-0000-000000000023',
    '10000000-0000-0000-0000-000000000008',
    '60000000-0000-0000-0000-000000000002', -- Hoa Vàng Cỏ Xanh
    '90000000-0000-0000-0000-000000000008',
    5,
    'Tuổi thơ tuyệt vời',
    'Đọc lại tuổi thơ, khóc rất nhiều. Nguyễn Nhật Ánh viết hay quá. Câu chuyện về tình anh em và tuổi thơ nghèo khó cảm động vô cùng.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days'
),

-- Review 24: Conan - 4 stars
(
    'd0000000-0000-0000-0000-000000000024',
    '10000000-0000-0000-0000-000000000010',
    '60000000-0000-0000-0000-000000000033', -- Conan 1
    '90000000-0000-0000-0000-000000000010',
    4,
    'Hay và hấp dẫn',
    'Truyện trinh thám hấp dẫn. Mua cho con đọc nhưng người lớn đọc cũng thích. In màu đẹp.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'
),

-- Review 25: Doraemon 2 - 5 stars
(
    'd0000000-0000-0000-0000-000000000025',
    '10000000-0000-0000-0000-000000000010',
    '60000000-0000-0000-0000-000000000034', -- Doraemon 2
    '90000000-0000-0000-0000-000000000010',
    5,
    'Con đọc mãi không chán',
    'Truyện Doraemon không bao giờ lỗi thời. Con đọc đi đọc lại nhiều lần. Giá cả hợp lý, chất lượng tốt.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'
),

-- Review 26: Python Crash - 4 stars
(
    'd0000000-0000-0000-0000-000000000026',
    '10000000-0000-0000-0000-000000000009',
    '60000000-0000-0000-0000-000000000051', -- Python Crash
    '90000000-0000-0000-0000-000000000009',
    4,
    'Good Python introduction',
    'Perfect for beginners. Clear explanations and practical projects. Covers basics well but lacks advanced topics.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'
),

-- Review 27: Zero to One - 4 stars
(
    'd0000000-0000-0000-0000-000000000027',
    '10000000-0000-0000-0000-000000000002',
    '60000000-0000-0000-0000-000000000013', -- Zero to One
    '90000000-0000-0000-0000-000000000002',
    4,
    'Innovative thinking',
    'Peter Thiel shares unique perspectives on startups and innovation. Thought-provoking read. Some ideas controversial but interesting.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '1 month 14 days', NOW() - INTERVAL '1 month 14 days'
),

-- Review 28: Good to Great - 3 stars
(
    'd0000000-0000-0000-0000-000000000028',
    '10000000-0000-0000-0000-000000000006',
    '60000000-0000-0000-0000-000000000016', -- Good to Great
    '90000000-0000-0000-0000-000000000006',
    3,
    'Decent business book',
    'Some good insights but examples are mostly from big old companies. Not sure how applicable to modern startups.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days'
),

-- Review 29: Deep Work - 4 stars
(
    'd0000000-0000-0000-0000-000000000029',
    '10000000-0000-0000-0000-000000000012',
    '60000000-0000-0000-0000-000000000024', -- Deep Work
    '90000000-0000-0000-0000-000000000012',
    4,
    'Important productivity book',
    'Cal Newport makes a strong case for focused work. Practical strategies to minimize distractions. Helped me improve my productivity significantly.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '22 hours', NOW() - INTERVAL '22 hours'
),

-- Review 30: Outliers - 4 stars
(
    'd0000000-0000-0000-0000-000000000030',
    '10000000-0000-0000-0000-000000000012',
    '60000000-0000-0000-0000-000000000026', -- Outliers
    '90000000-0000-0000-0000-000000000012',
    4,
    'Interesting success stories',
    'Malcolm Gladwell explores factors behind extraordinary success. The 10,000-hour rule is famous. Engaging stories but some conclusions debatable.',
    NULL,
    true, true, false, NULL,
    NOW() - INTERVAL '20 hours', NOW() - INTERVAL '20 hours'
);

-- =====================================================
-- UPDATE BOOK RATING STATS (Trigger will auto-update)
-- =====================================================
-- Manually trigger to ensure all books have correct ratings
DO $$
DECLARE
    book_record RECORD;
BEGIN
    FOR book_record IN 
        SELECT DISTINCT book_id FROM reviews WHERE is_approved = true
    LOOP
        UPDATE books
        SET 
            rating_average = (
                SELECT ROUND(AVG(rating)::numeric, 1)
                FROM reviews
                WHERE book_id = book_record.book_id AND is_approved = true
            ),
            rating_count = (
                SELECT COUNT(*)
                FROM reviews
                WHERE book_id = book_record.book_id AND is_approved = true
            )
        WHERE id = book_record.book_id;
    END LOOP;
END $$;

-- =====================================================
-- VERIFICATION QUERIES
-- =====================================================

-- Review statistics
SELECT 
    'Total Reviews' as metric,
    COUNT(*)::text as value
FROM reviews
UNION ALL
SELECT 'Approved Reviews', COUNT(*)::text FROM reviews WHERE is_approved = true
UNION ALL
SELECT 'Pending Reviews', COUNT(*)::text FROM reviews WHERE is_approved = false
UNION ALL
SELECT 'Featured Reviews', COUNT(*)::text FROM reviews WHERE is_featured = true;

-- Rating distribution
SELECT 
    rating,
    COUNT(*) as count,
    ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER(), 2) as percentage
FROM reviews
WHERE is_approved = true
GROUP BY rating
ORDER BY rating DESC;

-- Books with most reviews
SELECT 
    b.title,
    b.rating_average,
    b.rating_count,
    COUNT(r.id) as total_reviews,
    COUNT(r.id) FILTER (WHERE r.is_approved = true) as approved_reviews
FROM books b
LEFT JOIN reviews r ON b.id = r.book_id
GROUP BY b.id, b.title, b.rating_average, b.rating_count
HAVING COUNT(r.id) > 0
ORDER BY b.rating_count DESC
LIMIT 10;

-- Users with most reviews
SELECT 
    u.full_name,
    COUNT(r.id) as review_count,
    ROUND(AVG(r.rating), 1) as avg_rating_given
FROM users u
JOIN reviews r ON u.id = r.user_id
WHERE r.is_approved = true
GROUP BY u.id, u.full_name
ORDER BY review_count DESC;

-- Books rating verification
SELECT 
    b.title,
    b.rating_average as book_avg,
    ROUND(AVG(r.rating)::numeric, 1) as calculated_avg,
    b.rating_count as book_count,
    COUNT(r.id) as calculated_count
FROM books b
LEFT JOIN reviews r ON b.id = r.book_id AND r.is_approved = true
WHERE b.rating_count > 0
GROUP BY b.id, b.title, b.rating_average, b.rating_count
HAVING b.rating_average != ROUND(AVG(r.rating)::numeric, 1) OR b.rating_count != COUNT(r.id);
