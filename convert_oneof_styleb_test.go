package schema_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStyleBOneOfGeneratesProtoOneof verifies that a wire-compatible style-B schema
// (an object with one optional $ref property per variant plus a oneOf of
// single-required branches and no discriminator) is built as a protobuf message
// containing a oneof group. Each variant field keeps its locked number and its
// snake_case json_name annotation, and no Go union is produced.
func TestStyleBOneOfGeneratesProtoOneof(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
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
          type: string
`

	expected := `syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1";

message Event {
  oneof event {
    Cat cat_event = 1 [json_name = "cat_event"];
    Dog dog_event = 2 [json_name = "dog_event"];
  }
}

message Cat {
  string pet_name = 1 [json_name = "pet_name"];
}

message Dog {
  string pet_name = 1 [json_name = "pet_name"];
}

`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected, string(result.Protobuf))
	assert.Empty(t, result.Golang)
	assert.Equal(t, schema.TypeLocationProto, result.TypeMap["Event"].Location)
}

// TestStyleBOneOfWithRegularFields verifies a style-B union may coexist with normal
// always-present fields in the same message: proto3 permits a oneof next to plain
// fields. The non-variant field renders outside the oneof group; the variant fields
// render inside it, keeping their numbers.
func TestStyleBOneOfWithRegularFields(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        id:
          type: string
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
          type: string
`

	expectedMessage := `message Event {
  string id = 1 [json_name = "id"];
  oneof event {
    Cat cat_event = 2 [json_name = "cat_event"];
    Dog dog_event = 3 [json_name = "dog_event"];
  }
}`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, string(result.Protobuf), expectedMessage)
	assert.Empty(t, result.Golang)
}

// TestStyleBOneOfFieldNumbersAppendOnly verifies that oneof members are ordinary
// numbered fields: a supplied FieldNumbers mapping (the fieldmap-lock path) pins each
// variant field's number, and the oneof grouping never renumbers or reuses a member's
// number.
func TestStyleBOneOfFieldNumbersAppendOnly(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
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
          type: string
`

	expectedMessage := `message Event {
  oneof event {
    Cat cat_event = 5 [json_name = "cat_event"];
    Dog dog_event = 9 [json_name = "dog_event"];
  }
}`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
		FieldNumbers: &schema.FieldNumbers{
			Messages: map[string]schema.MessageNumbers{
				"Event": {Fields: map[string]int{"cat_event": 5, "dog_event": 9}},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, string(result.Protobuf), expectedMessage)
}

// TestStyleBOneOfRemovedVariantReserved verifies that removing a variant from a
// style-B schema and regenerating reserves the removed field's number permanently
// (a tombstone) while the surviving variant keeps its number. The reserved statement
// renders at message level, outside the oneof group.
func TestStyleBOneOfRemovedVariantReserved(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        cat_event:
          $ref: '#/components/schemas/Cat'
      oneOf:
        - required: [cat_event]
    Cat:
      type: object
      properties:
        pet_name:
          type: string
`

	expectedMessage := `message Event {
  oneof event {
    Cat cat_event = 1 [json_name = "cat_event"];
  }
  reserved 2;
}`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
		FieldNumbers: &schema.FieldNumbers{
			Messages: map[string]schema.MessageNumbers{
				"Event": {Fields: map[string]int{"cat_event": 1}, Reserved: []int{2}},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, string(result.Protobuf), expectedMessage)
}

// TestStyleBOneOfMalformedRejected verifies each illegal style-B shape is rejected at
// validation time (through Convert) rather than passed through or discovered downstream
// by protoc.
func TestStyleBOneOfMalformedRejected(t *testing.T) {
	for _, test := range []struct {
		name    string
		given   string
		wantErr string
	}{
		{
			name: "branch with more than one required entry",
			given: `openapi: 3.0.0
info:
  title: Test API
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
        - required: [cat_event, dog_event]
    Cat:
      type: object
      properties:
        pet_name:
          type: string
    Dog:
      type: object
      properties:
        pet_name:
          type: string
`,
			wantErr: "exactly one required",
		},
		{
			name: "branch names a property absent from properties",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        cat_event:
          $ref: '#/components/schemas/Cat'
      oneOf:
        - required: [cat_event]
        - required: [bird_event]
    Cat:
      type: object
      properties:
        pet_name:
          type: string
`,
			wantErr: "bird_event",
		},
		{
			name: "two branches requiring the same property",
			given: `openapi: 3.0.0
info:
  title: Test API
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
          type: string
`,
			wantErr: "duplicate",
		},
		{
			name: "variant property whose schema is an array",
			given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        cats:
          type: array
          items:
            $ref: '#/components/schemas/Cat'
        dog_event:
          $ref: '#/components/schemas/Dog'
      oneOf:
        - required: [cats]
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
          type: string
`,
			wantErr: "cats",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
				PackageName: "testpkg",
				PackagePath: "github.com/example/proto/v1",
			})
			require.Error(t, err)
			require.Nil(t, result)
			assert.ErrorContains(t, err, test.wantErr)
		})
	}
}

// TestDiscriminatedOneOfStillRoutedToGo verifies the feature does not open a path for
// the flat/discriminated oneOf: a oneOf with a discriminator is still routed to the Go
// union path (not the proto oneof builder), preserving existing behavior.
func TestDiscriminatedOneOfStillRoutedToGo(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, schema.TypeLocationGolang, result.TypeMap["Pet"].Location)
	assert.NotEmpty(t, result.Golang)
	assert.NotContains(t, string(result.Protobuf), "oneof")
}

// TestStyleBMalformedBranchHasClearMessage verifies that a nested-union attempt with a
// branch missing `required` (no discriminator, no $ref branches) is rejected with a
// message that names the real problem — the required-property contract — rather than
// the misleading "oneOf requires discriminator", which would point the author at a
// discriminator that does not apply to inline branches.
func TestStyleBMalformedBranchHasClearMessage(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test API
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
        - description: "oops, no required here"
    Cat:
      type: object
      properties:
        pet_name:
          type: string
    Dog:
      type: object
      properties:
        pet_name:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		PackageName: "testpkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.Error(t, err)
	require.Nil(t, result)
	assert.ErrorContains(t, err, "required property")
	assert.NotContains(t, err.Error(), "requires discriminator")
}
