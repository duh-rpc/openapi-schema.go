package schema_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertFieldNumbersOverride verifies that a supplied FieldNumbers fully
// drives message field numbering by JSON name (independent of declaration order)
// and renders a reserved statement for retired numbers.
func TestConvertFieldNumbersOverride(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    CreateUserRequest:
      type: object
      properties:
        displayName:
          type: string
        email:
          type: string
        name:
          type: string`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message CreateUserRequest {
  string name = 1 [json_name = "name"];
  string email = 2 [json_name = "email"];
  string displayName = 4 [json_name = "displayName"];
  reserved 3;
}

`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
		FieldNumbers: &schema.FieldNumbers{
			Messages: map[string]schema.MessageNumbers{
				"CreateUserRequest": {
					Fields:   map[string]int{"name": 1, "email": 2, "displayName": 4},
					Reserved: []int{3},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected, string(result.Protobuf))
}

// TestConvertFieldNumbersMissingFieldErrors verifies the library errors (rather
// than silently falling back to positional) when a live field has no mapped number,
// which signals a caller reconciliation bug.
func TestConvertFieldNumbersMissingFieldErrors(t *testing.T) {
	given := `openapi: 3.0.0
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

	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
		FieldNumbers: &schema.FieldNumbers{
			Messages: map[string]schema.MessageNumbers{
				"Thing": {Fields: map[string]int{"a": 1}},
			},
		},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "b")
}
