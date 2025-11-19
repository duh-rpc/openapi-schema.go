package conv_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	conv "github.com/duh-rpc/openapi-proto.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertProtoOnlyTypeMap validates that all schemas in TypeMap are proto-only
func assertProtoOnlyTypeMap(t *testing.T, result *conv.ConvertResult, expectedSchemas []string) {
	require.NotNil(t, result.TypeMap)
	assert.Len(t, result.TypeMap, len(expectedSchemas))

	for _, schema := range expectedSchemas {
		info, exists := result.TypeMap[schema]
		require.True(t, exists, "schema %s not in TypeMap", schema)
		assert.Equal(t, conv.TypeLocationProto, info.Location)
		assert.Empty(t, info.Reason)
	}
}

func TestConvertBasics(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    []byte
		opts     conv.ConvertOptions
		expected string
		wantErr  string
	}{
		{
			name:    "empty openapi bytes",
			given:   []byte{},
			opts:    conv.ConvertOptions{PackageName: "testpkg", PackagePath: "github.com/example/proto/v1"},
			wantErr: "openapi input cannot be empty",
		},
		{
			name:    "empty package name",
			given:   []byte("openapi: 3.0.0"),
			opts:    conv.ConvertOptions{PackagePath: "github.com/example/proto/v1"},
			wantErr: "package name cannot be empty",
		},
		{
			name:    "empty package path",
			given:   []byte("openapi: 3.0.0"),
			opts:    conv.ConvertOptions{PackageName: "testpkg"},
			wantErr: "package path cannot be empty",
		},
		{
			name:    "both empty",
			given:   []byte("openapi: 3.0.0"),
			opts:    conv.ConvertOptions{},
			wantErr: "package name cannot be empty",
		},
		{
			name:    "invalid YAML syntax",
			given:   []byte("this is not valid: [yaml"),
			opts:    conv.ConvertOptions{PackageName: "testpkg", PackagePath: "github.com/example/proto/v1"},
			wantErr: "failed to parse OpenAPI document",
		},
		{
			name: "valid minimal OpenAPI 3.0",
			given: []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}

`),
			opts:     conv.ConvertOptions{PackageName: "testpkg", PackagePath: "github.com/example/proto/v1"},
			expected: "syntax = \"proto3\";\n\npackage testpkg;\n\noption go_package = \"github.com/example/proto/v1\";\n\n",
		},
		{
			name: "OpenAPI 2.0 Swagger",
			given: []byte(`swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths: {}

`),
			opts:    conv.ConvertOptions{PackageName: "testpkg", PackagePath: "github.com/example/proto/v1"},
			wantErr: "supplied spec is a different version",
		},
		{
			name: "valid JSON OpenAPI",
			given: []byte(`{
  "openapi": "3.0.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {}
}`),
			opts:     conv.ConvertOptions{PackageName: "testpkg", PackagePath: "github.com/example/proto/v1"},
			expected: "syntax = \"proto3\";\n\npackage testpkg;\n\noption go_package = \"github.com/example/proto/v1\";\n\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.Convert(test.given, test.opts)

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Empty(t, result.Golang)
			assert.NotNil(t, result.TypeMap)
			assert.Equal(t, test.expected, string(result.Protobuf))
		})
	}
}

func TestConvertParseDocument(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
		wantErr  string
	}{
		{
			name: "parse valid OpenAPI 3.0 YAML",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}

`,
			expected: "syntax = \"proto3\";\n\npackage testpkg;\n\noption go_package = \"github.com/example/proto/v1\";\n\n",
		},
		{
			name: "parse valid OpenAPI 3.0 JSON",
			given: `{
  "openapi": "3.0.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {}
}`,
			expected: "syntax = \"proto3\";\n\npackage testpkg;\n\noption go_package = \"github.com/example/proto/v1\";\n\n",
		},
		{
			name:    "non-OpenAPI document",
			given:   `title: Some Random YAML`,
			wantErr: "spec type not supported",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.Convert([]byte(test.given), conv.ConvertOptions{
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
			assert.Empty(t, result.Golang)
			assert.NotNil(t, result.TypeMap)
			assert.Equal(t, test.expected, string(result.Protobuf))
		})
	}
}

