package conv_test

import (
	"encoding/json"
	"testing"

	conv "github.com/duh-rpc/openapi-proto.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToExamplesAllHeuristicsTogether(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Response:
      type: object
      properties:
        cursor:
          type: string
        message:
          type: string
        code:
          type: integer
        value:
          type: number
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

	require.Contains(t, value, "cursor")
	cursor := value["cursor"].(string)
	assert.GreaterOrEqual(t, len(cursor), 16)
	assert.LessOrEqual(t, len(cursor), 32)

	require.Contains(t, value, "message")
	assert.Equal(t, "This is a message", value["message"])

	require.Contains(t, value, "code")
	code := int(value["code"].(float64))
	assert.GreaterOrEqual(t, code, 1)
	assert.LessOrEqual(t, code, 100)

	require.Contains(t, value, "value")
	valueNum := value["value"].(float64)
	assert.GreaterOrEqual(t, valueNum, 1.0)
	assert.LessOrEqual(t, valueNum, 100.0)
}

func TestConvertToExamplesRealisticErrorResponse(t *testing.T) {
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
        error:
          type: string
        message:
          type: string
        timestamp:
          type: string
          format: date-time
`

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		FieldOverrides: map[string]interface{}{
			"code": 500,
		},
		SchemaNames: []string{"ErrorResponse"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "ErrorResponse")

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["ErrorResponse"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "code")
	assert.Equal(t, float64(500), value["code"])

	require.Contains(t, value, "error")
	assert.Equal(t, "An error occurred", value["error"])

	require.Contains(t, value, "message")
	assert.Equal(t, "This is a message", value["message"])

	require.Contains(t, value, "timestamp")
	timestamp := value["timestamp"].(string)
	assert.NotEmpty(t, timestamp)
}

func TestConvertToExamplesPaginationResponse(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    PageInfo:
      type: object
      properties:
        cursor:
          type: string
        first:
          type: string
        after:
          type: string
        hasNext:
          type: boolean
    PaginatedResponse:
      type: object
      properties:
        data:
          type: array
          minItems: 2
          items:
            type: object
            properties:
              id:
                type: string
                format: uuid
              name:
                type: string
        pageInfo:
          $ref: '#/components/schemas/PageInfo'
`

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		SchemaNames: []string{"PaginatedResponse"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "PaginatedResponse")

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["PaginatedResponse"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "data")
	data := value["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 2)

	require.Contains(t, value, "pageInfo")
	pageInfo := value["pageInfo"].(map[string]interface{})

	require.Contains(t, pageInfo, "cursor")
	cursor := pageInfo["cursor"].(string)
	assert.GreaterOrEqual(t, len(cursor), 16)
	assert.LessOrEqual(t, len(cursor), 32)
	for _, ch := range cursor {
		valid := (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '+' || ch == '/'
		assert.True(t, valid)
	}

	require.Contains(t, pageInfo, "first")
	first := pageInfo["first"].(string)
	assert.GreaterOrEqual(t, len(first), 16)
	assert.LessOrEqual(t, len(first), 32)

	require.Contains(t, pageInfo, "after")
	after := pageInfo["after"].(string)
	assert.GreaterOrEqual(t, len(after), 16)
	assert.LessOrEqual(t, len(after), 32)

	require.Contains(t, pageInfo, "hasNext")
	assert.IsType(t, true, pageInfo["hasNext"])
}

func TestConvertToExamplesFieldOverridesWithNestedObjects(t *testing.T) {
	openapi := `openapi: 3.0.0
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
        code:
          type: integer
    User:
      type: object
      properties:
        name:
          type: string
        code:
          type: integer
        address:
          $ref: '#/components/schemas/Address'
`

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		FieldOverrides: map[string]interface{}{
			"code": 42,
		},
		SchemaNames: []string{"User"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "User")

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["User"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "code")
	assert.Equal(t, float64(42), value["code"])

	require.Contains(t, value, "address")
	address := value["address"].(map[string]interface{})

	require.Contains(t, address, "code")
	assert.Equal(t, float64(42), address["code"])
}

func TestConvertToExamplesRandomDefaultsConsistency(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Data:
      type: object
      properties:
        count1:
          type: integer
        count2:
          type: integer
        count3:
          type: integer
        value1:
          type: number
        value2:
          type: number
`

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		SchemaNames: []string{"Data"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "Data")

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["Data"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "count1")
	count1 := int(value["count1"].(float64))
	assert.GreaterOrEqual(t, count1, 1)
	assert.LessOrEqual(t, count1, 100)
	assert.NotEqual(t, 0, count1)

	require.Contains(t, value, "count2")
	count2 := int(value["count2"].(float64))
	assert.GreaterOrEqual(t, count2, 1)
	assert.LessOrEqual(t, count2, 100)
	assert.NotEqual(t, 0, count2)

	require.Contains(t, value, "count3")
	count3 := int(value["count3"].(float64))
	assert.GreaterOrEqual(t, count3, 1)
	assert.LessOrEqual(t, count3, 100)
	assert.NotEqual(t, 0, count3)

	require.Contains(t, value, "value1")
	value1 := value["value1"].(float64)
	assert.GreaterOrEqual(t, value1, 1.0)
	assert.LessOrEqual(t, value1, 100.0)
	assert.NotEqual(t, 0.0, value1)

	require.Contains(t, value, "value2")
	value2 := value["value2"].(float64)
	assert.GreaterOrEqual(t, value2, 1.0)
	assert.LessOrEqual(t, value2, 100.0)
	assert.NotEqual(t, 0.0, value2)
}

func TestConvertToExamplesHeuristicsWithOverridePriority(t *testing.T) {
	openapi := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Response:
      type: object
      properties:
        cursor:
          type: string
        message:
          type: string
          example: "Example message"
        error:
          type: string
          default: "Default error"
`

	result, err := conv.ConvertToExamples([]byte(openapi), conv.ExampleOptions{
		FieldOverrides: map[string]interface{}{
			"cursor": "custom-cursor-value",
		},
		SchemaNames: []string{"Response"},
		Seed:        42,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Examples, "Response")

	var value map[string]interface{}
	err = json.Unmarshal(result.Examples["Response"], &value)
	require.NoError(t, err)

	require.Contains(t, value, "cursor")
	assert.Equal(t, "custom-cursor-value", value["cursor"])

	require.Contains(t, value, "message")
	assert.Equal(t, "Example message", value["message"])

	require.Contains(t, value, "error")
	assert.Equal(t, "Default error", value["error"])
}
