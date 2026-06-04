package schema_test

import (
	"strings"
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertEnumNumbersReorderInvariant proves the determinism invariant on enum
// variants: "emit variants in number order for a deterministic, reorder-invariant
// proto" (internal/proto/builder.go). The same enum with the same EnumNumbers
// mapping, declared in two different YAML orders, must produce byte-identical
// proto. Without the number-order sort the output follows declaration order, so the
// two encodings diverge and regenerating from a reordered spec silently rewrites the
// proto. The reserved/zero-value checks (a mutex-equivalent that already passes)
// cannot satisfy this — only the ordering itself can.
func TestConvertEnumNumbersReorderInvariant(t *testing.T) {
	// Variant 0 is declared first in both orders so the proto3 zero-value check
	// passes regardless of the sort; only the non-zero variants are scrambled.
	declOrderA := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Response:
      type: object
      properties:
        code:
          $ref: '#/components/schemas/Code'
    Code:
      type: integer
      enum:
        - 0
        - 404
        - 200`

	declOrderB := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Response:
      type: object
      properties:
        code:
          $ref: '#/components/schemas/Code'
    Code:
      type: integer
      enum:
        - 0
        - 200
        - 404`

	nums := &schema.FieldNumbers{
		Enums: map[string]schema.EnumNumbers{
			"Code": {Variants: map[string]int{"0": 0, "200": 1, "404": 2}},
		},
	}

	resultA, err := schema.Convert([]byte(declOrderA), schema.ConvertOptions{
		PackageName:  "testpkg",
		PackagePath:  "github.com/example/proto/v1",
		FieldNumbers: nums,
	})
	require.NoError(t, err)
	require.NotNil(t, resultA)

	resultB, err := schema.Convert([]byte(declOrderB), schema.ConvertOptions{
		PackageName:  "testpkg",
		PackagePath:  "github.com/example/proto/v1",
		FieldNumbers: nums,
	})
	require.NoError(t, err)
	require.NotNil(t, resultB)

	// Reorder-invariance: identical input numbers, different declaration order, byte-identical proto.
	assert.Equal(t, string(resultA.Protobuf), string(resultB.Protobuf))

	// Canonical order is proto-number order: CODE_0(0), CODE_200(1), CODE_404(2).
	out := string(resultA.Protobuf)
	pos0 := strings.Index(out, "CODE_0 = 0")
	pos200 := strings.Index(out, "CODE_200 = 1")
	pos404 := strings.Index(out, "CODE_404 = 2")
	require.NotEqual(t, -1, pos0)
	require.NotEqual(t, -1, pos200)
	require.NotEqual(t, -1, pos404)
	assert.Less(t, pos0, pos200)
	assert.Less(t, pos200, pos404)
}

// TestConvertStyleBOneOfReorderInvariant proves the determinism invariant on oneof
// members: "emitted in field-number order for deterministic output" (attachOneof in
// internal/proto/builder.go). A style-B union whose oneOf branches are declared in a
// different order than the variant fields' numbers must still emit the oneof members
// in number order, so two specs that differ only in branch order produce byte-identical
// proto. The oneof grouping references fields by identity and the per-field numbers are
// locked (the already-passing append-only guard) — that cannot make the *member order*
// deterministic; only the group's number-order sort can.
func TestConvertStyleBOneOfReorderInvariant(t *testing.T) {
	// Properties are declared cat_event then dog_event in both specs, so positional
	// numbering pins cat_event=1, dog_event=2 identically. Only the oneOf BRANCH order
	// differs, which is exactly what the group's number-order sort must normalize.
	branchOrderCatDog := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        cat_event:
          $ref: '#/components/schemas/Cat'
        dog_event:
          $ref: '#/components/schemas/Dog'
      oneOf:
        - required: [cat_event]
        - required: [dog_event]
    Cat:
      type: object
      properties:
        pet_name:
          type: string
    Dog:
      type: object
      properties:
        pet_name:
          type: string`

	branchOrderDogCat := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        cat_event:
          $ref: '#/components/schemas/Cat'
        dog_event:
          $ref: '#/components/schemas/Dog'
      oneOf:
        - required: [dog_event]
        - required: [cat_event]
    Cat:
      type: object
      properties:
        pet_name:
          type: string
    Dog:
      type: object
      properties:
        pet_name:
          type: string`

	resultCatDog, err := schema.Convert([]byte(branchOrderCatDog), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, resultCatDog)

	resultDogCat, err := schema.Convert([]byte(branchOrderDogCat), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, resultDogCat)

	// Reorder-invariance: branch order must not affect the emitted proto.
	assert.Equal(t, string(resultCatDog.Protobuf), string(resultDogCat.Protobuf))

	// Canonical member order is field-number order: cat_event(1) before dog_event(2).
	out := string(resultCatDog.Protobuf)
	posCat := strings.Index(out, "cat_event = 1")
	posDog := strings.Index(out, "dog_event = 2")
	require.NotEqual(t, -1, posCat)
	require.NotEqual(t, -1, posDog)
	assert.Less(t, posCat, posDog)
}