func TestConvertExtractSchemas(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "extract schemas in YAML insertion order",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
    Product:
      type: object
    Order:
      type: object
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
}

message Product {
}

message Order {
}

`,
		},
		{
			name: "document with no components section",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}

`,
			expected: "syntax = \"proto3\";\n\npackage testpkg;\n\noption go_package = \"github.com/example/proto/v1\";\n\n",
		},
		{
			name: "document with empty components/schemas",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas: {}

`,
			expected: "syntax = \"proto3\";\n\npackage testpkg;\n\noption go_package = \"github.com/example/proto/v1\";\n\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.Convert([]byte(test.given), conv.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Empty(t, result.Golang)
			assert.NotNil(t, result.TypeMap)
			assert.Equal(t, test.expected, string(result.Protobuf))
		})
	}
}

func TestConvertTypeMapValidation(t *testing.T) {
	given := `openapi: 3.0.0
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
    Product:
      type: object
      properties:
        title:
          type: string
    Order:
      type: object
      properties:
        orderId:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assertProtoOnlyTypeMap(t, result, []string{"User", "Product", "Order"})
}

func TestConvertSimpleMessage(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
		wantErr  string
	}{
		{
			name: "object schema with multiple scalar fields",
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
        email:
          type: string
        age:
          type: integer
`,
			expected: `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message User {
  string userId = 1 [json_name = "userId"];
  string email = 2 [json_name = "email"];
  int32 age = 3 [json_name = "age"];
}

`,
		},
		{
			name: "top-level primitive schema",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    SimpleString:
      type: string
`,
			wantErr: "only objects and enums supported at top level",
		},
		{
			name: "top-level array schema",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    StringList:
      type: array
      items:
        type: string
`,
			wantErr: "only objects and enums supported at top level",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.Convert([]byte(test.given), conv.ConvertOptions{
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
			assert.Empty(t, result.Golang)
			assert.NotNil(t, result.TypeMap)
			assert.Equal(t, test.expected, string(result.Protobuf))
		})
	}
}

func TestConvertFieldOrdering(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Order:
      type: object
      properties:
        orderId:
          type: string
        customerId:
          type: string
        amount:
          type: number
        status:
          type: string
`
	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Order {
  string orderId = 1 [json_name = "orderId"];
  string customerId = 2 [json_name = "customerId"];
  double amount = 3 [json_name = "amount"];
  string status = 4 [json_name = "status"];
}

`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Golang)
	assert.NotNil(t, result.TypeMap)
	assert.Equal(t, expected, string(result.Protobuf))
}

func TestConvertCompleteExample(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: E-Commerce API
  version: 1.0.0
paths: {}
components:
  schemas:
    OrderStatus:
      type: string
      description: Status of an order
      enum:
        - pending
        - confirmed
        - shipped
        - delivered
        - cancelled

    Address:
      type: object
      description: Shipping or billing address
      properties:
        street:
          type: string
          description: Street address
        city:
          type: string
        state:
          type: string
        zipCode:
          type: string
        country:
          type: string

    Product:
      type: object
      description: Product in the catalog
      properties:
        productId:
          type: string
          description: Unique product identifier
        name:
          type: string
        description:
          type: string
        price:
          type: number
          format: double
        inStock:
          type: boolean
        category:
          type: string
          enum:
            - electronics
            - clothing
            - books
            - home

    OrderItem:
      type: object
      description: Item in an order
      properties:
        product:
          $ref: '#/components/schemas/Product'
        quantity:
          type: integer
          format: int32
        unitPrice:
          type: number
          format: double

    Order:
      type: object
      description: Customer order
      properties:
        orderId:
          type: string
          description: Unique order identifier
        customerId:
          type: string
        item:
          type: array
          items:
            $ref: '#/components/schemas/OrderItem'
        status:
          $ref: '#/components/schemas/OrderStatus'
        shippingAddress:
          $ref: '#/components/schemas/Address'
        totalAmount:
          type: number
          format: double
        createdAt:
          type: string
          format: date-time
`
	expected := `syntax = "proto3";

package ecommerce;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/example/proto/v1";

// Shipping or billing address
message Address {
  // Street address
  string street = 1 [json_name = "street"];
  string city = 2 [json_name = "city"];
  string state = 3 [json_name = "state"];
  string zipCode = 4 [json_name = "zipCode"];
  string country = 5 [json_name = "country"];
}

// Product in the catalog
message Product {
  // Unique product identifier
  string productId = 1 [json_name = "productId"];
  string name = 2 [json_name = "name"];
  string description = 3 [json_name = "description"];
  double price = 4 [json_name = "price"];
  bool inStock = 5 [json_name = "inStock"];
  // enum: [electronics, clothing, books, home]
  string category = 6 [json_name = "category"];
}

// Item in an order
message OrderItem {
  Product product = 1 [json_name = "product"];
  int32 quantity = 2 [json_name = "quantity"];
  double unitPrice = 3 [json_name = "unitPrice"];
}

// Customer order
message Order {
  // Unique order identifier
  string orderId = 1 [json_name = "orderId"];
  string customerId = 2 [json_name = "customerId"];
  repeated OrderItem item = 3 [json_name = "item"];
  // Status of an order
  // enum: [pending, confirmed, shipped, delivered, cancelled]
  string status = 4 [json_name = "status"];
  Address shippingAddress = 5 [json_name = "shippingAddress"];
  double totalAmount = 6 [json_name = "totalAmount"];
  google.protobuf.Timestamp createdAt = 7 [json_name = "createdAt"];
}

`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "ecommerce",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Golang)
	assert.NotNil(t, result.TypeMap)
	assert.Equal(t, expected, string(result.Protobuf))
}

