package example_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToExamplesStringFormats(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "email format",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        email:
          type: string
          format: email
`,
			schema:   "User",
			expected: `{"email":"user@example.com"}`,
		},
		{
			name: "uuid format",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Resource:
      type: object
      properties:
        id:
          type: string
          format: uuid
`,
			schema:   "Resource",
			expected: `{"id":"123e4567-e89b-12d3-a456-426614174000"}`,
		},
		{
			name: "uri format",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Link:
      type: object
      properties:
        url:
          type: string
          format: uri
`,
			schema:   "Link",
			expected: `{"url":"https://example.com"}`,
		},
		{
			name: "date format",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Event:
      type: object
      properties:
        date:
          type: string
          format: date
`,
			schema:   "Event",
			expected: `{"date":"2024-01-15"}`,
		},
		{
			name: "date-time format",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Timestamp:
      type: object
      properties:
        createdAt:
          type: string
          format: date-time
`,
			schema:   "Timestamp",
			expected: `{"createdAt":"2024-01-15T10:30:00Z"}`,
		},
		{
			name: "hostname format",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Server:
      type: object
      properties:
        host:
          type: string
          format: hostname
`,
			schema:   "Server",
			expected: `{"host":"example.com"}`,
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

func TestConvertToExamplesStringLengthConstraints(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		expected string
	}{
		{
			name: "string with minLength",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        username:
          type: string
          minLength: 5
`,
			schema:   "User",
			expected: `{"username":"dl2IN"}`,
		},
		{
			name: "string with maxLength",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        code:
          type: string
          maxLength: 8
`,
			schema:   "User",
			expected: `{"code":"l2INvNSQ"}`,
		},
		{
			name: "string with minLength and maxLength",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Product:
      type: object
      properties:
        sku:
          type: string
          minLength: 10
          maxLength: 15
`,
			schema:   "Product",
			expected: `{"sku":"l2INvNSQTZ5zQu9"}`,
		},
		{
			name: "email format with minLength padding",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Contact:
      type: object
      properties:
        email:
          type: string
          format: email
          minLength: 30
`,
			schema:   "Contact",
			expected: `{"email":"user@example.comxxxxxxxxxxxxxx"}`,
		},
		{
			name: "uuid format with maxLength truncation",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    ShortId:
      type: object
      properties:
        id:
          type: string
          format: uuid
          maxLength: 10
`,
			schema:   "ShortId",
			expected: `{"id":"123e4567-e"}`,
		},
		{
			name: "invalid constraints - minLength greater than maxLength",
			openapi: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Invalid:
      type: object
      properties:
        text:
          type: string
          minLength: 20
          maxLength: 10
`,
			schema:   "Invalid",
			expected: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.ConvertToExamples([]byte(test.openapi), schema.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})
			require.NoError(t, err)
			require.NotNil(t, result)

			if test.expected == "" {
				assert.Empty(t, result.Examples)
				return
			}

			require.Contains(t, result.Examples, test.schema)
			assert.JSONEq(t, test.expected, string(result.Examples[test.schema]))
		})
	}
}
