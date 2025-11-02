package category

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================
// 📚 KHÁI NIỆM: Value Object vs Entity
// ============================================================
// VALUE OBJECT:
//   - Không có identity (không quan tâm ID)
//   - Immutable (không thay đổi sau khi tạo)
//   - So sánh bằng value, không bằng reference
//   - VÍ DỤ: Money(100, "VND"), Address("123 Street"), TreePath([1, 2, 3])
//
// ENTITY:
//   - Có identity unique (ID)
//   - Mutable (có thể thay đổi)
//   - So sánh bằng ID
//   - VÍ DỤ: Category(id=123, name="Book"), User(id=456, email="...")
//
// TRONG BÀI: Category là ENTITY, TreePath là VALUE OBJECT

// ============================================================
// ENTITY: Category
// ============================================================
// Category đại diện 1 danh mục sản phẩm có ID unique
// Nó tuân theo mô hình cây (Tree) với parent_id
//
// PROPERTIES:
// - Identity: ID (UUID)
// - State: Name, Slug, ParentID, IsActive, etc.
// - Behavior: Update, SetActive, CanDelete, etc.
//
// DATABASE MAPPING:
// ┌─────────────────────────┐
// │    categories table      │
// ├─────────────────────────┤
// │ id (UUID) - PRIMARY KEY │
// │ name (TEXT)             │
// │ slug (TEXT) - UNIQUE    │
// │ parent_id (UUID) - FK   │
// │ sort_order (INT)        │
// │ description (TEXT)      │
// │ icon_url (TEXT)         │
// │ is_active (BOOLEAN)     │
// │ created_at              │
// │ updated_at              │
// └─────────────────────────┘
type Category struct {
	// ========== Identity ==========
	// ID là định danh duy nhất của category
	// Format: UUID v4 (chuỗi 36 ký tự)
	// Ví dụ: "550e8400-e29b-41d4-a716-446655440000"
	// Generated: PostgreSQL tự generate hoặc Go generate
	ID uuid.UUID

	// ========== Basic Info ==========
	// Name: Tên hiển thị (ví dụ: "Tiểu thuyết")
	// Constraint: NOT NULL, max 255 chars
	Name string

	// Slug: URL-friendly version (ví dụ: "tieu-thuyet")
	// Constraint: UNIQUE NOT NULL
	// Dùng cho: API endpoint, SEO, bookmarks
	// Generated: Auto từ Name
	Slug string

	// ========== Hierarchy ==========
	// ParentID: Reference tới category cha
	// NULL => Root category (cấp 1)
	// UUID => Child category (cấp 2+)
	// Ý nghĩa: Tạo quan hệ cha-con trong cây
	//
	// TREE EXAMPLE:
	// ├─ Văn học (ParentID: nil, level: 1)
	// │  ├─ Tiểu thuyết (ParentID: văn_học_id, level: 2)
	// │  │  ├─ Trinh thám (ParentID: tiểu_thuyết_id, level: 3)
	// │  │  └─ Tình cảm (ParentID: tiểu_thuyết_id, level: 3)
	// │  └─ Thơ (ParentID: văn_học_id, level: 2)
	ParentID *uuid.UUID

	// SortOrder: Thứ tự hiển thị trong cùng level
	// Constraint: 0-999
	// Dùng: Order By trong query
	// Ví dụ: Nếu parent = "Văn học"
	//   - Tiểu thuyết: sort_order = 0
	//   - Thơ: sort_order = 1
	//   - Triết học: sort_order = 2
	SortOrder int

	// ========== Display/UI ==========
	// Description: Mô tả chi tiết (dùng product page)
	// Constraint: max 1000 chars
	Description string

	// IconURL: Link đến icon (dùng UI)
	// Ví dụ: "https://cdn.bookstore.com/icons/tieu-thuyet.png"
	IconURL string

	// IsActive: Cờ ẩn/hiện category
	// true => Hiển thị
	// false => Ẩn (inactive)
	// Lợi ích: Soft feature instead of hard delete
	IsActive bool

	// ========== Timestamps ==========
	// CreatedAt: Thời điểm tạo
	// Format: RFC3339 (2024-11-02T10:52:00+07:00)
	CreatedAt time.Time

	// UpdatedAt: Thời điểm cập nhật lần cuối
	// Được auto update bởi trigger DB
	UpdatedAt time.Time

	// ========== Runtime Metadata (không lưu DB) ==========
	// Level: Độ sâu trong cây (1=root, 2=child, 3=grandchild)
	// Set bởi Repository sau query
	Level *int

	// FullPath: Breadcrumb đầy đủ
	// VÍ DỤ: "Văn học > Tiểu thuyết > Trinh thám"
	// Dùng: UI breadcrumb, admin view
	FullPath *string

	// ChildCount: Số con trực tiếp
	// Dùng: Check xem có thể delete không
	ChildCount *int

	// TotalBooksCount: Tổng books (bao gồm descendants)
	// Dùng: Display badge "245 cuốn sách"
	TotalBooksCount *int64
}

