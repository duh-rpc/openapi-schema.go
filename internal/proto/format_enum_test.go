package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatEnumComment(t *testing.T) {
	for _, test := range []struct {
		name     string
		values   []string
		indent   string
		expected string
	}{
		{
			name:     "simple enum values",
			values:   []string{"active", "inactive", "pending"},
			indent:   "",
			expected: "// enum: [active, inactive, pending]\n",
		},
		{
			name:     "enum with indentation",
			values:   []string{"active", "inactive"},
			indent:   "  ",
			expected: "  // enum: [active, inactive]\n",
		},
		{
			name:     "empty enum values",
			values:   []string{},
			indent:   "",
			expected: "",
		},
		{
			name:     "nil enum values",
			values:   nil,
			indent:   "",
			expected: "",
		},
		{
			name:     "enum with quotes",
			values:   []string{"a\"b", "c"},
			indent:   "",
			expected: "// enum: [a\"b, c]\n",
		},
		{
			name:     "enum with brackets",
			values:   []string{"a[b]", "c"},
			indent:   "",
			expected: "// enum: [a[b], c]\n",
		},
		{
			name:     "enum with spaces",
			values:   []string{"foo bar", "baz"},
			indent:   "",
			expected: "// enum: [foo bar, baz]\n",
		},
		{
			name:     "enum with special characters",
			values:   []string{"a\"b", "c[d]", "e f"},
			indent:   "",
			expected: "// enum: [a\"b, c[d], e f]\n",
		},
		{
			name:     "single value",
			values:   []string{"active"},
			indent:   "",
			expected: "// enum: [active]\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := formatEnumComment(test.values, test.indent)
			assert.Equal(t, test.expected, result)
		})
	}
}
