package utils

import (
	"regexp"
	"strings"
)

// FormatWhatsApp standardizes phone numbers to 628... format.
func FormatWhatsApp(phone string) string {
	// 1. Remove all non-digit characters
	reg := regexp.MustCompile(`[^0-9]`)
	clean := reg.ReplaceAllString(phone, "")

	// 2. Format
	if strings.HasPrefix(clean, "08") {
		return "62" + clean[1:]
	}
	if strings.HasPrefix(clean, "8") {
		return "62" + clean
	}
	if strings.HasPrefix(clean, "628") {
		return clean
	}

	return clean
}