func TestOneOfWithDiscriminatorAccepted(t *testing.T) {
	given := `openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.TypeMap)
}

func TestOneOfWithoutDiscriminatorRejected(t *testing.T) {
	given := `openapi: 3.0.0
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
    Dog:
      type: object
      properties:
        bark:
          type: string
    Cat:
      type: object
      properties:
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.ErrorContains(t, err, "oneOf requires discriminator")
	require.Nil(t, result)
}

func TestOneOfWithInlineVariantRejected(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - type: object
          properties:
            bark:
              type: string
        - $ref: '#/components/schemas/Cat'
      discriminator:
        propertyName: petType
    Cat:
      type: object
      properties:
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.ErrorContains(t, err, "must use $ref")
	require.Nil(t, result)
}

func TestTypeMapClassifiesUnionTypes(t *testing.T) {
	given := `openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	info, exists := result.TypeMap["Pet"]
	require.True(t, exists)
	assert.Equal(t, conv.TypeLocationGolang, info.Location)
	assert.Equal(t, "contains oneOf", info.Reason)
}

func TestTypeMapClassifiesVariants(t *testing.T) {
	given := `openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	dogInfo, exists := result.TypeMap["Dog"]
	require.True(t, exists)
	assert.Equal(t, conv.TypeLocationGolang, dogInfo.Location)
	assert.Equal(t, "variant of union type Pet", dogInfo.Reason)

	catInfo, exists := result.TypeMap["Cat"]
	require.True(t, exists)
	assert.Equal(t, conv.TypeLocationGolang, catInfo.Location)
	assert.Equal(t, "variant of union type Pet", catInfo.Reason)
}

func TestTypeMapClassifiesReferencingTypes(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      type: object
      properties:
        name:
          type: string
        pet:
          $ref: '#/components/schemas/Pet'
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	ownerInfo, exists := result.TypeMap["Owner"]
	require.True(t, exists)
	assert.Equal(t, conv.TypeLocationGolang, ownerInfo.Location)
	assert.Equal(t, "references union type Pet", ownerInfo.Reason)
}