// ============================================================
// VALUE OBJECT: TreePath
// ============================================================
// TreePath đại diện 1 nút trong cây phân cấp
// Nó chứa metadata để traverse cây
//
// KHÁI NIỆM - Path là gì?
// Path là mảng sort_order từ root tới node hiện tại
// Ví dụ: [0, 1, 2]
//   - [0] = root category (first root)
//   - [0, 1] = child của root đó (second child)
//   - [0, 1, 2] = grandchild (third grandchild)
//
// Dùng để:
// 1. So sánh thứ tự (sort)
// 2. Detect depth (level = len(path))
// 3. Build full path (tên đầy đủ)
type TreePath struct {
	// Path: Mảng sort_order từ root tới node
	// VÍ DỤ:
	//   Root: []
	//   Child: [1]
	//   Grandchild: [1, 0]
	Path []int

	// Level: Độ sâu (length của path + 1)
	// Root: level=1
	// Child: level=2
	// Grandchild: level=3
	Level int

	// FullPath: Tên dễ đọc
	// VÍ DỤ: "Văn học > Tiểu thuyết > Trinh thám"
	// Dùng: Breadcrumb, UI display
	FullPath string
}

// ============================================================
// VALUE OBJECT: CategoryFilter
// ============================================================
// CategoryFilter dùng để filter khi query danh sách categories
// Nó là value object vì không có identity, chỉ là điều kiện filter
//
// KHÁI NIỆM - Filter là gì?
// Filter là tập hợp điều kiện để lọc dữ liệu
// Ví dụ:
//   - GetAll(filter={IsActive: true, Limit: 10})
//   - SELECT * FROM categories WHERE is_active = true LIMIT 10
//
// Lợi ích:
// - Dễ thêm filter mới (không cần thay đổi function signature)
// - Dễ test (mock filter)
// - Type-safe (so với varargs hay map[string]interface{})
type CategoryFilter struct {
	// IsActive: Chỉ active categories
	// nil => ignore (lấy tất cả)
	// true => chỉ active
	// false => chỉ inactive
	IsActive *bool

	// ParentID: Filter by parent
	// nil => root categories hoặc tất cả
	// UUID => chỉ children của parent này
	ParentID *uuid.UUID

	// IncludeInactive: Bao gồm inactive
	// Dùng cho admin view
	IncludeInactive bool

	// Pagination
	Limit  int // Default: 10, Max: 100
	Offset int // Default: 0
}

