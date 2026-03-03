package schema_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToExamplesValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		openapi []byte
		opts    schema.ExampleOptions
		wantErr string
	}{
		{
			name:    "empty openapi bytes",
			openapi: []byte{},
			opts:    schema.ExampleOptions{IncludeAll: true},
			wantErr: "openapi input cannot be empty",
		},
		{
			name: "no schema names and no include all",
			openapi: []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`),
			opts:    schema.ExampleOptions{},
			wantErr: "must specify SchemaNames or set IncludeAll",
		},
		{
			name:    "invalid openapi document",
			openapi: []byte(`this is not valid: [yaml`),
			opts:    schema.ExampleOptions{IncludeAll: true},
			wantErr: "failed to parse OpenAPI document",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.ConvertToExamples(test.openapi, test.opts)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestConvertToExamplesScalarTypes(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "simple string field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
`,
			schema:   "User",
			expected: `{"name":"dl2INvNSQT"}`,
		},
		{
			name: "integer field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Product:
      type: object
      properties:
        quantity:
          type: integer
`,
			schema:   "Product",
			expected: `{"quantity":6}`,
		},
		{
			name: "boolean field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Settings:
      type: object
      properties:
        enabled:
          type: boolean
`,
			schema:   "Settings",
			expected: `{"enabled":true}`,
		},
		{
			name: "number field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Price:
      type: object
      properties:
        amount:
          type: number
`,
			schema:   "Price",
			expected: `{"amount":37.92980774361663}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesConstraints(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "integer with min and max",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Product:
      type: object
      properties:
        quantity:
          type: integer
          minimum: 10
          maximum: 50
`,
			schema:   "Product",
			expected: `{"quantity":47}`,
		},
		{
			name: "number with min and max",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Price:
      type: object
      properties:
        amount:
          type: number
          minimum: 1.5
          maximum: 99.99
`,
			schema:   "Price",
			expected: `{"amount":38.239563279482844}`,
		},
		{
			name: "default value used",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Settings:
      type: object
      properties:
        timeout:
          type: integer
          default: 30
`,
			schema:   "Settings",
			expected: `{"timeout":30}`,
		},
		{
			name: "example value used",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
          example: "John Doe"
`,
			schema:   "User",
			expected: `{"name":"John Doe"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesEnums(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Status:
      type: object
      properties:
        state:
          type: string
          enum:
            - pending
            - active
            - completed
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"Status"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "Status")
	assert.JSONEq(t, `{"state":"pending"}`, string(result.Examples["Status"]))
}

func TestConvertToExamplesDeterministic(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Product:
      type: object
      properties:
        quantity:
          type: integer
          minimum: 1
          maximum: 100
        price:
          type: number
          minimum: 0.01
          maximum: 999.99
        active:
          type: boolean
`

	const seed = int64(12345)

	result1, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"Product"},
		Seed:        seed,
	})
	require.NoError(t, err)

	result2, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"Product"},
		Seed:        seed,
	})
	require.NoError(t, err)

	assert.Equal(t, result1.Examples["Product"], result2.Examples["Product"])
}

func TestConvertToExamplesIncludeAll(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
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
        id:
          type: integer
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		IncludeAll: true,
		Seed:       42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.Examples, 3)
	assert.Contains(t, result.Examples, "User")
	assert.Contains(t, result.Examples, "Product")
	assert.Contains(t, result.Examples, "Order")
}

func TestConvertToExamplesSchemaFiltering(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
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
        id:
          type: integer
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"User", "Product"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.Examples, 2)
	assert.Contains(t, result.Examples, "User")
	assert.Contains(t, result.Examples, "Product")
	assert.NotContains(t, result.Examples, "Order")
}

func TestConvertToExamplesDefaultMaxDepth(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"User"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Examples, "User")
}

