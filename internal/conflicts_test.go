package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertNameConflicts(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "duplicate message names get numeric suffixes",
			given: `openapi: 3.0.0
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
    user:
      type: object
      properties:
        email:
          type: string
    USER:
      type: object
      properties:
        id:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string name = 1 [json_name = "name"];
}

message User_2 {
  string email = 1 [json_name = "email"];
}

message User_3 {
  int32 id = 1 [json_name = "id"];
}

`,
		},
		{
			name: "duplicate enum names get numeric suffixes",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Status:
      type: integer
      enum:
        - 1
        - 2
    status:
      type: integer
      enum:
        - 10
        - 20
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_1 = 1;
  STATUS_2 = 2;
}

enum Status_2 {
  STATUS_2_UNSPECIFIED = 0;
  STATUS_2_10 = 1;
  STATUS_2_20 = 2;
}

`,
		},
		{
			name: "mixed message and enum with same name",
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
    item:
      type: integer
      enum:
        - 1
        - 2
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Item {
  string name = 1 [json_name = "name"];
}

enum Item_2 {
  ITEM_2_UNSPECIFIED = 0;
  ITEM_2_1 = 1;
  ITEM_2_2 = 2;
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
