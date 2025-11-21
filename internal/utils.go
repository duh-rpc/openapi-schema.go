package internal

import (
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// Contains checks if a slice contains a string (case-insensitive)
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// ExtractReferenceName extracts the schema name from a reference string.
// Example: "#/components/schemas/Address" → "Address"
func ExtractReferenceName(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("reference string is empty")
	}

	// Split by '/' and validate standard format: "#/components/schemas/Name"
	parts := strings.Split(ref, "/")
	if len(parts) < 4 || parts[0] != "#" || parts[1] != "components" || parts[2] != "schemas" {
		return "", fmt.Errorf("invalid reference format: %s (expected #/components/schemas/Name)", ref)
	}

	name := parts[len(parts)-1]
	if name == "" {
		return "", fmt.Errorf("reference has empty name segment: %s", ref)
	}

	return name, nil
}

// IsEnumSchema returns true if schema defines an enum
func IsEnumSchema(schema *base.Schema) bool {
	return len(schema.Enum) > 0
}