func TestConvertToExamplesObjects(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "simple object with scalar properties",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
        active:
          type: boolean
`,
			schema:   "User",
			expected: `{"active":true,"age":30,"name":"dl2INvNSQT"}`,
		},
		{
			name: "object with mixed types",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Product:
      type: object
      properties:
        title:
          type: string
        price:
          type: number
        quantity:
          type: integer
        inStock:
          type: boolean
`,
			schema:   "Product",
			expected: `{"inStock":true,"price":73.8273024155778,"quantity":68,"title":"dl2INvNSQT"}`,
		},
		{
			name: "empty object",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Empty:
      type: object
`,
			schema:   "Empty",
			expected: `{}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesNestedObjects(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "object with nested object property",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        address:
          type: object
          properties:
            street:
              type: string
            city:
              type: string
            zipCode:
              type: integer
`,
			schema:   "User",
			expected: `{"address":{"city":"GyAVmNkB33","street":"Z5zQu9MxNm","zipCode":83},"name":"dl2INvNSQT"}`,
		},
		{
			name: "deeply nested objects",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Company:
      type: object
      properties:
        name:
          type: string
        headquarters:
          type: object
          properties:
            address:
              type: object
              properties:
                street:
                  type: string
                location:
                  type: object
                  properties:
                    lat:
                      type: number
                    lng:
                      type: number
`,
			schema:   "Company",
			expected: `{"headquarters":{"address":{"location":{"lat":12.813847879609565,"lng":34.67652672737327},"street":"Z5zQu9MxNm"}},"name":"dl2INvNSQT"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesDepthLimit(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Level1:
      type: object
      properties:
        name:
          type: string
        level2:
          type: object
          properties:
            name:
              type: string
            level3:
              type: object
              properties:
                name:
                  type: string
                level4:
                  type: object
                  properties:
                    name:
                      type: string
                    level5:
                      type: object
                      properties:
                        name:
                          type: string
                        level6:
                          type: object
                          properties:
                            name:
                              type: string
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"Level1"},
		MaxDepth:    3,
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.JSONEq(t, `{"level2":{"level3":{"name":"GyAVmNkB33"},"name":"Z5zQu9MxNm"},"name":"dl2INvNSQT"}`, string(result.Examples["Level1"]))
}

func TestConvertToExamplesArrays(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "array with scalar items",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    TagList:
      type: object
      properties:
        tags:
          type: array
          items:
            type: string
`,
			schema:   "TagList",
			expected: `{"tags":["dl2INvNSQT"]}`,
		},
		{
			name: "array with integer items",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Numbers:
      type: object
      properties:
        values:
          type: array
          items:
            type: integer
`,
			schema:   "Numbers",
			expected: `{"values":[6]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesArrayConstraints(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "array with minItems",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    TagList:
      type: object
      properties:
        tags:
          type: array
          minItems: 3
          items:
            type: string
`,
			schema:   "TagList",
			expected: `{"tags":["dl2INvNSQT","Z5zQu9MxNm","GyAVmNkB33"]}`,
		},
		{
			name: "array with maxItems",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Limited:
      type: object
      properties:
        items:
          type: array
          minItems: 5
          maxItems: 5
          items:
            type: integer
`,
			schema:   "Limited",
			expected: `{"items":[6,88,69,51,24]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesArrayOfObjects(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    UserList:
      type: object
      properties:
        users:
          type: array
          minItems: 2
          items:
            type: object
            properties:
              name:
                type: string
              age:
                type: integer
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"UserList"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.JSONEq(t, `{"users":[{"age":30,"name":"dl2INvNSQT"},{"age":35,"name":"5zQu9MxNmG"}]}`, string(result.Examples["UserList"]))
}

func TestConvertToExamplesInvalidArraySchema(t *testing.T) {
	for _, test := range []struct {
		name       string
		openapi    string
		schemaName string
	}{
		{
			name: "array without items",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    BadArray:
      type: object
      properties:
        items:
          type: array
`,
			schemaName: "BadArray",
		},
		{
			name: "array with minItems greater than maxItems",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    InvalidArray:
      type: object
      properties:
        items:
          type: array
          minItems: 10
          maxItems: 5
          items:
            type: string
`,
			schemaName: "InvalidArray",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schemaName},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Empty(t, result.Examples)
		})
	}
}

func TestConvertToExamplesReferences(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "simple reference",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
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
          type: string
        address:
          $ref: '#/components/schemas/Address'
`,
			schema:   "User",
			expected: `{"address":{"city":"GyAVmNkB33","street":"Z5zQu9MxNm"},"name":"dl2INvNSQT"}`,
		},
		{
			name: "nested references",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    City:
      type: object
      properties:
        name:
          type: string
        zipCode:
          type: integer
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          $ref: '#/components/schemas/City'
    User:
      type: object
      properties:
        name:
          type: string
        address:
          $ref: '#/components/schemas/Address'
`,
			schema:   "User",
			expected: `{"address":{"city":{"name":"GyAVmNkB33","zipCode":83},"street":"Z5zQu9MxNm"},"name":"dl2INvNSQT"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesCircularReferences(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "direct circular reference",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Node:
      type: object
      properties:
        value:
          type: integer
        next:
          $ref: '#/components/schemas/Node'
`,
			schema:   "Node",
			expected: `{"value":6}`,
		},
		{
			name: "indirect circular reference",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        address:
          $ref: '#/components/schemas/Address'
    Address:
      type: object
      properties:
        street:
          type: string
        owner:
          $ref: '#/components/schemas/User'
`,
			schema:   "User",
			expected: `{"address":{"street":"Z5zQu9MxNm"},"name":"dl2INvNSQT"}`,
		},
		{
			name: "three-way circular reference",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    A:
      type: object
      properties:
        name:
          type: string
        b:
          $ref: '#/components/schemas/B'
    B:
      type: object
      properties:
        value:
          type: integer
        c:
          $ref: '#/components/schemas/C'
    C:
      type: object
      properties:
        flag:
          type: boolean
        a:
          $ref: '#/components/schemas/A'
`,
			schema:   "A",
			expected: `{"b":{"c":{"flag":true},"value":30},"name":"dl2INvNSQT"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesFieldOverrides(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    ErrorResponse:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames:    []string{"ErrorResponse"},
		Seed:           42,
		FieldOverrides: map[string]interface{}{"code": 400},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "ErrorResponse")
	assert.JSONEq(t, `{"code":400,"message":"This is a message"}`, string(result.Examples["ErrorResponse"]))
}

func TestConvertToExamplesRandomDefaults(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "integer without constraints generates random 1-100",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Product:
      type: object
      properties:
        quantity:
          type: integer
`,
			schema:   "Product",
			expected: `{"quantity":6}`,
		},
		{
			name: "number without constraints generates random 1.0-100.0",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Price:
      type: object
      properties:
        amount:
          type: number
`,
			schema:   "Price",
			expected: `{"amount":37.92980774361663}`,
		},
		{
			name: "deterministic with fixed seed",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Data:
      type: object
      properties:
        count:
          type: integer
        value:
          type: number
`,
			schema:   "Data",
			expected: `{"count":6,"value":7.534049182558273}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesCursorHeuristics(t *testing.T) {
	for _, test := range []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "cursor field lowercase",
			fieldName: "cursor",
			expected:  `{"cursor":"le+FHLiWt5VNCmTe5VqQw"}`,
		},
		{
			name:      "first field lowercase",
			fieldName: "first",
			expected:  `{"first":"le+FHLiWt5VNCmTe5VqQw"}`,
		},
		{
			name:      "after field lowercase",
			fieldName: "after",
			expected:  `{"after":"le+FHLiWt5VNCmTe5VqQw"}`,
		},
		{
			name:      "Cursor field capitalized",
			fieldName: "Cursor",
			expected:  `{"Cursor":"le+FHLiWt5VNCmTe5VqQw"}`,
		},
		{
			name:      "other field does not match",
			fieldName: "other",
			expected:  `{"other":"dl2INvNSQT"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pagination:
      type: object
      properties:
        ` + test.fieldName + `:
          type: string
`
			result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
				SchemaNames: []string{"Pagination"},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, "Pagination")
			assert.JSONEq(t, test.expected, string(result.Examples["Pagination"]))
		})
	}
}

func TestConvertToExamplesMessageHeuristics(t *testing.T) {
	for _, test := range []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "error field lowercase",
			fieldName: "error",
			expected:  `{"error":"An error occurred"}`,
		},
		{
			name:      "message field lowercase",
			fieldName: "message",
			expected:  `{"message":"This is a message"}`,
		},
		{
			name:      "Error field capitalized",
			fieldName: "Error",
			expected:  `{"Error":"An error occurred"}`,
		},
		{
			name:      "description field does not match",
			fieldName: "description",
			expected:  `{"description":"dl2INvNSQT"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Response:
      type: object
      properties:
        ` + test.fieldName + `:
          type: string
`
			result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
				SchemaNames: []string{"Response"},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, "Response")
			assert.JSONEq(t, test.expected, string(result.Examples["Response"]))
		})
	}
}

