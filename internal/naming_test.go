package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertJSONNameAnnotation(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "userId gets json_name annotation",
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
        userId:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string userId = 1 [json_name = "userId"];
}

`,
		},
		{
			name: "email gets json_name annotation (always included)",
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
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string email = 1 [json_name = "email"];
}

`,
		},
		{
			name: "HTTPStatus preserved (field name preservation)",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Response:
      type: object
      properties:
        HTTPStatus:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Response {
  int32 HTTPStatus = 1 [json_name = "HTTPStatus"];
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

func TestConvertAlwaysIncludesJsonName(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    MixedNaming:
      type: object
      properties:
        userId:
          type: string
        user_id:
          type: string
        HTTPStatus:
          type: integer
        status_code:
          type: integer
        email:
          type: string
`
	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message MixedNaming {
  string userId = 1 [json_name = "userId"];
  string user_id = 2 [json_name = "user_id"];
  int32 HTTPStatus = 3 [json_name = "HTTPStatus"];
  int32 status_code = 4 [json_name = "status_code"];
  string email = 5 [json_name = "email"];
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

func TestConvertFieldNameSanitization(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "hyphen replaced with underscore",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Response:
      type: object
      properties:
        status-code:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Response {
  string status_code = 1 [json_name = "status-code"];
}

`,
		},
		{
			name: "dot replaced with underscore",
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
        user.name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string user_name = 1 [json_name = "user.name"];
}

`,
		},
		{
			name: "space replaced with underscore",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Person:
      type: object
      properties:
        first name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Person {
  string first_name = 1 [json_name = "first name"];
}

`,
		},
		{
			name: "multiple invalid chars collapsed",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        user--name..test:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string user_name_test = 1 [json_name = "user--name..test"];
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

func TestConvertInvalidFieldNames(t *testing.T) {
	for _, test := range []struct {
		name    string
		given   string
		wantErr string
	}{
		{
			name: "field starting with digit",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Invalid:
      type: object
      properties:
        2ndValue:
          type: string
`,
			wantErr: "field name must start with a letter",
		},
		{
			name: "field starting with underscore",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Invalid:
      type: object
      properties:
        _private:
          type: string
`,
			wantErr: "field name cannot start with underscore",
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

func TestConvertMixedValidAndInvalidChars(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "mixed case with hyphen",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        user-ID:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string user_ID = 1 [json_name = "user-ID"];
}

`,
		},
		{
			name: "uppercase with hyphen",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        HTTP-Status:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string HTTP_Status = 1 [json_name = "HTTP-Status"];
}

`,
		},
		{
			name: "dots in version string",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        api.v2.endpoint:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string api_v2_endpoint = 1 [json_name = "api.v2.endpoint"];
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

func TestConvertConsecutiveInvalidChars(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "consecutive hyphens collapsed",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        status---code:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string status_code = 1 [json_name = "status---code"];
}

`,
		},
		{
			name: "consecutive dots collapsed",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        user...name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string user_name = 1 [json_name = "user...name"];
}

`,
		},
		{
			name: "consecutive spaces collapsed",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        first  name:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string first_name = 1 [json_name = "first  name"];
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

func TestConvertTrailingInvalidChars(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "trailing hyphen trimmed",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        status-:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string status = 1 [json_name = "status-"];
}

`,
		},
		{
			name: "trailing underscore preserved",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        user_:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string user_ = 1 [json_name = "user_"];
}

`,
		},
		{
			name: "complex trailing chars sanitized and trimmed",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Test:
      type: object
      properties:
        name-_-:
          type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Test {
  string name_ = 1 [json_name = "name-_-"];
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
