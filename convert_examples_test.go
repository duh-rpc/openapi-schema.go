package conv_test

import (
	"encoding/json"
	"testing"

	conv "github.com/duh-rpc/openapi-proto.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToExamplesValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		openapi []byte
		opts    conv.ExampleOptions
		wantErr string
	}{
		{
			name:    "empty openapi bytes",
			openapi: []byte{},
			opts:    conv.ExampleOptions{IncludeAll: true},
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
			opts:    conv.ExampleOptions{},
			wantErr: "must specify SchemaNames or set IncludeAll",
		},
		{
			name: "invalid openapi document",
			openapi: []byte(`this is not valid: [yaml`),
			opts:    conv.ExampleOptions{IncludeAll: true},
			wantErr: "failed to parse OpenAPI document",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := conv.ConvertToExamples(test.openapi, test.opts)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestConvertToExamplesScalarTypes(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				assert.IsType(t, "", m["name"])
			},
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
			schema: "Product",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "quantity")
				_, ok := m["quantity"].(float64)
				assert.True(t, ok)
			},
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
			schema: "Settings",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "enabled")
				assert.IsType(t, true, m["enabled"])
			},
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
			schema: "Price",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "amount")
				assert.IsType(t, 0.0, m["amount"])
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
		})
	}
}

func TestConvertToExamplesConstraints(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "Product",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "quantity")
				quantity := int(m["quantity"].(float64))
				assert.GreaterOrEqual(t, quantity, 10)
				assert.LessOrEqual(t, quantity, 50)
			},
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
			schema: "Price",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "amount")
				amount := m["amount"].(float64)
				assert.GreaterOrEqual(t, amount, 1.5)
				assert.LessOrEqual(t, amount, 99.99)
			},
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
			schema: "Settings",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "timeout")
				assert.Equal(t, float64(30), m["timeout"])
			},
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				assert.Equal(t, "John Doe", m["name"])
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
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

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		SchemaNames: []string{"Status"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "Status")

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["Status"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "state")
	assert.Equal(t, "pending", value["state"])
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

	result1, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		SchemaNames: []string{"Product"},
		Seed:        seed,
	})
	require.NoError(t, err)

	result2, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
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

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
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

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
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

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
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
		validate func(t *testing.T, value interface{})
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				require.Contains(t, m, "age")
				require.Contains(t, m, "active")
				assert.IsType(t, "", m["name"])
				assert.IsType(t, float64(0), m["age"])
				assert.IsType(t, true, m["active"])
			},
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
			schema: "Product",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "title")
				require.Contains(t, m, "price")
				require.Contains(t, m, "quantity")
				require.Contains(t, m, "inStock")
			},
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
			schema: "Empty",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				assert.Empty(t, m)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
		})
	}
}

func TestConvertToExamplesNestedObjects(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				require.Contains(t, m, "address")

				address := m["address"].(map[string]interface{})
				require.Contains(t, address, "street")
				require.Contains(t, address, "city")
				require.Contains(t, address, "zipCode")
				assert.IsType(t, "", address["street"])
				assert.IsType(t, "", address["city"])
				assert.IsType(t, float64(0), address["zipCode"])
			},
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
			schema: "Company",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				require.Contains(t, m, "headquarters")

				hq := m["headquarters"].(map[string]interface{})
				require.Contains(t, hq, "address")

				address := hq["address"].(map[string]interface{})
				require.Contains(t, address, "street")
				require.Contains(t, address, "location")

				location := address["location"].(map[string]interface{})
				require.Contains(t, location, "lat")
				require.Contains(t, location, "lng")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
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

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		SchemaNames: []string{"Level1"},
		MaxDepth:    3,
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["Level1"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "name")
	require.Contains(t, value, "level2")

	level2 := value["level2"].(map[string]interface{})
	require.Contains(t, level2, "name")
	require.Contains(t, level2, "level3")

	level3 := level2["level3"].(map[string]interface{})
	require.Contains(t, level3, "name")
	assert.NotContains(t, level3, "level4")
}

func TestConvertToExamplesArrays(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "TagList",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "tags")
				tags := m["tags"].([]interface{})
				assert.Len(t, tags, 1)
				assert.IsType(t, "", tags[0])
			},
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
			schema: "Numbers",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "values")
				values := m["values"].([]interface{})
				assert.Len(t, values, 1)
				assert.IsType(t, float64(0), values[0])
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
		})
	}
}

