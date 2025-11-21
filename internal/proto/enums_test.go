package proto_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopLevelEnum(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - pending
        - isActive`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected, string(result.Protobuf))
}

func TestEnumWithDashes(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - in-progress
        - not-started
        - completed
    Task:
      type: object
      properties:
        status:
          $ref: '#/components/schemas/Status'`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Task {
  // enum: [in-progress, not-started, completed]
  string status = 1 [json_name = "status"];
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

func TestEnumWithNumbers(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Code:
      type: integer
      enum:
        - 200
        - 401
        - 404
        - 500`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

enum Code {
  CODE_UNSPECIFIED = 0;
  CODE_200 = 1;
  CODE_401 = 2;
  CODE_404 = 3;
  CODE_500 = 4;
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

func TestEnumWithDescription(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      description: Status of the operation
      enum:
        - active
        - inactive
    Task:
      type: object
      properties:
        status:
          $ref: '#/components/schemas/Status'`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Task {
  // Status of the operation
  // enum: [active, inactive]
  string status = 1 [json_name = "status"];
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

func TestInlineEnum(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        status:
          type: string
          description: User status
          enum:
            - active
            - inactive
            - notStarted`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string name = 1 [json_name = "name"];
  // User status
  // enum: [active, inactive, notStarted]
  string status = 2 [json_name = "status"];
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

func TestMultipleInlineEnums(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        status:
          type: string
          enum:
            - active
            - inactive
        role:
          type: string
          enum:
            - admin
            - user
            - superAdmin`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  // enum: [active, inactive]
  string status = 1 [json_name = "status"];
  // enum: [admin, user, superAdmin]
  string role = 2 [json_name = "role"];
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

func TestEnumAndMessageMixed(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
    User:
      type: object
      properties:
        name:
          type: string
    Priority:
      type: string
      enum:
        - high
        - low`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string name = 1 [json_name = "name"];
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

func TestStringEnumReference(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
    Task:
      type: object
      properties:
        status:
          $ref: '#/components/schemas/Status'`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Task {
  // enum: [active, inactive]
  string status = 1 [json_name = "status"];
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

func TestMixedEnumTypes(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Task:
      type: object
      properties:
        status:
          type: string
          enum:
            - active
            - inactive
        code:
          type: integer
          enum:
            - 200
            - 404`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

enum Code {
  CODE_UNSPECIFIED = 0;
  CODE_200 = 1;
  CODE_404 = 2;
}

message Task {
  // enum: [active, inactive]
  string status = 1 [json_name = "status"];
  Code code = 2 [json_name = "code"];
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

func TestStringEnumInArray(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Article:
      type: object
      properties:
        tag:
          type: array
          items:
            type: string
            enum:
              - draft
              - published`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Article {
  // enum: [draft, published]
  repeated string tag = 1 [json_name = "tag"];
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

func TestStringEnumArrayReference(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    OrderStatus:
      type: string
      enum:
        - pending
        - shipped
    Report:
      type: object
      properties:
        statuses:
          type: array
          items:
            $ref: '#/components/schemas/OrderStatus'`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Report {
  // enum: [pending, shipped]
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
}

func TestEnumValidationErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		given   string
		wantErr string
	}{
		{
			name: "enum without type field",
			given: `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      enum:
        - active
        - inactive`,
			wantErr: "enum must have explicit type field",
		},
		{
			name: "enum with mixed types",
			given: `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Code:
      type: string
      enum:
        - active
        - 200`,
			wantErr: "enum contains mixed types (string and integer)",
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
}

func TestStringEnumSpecialCharacters(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Task:
      type: object
      properties:
        value:
          type: string
          enum:
            - foo bar
            - a"b
            - c[d]`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Task {
  // enum: [foo bar, a"b, c[d]]
  string value = 1 [json_name = "value"];
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

func TestIntegerEnumDescriptionPreserved(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Task:
      type: object
      properties:
        code:
          type: integer
          description: HTTP status code
          enum:
            - 200
            - 404`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

// HTTP status code
enum Code {
  CODE_UNSPECIFIED = 0;
  CODE_200 = 1;
  CODE_404 = 2;
}

message Task {
  Code code = 1 [json_name = "code"];
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
