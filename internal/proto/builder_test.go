package proto

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestIsStringEnum(t *testing.T) {
	for _, test := range []struct {
		name     string
		schema   *base.Schema
		expected bool
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: false,
		},
		{
			name: "string enum",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "active"},
					{Value: "inactive"},
				},
			},
			expected: true,
		},
		{
			name: "integer enum",
			schema: &base.Schema{
				Type: []string{"integer"},
				Enum: []*yaml.Node{
					{Value: "200"},
					{Value: "404"},
				},
			},
			expected: false,
		},
		{
			name: "string without enum",
			schema: &base.Schema{
				Type: []string{"string"},
			},
			expected: false,
		},
		{
			name: "empty enum",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{},
			},
			expected: false,
		},
		{
			name: "no type field",
			schema: &base.Schema{
				Enum: []*yaml.Node{
					{Value: "active"},
				},
			},
			expected: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := isStringEnum(test.schema)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestIsIntegerEnum(t *testing.T) {
	for _, test := range []struct {
		name     string
		schema   *base.Schema
		expected bool
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: false,
		},
		{
			name: "integer enum",
			schema: &base.Schema{
				Type: []string{"integer"},
				Enum: []*yaml.Node{
					{Value: "200"},
					{Value: "404"},
				},
			},
			expected: true,
		},
		{
			name: "string enum",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "active"},
					{Value: "inactive"},
				},
			},
			expected: false,
		},
		{
			name: "integer without enum",
			schema: &base.Schema{
				Type: []string{"integer"},
			},
			expected: false,
		},
		{
			name: "empty enum",
			schema: &base.Schema{
				Type: []string{"integer"},
				Enum: []*yaml.Node{},
			},
			expected: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := isIntegerEnum(test.schema)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestExtractEnumValues(t *testing.T) {
	for _, test := range []struct {
		name     string
		schema   *base.Schema
		expected []string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: []string{},
		},
		{
			name: "string enum values",
			schema: &base.Schema{
				Enum: []*yaml.Node{
					{Value: "active"},
					{Value: "inactive"},
					{Value: "pending"},
				},
			},
			expected: []string{"active", "inactive", "pending"},
		},
		{
			name: "integer enum values",
			schema: &base.Schema{
				Enum: []*yaml.Node{
					{Value: "200"},
					{Value: "401"},
					{Value: "404"},
				},
			},
			expected: []string{"200", "401", "404"},
		},
		{
			name: "empty enum",
			schema: &base.Schema{
				Enum: []*yaml.Node{},
			},
			expected: []string{},
		},
		{
			name: "enum with special characters",
			schema: &base.Schema{
				Enum: []*yaml.Node{
					{Value: "foo bar"},
					{Value: "a\"b"},
					{Value: "c[d]"},
				},
			},
			expected: []string{"foo bar", "a\"b", "c[d]"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := extractEnumValues(test.schema)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestValidateEnumSchema(t *testing.T) {
	for _, test := range []struct {
		name    string
		schema  *base.Schema
		wantErr string
	}{
		{
			name:    "nil schema",
			schema:  nil,
			wantErr: "",
		},
		{
			name: "valid string enum",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "active"},
					{Value: "inactive"},
				},
			},
			wantErr: "",
		},
		{
			name: "valid integer enum",
			schema: &base.Schema{
				Type: []string{"integer"},
				Enum: []*yaml.Node{
					{Value: "200"},
					{Value: "404"},
				},
			},
			wantErr: "",
		},
		{
			name: "enum without type field",
			schema: &base.Schema{
				Enum: []*yaml.Node{
					{Value: "active"},
				},
			},
			wantErr: "enum must have explicit type field",
		},
		{
			name: "enum with null value",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "active"},
					nil,
				},
			},
			wantErr: "enum cannot contain null values",
		},
		{
			name: "enum with empty string value",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "active"},
					{Value: ""},
				},
			},
			wantErr: "enum cannot contain null values",
		},
		{
			name: "enum with mixed types",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "active"},
					{Value: "200"},
				},
			},
			wantErr: "enum contains mixed types (string and integer)",
		},
		{
			name: "empty enum array",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{},
			},
			wantErr: "",
		},
		{
			name: "duplicate enum values",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "active"},
					{Value: "active"},
				},
			},
			wantErr: "",
		},
		{
			name: "case-sensitive enum values",
			schema: &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{
					{Value: "Active"},
					{Value: "active"},
				},
			},
			wantErr: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateEnumSchema(test.schema, "Status")
			if test.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, test.wantErr)
			}
		})
	}
}