func TestConvertToExamplesArrayConstraints(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "TagList",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "tags")
				tags := m["tags"].([]interface{})
				assert.GreaterOrEqual(t, len(tags), 3)
			},
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
			schema: "Limited",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "items")
				items := m["items"].([]interface{})
				assert.Len(t, items, 5)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
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

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		SchemaNames: []string{"UserList"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["UserList"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "users")
	users := value["users"].([]interface{})
	assert.GreaterOrEqual(t, len(users), 2)

	user := users[0].(map[string]interface{})
	require.Contains(t, user, "name")
	require.Contains(t, user, "age")
	assert.IsType(t, "", user["name"])
	assert.IsType(t, float64(0), user["age"])
}

func TestConvertToExamplesInvalidArraySchema(t *testing.T) {
	for _, test := range []struct {
		name    string
		openapi string
		wantErr string
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
			wantErr: "array must have items defined",
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
			wantErr: "invalid schema: minItems > maxItems",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{"BadArray", "InvalidArray"},
				Seed:        42,
			})
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestConvertToExamplesReferences(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				require.Contains(t, m, "address")

				address := m["address"].(map[string]interface{})
				require.Contains(t, address, "street")
				require.Contains(t, address, "city")
			},
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				require.Contains(t, m, "address")

				address := m["address"].(map[string]interface{})
				require.Contains(t, address, "street")
				require.Contains(t, address, "city")

				city := address["city"].(map[string]interface{})
				require.Contains(t, city, "name")
				require.Contains(t, city, "zipCode")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
		})
	}
}

func TestConvertToExamplesCircularReferences(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "Node",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "value")
				assert.NotContains(t, m, "next")
			},
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				require.Contains(t, m, "address")

				address := m["address"].(map[string]interface{})
				require.Contains(t, address, "street")
				assert.NotContains(t, address, "owner")
			},
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
			schema: "A",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "name")
				require.Contains(t, m, "b")

				b := m["b"].(map[string]interface{})
				require.Contains(t, b, "value")
				require.Contains(t, b, "c")

				c := b["c"].(map[string]interface{})
				require.Contains(t, c, "flag")
				assert.NotContains(t, c, "a")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
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

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		SchemaNames:    []string{"ErrorResponse"},
		Seed:           42,
		FieldOverrides: map[string]interface{}{"code": 400},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "ErrorResponse")

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["ErrorResponse"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "code")
	require.Contains(t, value, "message")
}

func TestConvertToExamplesRandomDefaults(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "Product",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "quantity")
				quantity := int(m["quantity"].(float64))
				assert.GreaterOrEqual(t, quantity, 1)
				assert.LessOrEqual(t, quantity, 100)
				assert.NotEqual(t, 0, quantity)
			},
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
			schema: "Price",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "amount")
				amount := m["amount"].(float64)
				assert.GreaterOrEqual(t, amount, 1.0)
				assert.LessOrEqual(t, amount, 100.0)
				assert.NotEqual(t, 0.0, amount)
			},
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
			schema: "Data",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "count")
				require.Contains(t, m, "value")
				count := int(m["count"].(float64))
				valueNum := m["value"].(float64)
				assert.GreaterOrEqual(t, count, 1)
				assert.LessOrEqual(t, count, 100)
				assert.GreaterOrEqual(t, valueNum, 1.0)
				assert.LessOrEqual(t, valueNum, 100.0)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, test.schema)

			var value interface{}
			err = json.Unmarshal(result.Examples[test.schema], &value)
			require.NoError(t, err)

			test.validate(t, value)
		})
	}
}

