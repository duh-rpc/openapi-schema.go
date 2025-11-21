package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoNestedStructsInVariant validates that variant structs with nested referenced types
func TestGoNestedStructsInVariant(t *testing.T) {
	given := `openapi: 3.0.0
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
        friend:
          $ref: '#/components/schemas/Cat'
    Cat:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "Friend *Cat")
	assert.Contains(t, goCode, "type Cat struct")
	assert.Contains(t, goCode, "Name string")
}

// TestGoMixedFieldTypes validates variant structs with mixed scalar, reference, and array fields
func TestGoMixedFieldTypes(t *testing.T) {
	given := `openapi: 3.0.0
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
        name:
          type: string
        age:
          type: integer
          format: int32
        weight:
          type: number
          format: float
        isVaccinated:
          type: boolean
        tags:
          type: array
          items:
            type: string
        bestFriend:
          $ref: '#/components/schemas/Cat'
        birthDate:
          type: string
          format: date
        siblings:
          type: array
          items:
            $ref: '#/components/schemas/Dog'
    Cat:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
        meow:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "Name string")
	assert.Contains(t, goCode, "Age int32")
	assert.Contains(t, goCode, "Weight float32")
	assert.Contains(t, goCode, "IsVaccinated bool")
	assert.Contains(t, goCode, "Tags []string")
	assert.Contains(t, goCode, "BestFriend *Cat")
	assert.Contains(t, goCode, "BirthDate time.Time")
	assert.Contains(t, goCode, "Siblings []*Dog")
	assert.Contains(t, goCode, `"time"`)
}

// TestGoPackageNameExtraction validates package name extraction from GoPackagePath
func TestGoPackageNameExtraction(t *testing.T) {
	for _, test := range []struct {
		name        string
		packagePath string
		wantPkg     string
	}{
		{name: "v1 suffix uses second-to-last", packagePath: "github.com/example/types/v1", wantPkg: "package types"},
		{name: "v2 suffix uses second-to-last", packagePath: "github.com/example/api/v2", wantPkg: "package api"},
		{name: "no version uses last component", packagePath: "github.com/example/types", wantPkg: "package types"},
		{name: "api suffix no version", packagePath: "github.com/example/api", wantPkg: "package api"},
		{name: "single component", packagePath: "models", wantPkg: "package models"},
	} {
		t.Run(test.name, func(t *testing.T) {
			given := `openapi: 3.0.0
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
        name:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
`

			result, err := schema.Convert([]byte(given), schema.ConvertOptions{
				GoPackagePath: test.packagePath,
				PackageName:   "testpkg",
				PackagePath:   "github.com/example/proto/v1",
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Golang)

			goCode := string(result.Golang)
			assert.Contains(t, goCode, test.wantPkg)
		})
	}
}

// TestGoFieldJSONTags validates that JSON tags preserve original OpenAPI property names
func TestGoFieldJSONTags(t *testing.T) {
	given := `openapi: 3.0.0
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
        propertyName: pet_type
    Dog:
      type: object
      properties:
        pet_type:
          type: string
        HTTPStatus:
          type: string
        camelCase:
          type: string
    Cat:
      type: object
      properties:
        pet_type:
          type: string
        HTTPStatus:
          type: string
        camelCase:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, `PetType string `+"`json:\"pet_type\"`")
	assert.Contains(t, goCode, `HTTPStatus string `+"`json:\"HTTPStatus\"`")
	assert.Contains(t, goCode, `CamelCase string `+"`json:\"camelCase\"`")
}

// TestGoPointerFieldsForReferences validates pointer usage for referenced variant types
func TestGoPointerFieldsForReferences(t *testing.T) {
	given := `openapi: 3.0.0
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
        name:
          type: string
        age:
          type: integer
        bestFriend:
          $ref: '#/components/schemas/Cat'
        catFriends:
          type: array
          items:
            $ref: '#/components/schemas/Cat'
    Cat:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, "Name string")
	assert.Contains(t, goCode, "Age int32")
	assert.Contains(t, goCode, "BestFriend *Cat")
	assert.Contains(t, goCode, "CatFriends []*Cat")
}

// TestGoMultipleVariants validates union types with multiple variant structs
func TestGoMultipleVariants(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Bird'
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
    Bird:
      type: object
      properties:
        petType:
          type: string
        chirp:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "Dog *Dog")
	assert.Contains(t, goCode, "Cat *Cat")
	assert.Contains(t, goCode, "Bird *Bird")

	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "Bark string")
	assert.Contains(t, goCode, "type Cat struct")
	assert.Contains(t, goCode, "Meow string")
	assert.Contains(t, goCode, "type Bird struct")
	assert.Contains(t, goCode, "Chirp string")
}
