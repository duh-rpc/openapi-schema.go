# OpenAPI to Protobuf Converter

A Go library that converts OpenAPI 3.x (3.0, 3.1, 3.2) schema definitions to Protocol Buffer 3 (proto3) format.

[![Go Version](https://img.shields.io/github/go-mod/go-version/duh-rpc/openapi-proto.go)](https://golang.org/dl/)
[![CI Status](https://github.com/duh-rpc/openapi-proto.go/workflows/CI/badge.svg)](https://github.com/duh-rpc/openapi-proto.go/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/duh-rpc/openapi-proto.go)](https://goreportcard.com/report/github.com/duh-rpc/openapi-proto.go)

## Overview

This library parses OpenAPI 3.x specifications (3.0, 3.1, and 3.2) and generates corresponding `.proto` files with proper type mappings, JSON field name annotations, and protobuf conventions. It's designed for projects that need to support both OpenAPI and protobuf interfaces.

## Installation

```bash
go get github.com/duh-rpc/openapi-proto.go
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "os"

    conv "github.com/duh-rpc/openapi-proto.go"
)

func main() {
    // Read OpenAPI specification
    openapi, err := os.ReadFile("api.yaml")
    if err != nil {
        panic(err)
    }

    // Convert to proto3 and Go
    result, err := conv.Convert(openapi, conv.ConvertOptions{
        PackageName: "myapi",
        PackagePath: "github.com/example/proto/v1",
    })
    if err != nil {
        panic(err)
    }

    // Write proto file (if generated)
    if len(result.Protobuf) > 0 {
        err = os.WriteFile("api.proto", result.Protobuf, 0644)
        if err != nil {
            panic(err)
        }
    }

    // Write Go file (if generated - for types with unions)
    if len(result.Golang) > 0 {
        err = os.WriteFile("types.go", result.Golang, 0644)
        if err != nil {
            panic(err)
        }
    }
}
```

### Go-Only Conversion

If you need Go struct types without Protocol Buffer definitions, use `ConvertToStruct()` to generate pure Go code:

```go
package main

import (
    "fmt"
    "os"

    conv "github.com/duh-rpc/openapi-proto.go"
)

func main() {
    // Read OpenAPI specification
    openapi, err := os.ReadFile("api.yaml")
    if err != nil {
        panic(err)
    }

    // Convert ALL schemas to Go structs (no protobuf)
    result, err := conv.ConvertToStruct(openapi, conv.ConvertOptions{
        GoPackagePath: "github.com/example/types/v1",
    })
    if err != nil {
        panic(err)
    }

    // Write Go file
    err = os.WriteFile("types.go", result.Golang, 0644)
    if err != nil {
        panic(err)
    }

    // TypeMap shows all types as Go structs
    for typeName, info := range result.TypeMap {
        fmt.Printf("%s: %s (%s)\n", typeName, info.Location, info.Reason)
    }
}
```

**Key Differences from `Convert()`:**

| Feature | `Convert()` | `ConvertToStruct()` |
|---------|-------------|---------------------|
| Output | Proto + Go (for unions) | Go only |
| Type classification | Transitive closure filtering | All schemas become Go |
| Union handling | Custom marshaling | Custom marshaling |
| Regular types | Proto messages | Go structs with JSON tags |
| Use case | Dual proto/Go interface | Pure Go types |

**When to use `ConvertToStruct()`:**
- You need Go types but not protobuf definitions
- You want all schemas as Go structs for a consistent API
- You're building a pure Go application without gRPC
- You want simpler type management (everything in one Go file)

### JSON Example Generation

Generate JSON examples from OpenAPI schemas for documentation, testing, or API design. The `ConvertToExamples()` function creates realistic examples that honor schema constraints like min/max values, string formats, enums, and required fields.

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    conv "github.com/duh-rpc/openapi-proto.go"
)

func main() {
    openapi := []byte(`openapi: 3.0.0
info:
  title: User API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        age:
          type: integer
          minimum: 18
          maximum: 120
        status:
          type: string
          enum: [active, inactive]
`)

    // Generate examples for all schemas
    result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
        IncludeAll: true,
        MaxDepth:   5,
        Seed:       12345, // For deterministic generation
    })
    if err != nil {
        panic(err)
    }

    // Access generated examples
    userJSON := result.Examples["User"]
    fmt.Printf("User example: %s\n", string(userJSON))

    // Or unmarshal to validate structure
    var user map[string]interface{}
    json.Unmarshal(userJSON, &user)
    fmt.Printf("Email: %s, Age: %d\n", user["email"], int(user["age"].(float64)))
}
```

**Example output:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "email": "user@example.com",
  "age": 42,
  "status": "active"
}
```

**ExampleOptions:**
- `IncludeAll`: Generate examples for all schemas (takes precedence over SchemaNames)
- `SchemaNames`: Specific schemas to generate examples for (used when IncludeAll is false)
- `MaxDepth`: Maximum nesting depth for circular references (default: 5)
- `Seed`: Random seed for deterministic generation (0 = time-based randomness)

**Constraint Handling:**

The example generator honors OpenAPI schema constraints:

| Constraint | Behavior |
|------------|----------|
| `minimum` / `maximum` | Generates numbers within range |
| `minLength` / `maxLength` | Generates strings within length limits |
| `minItems` / `maxItems` | Generates arrays within item count limits |
| `enum` | Picks first value for deterministic output |
| `format` | Generates format-specific values (email, uuid, uri, date, date-time) |
| `default` | Uses default value if specified |
| `example` | Uses example value if specified (highest priority) |

**Circular Reference Handling:**

Circular references are automatically detected and broken to prevent infinite recursion:

```go
// Schema with circular reference
openapi := []byte(`openapi: 3.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        friends:
          type: array
          items:
            $ref: '#/components/schemas/User'
`)

result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
    MaxDepth:   3, // Limit nesting depth
})
// The 'friends' array will be generated but nested User objects
// will be omitted once the depth limit is reached
```

**See [docs/examples.md](docs/examples.md) for detailed documentation.**

### Input: OpenAPI 3.x YAML

```yaml
openapi: 3.0.0
info:
  title: User API
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      description: A user account
      properties:
        userId:
          type: string
          description: Unique user identifier
        email:
          type: string
        age:
          type: integer
        isActive:
          type: boolean
```

### Output: Proto3

```protobuf
syntax = "proto3";

package myapi;

// A user account
message User {
  // Unique user identifier
  string userId = 1 [json_name = "userId"];
  string email = 2 [json_name = "email"];
  int32 age = 3 [json_name = "age"];
  bool isActive = 4 [json_name = "isActive"];
}
```

## Union Support with OneOf

This library supports OpenAPI `oneOf` schemas with discriminators by generating **Go structs with custom JSON marshaling** instead of Protocol Buffer messages. This approach maintains complete JSON compatibility with the OpenAPI specification while avoiding protobuf's incompatible `oneof` format.

### Why Go Code Generation?

OpenAPI's `oneOf` with discriminators produces flat JSON like `{"petType": "dog", "bark": "woof"}`, but protobuf's `oneof` wraps the variant: `{"dog": {"petType": "dog", "bark": "woof"}}`. These formats are incompatible. To maintain OpenAPI's JSON contract, union types are generated as Go code with custom marshaling. See [discriminated-unions.md](docs/discriminated-unions.md) for details.

### Basic Union Example

**OpenAPI:**
```yaml
openapi: 3.0.0
info:
  title: Pet API
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
          enum: [dog]
        bark:
          type: string

    Cat:
      type: object
      properties:
        petType:
          type: string
          enum: [cat]
        meow:
          type: string
```

**Generated Go Code:**
```go
package mypkg

import (
    "encoding/json"
    "fmt"
    "strings"
)

// Union wrapper with pointer fields to variants
type Pet struct {
    Dog *Dog
    Cat *Cat
}

// Custom marshaling to match flat OpenAPI JSON
func (u *Pet) MarshalJSON() ([]byte, error) {
    if u.Dog != nil {
        return json.Marshal(u.Dog)
    }
    if u.Cat != nil {
        return json.Marshal(u.Cat)
    }
    return nil, fmt.Errorf("Pet: no variant set")
}

func (u *Pet) UnmarshalJSON(data []byte) error {
    var discriminator struct {
        PetType string `json:"petType"`
    }
    if err := json.Unmarshal(data, &discriminator); err != nil {
        return err
    }

    // Case-insensitive discriminator matching
    switch strings.ToLower(discriminator.PetType) {
    case "dog":
        u.Dog = &Dog{}
        return json.Unmarshal(data, u.Dog)
    case "cat":
        u.Cat = &Cat{}
        return json.Unmarshal(data, u.Cat)
    default:
        return fmt.Errorf("unknown petType: %s", discriminator.PetType)
    }
}

type Dog struct {
    PetType string `json:"petType"`
    Bark    string `json:"bark"`
}

type Cat struct {
    PetType string `json:"petType"`
    Meow    string `json:"meow"`
}
```

**JSON Format** (matches OpenAPI spec exactly):
```json
{"petType": "dog", "bark": "woof"}
```

### Using ConvertResult and TypeMap

When schemas contain unions, `Convert()` returns a `ConvertResult` with separate proto and Go outputs. Similarly, `ConvertToStruct()` returns a `StructResult` with Go-only output:

```go
result, err := conv.Convert(openapi, conv.ConvertOptions{
    PackageName:   "myapi",
    PackagePath:   "github.com/example/proto/v1",
    GoPackagePath: "github.com/example/types/v1",  // Optional, defaults to PackagePath
})
if err != nil {
    panic(err)
}

// TypeMap tells you where each type is generated
for typeName, info := range result.TypeMap {
    fmt.Printf("%s: %s (%s)\n", typeName, info.Location, info.Reason)
}
// Output:
// Pet: golang (contains oneOf)
// Dog: golang (variant of union type Pet)
// Cat: golang (variant of union type Pet)
// Owner: golang (references union type Pet)
// Address: proto ()

// Protobuf contains types that don't use unions
if len(result.Protobuf) > 0 {
    os.WriteFile("api.proto", result.Protobuf, 0644)
}

// Golang contains union types and anything that references them
if len(result.Golang) > 0 {
    os.WriteFile("types.go", result.Golang, 0644)
}
```

### Transitive Closure for Union Types

When a schema contains or references a union, it becomes a Go type. This applies transitively:

- **Union types** (Pet with oneOf) → Go
- **Union variants** (Dog, Cat referenced in oneOf) → Go
- **Types referencing unions** (Owner with `pet: $ref Pet`) → Go
- **Proto-only types** (Address with no union connection) → Proto

The `TypeMap` provides complete visibility into why each type is generated where it is.

### Union Requirements

For Phase 1 support, unions must meet these requirements:

- **Discriminator required**: All `oneOf` schemas must have a `discriminator.propertyName`
- **Reference-based variants**: All variants must use `$ref` (no inline schemas)
- **Discriminator in variants**: Each variant schema must include the discriminator property
- **Case-insensitive matching**: Discriminator values match schema names case-insensitively

**Supported:**
```yaml
Pet:
  oneOf:
    - $ref: '#/components/schemas/Dog'
    - $ref: '#/components/schemas/Cat'
  discriminator:
    propertyName: petType
```

**Not supported (will error):**
```yaml
# Missing discriminator
Pet:
  oneOf:
    - $ref: '#/components/schemas/Dog'
    - $ref: '#/components/schemas/Cat'

# Inline variant (not $ref)
Pet:
  oneOf:
    - type: object
      properties:
        bark: {type: string}
  discriminator:
    propertyName: petType
```

## Supported Features

### OpenAPI Features
- ✅ Object schemas with properties
- ✅ Scalar types (string, integer, number, boolean)
- ✅ String enums (mapped to string fields with enum comments)
- ✅ Integer enums (mapped to protobuf enum types)
- ✅ Arrays (repeated fields)
- ✅ Nested objects
- ✅ Schema references (`$ref`)
- ✅ Descriptions (converted to comments)
- ✅ Multiple format specifiers (int32, int64, float, double, byte, binary, date, date-time)

### Proto3 Features
- ✅ Message definitions
- ✅ Enum definitions with UNSPECIFIED values
- ✅ Repeated fields
- ✅ Nested messages
- ✅ JSON name annotations
- ✅ Field numbering (sequential based on YAML order)
- ✅ Comments from descriptions

## Unsupported Features

### OpenAPI Features Not Supported
- ✅ `oneOf` with discriminators (generates Go code with custom marshaling)
- ✅ Nullable type arrays (OpenAPI 3.1+ `type: [string, null]` syntax)
- ❌ Schema composition: `allOf`, `anyOf`, `not`
- ❌ `oneOf` without discriminators
- ❌ Inline oneOf variants (must use `$ref`)
- ❌ External file references (only internal `#/components/schemas` refs)
- ❌ Nested arrays (e.g., `array` of `array`)
- ❌ Truly multi-type properties (e.g., `type: [string, integer]`) - only nullable variants allowed
- ❌ Map types via `additionalProperties`
- ❌ Validation constraints (min, max, pattern, etc. are ignored)
- ❌ OpenAPI 2.0 (Swagger) - only 3.x supported

### Proto3 Features Not Generated
- ❌ Service definitions
- ❌ Multiple output files (single file only)
- ❌ Import statements
- ❌ Proto options beyond `json_name`
- ❌ Map types
- ❌ `optional` keyword (all fields follow proto3 default semantics)
- ❌ Wrapper types for nullable fields

### Nullable Field Handling

The library supports nullable types in both OpenAPI 3.0 and 3.1+ syntax:

**OpenAPI 3.0 nullable syntax:**
```yaml
properties:
  name:
    type: string
    nullable: true
```

**OpenAPI 3.1+ type array syntax:**
```yaml
properties:
  name:
    type: [string, null]
```

Both are converted to the same proto3 field:
```protobuf
string name = 1 [json_name = "name"];
```

**Important:** Proto3 doesn't have a nullable concept - it uses zero values to indicate "not set" (empty string for strings, 0 for numbers, false for booleans, null for messages). The `nullable` keyword and `null` type are processed but don't change the proto3 output, since proto3 fields are inherently nullable through zero values.

### Ignored OpenAPI Directives
- The `required` array is ignored (proto3 has no required keyword)
- The `nullable` field is ignored (proto3 uses zero values for optional semantics)

## Type Mapping

| OpenAPI Type | OpenAPI Format | Proto3 Type | Notes |
|--------------|----------------|-------------|-------|
| string       | (none)         | string      |       |
| string       | byte           | bytes       |       |
| string       | binary         | bytes       |       |
| string       | date           | string      |       |
| string       | date-time      | string      |       |
| string + enum | (none)        | string      | Enum values in comments |
| integer      | (none)         | int32       |       |
| integer      | int32          | int32       |       |
| integer      | int64          | int64       |       |
| integer + enum | (none)       | enum        | Protobuf enum type |
| number       | (none)         | double      |       |
| number       | float          | float       |       |
| number       | double         | double      |       |
| boolean      | (any)          | bool        |       |
| object       | (any)          | message     |       |
| array        | (any)          | repeated    |       |

## Naming Conventions

### Field Names: Preservation

The library preserves original OpenAPI field names when they're valid proto3 syntax:
- `HTTPStatus` → `HTTPStatus` (preserved)
- `userId` → `userId` (preserved)
- `user_id` → `user_id` (preserved)

Invalid characters are replaced with underscores:
- `status-code` → `status_code` (hyphen → underscore)
- `user.name` → `user_name` (dot → underscore)
- `first name` → `first_name` (space → underscore)

All fields include a `json_name` annotation to explicitly map to the original OpenAPI field name.

#### Proto3 Field Name Requirements

Field names must:
- Start with an ASCII letter (A-Z or a-z) - non-ASCII letters like `ñ` are not allowed
- Contain only ASCII letters, digits (0-9), and underscores (_)
- Field names starting with digits or underscores will cause errors
- Field names that are proto3 reserved keywords (like `message`, `enum`, `package`) will cause protoc compilation errors - the library does not detect or prevent these

**Note on Reserved Keywords:** Proto3 has reserved keywords like `message`, `enum`, `service`, `package`, `import`, `option`, etc. If your OpenAPI schema has field names that match these keywords, the generated proto file will fail to compile with protoc. This is intentional - the library lets protoc handle keyword validation rather than maintaining a keyword list that might change across proto versions.

#### Best Practices

While proto3 syntax allows mixed-case field names, the [Protocol Buffers style guide](https://protobuf.dev/programming-guides/style/) recommends snake_case for consistency across languages. If you control your OpenAPI schema, consider using snake_case field names to align with proto3 conventions.

#### BREAKING CHANGE Notice

**This represents a breaking change from previous library behavior.**

**Previous behavior:**
- Field names were converted to snake_case: `HTTPStatus` → `h_t_t_p_status`
- Simple letter-by-letter conversion with no acronym detection

**New behavior:**
- Field names are preserved when valid: `HTTPStatus` → `HTTPStatus`
- Only invalid characters are replaced: `status-code` → `status_code`

**Migration:**
If you have existing code that references generated proto field names, you will need to update those references. For example:
- Proto references: `message.h_t_t_p_status` → `message.HTTPStatus`
- Any tooling parsing .proto files needs adjustment for new field names

**Rationale:**
Preserving original names provides more intuitive mapping between OpenAPI and proto, respects your naming choices, and avoids surprising transformations like `HTTPStatus` → `h_t_t_p_status`.

### Message Names: PascalCase

Schema names and nested message names are converted to PascalCase:
- `user_account` → `UserAccount`
- `shippingAddress` → `ShippingAddress`

### Enum Values: UPPERCASE_SNAKE_CASE (Integer Enums Only)

Integer enum values are prefixed with the enum name and converted to uppercase:
- Enum `Code` with value `200` → `CODE_200`
- Enum `Code` with value `404` → `CODE_404`

All integer enums automatically include an `UNSPECIFIED` value at position 0 following proto3 conventions.

String enums do not generate protobuf enum types - they become `string` fields with enum values documented in comments.

### Plural Name Validation

When using inline objects or enums in arrays, property names **must be singular**:

```yaml
# ✅ GOOD - singular property name
properties:
  contact:
    type: array
    items:
      type: object
      properties:
        name:
          type: string

# ❌ BAD - plural property name
properties:
  contacts:  # Will cause error
    type: array
    items:
      type: object
```

**Why?** The library derives message names from property names. A plural property name like `contacts` would generate a message named `Contacts`, which is confusing. Instead:
- Use singular names: `contact` → `Contact` message
- Or use `$ref` to reference a named schema


## Examples

### Enums

The library handles string enums and integer enums differently to preserve JSON wire format compatibility.

#### String Enums

String enums map to `string` fields with enum value annotations in comments:

**OpenAPI:**
```yaml
components:
  schemas:
    Order:
      type: object
      properties:
        status:
          type: string
          description: Status of the order
          enum:
            - pending
            - confirmed
            - shipped
```

**Proto3:**
```protobuf
message Order {
  // Status of the order
  // enum: [pending, confirmed, shipped]
  string status = 1 [json_name = "status"];
}
```

String enums preserve JSON wire format exactly - the JSON will contain `"pending"` not `1` or `"ORDER_STATUS_PENDING"`.

#### Integer Enums

Integer enums map to protobuf enum types:

**OpenAPI:**
```yaml
components:
  schemas:
    Code:
      type: integer
      enum:
        - 200
        - 400
        - 404
        - 500
```

**Proto3:**
```protobuf
enum Code {
  CODE_UNSPECIFIED = 0;
  CODE_200 = 1;
  CODE_400 = 2;
  CODE_404 = 3;
  CODE_500 = 4;
}
```

### Nested Objects

**OpenAPI:**
```yaml
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        address:
          type: object
          properties:
            street:
              type: string
            city:
              type: string
```

**Proto3:**
```protobuf
message User {
  message Address {
    string street = 1 [json_name = "street"];
    string city = 2 [json_name = "city"];
  }

  string name = 1 [json_name = "name"];
  Address address = 2 [json_name = "address"];
}
```

### Arrays with References

**OpenAPI:**
```yaml
components:
  schemas:
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          type: string

    User:
      type: object
      properties:
        name:
          type: string
        address:
          type: array
          items:
            $ref: '#/components/schemas/Address'
```

**Proto3:**
```protobuf
message Address {
  string street = 1 [json_name = "street"];
  string city = 2 [json_name = "city"];
}

message User {
  string name = 1 [json_name = "name"];
  repeated Address address = 2 [json_name = "address"];
}
```

### Name Conflict Resolution

When multiple schemas have the same name, numeric suffixes are automatically added:

**OpenAPI:**
```yaml
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string

    User:  # Duplicate name
      type: object
      properties:
        name:
          type: string
```

**Proto3:**
```protobuf
message User {
  string id = 1 [json_name = "id"];
}

message User_2 {
  string name = 1 [json_name = "name"];
}
```

## Best Practices

1. **Use singular property names** for arrays with inline objects/enums, or use `$ref` to reference named schemas
2. **Consider snake_case field names** in your OpenAPI schema to align with proto3 style guide conventions
3. **Use descriptions** liberally - they become useful comments in the generated proto
4. **Order schemas intentionally** in your OpenAPI YAML - the output order will match
5. **Test with protoc** after generation to catch any proto3 reserved keywords

## Development

### Running Tests

```bash
make test
```

### Test Coverage

```bash
make coverage
```

### Linting

```bash
make lint
```

### Detailed Documentation
See the following links for more details:
- [Enums](docs/enums.md) - How string enums are converted and their limitations
- [Scalar Types](docs/scalar.md) - Type mapping between OpenAPI and proto3
- [Objects](docs/objects.md) - Message generation and nested objects
- [Discriminated Unions](docs/discriminated-unions.md) - How oneOf with discriminators generates Go code

## License

MIT License - see LICENSE file for details

## Related Projects

- [pb33f/libopenapi](https://github.com/pb33f/libopenapi) - OpenAPI parser used by this library
- [protocolbuffers/protobuf](https://github.com/protocolbuffers/protobuf) - Protocol Buffers

## Acknowledgments

This library uses the excellent [libopenapi](https://github.com/pb33f/libopenapi) for OpenAPI parsing, which provides
 support for OpenAPI 3.0 and 3.1 specifications.
