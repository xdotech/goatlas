package parser

import (
	"strings"
	"unicode"
)

// NormalizePath converts dynamic path segments to {id} placeholders for matching.
// Examples:
//
//	/users/123          → /users/{id}
//	/transfer-orders/45 → /transfer-orders/{id}
//	/api/v1/users       → /api/v1/users  (unchanged)
//	/items/:id          → /items/{id}
//	/items/$id          → /items/{id}
func NormalizePath(path string) string {
	// Strip /api prefix for consistent comparison
	normalized := strings.TrimPrefix(path, "/api")
	if normalized == "" {
		normalized = path
	} else if !strings.HasPrefix(normalized, "/") {
		normalized = path // only strip when result still starts with /
	}

	segments := strings.Split(normalized, "/")
	for i, seg := range segments {
		if isDynamicSegment(seg) {
			segments[i] = "{id}"
		}
	}
	return strings.Join(segments, "/")
}

// isDynamicSegment returns true for segments that represent path parameters.
func isDynamicSegment(seg string) bool {
	if seg == "" {
		return false
	}
	// Explicit param syntax
	if strings.HasPrefix(seg, ":") || strings.HasPrefix(seg, "$") || strings.HasPrefix(seg, "{") {
		return true
	}
	// Strip template literal suffix like /${uid} → check the whole thing
	if strings.Contains(seg, "${") {
		return true
	}
	// All digits → numeric ID
	if isNumericID(seg) {
		return true
	}
	// UUID-like hex (8-4-4-4-12 or all hex, 32 chars)
	if isUUID(seg) {
		return true
	}
	return false
}

func isNumericID(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

func isUUID(s string) bool {
	if len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' {
		return true
	}
	if len(s) == 32 {
		for _, r := range s {
			if !unicode.Is(unicode.Hex_Digit, r) {
				return false
			}
		}
		return true
	}
	return false
}