// ============================================================
// FACTORY METHOD: NewCategory
// ============================================================
// Factory method là design pattern để tạo instance
// Thay vì dùng &Category{...}, dùng NewCategory(...)
//
// LỢI ÍCH:
// 1. Validation: Đảm bảo object hợp lệ từ khi tạo
// 2. Initialization: Set default values, timestamps
// 3. Encapsulation: Control cách tạo object
//
// VÍ DỤ SO SÁNH:
// ❌ BAD:
//
//	cat := &Category{Name: "", Slug: ""}  // Có thể tạo invalid object
//
// ✅ GOOD:
//
//	cat, err := NewCategory("Tiểu Thuyết", nil, "", "", 0)
//	if err != nil {
//	  return err  // Validation fail, không tạo object
//	}
//
// FLOW:
// 1. Validate input
// 2. Generate slug
// 3. Create instance
// 4. Return với error check
func NewCategory(
	name string,
	parentID *uuid.UUID,
	description string,
	iconURL string,
	sortOrder int,
) (*Category, error) {
	// ========== VALIDATION LAYER ==========
	// Validate là bước kiểm tra dữ liệu
	// Lợi ích:
	// - Fail fast: Lỗi được phát hiện sớm
	// - User-friendly errors: Error message rõ ràng
	// - Security: Prevent invalid data vào DB
	//
	// VALIDATION STRATEGY:
	// 1. Required fields (not empty)
	// 2. Length limits (255 chars)
	// 3. Type validation (sortOrder: 0-999)
	// 4. Business rules (slug unique - check ở Repository)

	// 1. Validate Name
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("category name cannot be empty")
	}

	if len(name) > 255 {
		return nil, fmt.Errorf("category name must not exceed 255 characters (got %d)", len(name))
	}

	// 2. Validate Description
	if len(description) > 1000 {
		return nil, fmt.Errorf("category description must not exceed 1000 characters (got %d)", len(description))
	}

	// 3. Validate SortOrder
	if sortOrder < 0 || sortOrder > 999 {
		return nil, fmt.Errorf("sort_order must be between 0 and 999 (got %d)", sortOrder)
	}

	// ========== SLUG GENERATION ==========
	// GenerateSlug là function để tạo slug từ name
	// FLOW:
	// 1. "Tiểu Thuyết" (input)
	// 2. "tiểu thuyết" (lowercase)
	// 3. "tieu thuyet" (remove diacritics)
	// 4. "tieu-thuyet" (replace spaces with dashes)
	// 5. "tieu-thuyet" (remove special chars)
	//
	// OUTPUT: "tieu-thuyet" (URL-friendly)
	slug := GenerateSlug(name)

	// ========== CREATE INSTANCE ==========
	now := time.Now()
	category := &Category{
		ID:          uuid.New(), // Generate new UUID
		Name:        strings.TrimSpace(name),
		Slug:        slug,
		ParentID:    parentID,
		SortOrder:   sortOrder,
		Description: description,
		IconURL:     iconURL,
		IsActive:    true, // Default: active
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return category, nil
}

// ============================================================
// DOMAIN METHOD: Update
// ============================================================
// Update là domain method để modify category
// Domain method khác với factory vì nó modify state của object
//
// LỢI ÍCH:
// 1. Encapsulation: Logic update tập trung ở entity
// 2. Consistency: Validation được apply mỗi khi update
// 3. Maintainability: Dễ change logic sau này
//
// FLOW:
// 1. Validate input
// 2. Update fields
// 3. Update timestamp
//
// IMPORTANT: Không update ID, CreatedAt, ParentID ở đây
// - ID: Không thay đổi (identity)
// - CreatedAt: Chỉ set khi tạo
// - ParentID: Dùng MoveToParent() (check circular reference riêng)
func (c *Category) Update(
	name string,
	description string,
	iconURL string,
	sortOrder int,
) error {
	// Validate tương tự NewCategory
	if strings.TrimSpace(name) == "" {
		return errors.New("category name cannot be empty")
	}

	if len(name) > 255 {
		return fmt.Errorf("category name must not exceed 255 characters (got %d)", len(name))
	}

	if len(description) > 1000 {
		return fmt.Errorf("category description must not exceed 1000 characters (got %d)", len(description))
	}

	if sortOrder < 0 || sortOrder > 999 {
		return fmt.Errorf("sort_order must be between 0 and 999 (got %d)", sortOrder)
	}

	// Update fields
	c.Name = strings.TrimSpace(name)
	c.Slug = GenerateSlug(name)
	c.Description = description
	c.IconURL = iconURL
	c.SortOrder = sortOrder

	// Update timestamp (auto update khi có change)
	c.UpdatedAt = time.Now()

	return nil
}

// ============================================================
// DOMAIN METHOD: SetActive / SetInactive
// ============================================================
// SetActive / SetInactive thay đổi trạng thái category
// Simple setter methods
func (c *Category) SetActive(active bool) {
	c.IsActive = active
	c.UpdatedAt = time.Now()
}