func TestTypeMapTransitiveClosure(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    A:
      type: object
      properties:
        b:
          $ref: '#/components/schemas/B'
    B:
      type: object
      properties:
        c:
          $ref: '#/components/schemas/C'
    C:
      oneOf:
        - $ref: '#/components/schemas/D'
        - $ref: '#/components/schemas/E'
      discriminator:
        propertyName: type
    D:
      type: object
      properties:
        type:
          type: string
        value:
          type: string
    E:
      type: object
      properties:
        type:
          type: string
        count:
          type: integer
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	// All should be golang because of transitive closure
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		info, exists := result.TypeMap[name]
		require.True(t, exists)
		assert.Equal(t, conv.TypeLocationGolang, info.Location)
	}

	// Check specific reasons
	assert.Equal(t, "contains oneOf", result.TypeMap["C"].Reason)
	assert.Equal(t, "variant of union type C", result.TypeMap["D"].Reason)
	assert.Equal(t, "variant of union type C", result.TypeMap["E"].Reason)
	assert.Equal(t, "references union type C", result.TypeMap["B"].Reason)
	assert.Equal(t, "references union type C", result.TypeMap["A"].Reason)
}

func TestOneOfBasicGeneration(t *testing.T) {
	given := `openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
		GoPackagePath: "github.com/example/types/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Proto should be empty (all types are Go-only)
	assert.Empty(t, result.Protobuf)

	// Go should be populated
	require.NotEmpty(t, result.Golang)
	goCode := string(result.Golang)

	// Check package declaration
	assert.Contains(t, goCode, "package types")

	// Check imports
	assert.Contains(t, goCode, `"encoding/json"`)
	assert.Contains(t, goCode, `"fmt"`)
	assert.Contains(t, goCode, `"strings"`)

	// Check Pet union struct with pointer fields
	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "Dog *Dog")
	assert.Contains(t, goCode, "Cat *Cat")

	// Check variant structs
	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "PetType string")
	assert.Contains(t, goCode, "Bark string")

	assert.Contains(t, goCode, "type Cat struct")
	assert.Contains(t, goCode, "Meow string")

	// Check MarshalJSON
	assert.Contains(t, goCode, "func (u *Pet) MarshalJSON() ([]byte, error)")

	// Check UnmarshalJSON
	assert.Contains(t, goCode, "func (u *Pet) UnmarshalJSON(data []byte) error")

	// Verify TypeMap
	require.NotNil(t, result.TypeMap)
	assert.Equal(t, conv.TypeLocationGolang, result.TypeMap["Pet"].Location)
	assert.Equal(t, "contains oneOf", result.TypeMap["Pet"].Reason)
	assert.Equal(t, conv.TypeLocationGolang, result.TypeMap["Dog"].Location)
	assert.Equal(t, "variant of union type Pet", result.TypeMap["Dog"].Reason)
	assert.Equal(t, conv.TypeLocationGolang, result.TypeMap["Cat"].Location)
	assert.Equal(t, "variant of union type Pet", result.TypeMap["Cat"].Reason)
}

func TestOneOfWithDiscriminatorMapping(t *testing.T) {
	given := `openapi: 3.0.0
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
        mapping:
          canine: '#/components/schemas/Dog'
          feline: '#/components/schemas/Cat'
    Dog:
      type: object
      properties:
        petType:
          type: string
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
		GoPackagePath: "github.com/example/types/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Check that UnmarshalJSON uses the mapping values
	assert.Contains(t, goCode, "case \"canine\":")
	assert.Contains(t, goCode, "case \"feline\":")
}

