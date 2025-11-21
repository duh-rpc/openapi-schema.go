package internal

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestContains(t *testing.T) {
	for _, test := range []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "exact match",
			slice:    []string{"string", "integer", "boolean"},
			item:     "string",
			expected: true,
		},
		{
			name:     "case insensitive match",
			slice:    []string{"String", "Integer", "Boolean"},
			item:     "string",
			expected: true,
		},
		{
			name:     "not found",
			slice:    []string{"string", "integer", "boolean"},
			item:     "object",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "string",
			expected: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := Contains(test.slice, test.item)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestExtractReferenceName(t *testing.T) {
	for _, test := range []struct {
		name        string
		ref         string
		expected    string
		expectError bool
	}{
		{
			name:     "valid reference",
			ref:      "#/components/schemas/Address",
			expected: "Address",
		},
		{
			name:     "valid reference with nested name",
			ref:      "#/components/schemas/User/Address",
			expected: "Address",
		},
		{
			name:        "empty reference",
			ref:         "",
			expectError: true,
		},
		{
			name:        "invalid format - missing components",
			ref:         "#/schemas/Address",
			expectError: true,
		},
		{
			name:        "invalid format - wrong prefix",
			ref:         "$/components/schemas/Address",
			expectError: true,
		},
		{
			name:        "invalid format - empty name",
			ref:         "#/components/schemas/",
			expectError: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := ExtractReferenceName(test.ref)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestIsEnumSchema(t *testing.T) {
	for _, test := range []struct {
		name     string
		schema   *base.Schema
		expected bool
	}{
		{
			name: "schema with enum",
			schema: &base.Schema{
				Enum: []*yaml.Node{
					{Value: "value1"},
					{Value: "value2"},
				},
			},
			expected: true,
		},
		{
			name: "schema without enum",
			schema: &base.Schema{
				Enum: []*yaml.Node{},
			},
			expected: false,
		},
		{
			name:     "nil enum",
			schema:   &base.Schema{},
			expected: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := IsEnumSchema(test.schema)
			assert.Equal(t, test.expected, result)
		})
	}
}
