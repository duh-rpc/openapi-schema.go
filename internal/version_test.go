package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPI31Support(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "valid OpenAPI 3.1.0 document",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string name = 1 [json_name = "name"];
  int32 age = 2 [json_name = "age"];
}

`,
		},
		{
			name: "OpenAPI 3.1.0 with nullable types",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Product:
      type: object
      properties:
        name:
          type: [string, null]
        price:
          type: [number, null]
          format: double
        stock:
          type: [integer, null]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Product {
  string name = 1 [json_name = "name"];
  double price = 2 [json_name = "price"];
  int32 stock = 3 [json_name = "stock"];
}

`,
		},
		{
			name: "OpenAPI 3.1.0 with complex schema",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          type: string
    User:
      type: object
      properties:
        name:
          type: [string, null]
        email:
          type: string
        addresses:
          type: array
          items:
            $ref: '#/components/schemas/Address'
        age:
          type: [integer, null]
          format: int32
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Address {
  string street = 1 [json_name = "street"];
  string city = 2 [json_name = "city"];
}

message User {
  string name = 1 [json_name = "name"];
  string email = 2 [json_name = "email"];
  repeated Address addresses = 3 [json_name = "addresses"];
  int32 age = 4 [json_name = "age"];
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

func TestOpenAPI32Support(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "valid OpenAPI 3.2.0 document",
			given: `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string name = 1 [json_name = "name"];
  int32 age = 2 [json_name = "age"];
}

`,
		},
		{
			name: "OpenAPI 3.2.0 with nullable types",
			given: `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Product:
      type: object
      properties:
        name:
          type: [string, null]
        price:
          type: [number, null]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Product {
  string name = 1 [json_name = "name"];
  double price = 2 [json_name = "price"];
}

`,
		},
		{
			name: "OpenAPI 3.2.0 backward compatibility with 3.1",
			given: `openapi: 3.2.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - pending
    Record:
      type: object
      properties:
        id:
          type: string
        status:
          $ref: '#/components/schemas/Status'
        count:
          type: [integer, null]
        active:
          type: [boolean, null]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Record {
  string id = 1 [json_name = "id"];
  // enum: [active, inactive, pending]
  string status = 2 [json_name = "status"];
  int32 count = 3 [json_name = "count"];
  bool active = 4 [json_name = "active"];
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

func TestMultipleVersionsCompatibility(t *testing.T) {
	for _, test := range []struct {
		name          string
		version       string
		wantErr       string
		expectSuccess bool
	}{
		{
			name:          "OpenAPI 3.0.0 supported",
			version:       "3.0.0",
			expectSuccess: true,
		},
		{
			name:          "OpenAPI 3.0.1 supported",
			version:       "3.0.1",
			expectSuccess: true,
		},
		{
			name:          "OpenAPI 3.0.3 supported",
			version:       "3.0.3",
			expectSuccess: true,
		},
		{
			name:          "OpenAPI 3.1.0 supported",
			version:       "3.1.0",
			expectSuccess: true,
		},
		{
			name:          "OpenAPI 3.1.1 supported",
			version:       "3.1.1",
			expectSuccess: true,
		},
		{
			name:          "OpenAPI 3.2.0 supported",
			version:       "3.2.0",
			expectSuccess: true,
		},
		{
			name:    "OpenAPI 2.0 not supported",
			version: "2.0",
			wantErr: "spec is defined as an openapi spec, but is using a swagger (2.0), or unknown version",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			openapi := `openapi: ` + test.version + `
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
`

			result, err := schema.Convert([]byte(openapi), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}

			if test.expectSuccess {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Contains(t, string(result.Protobuf), "message User")
			}
		})
	}
}
