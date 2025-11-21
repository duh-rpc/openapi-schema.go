package schema_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateExamplesValidExample(t *testing.T) {
	openapi := `
openapi: 3.1.0
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
      example:
        name: "John"
        age: 30
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")

	userResult := result.Schemas["User"]
	assert.True(t, userResult.HasExamples)
	assert.True(t, userResult.Valid)
	assert.Empty(t, userResult.Issues)
}

func TestValidateExamplesInvalidExample(t *testing.T) {
	openapi := `
openapi: 3.1.0
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
      example:
        name: "John"
        age: "thirty"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")

	userResult := result.Schemas["User"]
	assert.True(t, userResult.HasExamples)
	assert.False(t, userResult.Valid)
	assert.NotEmpty(t, userResult.Issues)

	hasError := false
	for _, issue := range userResult.Issues {
		if issue.Severity == schema.IssueSeverityError {
			hasError = true
			assert.Equal(t, "example", issue.ExampleField)
		}
	}
	assert.True(t, hasError)
}

func TestValidateExamplesMissingExample(t *testing.T) {
	openapi := `
openapi: 3.1.0
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

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")

	userResult := result.Schemas["User"]
	assert.False(t, userResult.HasExamples)
	assert.True(t, userResult.Valid)
}

func TestValidateExamplesMultipleExamples(t *testing.T) {
	openapi := `
openapi: 3.1.0
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
      examples:
        - name: "Alice"
        - name: "Bob"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")

	userResult := result.Schemas["User"]
	assert.True(t, userResult.HasExamples)
	assert.True(t, userResult.Valid)
	assert.Empty(t, userResult.Issues)
}

func TestValidateExamplesBothExampleAndExamples(t *testing.T) {
	openapi := `
openapi: 3.1.0
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
      example:
        name: "John"
        age: 30
      examples:
        - name: "Alice"
          age: 25
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")

	userResult := result.Schemas["User"]
	assert.True(t, userResult.HasExamples)
	assert.True(t, userResult.Valid)
}

func TestValidateExamplesEnumValidation(t *testing.T) {
	openapi := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum: [active, inactive, pending]
      example: "deleted"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "Status")

	statusResult := result.Schemas["Status"]
	assert.True(t, statusResult.HasExamples)
	assert.False(t, statusResult.Valid)
	assert.NotEmpty(t, statusResult.Issues)
}

func TestValidateExamplesConstraintValidation(t *testing.T) {
	openapi := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Age:
      type: integer
      minimum: 0
      maximum: 120
      example: 150
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "Age")

	ageResult := result.Schemas["Age"]
	assert.True(t, ageResult.HasExamples)
	assert.False(t, ageResult.Valid)
	assert.NotEmpty(t, ageResult.Issues)
}

func TestValidateExamplesOpenAPI30Warning(t *testing.T) {
	openapi := `
openapi: 3.0.0
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
      example:
        name: "John"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")

	userResult := result.Schemas["User"]
	assert.True(t, userResult.HasExamples)
	assert.True(t, userResult.Valid)

	hasWarning := false
	for _, issue := range userResult.Issues {
		if issue.Severity == schema.IssueSeverityWarning {
			hasWarning = true
			assert.Contains(t, issue.Message, "OpenAPI 3.0 detected")
		}
	}
	assert.True(t, hasWarning)
}

func TestValidateExamplesSchemaNameFiltering(t *testing.T) {
	openapi := `
openapi: 3.1.0
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
      example:
        name: "John"
    Product:
      type: object
      properties:
        title:
          type: string
      example:
        title: "Widget"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		SchemaNames: []string{"User"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")
	require.NotContains(t, result.Schemas, "Product")

	userResult := result.Schemas["User"]
	assert.True(t, userResult.HasExamples)
	assert.True(t, userResult.Valid)
}

func TestValidateExamplesIncludeAllPriority(t *testing.T) {
	openapi := `
openapi: 3.1.0
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
      example:
        name: "John"
    Product:
      type: object
      properties:
        title:
          type: string
      example:
        title: "Widget"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		SchemaNames: []string{"User"},
		IncludeAll:  true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")
	require.Contains(t, result.Schemas, "Product")
}

func TestValidateExamplesEmptyOptionsError(t *testing.T) {
	openapi := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
`

	_, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{})

	require.Error(t, err)
	require.ErrorContains(t, err, "must specify SchemaNames or set IncludeAll")
}

func TestValidateExamplesEmptyInput(t *testing.T) {
	_, err := schema.ValidateExamples([]byte{}, schema.ValidateOptions{
		IncludeAll: true,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "openapi input cannot be empty")
}

func TestValidateExamplesMultipleErrorsCollected(t *testing.T) {
	openapi := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      required: [name, age, email]
      properties:
        name:
          type: string
        age:
          type: integer
        email:
          type: string
      example:
        name: 123
        age: "thirty"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "User")

	userResult := result.Schemas["User"]
	assert.True(t, userResult.HasExamples)
	assert.False(t, userResult.Valid)

	errorCount := 0
	for _, issue := range userResult.Issues {
		if issue.Severity == schema.IssueSeverityError {
			errorCount++
		}
	}
	assert.Greater(t, errorCount, 0)
}

func TestValidateExamplesStringLengthConstraint(t *testing.T) {
	openapi := `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    ShortString:
      type: string
      minLength: 5
      maxLength: 10
      example: "abc"
`

	result, err := schema.ValidateExamples([]byte(openapi), schema.ValidateOptions{
		IncludeAll: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.Schemas, "ShortString")

	stringResult := result.Schemas["ShortString"]
	assert.True(t, stringResult.HasExamples)
	assert.False(t, stringResult.Valid)
	assert.NotEmpty(t, stringResult.Issues)
}