func TestConvertToExamplesCursorHeuristics(t *testing.T) {
	for _, test := range []struct {
		name       string
		fieldName  string
		shouldMatch bool
	}{
		{
			name:       "cursor field lowercase",
			fieldName:  "cursor",
			shouldMatch: true,
		},
		{
			name:       "first field lowercase",
			fieldName:  "first",
			shouldMatch: true,
		},
		{
			name:       "after field lowercase",
			fieldName:  "after",
			shouldMatch: true,
		},
		{
			name:       "Cursor field capitalized",
			fieldName:  "Cursor",
			shouldMatch: true,
		},
		{
			name:       "other field does not match",
			fieldName:  "other",
			shouldMatch: false,
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
			result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
				SchemaNames: []string{"Pagination"},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, "Pagination")

			var value map[string]interface{}
			err = json.Unmarshal(result.Examples["Pagination"], &value)
			require.NoError(t, err)
			require.Contains(t, value, test.fieldName)

			fieldValue := value[test.fieldName].(string)

			if test.shouldMatch {
				assert.GreaterOrEqual(t, len(fieldValue), 16)
				assert.LessOrEqual(t, len(fieldValue), 32)

				for _, ch := range fieldValue {
					valid := (ch >= 'a' && ch <= 'z') ||
						(ch >= 'A' && ch <= 'Z') ||
						(ch >= '0' && ch <= '9') ||
						ch == '+' || ch == '/'
					assert.True(t, valid)
				}
			} else {
				assert.Len(t, fieldValue, 10)
			}
		})
	}
}

func TestConvertToExamplesMessageHeuristics(t *testing.T) {
	for _, test := range []struct {
		name       string
		fieldName  string
		expected   string
		shouldMatch bool
	}{
		{
			name:       "error field lowercase",
			fieldName:  "error",
			expected:   "An error occurred",
			shouldMatch: true,
		},
		{
			name:       "message field lowercase",
			fieldName:  "message",
			expected:   "This is a message",
			shouldMatch: true,
		},
		{
			name:       "Error field capitalized",
			fieldName:  "Error",
			expected:   "An error occurred",
			shouldMatch: true,
		},
		{
			name:       "description field does not match",
			fieldName:  "description",
			shouldMatch: false,
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
			result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
				SchemaNames: []string{"Response"},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Contains(t, result.Examples, "Response")

			var value map[string]interface{}
			err = json.Unmarshal(result.Examples["Response"], &value)
			require.NoError(t, err)
			require.Contains(t, value, test.fieldName)

			fieldValue := value[test.fieldName].(string)

			if test.shouldMatch {
				assert.Equal(t, test.expected, fieldValue)
			} else {
				assert.Len(t, fieldValue, 10)
			}
		})
	}
}

func TestConvertToExamplesFieldOverridePriority(t *testing.T) {
	for _, test := range []struct {
		name       string
		openapi    string
		overrides  map[string]interface{}
		validate   func(t *testing.T, value map[string]interface{})
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
			validate: func(t *testing.T, value map[string]interface{}) {
				require.Contains(t, value, "message")
				assert.Equal(t, "custom message", value["message"])
			},
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
			validate: func(t *testing.T, value map[string]interface{}) {
				require.Contains(t, value, "code")
				code := value["code"].(float64)
				assert.Equal(t, 200.0, code)
			},
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
			validate: func(t *testing.T, value map[string]interface{}) {
				require.Contains(t, value, "enabled")
				assert.Equal(t, false, value["enabled"])
			},
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
			validate: func(t *testing.T, value map[string]interface{}) {
				require.Contains(t, value, "code")
				require.Contains(t, value, "status")
				require.Contains(t, value, "message")
				code := value["code"].(float64)
				assert.Equal(t, 500.0, code)
				assert.Equal(t, "error", value["status"])
				assert.Equal(t, "Internal server error", value["message"])
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				FieldOverrides: test.overrides,
				IncludeAll:     true,
				Seed:           42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)

			var value map[string]interface{}
			schemaName := ""
			for name := range result.Examples {
				schemaName = name
				break
			}
			require.NotEmpty(t, schemaName)

			err = json.Unmarshal(result.Examples[schemaName], &value)
			require.NoError(t, err)

			test.validate(t, value)
		})
	}
}

func TestConvertToExamplesFieldOverrideTypeMismatch(t *testing.T) {
	for _, test := range []struct {
		name      string
		openapi   string
		overrides map[string]interface{}
		wantErr   string
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
			overrides: map[string]interface{}{"code": "not a number"},
			wantErr:   "field override for 'code' has wrong type",
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
			overrides: map[string]interface{}{"message": 123},
			wantErr:   "field override for 'message' has wrong type",
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
			overrides: map[string]interface{}{"enabled": "true"},
			wantErr:   "field override for 'enabled' has wrong type",
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
			overrides: map[string]interface{}{"count": 42.5},
			wantErr:   "field override for 'count' has wrong type",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				FieldOverrides: test.overrides,
				IncludeAll:     true,
				Seed:           42,
			})
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}
