package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDependencyGraphDiamond validates diamond dependency pattern classification
func TestDependencyGraphDiamond(t *testing.T) {
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
    Owner:
      type: object
      properties:
        name:
          type: string
        pet:
          $ref: '#/components/schemas/Pet'
    Home:
      type: object
      properties:
        address:
          type: string
        pet:
          $ref: '#/components/schemas/Pet'
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	// Pet is union
	petInfo, exists := result.TypeMap["Pet"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, petInfo.Location)
	assert.Contains(t, petInfo.Reason, "oneOf")

	// Dog and Cat are variants
	dogInfo, exists := result.TypeMap["Dog"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, dogInfo.Location)
	assert.Contains(t, dogInfo.Reason, "variant")

	catInfo, exists := result.TypeMap["Cat"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, catInfo.Location)
	assert.Contains(t, catInfo.Reason, "variant")

	// Both Owner and Home reference Pet (diamond pattern)
	ownerInfo, exists := result.TypeMap["Owner"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, ownerInfo.Location)
	assert.Contains(t, ownerInfo.Reason, "references union type")

	homeInfo, exists := result.TypeMap["Home"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, homeInfo.Location)
	assert.Contains(t, homeInfo.Reason, "references union type")

	// Verify all types are in Go output
	goCode := string(result.Golang)
	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "type Cat struct")
	assert.Contains(t, goCode, "type Owner struct")
	assert.Contains(t, goCode, "type Home struct")
}

// TestDependencyGraphDeepTransitive validates deep transitive closure (A->B->C->D->Union)
func TestDependencyGraphDeepTransitive(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Union:
      oneOf:
        - $ref: '#/components/schemas/Variant1'
        - $ref: '#/components/schemas/Variant2'
      discriminator:
        propertyName: type
    Variant1:
      type: object
      properties:
        type:
          type: string
        value:
          type: string
    Variant2:
      type: object
      properties:
        type:
          type: string
        count:
          type: integer
    D:
      type: object
      properties:
        name:
          type: string
        union:
          $ref: '#/components/schemas/Union'
    C:
      type: object
      properties:
        id:
          type: integer
        d:
          $ref: '#/components/schemas/D'
    B:
      type: object
      properties:
        active:
          type: boolean
        c:
          $ref: '#/components/schemas/C'
    A:
      type: object
      properties:
        label:
          type: string
        b:
          $ref: '#/components/schemas/B'
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	// All types should be Go-only due to transitive closure
	for _, typeName := range []string{"Union", "Variant1", "Variant2", "D", "C", "B", "A"} {
		info, exists := result.TypeMap[typeName]
		require.True(t, exists, "type %s not in TypeMap", typeName)
		assert.Equal(t, schema.TypeLocationGolang, info.Location, "type %s should be Go-only", typeName)
		assert.NotEmpty(t, info.Reason, "type %s should have a reason", typeName)
	}

	// Verify reasons
	unionInfo := result.TypeMap["Union"]
	assert.Contains(t, unionInfo.Reason, "oneOf")

	variant1Info := result.TypeMap["Variant1"]
	assert.Contains(t, variant1Info.Reason, "variant")

	dInfo := result.TypeMap["D"]
	assert.Contains(t, dInfo.Reason, "references union type")

	// C, B, A should all reference union transitively
	for _, typeName := range []string{"C", "B", "A"} {
		info := result.TypeMap[typeName]
		assert.Contains(t, info.Reason, "references union type", "type %s reason", typeName)
	}
}

