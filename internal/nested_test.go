package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNestedObject(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "simple nested object",
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
        name:
          type: string
        location:
          type: object
          properties:
            street:
              type: string
            city:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  message Location {
    string street = 1 [json_name = "street"];
    string city = 2 [json_name = "city"];
  }

  string name = 1 [json_name = "name"];
  Location location = 2 [json_name = "location"];
}

`,
		},
		{
			name: "nested object with independent field numbering",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Person:
      type: object
      properties:
        id:
          type: integer
        email:
          type: string
        contact:
          type: object
          properties:
            phone:
              type: string
            mobile:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Person {
  message Contact {
    string phone = 1 [json_name = "phone"];
    string mobile = 2 [json_name = "mobile"];
  }

  int32 id = 1 [json_name = "id"];
  string email = 2 [json_name = "email"];
  Contact contact = 3 [json_name = "contact"];
}

`,
		},
		{
			name: "nested object with camelCase conversion",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Order:
      type: object
      properties:
        orderId:
          type: string
        shippingInfo:
          type: object
          properties:
            streetName:
              type: string
            zipCode:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Order {
  message ShippingInfo {
    string streetName = 1 [json_name = "streetName"];
    string zipCode = 2 [json_name = "zipCode"];
  }

  string orderId = 1 [json_name = "orderId"];
  ShippingInfo shippingInfo = 2 [json_name = "shippingInfo"];
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

func TestDeeplyNestedObjects(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "three levels of nesting",
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
        name:
          type: string
        office:
          type: object
          properties:
            location:
              type: object
              properties:
                street:
                  type: string
                city:
                  type: string
            phone:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Company {
  message Office {
    message Location {
      string street = 1 [json_name = "street"];
      string city = 2 [json_name = "city"];
    }

    Location location = 1 [json_name = "location"];
    string phone = 2 [json_name = "phone"];
  }

  string name = 1 [json_name = "name"];
  Office office = 2 [json_name = "office"];
}

`,
		},
		{
			name: "multiple nested objects",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Profile:
      type: object
      properties:
        billing:
          type: object
          properties:
            card:
              type: string
        shipping:
          type: object
          properties:
            method:
              type: string
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Profile {
  message Billing {
    string card = 1 [json_name = "card"];
  }

  message Shipping {
    string method = 1 [json_name = "method"];
  }

  Billing billing = 1 [json_name = "billing"];
  Shipping shipping = 2 [json_name = "shipping"];
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

func TestNestedObjectPluralName(t *testing.T) {
	for _, test := range []struct {
		name        string
		given       string
		expectedErr string
	}{
		{
			name: "property not ending in 's' - should pass",
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
        profile:
          type: object
          properties:
            bio:
              type: string
`,
			expectedErr: "",
		},
		{
			name: "property ending in 's' for plural",
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
        contacts:
          type: object
          properties:
            phone:
              type: string
`,
			expectedErr: "cannot derive message name from property 'contacts'; use singular form or $ref",
		},
		{
			name: "property ending in 'es'",
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
        addresses:
          type: object
          properties:
            street:
              type: string
`,
			expectedErr: "cannot derive message name from property 'addresses'; use singular form or $ref",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			if test.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestNestedObjectWithDescription(t *testing.T) {
	given := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      description: A user profile
      properties:
        name:
          type: string
          description: User's full name
        profile:
          type: object
          description: User's profile
          properties:
            bio:
              type: string
              description: Biography
`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

// A user profile
message User {
  // User's profile
  message Profile {
    // Biography
    string bio = 1 [json_name = "bio"];
  }

  // User's full name
  string name = 1 [json_name = "name"];
  Profile profile = 2 [json_name = "profile"];
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