func TestOneOfCaseInsensitiveDiscriminator(t *testing.T) {
	given := `openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
		GoPackagePath: "github.com/example/types/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Check that UnmarshalJSON uses strings.ToLower for case-insensitive matching
	assert.Contains(t, goCode, "strings.ToLower(discriminator.PetType)")
	assert.Contains(t, goCode, "case \"dog\":")
	assert.Contains(t, goCode, "case \"cat\":")
}

func TestOneOfMultipleUnionFields(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Owner:
      type: object
      properties:
        name:
          type: string
        pet:
          $ref: '#/components/schemas/Pet'
        vehicle:
          $ref: '#/components/schemas/Vehicle'
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
    Vehicle:
      oneOf:
        - $ref: '#/components/schemas/Car'
        - $ref: '#/components/schemas/Bike'
      discriminator:
        propertyName: vehicleType
    Car:
      type: object
      properties:
        vehicleType:
          type: string
        doors:
          type: integer
    Bike:
      type: object
      properties:
        vehicleType:
          type: string
        gears:
          type: integer
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
		GoPackagePath: "github.com/example/types/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Check Owner has both union fields
	assert.Contains(t, goCode, "type Owner struct")
	assert.Contains(t, goCode, "Pet *Pet")
	assert.Contains(t, goCode, "Vehicle *Vehicle")

	// Check both union types exist
	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "type Vehicle struct")

	// Check all variants exist
	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "type Cat struct")
	assert.Contains(t, goCode, "type Car struct")
	assert.Contains(t, goCode, "type Bike struct")

	// All types should be Go-only
	for _, name := range []string{"Owner", "Pet", "Dog", "Cat", "Vehicle", "Car", "Bike"} {
		require.NotNil(t, result.TypeMap[name])
		assert.Equal(t, conv.TypeLocationGolang, result.TypeMap[name].Location)
	}
}

func TestGeneratedGoCodeCompiles(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`)

	result, err := conv.Convert(openapi, conv.ConvertOptions{
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
		GoPackagePath: "github.com/example/types",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	modContent := `module test
go 1.21
`
	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte(modContent), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "compilation failed:\n%s\nGenerated code:\n%s",
		string(output), string(result.Golang))
}

