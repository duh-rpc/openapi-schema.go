package proto_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertScalarTypes(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
		wantErr  string
	}{
		{
			name: "all scalar type mappings",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    AllTypes:
      type: object
      properties:
        int32Field:
          type: integer
          format: int32
        int64Field:
          type: integer
          format: int64
        floatField:
          type: number
          format: float
        doubleField:
          type: number
          format: double
        stringField:
          type: string
        bytesField:
          type: string
          format: byte
        binaryField:
          type: string
          format: binary
        dateField:
          type: string
          format: date
        dateTimeField:
          type: string
          format: date-time
        boolField:
          type: boolean
`,
			expected: `syntax = "proto3";

package testpkg;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/example/proto/v1";

message AllTypes {
  int32 int32Field = 1 [json_name = "int32Field"];
  int64 int64Field = 2 [json_name = "int64Field"];
  float floatField = 3 [json_name = "floatField"];
  double doubleField = 4 [json_name = "doubleField"];
  string stringField = 5 [json_name = "stringField"];
  bytes bytesField = 6 [json_name = "bytesField"];
  bytes binaryField = 7 [json_name = "binaryField"];
  google.protobuf.Timestamp dateField = 8 [json_name = "dateField"];
  google.protobuf.Timestamp dateTimeField = 9 [json_name = "dateTimeField"];
  bool boolField = 10 [json_name = "boolField"];
}

`,
		},
		{
			name: "default integer format",
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
        count:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Thing {
  int32 count = 1 [json_name = "count"];
}

`,
		},
		{
			name: "default number format",
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
        value:
          type: number
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Thing {
  double value = 1 [json_name = "value"];
}

`,
		},
		{
			name: "array of timestamps",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        timestamp:
          type: array
          items:
            type: string
            format: date-time
`,
			expected: `syntax = "proto3";

package testpkg;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/example/proto/v1";

message Event {
  repeated google.protobuf.Timestamp timestamp = 1 [json_name = "timestamp"];
}

`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, test.expected, string(result.Protobuf))
		})
	}
}
