package utils

import (
	"regexp"
	"strings"
)

// utils.GenerateSlug() implementation (in pkg/utils/slug.go)
func GenerateSlug(input string) string {
	// Step 1: Convert Vietnamese characters to ASCII
	// "Nguyễn Nhật Ánh" → "Nguyen Nhat Anh"
	ascii := RemoveDiacritics(input)

	// Step 2: Lowercase
	// "Nguyen Nhat Anh" → "nguyen nhat anh"
	lower := strings.ToLower(ascii)

	// Step 3: Replace spaces with hyphens
	// "nguyen nhat anh" → "nguyen-nhat-anh"
	hyphenated := strings.ReplaceAll(lower, " ", "-")

	// Step 4: Remove special characters
	// Keep only: a-z, 0-9, hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	cleaned := reg.ReplaceAllString(hyphenated, "")

	// Step 5: Remove multiple consecutive hyphens
	// "nguyen--nhat---anh" → "nguyen-nhat-anh"
	reg = regexp.MustCompile(`-+`)
	normalized := reg.ReplaceAllString(cleaned, "-")

	// Step 6: Trim leading/trailing hyphens
	trimmed := strings.Trim(normalized, "-")

	return trimmed
}

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
