package internal

import (
	"fmt"
	"strings"
	"unicode"
)

// ToSnakeCase converts camelCase/PascalCase to snake_case.
// Algorithm: Each uppercase letter becomes lowercase with underscore prefix (except first char).
// Examples: userId → user_id, HTTPStatus → h_t_t_p_status, email → email
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s) + 5)

	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToPascalCase converts snake_case/camelCase/ALLCAPS to PascalCase.
// Examples: user_id → UserId, shippingAddress → ShippingAddress, USER → User
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Check if the string is all uppercase (no lowercase letters)
	isAllCaps := true
	hasUnderscore := false
	for _, r := range s {
		if r == '_' {
			hasUnderscore = true
			continue
		}
		if unicode.IsLower(r) {
			isAllCaps = false
			break
		}
	}

	var result strings.Builder
	result.Grow(len(s))

	capitalizeNext := true
	for _, r := range s {
		if r == '_' {
			capitalizeNext = true
			continue
		}

		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			// Only lowercase if the entire string was all caps (like "USER")
			// For camelCase (like "OrderStatus"), preserve the original casing
			if isAllCaps && !hasUnderscore {
				result.WriteRune(unicode.ToLower(r))
			} else {
				result.WriteRune(r)
			}
		}
	}

	return result.String()
}

// ToEnumValueName converts a value to ENUM_PREFIX_VALUE_NAME format.
// Examples: (Status, active) → STATUS_ACTIVE, (Status, in-progress) → STATUS_IN_PROGRESS,
// (SortBy, createdAt) → SORT_BY_CREATED_AT.
//
// Values that are already in CONSTANT_CASE are normalized without re-splitting
// (so STATUS_UNSPECIFIED stays STATUS_UNSPECIFIED, not S_T_A_T_U_S_...), and a value
// that already carries the enum prefix is not double-prefixed
// (so (Status, STATUS_UNSPECIFIED) → STATUS_UNSPECIFIED, not STATUS_STATUS_UNSPECIFIED).
func ToEnumValueName(enumName, value string) string {
	upperEnum := strings.ToUpper(ToSnakeCase(enumName))
	upperValue := normalizeEnumValue(value)
	if upperValue == upperEnum || strings.HasPrefix(upperValue, upperEnum+"_") {
		return upperValue
	}
	return fmt.Sprintf("%s_%s", upperEnum, upperValue)
}

// normalizeEnumValue converts an enum value to CONSTANT_CASE. Mixed/camelCase
// values are snake-cased first (createdAt → created_at); values already lacking
// lowercase letters (active, STATUS_UNSPECIFIED) are only upper-cased so an
// already-formatted constant is preserved intact.
func normalizeEnumValue(value string) string {
	hasLower := strings.ContainsFunc(value, unicode.IsLower)
	normalized := value
	if hasLower {
		normalized = ToSnakeCase(value)
	}
	normalized = strings.ToUpper(normalized)
	return strings.ReplaceAll(normalized, "-", "_")
}

// SanitizeFieldName sanitizes an OpenAPI field name for proto3 syntax.
// Preserves the original name structure when valid, only modifying to meet
// proto3 requirements:
//   - Must start with a letter (A-Z, a-z)
//   - Can contain letters, digits, underscores
//   - Invalid characters replaced with underscores
//
// Returns error if name cannot be sanitized (e.g., starts with digit).
func SanitizeFieldName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("field name cannot be empty")
	}

	// Check first character must be ASCII letter
	firstChar := rune(name[0])
	if (firstChar < 'a' || firstChar > 'z') && (firstChar < 'A' || firstChar > 'Z') {
		if firstChar >= '0' && firstChar <= '9' {
			return "", fmt.Errorf("field name must start with a letter, got '%s'", name)
		}
		if firstChar == '_' {
			return "", fmt.Errorf("field name cannot start with underscore, got '%s'", name)
		}
		return "", fmt.Errorf("field name must start with a letter, got '%s'", name)
	}

	var result strings.Builder
	result.Grow(len(name))

	var lastWritten rune
	for i, r := range name {
		if isValidProtoFieldChar(r) {
			result.WriteRune(r)
			lastWritten = r
		} else {
			// Replace invalid char with underscore, but avoid consecutive underscores
			if lastWritten != '_' {
				result.WriteRune('_')
				lastWritten = '_'
			}
		}

		// Track if this is the last character
		if i == len(name)-1 {
			// Trim trailing underscore only if it was added by sanitization
			// (i.e., the original char was invalid)
			if !isValidProtoFieldChar(r) && lastWritten == '_' {
				s := result.String()
				if len(s) > 0 && s[len(s)-1] == '_' {
					return s[:len(s)-1], nil
				}
			}
		}
	}

	sanitized := result.String()
	if sanitized == "" {
		return "", fmt.Errorf("field name contains no valid characters")
	}

	return sanitized, nil
}

// isValidProtoFieldChar returns true if character is valid in proto3 field name.
func isValidProtoFieldChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// NameTracker tracks used names and generates unique names when conflicts occur.
type NameTracker struct {
	used map[string]int
}

// NewNameTracker creates a new NameTracker.
func NewNameTracker() *NameTracker {
	return &NameTracker{
		used: make(map[string]int),
	}
}

// UniqueName returns a unique name, adding numeric suffix if needed (_2, _3, etc.).
func (nt *NameTracker) UniqueName(name string) string {
	count, exists := nt.used[name]
	if !exists {
		nt.used[name] = 1
		return name
	}

	count++
	nt.used[name] = count
	return fmt.Sprintf("%s_%d", name, count)
}
