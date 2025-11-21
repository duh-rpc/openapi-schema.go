package proto_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrayOfScalars(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "array of strings",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        tags:
          type: array
          items:
            type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  repeated string tags = 1 [json_name = "tags"];
}

`,
		},
		{
			name: "array of integers",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Numbers:
      type: object
      properties:
        values:
          type: array
          items:
            type: integer
            format: int32
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Numbers {
  repeated int32 values = 1 [json_name = "values"];
}

`,
		},
		{
			name: "array of int64",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Data:
      type: object
      properties:
        ids:
          type: array
          items:
            type: integer
            format: int64
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Data {
  repeated int64 ids = 1 [json_name = "ids"];
}

`,
		},
		{
			name: "array of booleans",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Flags:
      type: object
      properties:
        enabled:
          type: array
          items:
            type: boolean
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Flags {
  repeated bool enabled = 1 [json_name = "enabled"];
}

`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, test.expected, string(result.Protobuf))
		})
	}
}

func TestArrayOfReferences(t *testing.T) {
	given := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Address:
      type: object
      properties:
        street:
          type: string
    User:
      type: object
      properties:
        addresses:
          type: array
          items:
            $ref: '#/components/schemas/Address'
`
	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Address {
  string street = 1 [json_name = "street"];
}

message User {
  repeated Address addresses = 1 [json_name = "addresses"];
}

`
	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected, string(result.Protobuf))
}

func TestArrayOfInlineObjects(t *testing.T) {
	// Test array with singular property name (should work)
	singular := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Company:
      type: object
      properties:
        contact:
          type: array
          items:
            type: object
            properties:
              name:
                type: string
`
	singularExpected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Company {
  message Contact {
    string name = 1 [json_name = "name"];
  }

  repeated Contact contact = 1 [json_name = "contact"];
}

`
	result, err := schema.Convert([]byte(singular), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, singularExpected, string(result.Protobuf))
}

func TestArrayOfInlineEnums(t *testing.T) {
	// Test array with singular property name (should work)
	singular := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        level:
          type: array
          items:
            type: string
            enum:
              - low
              - medium
              - high
`
	singularExpected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Config {
  // enum: [low, medium, high]
  repeated string level = 1 [json_name = "level"];
}

`
	result, err := schema.Convert([]byte(singular), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, singularExpected, string(result.Protobuf))
}

func TestArrayPluralName(t *testing.T) {
	for _, test := range []struct {
		name    string
		given   string
		wantErr string
	}{
		{
			name: "inline object with plural name ending in 's'",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Company:
      type: object
      properties:
        contacts:
          type: array
          items:
            type: object
            properties:
              name:
                type: string
`,
			wantErr: "cannot derive message name from plural array property 'contacts'",
		},
		{
			name: "inline object with plural name ending in 'es'",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Company:
      type: object
      properties:
        addresses:
          type: array
          items:
            type: object
            properties:
              street:
                type: string
`,
			wantErr: "cannot derive message name from plural array property 'addresses'",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			require.ErrorContains(t, err, test.wantErr)
		})
	}

	// String enums in arrays are now allowed with plural names
	t.Run("inline string enum with plural name is allowed", func(t *testing.T) {
		given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        statuses:
          type: array
          items:
            type: string
            enum:
              - active
              - inactive`

		expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Config {
  // enum: [active, inactive]
  repeated string statuses = 1 [json_name = "statuses"];
}

`
		result, err := schema.Convert([]byte(given), schema.ConvertOptions{
			PackageName: "testpkg",
			PackagePath: "github.com/example/proto/v1",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected, string(result.Protobuf))
	})
}

func TestNestedArrays(t *testing.T) {
	given := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Matrix:
      type: object
      properties:
        data:
          type: array
          items:
            type: array
            items:
              type: integer
`
	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "nested arrays not supported")
}

func TestArrayWithoutItems(t *testing.T) {
	given := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    BadArray:
      type: object
      properties:
        data:
          type: array
`
	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "array must have items defined")
}
