package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsupportedAllOf(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "allOf at top level",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Combined:
      allOf:
        - type: object
          properties:
            id:
              type: string
        - type: object
          properties:
            name:
              type: string
`,
			expected: "uses 'allOf' which is not supported",
		},
		{
			name: "allOf in property",
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
          allOf:
            - type: object
              properties:
                name:
                  type: string
`,
			expected: "uses 'allOf' which is not supported",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, test.expected)
		})
	}
}

func TestUnsupportedAnyOf(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "anyOf at top level",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      anyOf:
        - type: object
          properties:
            bark:
              type: boolean
        - type: object
          properties:
            meow:
              type: boolean
`,
			expected: "uses 'anyOf' which is not supported",
		},
		{
			name: "anyOf in property",
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
        contact:
          anyOf:
            - type: string
            - type: object
              properties:
                email:
                  type: string
`,
			expected: "uses 'anyOf' which is not supported",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, test.expected)
		})
	}
}

func TestUnsupportedOneOf(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "oneOf at top level without discriminator",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Shape:
      oneOf:
        - type: object
          properties:
            radius:
              type: number
        - type: object
          properties:
            width:
              type: number
`,
			expected: "oneOf requires discriminator",
		},
		{
			name: "oneOf in property with inline schemas",
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
        identifier:
          oneOf:
            - type: string
            - type: integer
`,
			expected: "oneOf in property 'identifier' requires discriminator",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, test.expected)
		})
	}
}

func TestUnsupportedNot(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		expected string
	}{
		{
			name: "not at top level",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    NotString:
      not:
        type: string
`,
			expected: "uses 'not' which is not supported",
		},
		{
			name: "not in property",
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
        value:
          not:
            type: string
`,
			expected: "uses 'not' which is not supported",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, test.expected)
		})
	}
}

func TestPropertyNoType(t *testing.T) {
	given := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        mystery:
          description: A property with no type or $ref
`

	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "property must have type or $ref")
}

func TestTopLevelPrimitive(t *testing.T) {
	for _, test := range []struct {
		name     string
		given    string
		primType string
	}{
		{
			name: "top level string",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    SimpleString:
      type: string
`,
			primType: "string",
		},
		{
			name: "top level integer",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    SimpleInt:
      type: integer
`,
			primType: "integer",
		},
		{
			name: "top level number",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    SimpleNumber:
      type: number
`,
			primType: "number",
		},
		{
			name: "top level boolean",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    SimpleBool:
      type: boolean
`,
			primType: "boolean",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, "only objects and enums supported at top level")
		})
	}
}

func TestTopLevelArray(t *testing.T) {
	given := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    StringArray:
      type: array
      items:
        type: string
`

	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "only objects and enums supported at top level")
}

func TestErrorContext(t *testing.T) {
	for _, test := range []struct {
		name           string
		given          string
		expectedSchema string
		expectedProp   string
	}{
		{
			name: "schema name in error",
			given: `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    MyUser:
      type: object
      properties:
        bad:
          allOf:
            - type: string
`,
			expectedSchema: "MyUser",
			expectedProp:   "bad",
		},
		{
			name: "nested property in error",
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
        item:
          type: object
          properties:
            price:
              anyOf:
                - type: number
                - type: string
`,
			expectedSchema: "Order",
			expectedProp:   "price",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, test.expectedSchema)
			assert.ErrorContains(t, err, test.expectedProp)
		})
	}
}

func TestMultiTypeProperty(t *testing.T) {
	given := `
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        value:
          type:
            - string
            - integer
`

	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "multi-type properties not supported")
}