// ============================================================
// DOMAIN METHOD: CanDelete
// ============================================================
// CanDelete kiểm tra xem category có thể delete không
// RULES:
// 1. Không có children
// 2. Không có books
//
// FLOW:
// delete_handler -> category_service.Delete() -> repository.Delete()
// -> repository check: CanDelete()
//
// LỢI ÍCH: Validate trước khi query DB
func (c *Category) CanDelete() bool {
	// Nếu có children, không delete
	if c.ChildCount != nil && *c.ChildCount > 0 {
		return false
	}

	// Nếu có books, không delete
	if c.TotalBooksCount != nil && *c.TotalBooksCount > 0 {
		return false
	}

	return true
}

// ============================================================
// DOMAIN METHOD: IsRoot
// ============================================================
// IsRoot kiểm tra category là root hay không
// Root = cấp 1, ParentID = NULL
func (c *Category) IsRoot() bool {
	return c.ParentID == nil
}

// ============================================================
// DOMAIN METHOD: GetLevel
// ============================================================
// GetLevel trả về level (độ sâu) của category
func (c *Category) GetLevel() int {
	if c.Level == nil {
		return 1 // Default: root
	}
	return *c.Level
}

// ============================================================
// UTILITY FUNCTION: GenerateSlug
// ============================================================
// GenerateSlug tạo slug từ name
// Slug là URL-friendly version của name
//
// KHÁI NIỆM - Slug là gì?
// Slug là phần text trong URL thay vì ID
// VÍ DỤ:
//
//	❌ /category/550e8400-e29b-41d4-a716-446655440000 (ugly)
//	✅ /category/tieu-thuyet (beautiful, SEO-friendly)
//
// ALGORITHM:
// 1. Trim spaces
// 2. Lowercase
// 3. Remove Vietnamese diacritics (á, à, ả, ã, ạ => a)
// 4. Replace spaces with dashes
// 5. Remove special characters (keep only a-z, 0-9, -)
// 6. Remove multiple dashes
// 7. Trim leading/trailing dashes
//
// VÍ DỤ:
// - "Tiểu Thuyết" => "tieu-thuyet"
// - "Sách Kỹ Năng!!!" => "sach-ky-nang"
// - "Hello---World" => "hello-world"
func GenerateSlug(name string) string {
	// Step 1: Trim
	slug := strings.TrimSpace(name)

	// Step 2: Lowercase
	slug = strings.ToLower(slug)

	// Step 3: Remove diacritics
	slug = RemoveDiacritics(slug)

	// Step 4: Replace spaces with dashes
	slug = strings.ReplaceAll(slug, " ", "-")

	// Step 5: Remove special characters
	// Regex: [^a-z0-9-]+ (match anything NOT a-z, 0-9, -)
	reg := regexp.MustCompile("[^a-z0-9-]+")
	slug = reg.ReplaceAllString(slug, "")

	// Step 6: Remove multiple dashes
	// Regex: -+ (match multiple dashes)
	slug = regexp.MustCompile("-+").ReplaceAllString(slug, "-")

	// Step 7: Trim leading/trailing dashes
	slug = strings.Trim(slug, "-")

	return slug
}

