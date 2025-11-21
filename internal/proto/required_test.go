package proto_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertRequiredIgnored(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "required array is ignored",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      required:
        - email
        - name
      properties:
        email:
          type: string
        name:
          type: string
        age:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
  int32 age = 3 [json_name = "age"];
}

`,
		},
		{
			name: "all fields required - still no optional keyword",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Product:
      type: object
      required:
        - id
        - title
        - price
      properties:
        id:
          type: string
        title:
          type: string
        price:
          type: number
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Product {
  string id = 1 [json_name = "id"];
  string title = 2 [json_name = "title"];
  double price = 3 [json_name = "price"];
}

`,
		},
		{
			name: "no required array - fields identical",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      type: object
      properties:
        field1:
          type: string
        field2:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Thing {
  string field1 = 1 [json_name = "field1"];
  int32 field2 = 2 [json_name = "field2"];
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

func TestConvertNullableIgnored(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "nullable true is ignored",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Item:
      type: object
      properties:
        name:
          type: string
          nullable: true
        count:
          type: integer
          nullable: true
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Item {
  string name = 1 [json_name = "name"];
  int32 count = 2 [json_name = "count"];
}

`,
		},
		{
			name: "nullable false is ignored",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Data:
      type: object
      properties:
        value:
          type: string
          nullable: false
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Data {
  string value = 1 [json_name = "value"];
}

`,
		},
		{
			name: "no wrapper types for nullable fields",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Record:
      type: object
      properties:
        optionalInt:
          type: integer
          nullable: true
        optionalString:
          type: string
          nullable: true
        optionalBool:
          type: boolean
          nullable: true
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Record {
  int32 optionalInt = 1 [json_name = "optionalInt"];
  string optionalString = 2 [json_name = "optionalString"];
  bool optionalBool = 3 [json_name = "optionalBool"];
}

`,
		},
		{
			name: "mixed nullable and non-nullable fields are identical",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Mixed:
      type: object
      properties:
        nullable:
          type: string
          nullable: true
        nonNullable:
          type: string
        unspecified:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Mixed {
  string nullable = 1 [json_name = "nullable"];
  string nonNullable = 2 [json_name = "nonNullable"];
  string unspecified = 3 [json_name = "unspecified"];
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
