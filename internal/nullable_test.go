package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullableTypeHandling(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "nullable string becomes string field",
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
          type: [string, "null"]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string name = 1 [json_name = "name"];
}

`,
		},
		{
			name: "nullable integer becomes int32 field",
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
        quantity:
          type: [integer, "null"]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Product {
  int32 quantity = 1 [json_name = "quantity"];
}

`,
		},
		{
			name: "nullable int64 becomes int64 field",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Stats:
      type: object
      properties:
        count:
          type: [integer, "null"]
          format: int64
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Stats {
  int64 count = 1 [json_name = "count"];
}

`,
		},
		{
			name: "nullable number becomes double field",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Measurement:
      type: object
      properties:
        value:
          type: [number, "null"]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Measurement {
  double value = 1 [json_name = "value"];
}

`,
		},
		{
			name: "nullable float becomes float field",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Sensor:
      type: object
      properties:
        reading:
          type: [number, "null"]
          format: float
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Sensor {
  float reading = 1 [json_name = "reading"];
}

`,
		},
		{
			name: "nullable boolean becomes bool field",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Feature:
      type: object
      properties:
        enabled:
          type: [boolean, "null"]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Feature {
  bool enabled = 1 [json_name = "enabled"];
}

`,
		},
		{
			name: "nullable array becomes repeated field",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Container:
      type: object
      properties:
        items:
          type: [array, "null"]
          items:
            type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Container {
  repeated string items = 1 [json_name = "items"];
}

`,
		},
		{
			name: "nullable object becomes nested message",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Wrapper:
      type: object
      properties:
        data:
          type: [object, "null"]
          properties:
            value:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Wrapper {
  message Data {
    string value = 1 [json_name = "value"];
  }

  Data data = 1 [json_name = "data"];
}

`,
		},
		{
			name: "null first in type array",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      type: object
      properties:
        value:
          type: ["null", string]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Thing {
  string value = 1 [json_name = "value"];
}

`,
		},
		{
			name: "multiple properties with nullable types",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Record:
      type: object
      properties:
        name:
          type: [string, "null"]
        count:
          type: [integer, "null"]
        active:
          type: [boolean, "null"]
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Record {
  string name = 1 [json_name = "name"];
  int32 count = 2 [json_name = "count"];
  bool active = 3 [json_name = "active"];
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

func TestMultiTypeRejection(t *testing.T) {
	for _, test := range []struct {
		name    string
		given   string
		wantErr string
	}{
		{
			name: "string and integer rejected",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Bad:
      type: object
      properties:
        value:
          type: [string, integer]
`,
			wantErr: "multi-type properties not supported (only nullable variants allowed)",
		},
		{
			name: "string and number rejected",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Bad:
      type: object
      properties:
        value:
          type: [string, number]
`,
			wantErr: "multi-type properties not supported (only nullable variants allowed)",
		},
		{
			name: "three types with null rejected",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Bad:
      type: object
      properties:
        value:
          type: [string, integer, "null"]
`,
			wantErr: "multi-type properties not supported (only nullable variants allowed)",
		},
		{
			name: "boolean and integer rejected",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Bad:
      type: object
      properties:
        value:
          type: [boolean, integer]
`,
			wantErr: "multi-type properties not supported (only nullable variants allowed)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})

			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestGoClientNullableTypes(t *testing.T) {
	for _, test := range []struct {
		name        string
		given       string
		expectGoGen bool
	}{
		{
			name: "nullable string in Go client",
			given: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      properties:
        petType:
          type: string
        name:
          type: [string, "null"]
    Cat:
      type: object
      properties:
        petType:
          type: string
        name:
          type: [string, "null"]
`,
			expectGoGen: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName:   "testpkg",
				PackagePath:   "github.com/example/proto/v1",
				GoPackagePath: "github.com/example/types/v1",
			})

			require.NoError(t, err)
			require.NotNil(t, result)

			if test.expectGoGen {
				require.NotEmpty(t, result.Golang)
				goCode := string(result.Golang)
				assert.Contains(t, goCode, "Name string")
			}
		})
	}
}
