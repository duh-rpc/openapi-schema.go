# JSON Example Generation

This document explains how the library generates JSON examples from OpenAPI schemas, including constraint handling, format support, and circular reference detection.

## Overview

The `ConvertToExamples()` function generates realistic JSON examples from OpenAPI schema definitions. These examples honor schema constraints like min/max values, string formats, enums, and array size limits. The generated examples are useful for:

- **API Documentation**: Populate documentation with realistic example data
- **Testing**: Generate test fixtures that conform to schema constraints
- **API Design**: Validate schema definitions produce sensible examples
- **Mock Services**: Provide sample responses for API mocking

## Basic Usage

```go
package main

import (
    "encoding/json"
    "fmt"

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
        name:
          type: string
          minLength: 3
          maxLength: 50
        age:
          type: integer
          minimum: 18
          maximum: 120
`)

    result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
        IncludeAll: true,
        MaxDepth:   5,
        Seed:       12345,
    })
    if err != nil {
        panic(err)
    }

    var user map[string]interface{}
    json.Unmarshal(result.Examples["User"], &user)
    fmt.Printf("Generated User: %+v\n", user)
}
```

**Output:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "A9Xw2pQrLm",
  "age": 87
}
```

## ExampleOptions

The `ExampleOptions` struct configures example generation behavior:

```go
type ExampleOptions struct {
    SchemaNames  []string // Specific schemas to generate (ignored if IncludeAll is true)
    MaxDepth     int      // Maximum nesting depth (default 5)
    IncludeAll   bool     // If true, generate examples for all schemas
    Seed         int64    // Random seed for deterministic generation (0 = time-based)
}
```

### SchemaNames vs IncludeAll

Use `IncludeAll: true` to generate examples for all schemas in the OpenAPI document:

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
})
```

Or specify specific schemas with `SchemaNames`:

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    SchemaNames: []string{"User", "Product", "Order"},
})
```

**Note:** `IncludeAll` takes precedence over `SchemaNames`. If `IncludeAll` is false and `SchemaNames` is empty, the function returns an error.

### MaxDepth

Controls maximum nesting depth to prevent infinite recursion with circular references:

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
    MaxDepth:   3, // Limit to 3 levels of nesting
})
```

Default value is 5 if not specified or set to 0.

### Seed

Controls random number generation for deterministic output:

```go
// Deterministic generation (same seed = same output)
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
    Seed:       12345,
})

// Random generation (different output each time)
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
    Seed:       0, // 0 = use time-based seed
})
```

## Constraint Handling

The example generator honors OpenAPI schema constraints to produce realistic, valid examples.

### Numeric Constraints (minimum/maximum)

Integer and number fields respect min/max constraints:

**OpenAPI:**
```yaml
components:
  schemas:
    Product:
      type: object
      properties:
        price:
          type: number
          minimum: 0.01
          maximum: 9999.99
        quantity:
          type: integer
          minimum: 1
          maximum: 100
```

**Generated Example:**
```json
{
  "price": 4523.67,
  "quantity": 42
}
```

Values are randomly generated within the specified range.

### String Length Constraints (minLength/maxLength)

String fields respect length constraints:

**OpenAPI:**
```yaml
components:
  schemas:
    User:
      type: object
      properties:
        username:
          type: string
          minLength: 3
          maxLength: 20
        bio:
          type: string
          maxLength: 500
```

**Generated Example:**
```json
{
  "username": "A9Xw2pQr",
  "bio": "Lorem ipsum dolor sit amet..."
}
```

Generated strings are random alphanumeric values within the length range.

### Array Constraints (minItems/maxItems)

Array fields respect item count constraints:

**OpenAPI:**
```yaml
components:
  schemas:
    Article:
      type: object
      properties:
        tags:
          type: array
          items:
            type: string
          minItems: 2
          maxItems: 10
