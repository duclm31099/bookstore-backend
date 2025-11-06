package utils

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func ParseFloatToDecimal(number *float64) *decimal.Decimal {
	if number == nil {
		return nil
	}
	d := decimal.NewFromFloat(*number)
	return &d
}

func ParseStringToUUID(s string) uuid.UUID {
	uid, err := uuid.Parse(s)
	if err != nil || s == "" {
		return uuid.Nil
	}
	return uid
}

// Helper: Generate slug from title
func GenerateSlugBook(title string) string {
	// Remove Vietnamese accents
	title = RemoveVietnameseAccents(title)

	// Convert to lowercase
	title = strings.ToLower(title)

	// Replace spaces with hyphens
	title = strings.ReplaceAll(title, " ", "-")

	// Remove special characters (chỉ giữ a-z, 0-9, -)
	reg := regexp.MustCompile("[^a-z0-9-]+")
	title = reg.ReplaceAllString(title, "")

	// Remove duplicate hyphens
	title = regexp.MustCompile("-+").ReplaceAllString(title, "-")

	// Trim hyphens
	title = strings.Trim(title, "-")

	return title
}

// Helper: Remove Vietnamese accents
func RemoveVietnameseAccents(str string) string {
	// Map Vietnamese characters to ASCII
	replacements := map[string]string{
		"à": "a", "á": "a", "ả": "a", "ã": "a", "ạ": "a",
		"ă": "a", "ằ": "a", "ắ": "a", "ẳ": "a", "ẵ": "a", "ặ": "a",
		"â": "a", "ầ": "a", "ấ": "a", "ẩ": "a", "ẫ": "a", "ậ": "a",
		"đ": "d",
		"è": "e", "é": "e", "ẻ": "e", "ẽ": "e", "ẹ": "e",
		"ê": "e", "ề": "e", "ế": "e", "ể": "e", "ễ": "e", "ệ": "e",
		"ì": "i", "í": "i", "ỉ": "i", "ĩ": "i", "ị": "i",
		"ò": "o", "ó": "o", "ỏ": "o", "õ": "o", "ọ": "o",
		"ô": "o", "ồ": "o", "ố": "o", "ổ": "o", "ỗ": "o", "ộ": "o",
		"ơ": "o", "ờ": "o", "ớ": "o", "ở": "o", "ỡ": "o", "ợ": "o",
		"ù": "u", "ú": "u", "ủ": "u", "ũ": "u", "ụ": "u",
		"ư": "u", "ừ": "u", "ứ": "u", "ử": "u", "ữ": "u", "ự": "u",
		"ỳ": "y", "ý": "y", "ỷ": "y", "ỹ": "y", "ỵ": "y",
	}

	for viet, ascii := range replacements {
		str = strings.ReplaceAll(str, viet, ascii)
		str = strings.ReplaceAll(str, strings.ToUpper(viet), strings.ToUpper(ascii))
	}

	return str
}

// isValidUUID - Kiểm tra format UUID hợp lệ
func IsValidUUID(u string) bool {
	if len(u) != 36 {
		return false
	}
	// Simple validation: check dashes at correct positions
	if u[8] != '-' || u[13] != '-' || u[18] != '-' || u[23] != '-' {
		return false
	}
	return true
}