// TestDependencyGraphMultipleUnions validates schemas referencing multiple union types
func TestDependencyGraphMultipleUnions(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Union1:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Bird'
      discriminator:
        propertyName: type
    Union2:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Fish'
      discriminator:
        propertyName: type
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: string
    Bird:
      type: object
      properties:
        type:
          type: string
        chirp:
          type: string
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: string
    Fish:
      type: object
      properties:
        type:
          type: string
        swim:
          type: string
    Container:
      type: object
      properties:
        name:
          type: string
        pet1:
          $ref: '#/components/schemas/Union1'
        pet2:
          $ref: '#/components/schemas/Union2'
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	// Both unions should be Go-only
	union1Info, exists := result.TypeMap["Union1"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, union1Info.Location)

	union2Info, exists := result.TypeMap["Union2"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, union2Info.Location)

	// Container references both unions
	containerInfo, exists := result.TypeMap["Container"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, containerInfo.Location)
	assert.Contains(t, containerInfo.Reason, "references union type")

	// Variants should be Go-only
	dogInfo, exists := result.TypeMap["Dog"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, dogInfo.Location)

	catInfo, exists := result.TypeMap["Cat"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, catInfo.Location)
}

// TestDependencyGraphOrphanedTypes validates types with no dependencies stay proto-only
func TestDependencyGraphOrphanedTypes(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Union:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
      discriminator:
        propertyName: type
    Dog:
      type: object
      properties:
        type:
          type: string
        name:
          type: string
    Cat:
      type: object
      properties:
        type:
          type: string
        name:
          type: string
    Owner:
      type: object
      properties:
        name:
          type: string
        pet:
          $ref: '#/components/schemas/Union'
    Orphan1:
      type: object
      properties:
        id:
          type: integer
        value:
          type: string
    Orphan2:
      type: object
      properties:
        active:
          type: boolean
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	// Union and related types are Go-only
	unionInfo, exists := result.TypeMap["Union"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, unionInfo.Location)

	ownerInfo, exists := result.TypeMap["Owner"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, ownerInfo.Location)

	// Orphaned types should be proto-only
	orphan1Info, exists := result.TypeMap["Orphan1"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationProto, orphan1Info.Location)
	assert.Empty(t, orphan1Info.Reason)

	orphan2Info, exists := result.TypeMap["Orphan2"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationProto, orphan2Info.Location)
	assert.Empty(t, orphan2Info.Reason)

	// Verify orphaned types are in proto output, not Go
	protoCode := string(result.Protobuf)
	assert.Contains(t, protoCode, "message Orphan1")
	assert.Contains(t, protoCode, "message Orphan2")

	goCode := string(result.Golang)
	assert.NotContains(t, goCode, "type Orphan1 struct")
	assert.NotContains(t, goCode, "type Orphan2 struct")
}

// TestDependencyGraphSiblingReferences validates siblings of union variants
func TestDependencyGraphSiblingReferences(t *testing.T) {
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
        toy:
          $ref: '#/components/schemas/Toy'
    Cat:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
    Toy:
      type: object
      properties:
        name:
          type: string
        color:
          type: string
    Food:
      type: object
      properties:
        brand:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeMap)

	// Pet and variants are Go-only
	petInfo, exists := result.TypeMap["Pet"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, petInfo.Location)

	dogInfo, exists := result.TypeMap["Dog"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, dogInfo.Location)

	catInfo, exists := result.TypeMap["Cat"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, catInfo.Location)

	// Toy is referenced by Dog (a variant), but Dog doesn't reference a union
	// So Toy should be proto-only (not all variants referencing something makes it Go)
	toyInfo, exists := result.TypeMap["Toy"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationProto, toyInfo.Location)
	assert.Empty(t, toyInfo.Reason)

	// Food is orphaned and should also be proto-only
	foodInfo, exists := result.TypeMap["Food"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationProto, foodInfo.Location)
	assert.Empty(t, foodInfo.Reason)

	// Verify proto output includes both Toy and Food
	protoCode := string(result.Protobuf)
	assert.Contains(t, protoCode, "message Toy")
	assert.Contains(t, protoCode, "message Food")

	// Verify Go output includes variants but not Toy or Food
	goCode := string(result.Golang)
	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "type Cat struct")
	assert.NotContains(t, goCode, "type Toy struct")
	assert.NotContains(t, goCode, "type Food struct")
}