```

**Generated Example:**
```json
{
  "tags": ["tag1", "tag2"]
}
```

If `minItems` is specified, at least that many items are generated. If `maxItems` is specified and less than the generated count, the array is capped at `maxItems`.

### Enum Values

Enum fields use the first enum value for deterministic output:

**OpenAPI:**
```yaml
components:
  schemas:
    Order:
      type: object
      properties:
        status:
          type: string
          enum:
            - pending
            - confirmed
            - shipped
```

**Generated Example:**
```json
{
  "status": "pending"
}
```

The first enum value (`pending`) is always selected for consistent, deterministic output.

### Default and Example Values

Schema `default` and `example` values take precedence over generated values:

**OpenAPI:**
```yaml
components:
  schemas:
    Config:
      type: object
      properties:
        timeout:
          type: integer
          default: 30
        apiKey:
          type: string
          example: "sk_test_1234567890"
        retries:
          type: integer
          minimum: 1
          maximum: 10
```

**Generated Example:**
```json
{
  "timeout": 30,
  "apiKey": "sk_test_1234567890",
  "retries": 5
}
```

**Priority:** `example` > `default` > generated value

## Smart Field Heuristics

The example generator applies intelligent heuristics based on field names to produce more realistic and context-aware examples. These heuristics recognize common field naming patterns and generate appropriate values automatically.

### Cursor Fields

Fields with names commonly used for pagination cursors generate base64-looking strings that resemble real cursor tokens.

**Recognized field names (case-insensitive):**
- `cursor`
- `first`
- `after`

**Generated values:**
- Random alphanumeric strings using base64 character set: `[a-zA-Z0-9+/]`
- Length: randomly between 16-32 characters
- Example: `"dGhpc2lzYWN1cnNvcg"`, `"YWJjZGVmZ2hpamts"`

**OpenAPI:**
```yaml
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
```

**Generated Example:**
```json
{
  "cursor": "dGhpc2lzYWN1cnNvcg",
  "first": "YWJjZGVmZ2hpamts",
  "after": "bXlwYWdlY3Vyc29y",
  "hasNext": true
}
```

**Notes:**
- Case-insensitive matching: `Cursor`, `CURSOR`, and `cursor` all work
- Applies BEFORE format checking (cursor heuristic takes precedence over format)
- Only applies to string fields

### Message Fields

Fields with names commonly used for error messages and descriptions generate human-readable text.

**Recognized field names (case-insensitive):**
- `error` → `"An error occurred"`
- `message` → `"This is a message"`

**OpenAPI:**
```yaml
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
```

**Generated Example:**
```json
{
  "code": 42,
  "error": "An error occurred",
  "message": "This is a message",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Notes:**
- Case-insensitive matching: `Error`, `MESSAGE`, etc. all work
- Provides more realistic error responses than random strings
- Only applies to string fields

### Non-Zero Defaults for Numbers

Integers and numbers without min/max constraints generate random non-zero values instead of defaulting to zero. This produces more realistic examples that better represent actual data.

**Default behavior:**
- **Integers without constraints**: Random value between 1-100
- **Numbers without constraints**: Random value between 1.0-100.0

**OpenAPI:**
```yaml
components:
  schemas:
    Product:
      type: object
      properties:
        quantity:
          type: integer
        price:
          type: number
        rating:
          type: number
          minimum: 0.0
          maximum: 5.0
```

**Generated Example:**
```json
{
  "quantity": 42,
  "price": 67.34,
  "rating": 3.8
}
```

**Notes:**
- Only applies when BOTH `minimum` and `maximum` are not specified
- If either `minimum` or `maximum` is set, normal constraint logic applies
- Makes examples more visually distinct and realistic
- Deterministic with fixed seed

### Heuristic Priority

Field heuristics apply at a specific point in the value generation priority:

**Full priority order:** `example` > `default` > `FieldOverride` > **heuristics** > generated value

This means:
- Schema `example` values always take precedence
- Schema `default` values override heuristics
- Field overrides (see below) override heuristics
- Heuristics only apply when no higher-priority value is available

**Example:**
```yaml
properties:
  message:
    type: string
    default: "Default message"
```
**Generated:** `"Default message"` (default overrides message heuristic)

```yaml
properties:
  message:
    type: string
```
**Generated:** `"This is a message"` (message heuristic applies)

## Field Overrides

Field overrides allow you to specify exact values for specific field names across all schemas. This is useful for generating examples with consistent error codes, status values, or other standardized fields.

### Basic Usage

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    FieldOverrides: map[string]interface{}{
        "code":    500,
        "status":  "error",
        "message": "Internal server error",
    },
    IncludeAll: true,
})
```

### How Field Overrides Work

**Scope:** Field overrides apply to any field with a matching name across ALL schemas in the document.

**Matching:** Field names are matched case-sensitively to ensure exact JSON field name matches.

**Type validation:** Override values must match the schema's type or an error is returned:
- Integer fields: Accept numeric values (int or float64 without decimal)
- Number fields: Accept numeric values
- String fields: Accept string values
- Boolean fields: Accept boolean values

### Complete Example

**OpenAPI:**
```yaml
openapi: 3.0.0
components:
  schemas:
    ErrorResponse:
      type: object
      properties:
        code:
          type: integer
        status:
          type: string
        message:
          type: string
        timestamp:
          type: string
          format: date-time

    ValidationError:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
        field:
          type: string