func TestOneOfJSONRoundTrip(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`)

	result, err := conv.Convert(openapi, conv.ConvertOptions{
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
		GoPackagePath: "test/types",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()

	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(typesDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"test/types"
)

func main() {
	dogJSON := []byte(` + "`" + `{"petType":"dog","bark":"woof"}` + "`" + `)
	var pet types.Pet
	if err := json.Unmarshal(dogJSON, &pet); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet.Dog == nil {
		fmt.Fprintf(os.Stderr, "expected Dog to be set\n")
		os.Exit(1)
	}
	if pet.Dog.Bark != "woof" {
		fmt.Fprintf(os.Stderr, "expected bark=woof, got %s\n", pet.Dog.Bark)
		os.Exit(1)
	}

	marshaled, err := json.Marshal(&pet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		os.Exit(1)
	}

	var original, remarshaled map[string]interface{}
	json.Unmarshal(dogJSON, &original)
	json.Unmarshal(marshaled, &remarshaled)

	if original["petType"] != remarshaled["petType"] {
		fmt.Fprintf(os.Stderr, "petType mismatch\n")
		os.Exit(1)
	}
	if original["bark"] != remarshaled["bark"] {
		fmt.Fprintf(os.Stderr, "bark mismatch\n")
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testProg), 0644)
	require.NoError(t, err)

	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "test program failed:\n%s", string(output))
	assert.Contains(t, string(output), "OK")
}

func TestOneOfVariantsMarkedAsGolang(t *testing.T) {
	given := `openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := conv.Convert([]byte(given), conv.ConvertOptions{
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
		GoPackagePath: "github.com/example/types/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	// Pet should be marked as union
	petInfo := result.TypeMap["Pet"]
	require.NotNil(t, petInfo)
	assert.Equal(t, conv.TypeLocationGolang, petInfo.Location)
	assert.Equal(t, "contains oneOf", petInfo.Reason)

	// Dog should be marked as variant
	dogInfo := result.TypeMap["Dog"]
	require.NotNil(t, dogInfo)
	assert.Equal(t, conv.TypeLocationGolang, dogInfo.Location)
	assert.Equal(t, "variant of union type Pet", dogInfo.Reason)

	// Cat should be marked as variant
	catInfo := result.TypeMap["Cat"]
	require.NotNil(t, catInfo)
	assert.Equal(t, conv.TypeLocationGolang, catInfo.Location)
	assert.Equal(t, "variant of union type Pet", catInfo.Reason)
}

func TestConvertToStructBasics(t *testing.T) {
	for _, test := range []struct {
		name    string
		given   []byte
		opts    conv.ConvertOptions
		wantErr string
	}{
		{
			name:    "empty openapi bytes",
			given:   []byte{},
			opts:    conv.ConvertOptions{GoPackagePath: "github.com/example/models"},
			wantErr: "openapi input cannot be empty",
		},
		{
			name:    "empty GoPackagePath",
			given:   []byte("openapi: 3.0.0"),
			opts:    conv.ConvertOptions{},
			wantErr: "GoPackagePath cannot be empty",
		},
		{
			name:    "invalid YAML syntax",
			given:   []byte("this is not valid: [yaml"),
			opts:    conv.ConvertOptions{GoPackagePath: "github.com/example/models"},
			wantErr: "failed to parse OpenAPI document",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToStruct(test.given, test.opts)

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestConvertToStructSimpleSchema(t *testing.T) {
	input := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        age:
          type: integer
`)

	result, err := conv.ConvertToStruct(input, conv.ConvertOptions{
		GoPackagePath: "github.com/example/models",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)
	require.NotNil(t, result.TypeMap)

	goCode := string(result.Golang)
	assert.Contains(t, goCode, "package models")
	assert.Contains(t, goCode, "type User struct")
	assert.Contains(t, goCode, "Id")
	assert.Contains(t, goCode, "Name")
	assert.Contains(t, goCode, "Age")
	assert.Contains(t, goCode, "json:\"id\"")
	assert.Contains(t, goCode, "json:\"name\"")
	assert.Contains(t, goCode, "json:\"age\"")

	userInfo := result.TypeMap["User"]
	require.NotNil(t, userInfo)
	assert.Equal(t, conv.TypeLocationGolang, userInfo.Location)
	assert.Empty(t, userInfo.Reason)
}

func TestConvertToStructWithUnion(t *testing.T) {
	input := []byte(`openapi: 3.0.0
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
        propertyName: type
        mapping:
          dog: '#/components/schemas/Dog'
          cat: '#/components/schemas/Cat'
    Dog:
      type: object
      properties:
        type:
          type: string
        breed:
          type: string
    Cat:
      type: object
      properties:
        type:
          type: string
        color:
          type: string
    Product:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
`)

	result, err := conv.ConvertToStruct(input, conv.ConvertOptions{
		GoPackagePath: "github.com/example/models/v1",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)
	require.NotNil(t, result.TypeMap)

	goCode := string(result.Golang)
	assert.Contains(t, goCode, "package models")
	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "type Cat struct")
	assert.Contains(t, goCode, "type Product struct")

	assert.Contains(t, goCode, "func (u *Pet) MarshalJSON()")
	assert.Contains(t, goCode, "func (u *Pet) UnmarshalJSON(")

	require.Len(t, result.TypeMap, 4)

	petInfo := result.TypeMap["Pet"]
	require.NotNil(t, petInfo)
	assert.Equal(t, conv.TypeLocationGolang, petInfo.Location)
	assert.Equal(t, "contains oneOf", petInfo.Reason)

	dogInfo := result.TypeMap["Dog"]
	require.NotNil(t, dogInfo)
	assert.Equal(t, conv.TypeLocationGolang, dogInfo.Location)
	assert.Equal(t, "variant of union type Pet", dogInfo.Reason)

	catInfo := result.TypeMap["Cat"]
	require.NotNil(t, catInfo)
	assert.Equal(t, conv.TypeLocationGolang, catInfo.Location)
	assert.Equal(t, "variant of union type Pet", catInfo.Reason)

	productInfo := result.TypeMap["Product"]
	require.NotNil(t, productInfo)
	assert.Equal(t, conv.TypeLocationGolang, productInfo.Location)
	assert.Empty(t, productInfo.Reason)
}
