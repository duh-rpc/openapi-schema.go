package proto_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldNamesAcrossMessages(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "common field names across messages",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    CreateUserRequest:
      type: object
      properties:
        email:
          type: string
        name:
          type: string
    CreateUserResponse:
      type: object
      properties:
        email:
          type: string
        name:
          type: string
    UpdateUserRequest:
      type: object
      properties:
        email:
          type: string
        name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message CreateUserRequest {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

message CreateUserResponse {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

message UpdateUserRequest {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

`,
		},
		{
			name: "field numbering independence",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Message1:
      type: object
      properties:
        email:
          type: string
        name:
          type: string
    Message2:
      type: object
      properties:
        email:
          type: string
        name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Message1 {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

message Message2 {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

`,
		},
		{
			name: "sanitization collisions within message",
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
        user-name:
          type: string
        user_name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string user_name = 1 [json_name = "user-name"];
  string user_name_2 = 2 [json_name = "user_name"];
}

`,
		},
		{
			name: "original bug scenario",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    CreateUserRequest:
      type: object
      properties:
        email:
          type: string
        name:
          type: string
    CreateUserResponse:
      type: object
      properties:
        email:
          type: string
        name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message CreateUserRequest {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

message CreateUserResponse {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

`,
		},
		{
			name: "parent and nested with same field names",
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
        email:
          type: string
        location:
          type: object
          properties:
            email:
              type: string
            city:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  message Location {
    string email = 1 [json_name = "email"];
    string city = 2 [json_name = "city"];
  }

  string email = 1 [json_name = "email"];
  Location location = 2 [json_name = "location"];
}

`,
		},
		{
			name: "multiple nested in different parents",
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
        location:
          type: object
          properties:
            city:
              type: string
    Product:
      type: object
      properties:
        origin:
          type: object
          properties:
            city:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  message Location {
    string city = 1 [json_name = "city"];
  }

  Location location = 1 [json_name = "location"];
}

message Product {
  message Origin {
    string city = 1 [json_name = "city"];
  }

  Origin origin = 1 [json_name = "origin"];
}

`,
		},
		{
			name: "mixed top-level and nested",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    CreateUserRequest:
      type: object
      properties:
        email:
          type: string
    CreateUserResponse:
      type: object
      properties:
        email:
          type: string
    User:
      type: object
      properties:
        contact:
          type: object
          properties:
            email:
              type: string
    Product:
      type: object
      properties:
        supplier:
          type: object
          properties:
            email:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message CreateUserRequest {
  string email = 1 [json_name = "email"];
}

message CreateUserResponse {
  string email = 1 [json_name = "email"];
}

message User {
  message Contact {
    string email = 1 [json_name = "email"];
  }

  Contact contact = 1 [json_name = "contact"];
}

message Product {
  message Supplier {
    string email = 1 [json_name = "email"];
  }

  Supplier supplier = 1 [json_name = "supplier"];
}

`,
		},
		{
			name: "deeply nested recursive structures",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Organization:
      type: object
      properties:
        name:
          type: string
        department:
          type: object
          properties:
            name:
              type: string
            team:
              type: object
              properties:
                name:
                  type: string
                id:
                  type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Organization {
  message Department {
    message Team {
      string name = 1 [json_name = "name"];
      string id = 2 [json_name = "id"];
    }

    string name = 1 [json_name = "name"];
    Team team = 2 [json_name = "team"];
  }

  string name = 1 [json_name = "name"];
  Department department = 2 [json_name = "department"];
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