func TestConvertToExamplesFieldOverridePriority(t *testing.T) {
	for _, test := range []struct {
		name      string
		openapi   string
		overrides map[string]interface{}
		expected  string
	}{
		{
			name: "override takes precedence over heuristics",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Response:
      type: object
      properties:
        message:
          type: string
`,
			overrides: map[string]interface{}{"message": "custom message"},
			expected:  `{"message":"custom message"}`,
		},
		{
			name: "schema example takes precedence over override",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Response:
      type: object
      properties:
        code:
          type: integer
          example: 200
`,
			overrides: map[string]interface{}{"code": 400},
			expected:  `{"code":200}`,
		},
		{
			name: "schema default takes precedence over override",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        enabled:
          type: boolean
          default: false
`,
			overrides: map[string]interface{}{"enabled": true},
			expected:  `{"enabled":false}`,
		},
		{
			name: "multiple field overrides",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    ErrorResponse:
      type: object
      properties:
        code:
          type: integer
        status:
          type: string
        message:
          type: string
`,
			overrides: map[string]interface{}{
				"code":    500,
				"status":  "error",
				"message": "Internal server error",
			},
			expected: `{"code":500,"message":"Internal server error","status":"error"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				FieldOverrides: test.overrides,
				IncludeAll:     true,
				Seed:           42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)

			var schemaName string
			for name := range result.Examples {
				schemaName = name
				break
			}
			require.NotEmpty(t, schemaName)
			assert.JSONEq(t, test.expected, string(result.Examples[schemaName]))
		})
	}
}

func TestConvertToExamplesFieldOverrideTypeMismatch(t *testing.T) {
	for _, test := range []struct {
		name       string
		openapi    string
		overrides  map[string]interface{}
		schemaName string
	}{
		{
			name: "string value for integer field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Response:
      type: object
      properties:
        code:
          type: integer
`,
			overrides:  map[string]interface{}{"code": "not a number"},
			schemaName: "Response",
		},
		{
			name: "integer value for string field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Response:
      type: object
      properties:
        message:
          type: string
`,
			overrides:  map[string]interface{}{"message": 123},
			schemaName: "Response",
		},
		{
			name: "string value for boolean field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Config:
      type: object
      properties:
        enabled:
          type: boolean
`,
			overrides:  map[string]interface{}{"enabled": "true"},
			schemaName: "Config",
		},
		{
			name: "float with decimal for integer field",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Data:
      type: object
      properties:
        count:
          type: integer
`,
			overrides:  map[string]interface{}{"count": 42.5},
			schemaName: "Data",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				FieldOverrides: test.overrides,
				IncludeAll:     true,
				Seed:           42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotContains(t, result.Examples, test.schemaName)
		})
	}
}

func TestConvertToExamplesSchemaLevelExample(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "schema with complete example object",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Transfer:
      type: object
      example:
        id: "xfer_complete_example"
        amount: 100
        status: "completed"
      properties:
        id:
          type: string
        amount:
          type: integer
        status:
          type: string
`,
			schema:   "Transfer",
			expected: `{"amount":100,"id":"xfer_complete_example","status":"completed"}`,
		},
		{
			name: "schema-level example with nested object",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      example:
        name: "John Doe"
        address:
          street: "123 Main St"
          city: "Springfield"
      properties:
        name:
          type: string
        address:
          type: object
          properties:
            street:
              type: string
            city:
              type: string
`,
			schema:   "User",
			expected: `{"address":{"city":"Springfield","street":"123 Main St"},"name":"John Doe"}`,
		},
		{
			name: "schema-level example with array",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Order:
      type: object
      example:
        orderId: "ord_123"
        items:
          - "item1"
          - "item2"
          - "item3"
      properties:
        orderId:
          type: string
        items:
          type: array
          items:
            type: string
`,
			schema:   "Order",
			expected: `{"items":["item1","item2","item3"],"orderId":"ord_123"}`,
		},
		{
			name: "schema-level example takes precedence over property examples",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Product:
      type: object
      example:
        name: "Schema Level Name"
        price: 999
      properties:
        name:
          type: string
          example: "Property Level Name"
        price:
          type: integer
          example: 100
`,
			schema:   "Product",
			expected: `{"name":"Schema Level Name","price":999}`,
		},
		{
			name: "schema-level example with null value",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    NullableData:
      type: object
      example: null
      properties:
        value:
          type: string
`,
			schema:   "NullableData",
			expected: "null",
		},
		{
			name: "schema-level example with boolean values",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Flags:
      type: object
      example:
        enabled: true
        disabled: false
      properties:
        enabled:
          type: boolean
        disabled:
          type: boolean
`,
			schema:   "Flags",
			expected: `{"disabled":false,"enabled":true}`,
		},
		{
			name: "schema-level example with float values",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Measurement:
      type: object
      example:
        temperature: 98.6
        pressure: 1013.25
      properties:
        temperature:
          type: number
        pressure:
          type: number
`,
			schema:   "Measurement",
			expected: `{"pressure":1013.25,"temperature":98.6}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			if test.expected == "null" {
				assert.Equal(t, "null", string(result.Examples[test.schema]))
			} else {
				assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
			}
		})
	}
}

func TestConvertToExamplesSchemaLevelExamplesArray(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "schema with examples array uses first entry",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Payment:
      type: object
      examples:
        - id: "pay_first_example"
          amount: 50
        - id: "pay_second_example"
          amount: 100
      properties:
        id:
          type: string
        amount:
          type: integer
`,
			schema:   "Payment",
			expected: `{"amount":50,"id":"pay_first_example"}`,
		},
		{
			name: "example singular takes precedence over examples array",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Data:
      type: object
      example:
        value: "from singular example"
      examples:
        - value: "from examples array"
      properties:
        value:
          type: string
`,
			schema:   "Data",
			expected: `{"value":"from singular example"}`,
		},
		{
			name: "empty examples array falls back to generation",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Generated:
      type: object
      examples: []
      properties:
        name:
          type: string
`,
			schema:   "Generated",
			expected: `{"name":"dl2INvNSQT"}`,
		},
		{
			name: "examples array with nested objects",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    ComplexData:
      type: object
      examples:
        - user:
            name: "Alice"
            age: 30
          tags:
            - "premium"
            - "verified"
      properties:
        user:
          type: object
          properties:
            name:
              type: string
            age:
              type: integer
        tags:
          type: array
          items:
            type: string
`,
			schema:   "ComplexData",
			expected: `{"tags":["premium","verified"],"user":{"age":30,"name":"Alice"}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesPropertyLevelExamples(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "property with examples array uses first entry",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Payment:
      type: object
      properties:
        transactionId:
          type: string
          examples:
            - "txn_first_example"
            - "txn_second_example"
        amount:
          type: integer
`,
			schema:   "Payment",
			expected: `{"amount":6,"transactionId":"txn_first_example"}`,
		},
		{
			name: "property example takes precedence over examples array",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Data:
      type: object
      properties:
        value:
          type: string
          example: "from singular example"
          examples:
            - "from examples array"
`,
			schema:   "Data",
			expected: `{"value":"from singular example"}`,
		},
		{
			name: "property with empty examples array falls back to generation",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Generated:
      type: object
      properties:
        name:
          type: string
          examples: []
`,
			schema:   "Generated",
			expected: `{"name":"dl2INvNSQT"}`,
		},
		{
			name: "multiple properties with examples",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    MultipleExamples:
      type: object
      properties:
        id:
          type: string
          example: "id_from_example"
        code:
          type: string
          examples:
            - "code_from_examples"
        generated:
          type: string
`,
			schema:   "MultipleExamples",
			expected: `{"code":"code_from_examples","generated":"dl2INvNSQT","id":"id_from_example"}`,
		},
		{
			name: "integer property with examples array",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    NumericData:
      type: object
      properties:
        count:
          type: integer
          examples:
            - 42
            - 100
`,
			schema:   "NumericData",
			expected: `{"count":42}`,
		},
		{
			name: "boolean property with examples array",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Flags:
      type: object
      properties:
        enabled:
          type: boolean
          examples:
            - true
            - false
`,
			schema:   "Flags",
			expected: `{"enabled":true}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesPropertyLevelExampleObject(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "object property with example uses that example",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        address:
          type: object
          example:
            street: "123 Custom St"
            city: "Example City"
            zip: 12345
          properties:
            street:
              type: string
            city:
              type: string
            zip:
              type: integer
`,
			schema:   "User",
			expected: `{"address":{"city":"Example City","street":"123 Custom St","zip":12345},"name":"dl2INvNSQT"}`,
		},
		{
			name: "array property with example uses that example",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    TagList:
      type: object
      properties:
        tags:
          type: array
          example:
            - "custom-tag-1"
            - "custom-tag-2"
            - "custom-tag-3"
          items:
            type: string
`,
			schema:   "TagList",
			expected: `{"tags":["custom-tag-1","custom-tag-2","custom-tag-3"]}`,
		},
		{
			name: "object property with examples array uses first entry",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Order:
      type: object
      properties:
        metadata:
          type: object
          examples:
            - key: "first_key"
              value: "first_value"
            - key: "second_key"
              value: "second_value"
          properties:
            key:
              type: string
            value:
              type: string
`,
			schema:   "Order",
			expected: `{"metadata":{"key":"first_key","value":"first_value"}}`,
		},
		{
			name: "array property with examples array uses first entry",
			openapi: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    ItemList:
      type: object
      properties:
        items:
          type: array
          examples:
            - - "first-array-item-1"
              - "first-array-item-2"
            - - "second-array-item-1"
          items:
            type: string
`,
			schema:   "ItemList",
			expected: `{"items":["first-array-item-1","first-array-item-2"]}`,
		},
		{
			name: "nested object with complex example",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Order:
      type: object
      properties:
        customer:
          type: object
          example:
            name: "John Doe"
            addresses:
              - street: "123 Main St"
                city: "NYC"
              - street: "456 Oak Ave"
                city: "LA"
          properties:
            name:
              type: string
            addresses:
              type: array
              items:
                type: object
                properties:
                  street:
                    type: string
                  city:
                    type: string
`,
			schema:   "Order",
			expected: `{"customer":{"addresses":[{"city":"NYC","street":"123 Main St"},{"city":"LA","street":"456 Oak Ave"}],"name":"John Doe"}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesAllOf(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "allOf with two ref entries merges properties from both",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Name:
      type: object
      properties:
        first_name:
          type: string
        last_name:
          type: string
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          type: string
    Person:
      allOf:
        - $ref: '#/components/schemas/Name'
        - $ref: '#/components/schemas/Address'
`,
			schema:   "Person",
			expected: `{"city":"ionwj2qrsh","first_name":"dl2INvNSQT","last_name":"Z5zQu9MxNm","street":"GyAVmNkB33"}`,
		},
		{
			name: "allOf with inline schema entries merges properties",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Combined:
      allOf:
        - type: object
          properties:
            name:
              type: string
        - type: object
          properties:
            age:
              type: integer
`,
			schema:   "Combined",
			expected: `{"age":30,"name":"dl2INvNSQT"}`,
		},
		{
			name: "allOf with ref plus inline schema merges both",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: integer
    Extended:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            label:
              type: string
`,
			schema:   "Extended",
			expected: `{"id":6,"label":"l2INvNSQTZ"}`,
		},
		{
			name: "allOf with overlapping property names uses later entry",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Overlap:
      allOf:
        - type: object
          properties:
            name:
              type: string
              example: "first"
            code:
              type: integer
        - type: object
          properties:
            name:
              type: string
              example: "second"
            label:
              type: string
`,
			schema:   "Overlap",
			expected: `{"code":6,"label":"l2INvNSQTZ","name":"second"}`,
		},
		{
			name: "nested allOf produces correct merged output",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Alpha:
      type: object
      properties:
        alpha_field:
          type: string
    Beta:
      allOf:
        - $ref: '#/components/schemas/Alpha'
        - type: object
          properties:
            beta_field:
              type: integer
    Gamma:
      allOf:
        - $ref: '#/components/schemas/Beta'
        - type: object
          properties:
            gamma_field:
              type: boolean
`,
			schema:   "Gamma",
			expected: `{"alpha_field":"dl2INvNSQT","beta_field":30,"gamma_field":true}`,
		},
		{
			name: "allOf without type field does not error",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    NoType:
      allOf:
        - type: object
          properties:
            value:
              type: string
`,
			schema:   "NoType",
			expected: `{"value":"dl2INvNSQT"}`,
		},
		{
			name: "allOf with sibling properties merges both",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: integer
    WithSiblings:
      properties:
        sibling_field:
          type: string
      allOf:
        - $ref: '#/components/schemas/Base'
`,
			schema:   "WithSiblings",
			expected: `{"id":6,"sibling_field":"l2INvNSQTZ"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesAllOfAlongsideOtherSchemas(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Simple:
      type: object
      properties:
        name:
          type: string
    Base:
      type: object
      properties:
        id:
          type: integer
    Composed:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            extra:
              type: string
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"Simple", "Composed"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.Examples, "Simple")
	require.Contains(t, result.Examples, "Composed")

	assert.JSONEq(t, `{"name":"dl2INvNSQT"}`, string(result.Examples["Simple"]))
	assert.JSONEq(t, `{"extra":"5zQu9MxNmG","id":30}`, string(result.Examples["Composed"]))
}

func TestConvertToExamplesOneOf(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "oneOf with ref variants picks first variant",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Cat:
      type: object
      properties:
        purrs:
          type: boolean
    Dog:
      type: object
      properties:
        barks:
          type: boolean
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
`,
			schema:   "Pet",
			expected: `{"purrs":true}`,
		},
		{
			name: "oneOf without type field does not error",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Variant:
      oneOf:
        - type: object
          properties:
            name:
              type: string
        - type: object
          properties:
            code:
              type: integer
`,
			schema:   "Variant",
			expected: `{"name":"dl2INvNSQT"}`,
		},
		{
			name: "oneOf with inline schemas picks first variant",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    InlineVariant:
      oneOf:
        - type: object
          properties:
            alpha:
              type: string
        - type: object
          properties:
            beta:
              type: integer
`,
			schema:   "InlineVariant",
			expected: `{"alpha":"dl2INvNSQT"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesOneOfWithDiscriminator(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "discriminator sets property to schema name",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Cat:
      type: object
      properties:
        purrs:
          type: boolean
    Dog:
      type: object
      properties:
        barks:
          type: boolean
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
      discriminator:
        propertyName: petType
`,
			schema:   "Pet",
			expected: `{"petType":"Cat","purrs":true}`,
		},
		{
			name: "discriminator with mapping uses mapping key",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    SftpRequest:
      type: object
      properties:
        host:
          type: string
        port:
          type: integer
    HttpRequest:
      type: object
      properties:
        url:
          type: string
    DeliveryRequest:
      oneOf:
        - $ref: '#/components/schemas/SftpRequest'
        - $ref: '#/components/schemas/HttpRequest'
      discriminator:
        propertyName: type
        mapping:
          sftp: '#/components/schemas/SftpRequest'
          http: '#/components/schemas/HttpRequest'
`,
			schema:   "DeliveryRequest",
			expected: `{"host":"dl2INvNSQT","port":30,"type":"sftp"}`,
		},
		{
			name: "discriminator without mapping falls back to schema name",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Circle:
      type: object
      properties:
        radius:
          type: number
    Square:
      type: object
      properties:
        side:
          type: number
    Shape:
      oneOf:
        - $ref: '#/components/schemas/Circle'
        - $ref: '#/components/schemas/Square'
      discriminator:
        propertyName: shapeType
`,
			schema:   "Shape",
			expected: `{"radius":37.92980774361663,"shapeType":"Circle"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesAnyOf(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "anyOf picks first variant",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    StringOrInt:
      anyOf:
        - type: object
          properties:
            text:
              type: string
        - type: object
          properties:
            number:
              type: integer
`,
			schema:   "StringOrInt",
			expected: `{"text":"dl2INvNSQT"}`,
		},
		{
			name: "anyOf with ref variants picks first",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Email:
      type: object
      properties:
        address:
          type: string
          format: email
    Phone:
      type: object
      properties:
        number:
          type: string
    ContactInfo:
      anyOf:
        - $ref: '#/components/schemas/Email'
        - $ref: '#/components/schemas/Phone'
`,
			schema:   "ContactInfo",
			expected: `{"address":"user@example.com"}`,
		},
		{
			name: "anyOf with discriminator sets property correctly",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    AdminUser:
      type: object
      properties:
        permissions:
          type: string
    RegularUser:
      type: object
      properties:
        plan:
          type: string
    AnyUser:
      anyOf:
        - $ref: '#/components/schemas/AdminUser'
        - $ref: '#/components/schemas/RegularUser'
      discriminator:
        propertyName: role
        mapping:
          admin: '#/components/schemas/AdminUser'
          regular: '#/components/schemas/RegularUser'
`,
			schema:   "AnyUser",
			expected: `{"permissions":"dl2INvNSQT","role":"admin"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesCompositionWithSiblingProperties(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "object with properties and oneOf merges both",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    SftpRequest:
      type: object
      properties:
        host:
          type: string
        port:
          type: integer
    HttpRequest:
      type: object
      properties:
        url:
          type: string
    DeliveryCreateRequest:
      type: object
      properties:
        name:
          type: string
      oneOf:
        - $ref: '#/components/schemas/SftpRequest'
        - $ref: '#/components/schemas/HttpRequest'
`,
			schema:   "DeliveryCreateRequest",
			expected: `{"host":"Z5zQu9MxNm","name":"dl2INvNSQT","port":83}`,
		},
		{
			name: "sibling properties take precedence over composition properties",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        name:
          type: string
          example: "from-base"
        code:
          type: integer
    Override:
      type: object
      properties:
        name:
          type: string
          example: "from-sibling"
      allOf:
        - $ref: '#/components/schemas/Base'
`,
			schema:   "Override",
			expected: `{"code":6,"name":"from-sibling"}`,
		},
		{
			name: "object with properties and allOf merges both",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Timestamps:
      type: object
      properties:
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    Resource:
      type: object
      properties:
        id:
          type: string
          format: uuid
      allOf:
        - $ref: '#/components/schemas/Timestamps'
`,
			schema:   "Resource",
			expected: `{"created_at":"2024-01-15T10:30:00Z","id":"123e4567-e89b-12d3-a456-426614174000","updated_at":"2024-01-15T10:30:00Z"}`,
		},
		{
			name: "object with properties and anyOf merges both",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    EmailContact:
      type: object
      properties:
        email:
          type: string
          format: email
    PhoneContact:
      type: object
      properties:
        phone:
          type: string
    Person:
      type: object
      properties:
        name:
          type: string
      anyOf:
        - $ref: '#/components/schemas/EmailContact'
        - $ref: '#/components/schemas/PhoneContact'
`,
			schema:   "Person",
			expected: `{"email":"user@example.com","name":"dl2INvNSQT"}`,
		},
		{
			name: "discriminator value set correctly with sibling properties",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    SftpRequest:
      type: object
      properties:
        host:
          type: string
        port:
          type: integer
    HttpRequest:
      type: object
      properties:
        url:
          type: string
    DeliveryCreateRequest:
      type: object
      properties:
        name:
          type: string
      oneOf:
        - $ref: '#/components/schemas/SftpRequest'
        - $ref: '#/components/schemas/HttpRequest'
      discriminator:
        propertyName: type
        mapping:
          sftp: '#/components/schemas/SftpRequest'
          http: '#/components/schemas/HttpRequest'
`,
			schema:   "DeliveryCreateRequest",
			expected: `{"host":"Z5zQu9MxNm","name":"dl2INvNSQT","port":83,"type":"sftp"}`,
		},
		{
			name: "nested object where property uses composition",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Name:
      type: object
      properties:
        first:
          type: string
        last:
          type: string
    Age:
      type: object
      properties:
        years:
          type: integer
    Wrapper:
      type: object
      properties:
        person:
          allOf:
            - $ref: '#/components/schemas/Name'
            - $ref: '#/components/schemas/Age'
`,
			schema:   "Wrapper",
			expected: `{"person":{"first":"le+FHLiWt5VNCmTe5VqQw","last":"AVmNkB33io","years":16}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}

func TestConvertToExamplesOneOfAlongsideSimpleSchemas(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    SimpleSchema:
      type: object
      properties:
        name:
          type: string
    Cat:
      type: object
      properties:
        purrs:
          type: boolean
    Dog:
      type: object
      properties:
        barks:
          type: boolean
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
`

	result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
		SchemaNames: []string{"SimpleSchema", "Pet"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.Examples, "SimpleSchema")
	require.Contains(t, result.Examples, "Pet")

	assert.JSONEq(t, `{"name":"dl2INvNSQT"}`, string(result.Examples["SimpleSchema"]))
	assert.JSONEq(t, `{"purrs":true}`, string(result.Examples["Pet"]))
}

func TestConvertToExamplesErrorIsolation(t *testing.T) {
	t.Run("valid schema alongside erroring schema", func(t *testing.T) {
		openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    ValidSchema:
      type: object
      properties:
        name:
          type: string
    ErrorSchema:
      type: object
      properties:
        items:
          type: array
`

		result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
			IncludeAll: true,
			Seed:       42,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.NotContains(t, result.Examples, "ErrorSchema")

		require.Contains(t, result.Examples, "ValidSchema")
		assert.JSONEq(t, `{"name":"dl2INvNSQT"}`, string(result.Examples["ValidSchema"]))
	})

	t.Run("all valid schemas produce examples", func(t *testing.T) {
		openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
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
        price:
          type: number
`

		result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
			IncludeAll: true,
			Seed:       42,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Examples, "User")
		assert.Contains(t, result.Examples, "Product")
	})

	t.Run("all schemas error returns empty map", func(t *testing.T) {
		openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    BadArray1:
      type: object
      properties:
        items:
          type: array
    BadArray2:
      type: object
      properties:
        tags:
          type: array
`

		result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
			IncludeAll: true,
			Seed:       42,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Examples)
	})

	t.Run("oneOf schema alongside simple schema", func(t *testing.T) {
		openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    SimpleSchema:
      type: object
      properties:
        name:
          type: string
    SftpRequest:
      type: object
      properties:
        host:
          type: string
        port:
          type: integer
    HttpRequest:
      type: object
      properties:
        url:
          type: string
          format: uri
    DeliveryCreateRequest:
      type: object
      properties:
        name:
          type: string
      oneOf:
        - $ref: '#/components/schemas/SftpRequest'
        - $ref: '#/components/schemas/HttpRequest'
`

		result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
			SchemaNames: []string{"SimpleSchema", "DeliveryCreateRequest"},
			Seed:        42,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Contains(t, result.Examples, "SimpleSchema")
		require.Contains(t, result.Examples, "DeliveryCreateRequest")

		assert.JSONEq(t, `{"name":"dl2INvNSQT"}`, string(result.Examples["SimpleSchema"]))
		assert.JSONEq(t, `{"host":"GyAVmNkB33","name":"Z5zQu9MxNm","port":83}`, string(result.Examples["DeliveryCreateRequest"]))
	})

	t.Run("multiple valid schemas with one erroring schema in between", func(t *testing.T) {
		openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    First:
      type: object
      properties:
        alpha:
          type: string
    Broken:
      type: object
      properties:
        list:
          type: array
    Last:
      type: object
      properties:
        omega:
          type: integer
`

		result, err := schema.ConvertToExamples([]byte(openapi), schema.ExampleOptions{
			IncludeAll: true,
			Seed:       42,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.NotContains(t, result.Examples, "Broken")
		assert.Contains(t, result.Examples, "First")
		assert.Contains(t, result.Examples, "Last")
	})
}
