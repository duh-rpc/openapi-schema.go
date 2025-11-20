package conv_test

import (
	"encoding/json"
	"testing"

	conv "github.com/duh-rpc/openapi-proto.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToExamplesStringFormats(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "email")
				email := m["email"].(string)
				assert.Contains(t, email, "@")
				assert.Contains(t, email, ".")
			},
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
			schema: "Resource",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "id")
				id := m["id"].(string)
				assert.Contains(t, id, "-")
				assert.Len(t, id, 36)
			},
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
			schema: "Link",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "url")
				url := m["url"].(string)
				assert.Contains(t, url, "://")
			},
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
			schema: "Event",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "date")
				date := m["date"].(string)
				assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, date)
			},
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
			schema: "Timestamp",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "createdAt")
				createdAt := m["createdAt"].(string)
				assert.Contains(t, createdAt, "T")
				assert.Contains(t, createdAt, "Z")
			},
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
			schema: "Server",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "host")
				host := m["host"].(string)
				assert.Contains(t, host, ".")
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

func TestConvertToExamplesStringLengthConstraints(t *testing.T) {
	for _, test := range []struct {
		name     string
		openapi  string
		schema   string
		validate func(t *testing.T, value interface{})
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "username")
				username := m["username"].(string)
				assert.GreaterOrEqual(t, len(username), 5)
			},
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
			schema: "User",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "code")
				code := m["code"].(string)
				assert.LessOrEqual(t, len(code), 8)
			},
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
			schema: "Product",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "sku")
				sku := m["sku"].(string)
				assert.GreaterOrEqual(t, len(sku), 10)
				assert.LessOrEqual(t, len(sku), 15)
			},
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
			schema: "Contact",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "email")
				email := m["email"].(string)
				assert.GreaterOrEqual(t, len(email), 30)
				assert.Contains(t, email, "@")
			},
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
			schema: "ShortId",
			validate: func(t *testing.T, value interface{}) {
				m := value.(map[string]interface{})
				require.Contains(t, m, "id")
				id := m["id"].(string)
				assert.LessOrEqual(t, len(id), 10)
			},
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
			schema: "Invalid",
			validate: func(t *testing.T, value interface{}) {
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := conv.ConvertToExamples([]byte(test.openapi), conv.ExampleOptions{
				SchemaNames: []string{test.schema},
				Seed:        42,
			})

			if test.name == "invalid constraints - minLength greater than maxLength" {
				require.ErrorContains(t, err, "invalid schema: minLength > maxLength")
				return
			}

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