```

**Code:**
```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    FieldOverrides: map[string]interface{}{
        "code":    400,
        "message": "Validation failed",
    },
    IncludeAll: true,
    Seed:       42,
})
```

**Generated Examples:**
```json
// ErrorResponse
{
  "code": 400,
  "status": "random-string",
  "message": "Validation failed",
  "timestamp": "2024-01-15T10:30:00Z"
}

// ValidationError
{
  "code": 400,
  "message": "Validation failed",
  "field": "random-string"
}
```

Note that `code` and `message` are overridden in both schemas.

### Override Priority

Field overrides fit into the value generation priority as follows:

**Priority:** `example` > `default` > **FieldOverride** > heuristics > generated value

**Examples:**

```yaml
# Override is used (no example or default)
properties:
  code:
    type: integer
# With FieldOverrides: {"code": 400}
# Generated: 400
```

```yaml
# Default takes precedence over override
properties:
  code:
    type: integer
    default: 200
# With FieldOverrides: {"code": 400}
# Generated: 200 (default wins)
```

```yaml
# Example takes precedence over everything
properties:
  code:
    type: integer
    example: 201
# With FieldOverrides: {"code": 400}
# Generated: 201 (example wins)
```

```yaml
# Override takes precedence over heuristics
properties:
  message:
    type: string
# With FieldOverrides: {"message": "Custom error"}
# Generated: "Custom error" (override wins over "This is a message" heuristic)
```

### Type Mismatch Errors

If an override value doesn't match the schema type, an error is returned:

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    FieldOverrides: map[string]interface{}{
        "code": "not a number",  // Type mismatch
    },
    IncludeAll: true,
})
// err: "field override for 'code' has wrong type"
```

**Type matching rules:**
- Integer fields: Must be numeric and whole number (float64 without decimal portion)
- Number fields: Must be numeric (int or float64)
- String fields: Must be string
- Boolean fields: Must be boolean

### Limitations

**Exact name matching only:** Overrides use exact field name matching. Wildcards, patterns, or partial matches are not supported.

**Case-sensitive:** Field names must match exactly, including case: `{"Code": 400}` will NOT match a field named `"code"`.

**No path-based overrides:** Cannot target specific fields in specific schemas. Override applies to ALL fields with matching name.

**No nested overrides:** Cannot override nested object properties with dot notation like `"address.city"`.

## Format Support

The generator recognizes common OpenAPI string formats and generates appropriate values:

| Format | Generated Value | Example |
|--------|----------------|---------|
| `email` | Email address | `user@example.com` |
| `uuid` | UUID v4 | `123e4567-e89b-12d3-a456-426614174000` |
| `uri` / `url` | URL | `https://example.com` |
| `date` | ISO 8601 date | `2024-01-15` |
| `date-time` | ISO 8601 timestamp | `2024-01-15T10:30:00Z` |
| `hostname` | Hostname | `example.com` |

**OpenAPI:**
```yaml
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
        website:
          type: string
          format: uri
        createdAt:
          type: string
          format: date-time
```

**Generated Example:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "email": "user@example.com",
  "website": "https://example.com",
  "createdAt": "2024-01-15T10:30:00Z"
}
```

### Format with Length Constraints

When formats are combined with length constraints, the format template is adjusted:

**OpenAPI:**
```yaml
properties:
  shortEmail:
    type: string
    format: email
    maxLength: 10
```

**Generated:** `"user@ex..."` (truncated to 10 characters)

**OpenAPI:**
```yaml
properties:
  longId:
    type: string
    format: uuid
    minLength: 50
```

**Generated:** `"123e4567-e89b-12d3-a456-426614174000xxxxxxxxxx"` (padded to 50 characters)

## Circular Reference Handling

Circular references are automatically detected and broken to prevent infinite recursion.

### Example: Self-Referencing Schema

**OpenAPI:**
```yaml
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
```

**Generated Example (MaxDepth: 2):**
```json
{
  "name": "SomeRandomName",
  "friends": [
    {
      "name": "AnotherName",
      "friends": []
    }
  ]
}
```

At depth 2, the nested `friends` array is empty to prevent infinite recursion.

### Example: Mutual References

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
          $ref: '#/components/schemas/Address'

    Address:
      type: object
      properties:
        street:
          type: string
        resident:
          $ref: '#/components/schemas/User'
```

**Generated Example (MaxDepth: 3):**
```json
{
  "name": "John",
  "address": {
    "street": "123 Main St",
    "resident": {
      "name": "Jane"
    }
  }
}
```

The circular reference from `Address.resident` back to `User` is detected and the nested `User.address` field is omitted.

### How It Works

1. **Path Tracking**: The generator maintains a path of schema names currently being generated (e.g., `["User", "Address", "User"]`)
2. **Circular Detection**: Before generating a schema, it checks if the schema name is already in the path
3. **Breaking the Cycle**: If circular, the field is omitted (for objects) or the array is empty (for arrays)
4. **Depth Limiting**: Additionally, a depth counter prevents deep nesting even without circular references

## Complex Type Examples

### Objects with Nested Objects

**OpenAPI:**
```yaml
components:
  schemas:
    Address:
      type: object
      properties:
        street:
          type: string
          minLength: 5
        city:
          type: string
          minLength: 2

    User:
      type: object
      properties:
        name:
          type: string
        address:
          $ref: '#/components/schemas/Address'
```

**Generated Example:**
```json
{
  "name": "RandomName",
  "address": {
    "street": "12345",
    "city": "AB"
  }
}
```

### Arrays of Objects

**OpenAPI:**
```yaml
components:
  schemas:
    Tag:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string

    Article:
      type: object
      properties:
        title:
          type: string
        tags:
          type: array
          items:
            $ref: '#/components/schemas/Tag'
          minItems: 1
          maxItems: 3
```

**Generated Example:**
```json
{
  "title": "Article Title",
  "tags": [
    {
      "id": 0,
      "name": "tag1"
    }
  ]
}
```

### Mixed Constraints

**OpenAPI:**
```yaml
components:
  schemas:
    Product:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
          minLength: 3
          maxLength: 50
        price:
          type: number
          minimum: 0.01
          maximum: 9999.99
        categories:
          type: array
          items:
            type: string
            enum:
              - electronics
              - clothing
              - books
          minItems: 1
          maxItems: 3
        inStock:
          type: boolean
```

