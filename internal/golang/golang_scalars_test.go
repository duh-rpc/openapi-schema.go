package golang_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoIntegerFormats validates that integer format mappings generate correct Go types
// when part of a union schema (which triggers Go code generation)
func TestGoIntegerFormats(t *testing.T) {
	for _, test := range []struct {
		name     string
		format   string
		wantType string
	}{
		{name: "int8", format: "int8", wantType: "Value int8"},
		{name: "int16", format: "int16", wantType: "Value int16"},
		{name: "int32", format: "int32", wantType: "Value int32"},
		{name: "int64", format: "int64", wantType: "Value int64"},
		{name: "uint8", format: "uint8", wantType: "Value uint8"},
		{name: "uint16", format: "uint16", wantType: "Value uint16"},
		{name: "uint32", format: "uint32", wantType: "Value uint32"},
		{name: "uint64", format: "uint64", wantType: "Value uint64"},
		{name: "int (default)", format: "int", wantType: "Value int32"},
		{name: "no format (default)", format: "", wantType: "Value int32"},
	} {
		t.Run(test.name, func(t *testing.T) {
			formatLine := ""
			if test.format != "" {
				formatLine = "\n          format: " + test.format
			}

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
        value:
          type: integer` + formatLine + `
    Cat:
      type: object
      properties:
        petType:
          type: string
        value:
          type: integer` + formatLine

			result, err := schema.Convert([]byte(given), schema.ConvertOptions{
				GoPackagePath: "github.com/example/types/v1",
				PackageName:   "testpkg",
				PackagePath:   "github.com/example/proto/v1",
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Golang)

			goCode := string(result.Golang)
			assert.Contains(t, goCode, test.wantType)
		})
	}
}

// TestGoNumberFormats validates that number format mappings generate correct Go types
func TestGoNumberFormats(t *testing.T) {
	for _, test := range []struct {
		name     string
		format   string
		wantType string
	}{
		{name: "float", format: "float", wantType: "Value float32"},
		{name: "double", format: "double", wantType: "Value float64"},
		{name: "no format (default)", format: "", wantType: "Value float64"},
	} {
		t.Run(test.name, func(t *testing.T) {
			formatLine := ""
			if test.format != "" {
				formatLine = "\n          format: " + test.format
			}

			given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      properties:
        petType:
          type: string
        value:
          type: number` + formatLine + `
    Cat:
      type: object
      properties:
        petType:
          type: string
        value:
          type: number` + formatLine

			result, err := schema.Convert([]byte(given), schema.ConvertOptions{
				GoPackagePath: "github.com/example/types/v1",
				PackageName:   "testpkg",
				PackagePath:   "github.com/example/proto/v1",
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Golang)

			goCode := string(result.Golang)
			assert.Contains(t, goCode, test.wantType)
		})
	}
}

// TestGoStringFormats validates that string format mappings generate correct Go types
func TestGoStringFormats(t *testing.T) {
	for _, test := range []struct {
		name        string
		format      string
		wantType    string
		wantImports []string
	}{
		{name: "date", format: "date", wantType: "Value time.Time", wantImports: []string{"time"}},
		{name: "date-time", format: "date-time", wantType: "Value time.Time", wantImports: []string{"time"}},
		{name: "byte", format: "byte", wantType: "Value []byte"},
		{name: "binary", format: "binary", wantType: "Value []byte"},
		{name: "email", format: "email", wantType: "Value string"},
		{name: "uuid", format: "uuid", wantType: "Value string"},
		{name: "password", format: "password", wantType: "Value string"},
		{name: "no format (default)", format: "", wantType: "Value string"},
	} {
		t.Run(test.name, func(t *testing.T) {
			formatLine := ""
			if test.format != "" {
				formatLine = "\n          format: " + test.format
			}

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
        value:
          type: string` + formatLine + `
    Cat:
      type: object
      properties:
        petType:
          type: string
        value:
          type: string` + formatLine

			result, err := schema.Convert([]byte(given), schema.ConvertOptions{
				GoPackagePath: "github.com/example/types/v1",
				PackageName:   "testpkg",
				PackagePath:   "github.com/example/proto/v1",
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Golang)

			goCode := string(result.Golang)
			assert.Contains(t, goCode, test.wantType)

			for _, imp := range test.wantImports {
				assert.Contains(t, goCode, `"`+imp+`"`)
			}
		})
	}
}

// TestGoArrayTypes validates that array types generate correct Go slice syntax
func TestGoArrayTypes(t *testing.T) {
	for _, test := range []struct {
		name      string
		itemsType string
		wantType  string
	}{
		{name: "array of int32", itemsType: "type: integer\n            format: int32", wantType: "Values []int32"},
		{name: "array of int8", itemsType: "type: integer\n            format: int8", wantType: "Values []int8"},
		{name: "array of uint64", itemsType: "type: integer\n            format: uint64", wantType: "Values []uint64"},
		{name: "array of float32", itemsType: "type: number\n            format: float", wantType: "Values []float32"},
		{name: "array of string", itemsType: "type: string", wantType: "Values []string"},
		{name: "array of boolean", itemsType: "type: boolean", wantType: "Values []bool"},
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
        values:
          type: array
          items:
            ` + test.itemsType + `
    Cat:
      type: object
      properties:
        petType:
          type: string
        values:
          type: array
          items:
            ` + test.itemsType

			result, err := schema.Convert([]byte(given), schema.ConvertOptions{
				GoPackagePath: "github.com/example/types/v1",
				PackageName:   "testpkg",
				PackagePath:   "github.com/example/proto/v1",
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Golang)

			goCode := string(result.Golang)
			assert.Contains(t, goCode, test.wantType)
		})
	}
}

// TestGoArrayOfReferences validates arrays of referenced types use pointer syntax
func TestGoArrayOfReferences(t *testing.T) {
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
        toys:
          type: array
          items:
            $ref: '#/components/schemas/Toy'
    Cat:
      type: object
      properties:
        petType:
          type: string
        toys:
          type: array
          items:
            $ref: '#/components/schemas/Toy'
    Toy:
      type: object
      properties:
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
	assert.Contains(t, goCode, "Toys []*Toy")
}

// TestGoTimestampGeneration validates time.Time generates import
func TestGoTimestampGeneration(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Event'
        - $ref: '#/components/schemas/Task'
      discriminator:
        propertyName: eventType
    Event:
      type: object
      properties:
        eventType:
          type: string
        timestamp:
          type: string
          format: date-time
        date:
          type: string
          format: date
    Task:
      type: object
      properties:
        eventType:
          type: string
        timestamp:
          type: string
          format: date-time
        date:
          type: string
          format: date
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
	assert.Contains(t, goCode, "Timestamp time.Time")
	assert.Contains(t, goCode, "Date time.Time")
	assert.Contains(t, goCode, `"time"`)
}

// TestGoScalarTypeMappings validates comprehensive scalar type mapping in one struct
func TestGoScalarTypeMappings(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/AllTypes'
        - $ref: '#/components/schemas/SomeTypes'
      discriminator:
        propertyName: type
    AllTypes:
      type: object
      properties:
        type:
          type: string
        int8Val:
          type: integer
          format: int8
        int16Val:
          type: integer
          format: int16
        int32Val:
          type: integer
          format: int32
        int64Val:
          type: integer
          format: int64
        uint8Val:
          type: integer
          format: uint8
        uint16Val:
          type: integer
          format: uint16
        uint32Val:
          type: integer
          format: uint32
        uint64Val:
          type: integer
          format: uint64
        float32Val:
          type: number
          format: float
        float64Val:
          type: number
          format: double
        stringVal:
          type: string
        emailVal:
          type: string
          format: email
        uuidVal:
          type: string
          format: uuid
        passwordVal:
          type: string
          format: password
        dateVal:
          type: string
          format: date
        dateTimeVal:
          type: string
          format: date-time
        byteVal:
          type: string
          format: byte
        binaryVal:
          type: string
          format: binary
        boolVal:
          type: boolean
    SomeTypes:
      type: object
      properties:
        type:
          type: string
        stringVal:
          type: string
        int32Val:
          type: integer
          format: int32
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

	assert.Contains(t, goCode, "Int8Val int8")
	assert.Contains(t, goCode, "Int16Val int16")
	assert.Contains(t, goCode, "Int32Val int32")
	assert.Contains(t, goCode, "Int64Val int64")
	assert.Contains(t, goCode, "Uint8Val uint8")
	assert.Contains(t, goCode, "Uint16Val uint16")
	assert.Contains(t, goCode, "Uint32Val uint32")
	assert.Contains(t, goCode, "Uint64Val uint64")
	assert.Contains(t, goCode, "Float32Val float32")
	assert.Contains(t, goCode, "Float64Val float64")
	assert.Contains(t, goCode, "StringVal string")
	assert.Contains(t, goCode, "EmailVal string")
	assert.Contains(t, goCode, "UuidVal string")
	assert.Contains(t, goCode, "PasswordVal string")
	assert.Contains(t, goCode, "DateVal time.Time")
	assert.Contains(t, goCode, "DateTimeVal time.Time")
	assert.Contains(t, goCode, "ByteVal []byte")
	assert.Contains(t, goCode, "BinaryVal []byte")
	assert.Contains(t, goCode, "BoolVal bool")
	assert.Contains(t, goCode, `"time"`)
}
