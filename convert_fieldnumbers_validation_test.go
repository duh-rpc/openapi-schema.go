package schema_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertFieldNumbersValidation verifies that supplied FieldNumbers are
// validated the same way x-proto-number values are: numbers must be in range
// 1..536870911, must not fall in the reserved 19000-19999 range, must be unique,
// and a Reserved entry must not collide with an active field number. Without
// validation the library silently emits invalid proto3.
func TestConvertFieldNumbersValidation(t *testing.T) {
	const given = `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      type: object
      properties:
        a:
          type: string
        b:
          type: string`

	for _, test := range []struct {
		name    string
		numbers schema.MessageNumbers
		wantErr string
	}{
		{
			name:    "duplicate field number",
			numbers: schema.MessageNumbers{Fields: map[string]int{"a": 5, "b": 5}},
			wantErr: "5",
		},
		{
			name:    "zero field number",
			numbers: schema.MessageNumbers{Fields: map[string]int{"a": 0, "b": 2}},
			wantErr: "a",
		},
		{
			name:    "negative field number",
			numbers: schema.MessageNumbers{Fields: map[string]int{"a": -1, "b": 2}},
			wantErr: "a",
		},
		{
			name:    "field number in reserved range",
			numbers: schema.MessageNumbers{Fields: map[string]int{"a": 19000, "b": 2}},
			wantErr: "19000",
		},
		{
			name:    "field number above max",
			numbers: schema.MessageNumbers{Fields: map[string]int{"a": 536870912, "b": 2}},
			wantErr: "536870911",
		},
		{
			name:    "reserved collides with active field",
			numbers: schema.MessageNumbers{Fields: map[string]int{"a": 1, "b": 2}, Reserved: []int{2}},
			wantErr: "2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
				FieldNumbers: &schema.FieldNumbers{
					Messages: map[string]schema.MessageNumbers{"Thing": test.numbers},
				},
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, test.wantErr)
		})
	}
}

// TestConvertEnumNumbersRequiresZeroValue verifies that an EnumNumbers override
// which maps no variant to 0 is rejected. proto3 requires the first enum value to
// be 0; without a zero value the generated enum is invalid proto3.
func TestConvertEnumNumbersRequiresZeroValue(t *testing.T) {
	const given = `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Code:
      type: integer
      enum:
        - 200
        - 404`

	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
		FieldNumbers: &schema.FieldNumbers{
			Enums: map[string]schema.EnumNumbers{
				"Code": {Variants: map[string]int{"200": 1, "404": 2}},
			},
		},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "0")
}