**Generated Example:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "ProductName123",
  "price": 4523.67,
  "categories": ["electronics"],
  "inStock": true
}
```

## Error Handling

The generator returns errors for invalid schemas or options:

### Invalid Constraints

```go
openapi := []byte(`openapi: 3.0.0
components:
  schemas:
    Invalid:
      type: object
      properties:
        age:
          type: integer
          minimum: 100
          maximum: 50  # min > max
`)

result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
})
// err: "invalid schema: minimum > maximum"
```

### Missing Options

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: false,
    SchemaNames: []string{}, // Empty and IncludeAll is false
})
// err: "must specify SchemaNames or set IncludeAll"
```

### Invalid Array Schema

```go
openapi := []byte(`openapi: 3.0.0
components:
  schemas:
    Invalid:
      type: object
      properties:
        items:
          type: array
          # Missing 'items' specification
`)

result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
})
// err: "array schema missing items"
```

## Limitations

The example generator has the following limitations:

### Not Supported

- **Pattern Matching**: The `pattern` field (regex) is ignored. Only `format` is used for string generation.
- **Discriminated Unions**: `oneOf` schemas are not supported for example generation.
- **AllOf/AnyOf/Not**: Schema composition operators are not supported.
- **AdditionalProperties**: Map types via `additionalProperties` are not generated.
- **Nested Arrays**: Arrays of arrays (e.g., `array` of `array`) are not supported.
- **Multiple Examples**: Only one example per schema is generated (no `examples` array support).

### Behavior Notes

- **Required Fields**: All properties are generated, regardless of the `required` array. Both required and optional fields appear in examples.
- **Nullable Fields**: The `nullable` field is ignored. Fields are never set to `null` in examples.
- **Random Strings**: Strings without format or pattern constraints generate random alphanumeric values.
- **Boolean Values**: Boolean fields generate random `true`/`false` values unless `default` or `example` is specified.

## Best Practices

1. **Use Deterministic Seeds for Tests**: When using examples in tests, always specify a `Seed` value for reproducible results.

2. **Set Appropriate MaxDepth**: For schemas with deep nesting or circular references, adjust `MaxDepth` to control output size.

3. **Provide Example Values**: For critical fields, use the `example` field in your schema to ensure specific values appear in generated examples.

4. **Use Format Annotations**: Leverage `format` fields (`email`, `uuid`, `date-time`) to generate realistic string values.

5. **Validate Generated Examples**: After generation, unmarshal the JSON and validate it matches your expectations.

6. **Test Constraint Handling**: If your schema has complex constraints (min/max, minLength/maxLength), verify the generated examples honor them.

## Integration with Other Tools

### Populating OpenAPI Documentation

Generated examples can be injected into OpenAPI documentation tools:

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    IncludeAll: true,
    Seed:       12345,
})

// Inject into Swagger UI or Redoc configuration
for schemaName, exampleJSON := range result.Examples {
    // Add to documentation config
    fmt.Printf("Schema: %s\nExample: %s\n\n", schemaName, string(exampleJSON))
}
```

### Test Fixture Generation

Use examples as test fixtures:

```go
result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
    SchemaNames: []string{"User", "Product"},
    Seed:        99999, // Deterministic for tests
})

// Use in test cases
userJSON := result.Examples["User"]
var user User
json.Unmarshal(userJSON, &user)

// Test with generated user
response := apiClient.CreateUser(user)
assert.NotNil(t, response)
```

### Mock Service Responses

Generate mock API responses:

```go
http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
    result, _ := conv.ConvertToExamples(openapi, conv.ExampleOptions{
        SchemaNames: []string{"User"},
        Seed:        time.Now().UnixNano(), // Different each time
    })

    w.Header().Set("Content-Type", "application/json")
    w.Write(result.Examples["User"])
})
```

## Further Reading

- [OpenAPI Schema Object](https://swagger.io/specification/#schema-object) - Schema constraint documentation
- [JSON Schema Validation](https://json-schema.org/draft/2020-12/json-schema-validation.html) - Validation keyword reference
- [OpenAPI String Formats](https://swagger.io/docs/specification/data-models/data-types/#string) - Supported format values