// ============================================================
// UTILITY FUNCTION: RemoveDiacritics
// ============================================================
// RemoveDiacritics loại bỏ diacritics từ tiếng Việt
//
// KHÁI NIỆM - Diacritics là gì?
// Diacritics là ký tự phụ (tone marks) trong tiếng Việt
// VÍ DỤ:
// - á, à, ả, ã, ạ => Tất cả là "a" với tone marks khác nhau
// - é, è, ẻ, ẽ, ẹ => Tất cả là "e"
//
// TẠI SAO REMOVE?
// URL không support diacritics (encode thành %C3%A1 rất xấu)
// Slug cần phải clean, readable
// "tìm kiếm" => "tim-kiem" (readable, SEO-friendly)
//
// ALGORITHM:
// Dùng mapping table: char_with_diacritic => char_without
// VÍ DỤ:
// á => a
// à => a
// ả => a
// (tất cả các tone của "a" => "a")
func RemoveDiacritics(input string) string {
	// Mapping diacritics tới base character
	// Các key là ký tự với diacritic
	// Các value là ký tự base
	mappings := map[rune]rune{
		// Vowel A
		'á': 'a', 'à': 'a', 'ả': 'a', 'ã': 'a', 'ạ': 'a',
		'ă': 'a', 'ắ': 'a', 'ằ': 'a', 'ẳ': 'a', 'ẵ': 'a', 'ặ': 'a',
		'â': 'a', 'ấ': 'a', 'ầ': 'a', 'ẩ': 'a', 'ẫ': 'a', 'ậ': 'a',

		// Vowel E
		'é': 'e', 'è': 'e', 'ẻ': 'e', 'ẽ': 'e', 'ẹ': 'e',
		'ê': 'e', 'ế': 'e', 'ề': 'e', 'ể': 'e', 'ễ': 'e', 'ệ': 'e',

		// Vowel I
		'í': 'i', 'ì': 'i', 'ỉ': 'i', 'ĩ': 'i', 'ị': 'i',

		// Vowel O
		'ó': 'o', 'ò': 'o', 'ỏ': 'o', 'õ': 'o', 'ọ': 'o',
		'ô': 'o', 'ố': 'o', 'ồ': 'o', 'ổ': 'o', 'ỗ': 'o', 'ộ': 'o',
		'ơ': 'o', 'ớ': 'o', 'ờ': 'o', 'ở': 'o', 'ỡ': 'o', 'ợ': 'o',

		// Vowel U
		'ú': 'u', 'ù': 'u', 'ủ': 'u', 'ũ': 'u', 'ụ': 'u',
		'ư': 'u', 'ứ': 'u', 'ừ': 'u', 'ử': 'u', 'ữ': 'u', 'ự': 'u',

		// Vowel Y
		'ý': 'y', 'ỳ': 'y', 'ỷ': 'y', 'ỹ': 'y', 'ỵ': 'y',

		// Consonant D
		'đ': 'd',

		// UPPERCASE
		'Á': 'A', 'À': 'A', 'Ả': 'A', 'Ã': 'A', 'Ạ': 'A',
		'Ă': 'A', 'Ắ': 'A', 'Ằ': 'A', 'Ẳ': 'A', 'Ẵ': 'A', 'Ặ': 'A',
		'Â': 'A', 'Ấ': 'A', 'Ầ': 'A', 'Ẩ': 'A', 'Ẫ': 'A', 'Ậ': 'A',

		'É': 'E', 'È': 'E', 'Ẻ': 'E', 'Ẽ': 'E', 'Ẹ': 'E',
		'Ê': 'E', 'Ế': 'E', 'Ề': 'E', 'Ể': 'E', 'Ễ': 'E', 'Ệ': 'E',

		'Í': 'I', 'Ì': 'I', 'Ỉ': 'I', 'Ĩ': 'I', 'Ị': 'I',

		'Ó': 'O', 'Ò': 'O', 'Ỏ': 'O', 'Õ': 'O', 'Ọ': 'O',
		'Ô': 'O', 'Ố': 'O', 'Ồ': 'O', 'Ổ': 'O', 'Ỗ': 'O', 'Ộ': 'O',
		'Ơ': 'O', 'Ớ': 'O', 'Ờ': 'O', 'Ở': 'O', 'Ỡ': 'O', 'Ợ': 'O',

		'Ú': 'U', 'Ù': 'U', 'Ủ': 'U', 'Ũ': 'U', 'Ụ': 'U',
		'Ư': 'U', 'Ứ': 'U', 'Ừ': 'U', 'Ử': 'U', 'Ữ': 'U', 'Ự': 'U',

		'Ý': 'Y', 'Ỳ': 'Y', 'Ỷ': 'Y', 'Ỹ': 'Y', 'Ỵ': 'Y',

		'Đ': 'D',
	}

	// Convert string to rune array
	// Dùng rune vì Vietnamese ký tự không phải ASCII (multi-byte)
	result := make([]rune, 0, len(input))

	// Iterate qua mỗi character (rune)
	for _, r := range input {
		// Nếu char có trong mapping, replace
		if replacement, ok := mappings[r]; ok {
			result = append(result, replacement)
		} else {
			// Nếu không, giữ nguyên
			result = append(result, r)
		}
	}

	return string(result)
}

// ============================================================
// STRING REPRESENTATION
// ============================================================
func (c *Category) String() string {
	return fmt.Sprintf(
		"Category{ID: %s, Name: %s, Slug: %s, Level: %d, IsActive: %v}",
		c.ID,
		c.Name,
		c.Slug,
		c.GetLevel(),
		c.IsActive,
	)
}
