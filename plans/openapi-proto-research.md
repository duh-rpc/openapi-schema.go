> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# Technical Specification: OpenAPI 3.0 to Protobuf 3 Conversion Library

## Review Status
- Review Cycles Completed: 1
- Final Approval: Ready for implementation
- Outstanding Concerns: None

## 1. Overview

This specification defines a Go library that converts OpenAPI 3.0 schema definitions to Protocol Buffer 3 (proto3) format. The library will parse OpenAPI YAML specifications and generate corresponding `.proto` files with proper type mappings, JSON field name annotations, and protobuf conventions.

**Business Value:**
- Enables protobuf message generation from OpenAPI specifications (OpenAPI as source of truth)
- Provides consistent type definitions across REST API specifications and protobuf messages
- Automates conversion process, reducing manual effort and errors
- Supports integration into existing OpenAPI-based code generation workflows

## 2. Current State Analysis

**Affected modules:** New greenfield project (empty directory)

**Current behavior:** No existing implementation

**Relevant ADRs reviewed:** None (new project with no ADRs)

**Technical debt identified:** None (greenfield project)

## 3. Architectural Context

### Relevant ADRs
None - this is a new project with no existing architectural decisions.

### Architectural Principles
Since this is a greenfield project, the following principles should guide implementation:
- **Single Responsibility**: Library focuses solely on OpenAPI → Protobuf conversion
- **Dependency Management**: Minimal dependencies, leverage existing battle-tested libraries
- **Idempotency**: Repeated conversions of the same input produce identical output content
- **Fail-Fast**: Clear error messages for unsupported or invalid OpenAPI constructs
- **Simplicity Over Completeness**: Focus on common use cases, explicitly error on unsupported features

## 4. Requirements

### Functional Requirements

**REQ-001: Parse OpenAPI 3.0 YAML Files**
- Accept OpenAPI 3.0 YAML as input (bytes)
- Use `github.com/pb33f/libopenapi` for parsing
- Support schemas defined in `components/schemas` only
- Schemas defined elsewhere (parameters, request bodies, responses) are out of scope and should be ignored
- Acceptance: Successfully parse valid OpenAPI 3.0 YAML documents and extract components/schemas

**REQ-002: Convert OpenAPI Types to Proto3 Types**
- Map OpenAPI primitive types to proto3 scalar types
- Support simple types: string, integer (int32/int64), number (float/double), boolean
- Support complex types: objects (messages), arrays (repeated fields)
- Support enums with proper proto3 conventions
- Support `$ref` references to other schemas
- Acceptance: All supported OpenAPI types correctly map to proto3 equivalents

**REQ-003: Generate JSON Field Name Annotations**
- Convert OpenAPI field names to snake_case for proto field names
- Add `json_name` option when original OpenAPI name differs from generated snake_case name
- Preserve original OpenAPI field names for JSON serialization compatibility
- If OpenAPI field is already snake_case, no `json_name` annotation is needed
- Acceptance: Generated proto includes `[json_name = "originalName"]` only where necessary

**REQ-004: Preserve Field Ordering**
- Assign proto3 field numbers based on appearance order in OpenAPI YAML
- Use sequential numbering starting from 1 within each message
- Use libopenapi's `FromOldest()` method which preserves YAML insertion order
- Maintain deterministic ordering across conversions
- Acceptance: Field numbers match property order in OpenAPI YAML file

**REQ-005: Handle Enum Conversion**
- Convert OpenAPI enum values to proto3 enum definitions
- Generate enums as top-level definitions (never nested in messages)
- Add `ENUM_NAME_UNSPECIFIED = 0` as first enum value
- Shift original OpenAPI enum values to start at 1
- Preserve original enum value ordering (after UNSPECIFIED)
- For inline enum properties, hoist to top-level and derive name from property path
- Acceptance: All enums include UNSPECIFIED=0 and preserve original value order

**REQ-006: Convert Descriptions to Comments**
- Extract `description` fields from OpenAPI schemas and properties
- Generate proto3 comments above corresponding messages/fields
- Multi-line descriptions should have each line prefixed with `//`
- Preserve formatting where reasonable (no specific line wrapping required for v1)
- Acceptance: All OpenAPI descriptions appear as comments in proto files

**REQ-007: Handle Nested Objects**
- Generate nested message definitions for inline object properties
- Generate top-level message definitions for schemas in `components/schemas`
- Derive nested message names from property names (converted to PascalCase)
- Acceptance: Inline objects become nested messages, named schemas become top-level

**REQ-008: Resolve Schema References ($ref)**
- Detect `$ref` references in property definitions
- Resolve internal references (`#/components/schemas/...`) using libopenapi's resolution
- Use referenced schema name as field type in protobuf
- External file references (e.g., `./other-file.yaml#/...`) should return a clear error
- Acceptance: Properties with internal `$ref` correctly use referenced message type; external refs error

**REQ-009: Generate Valid Proto3 Syntax**
- Include `syntax = "proto3";` declaration
- Include package declaration from user-provided option
- Generate syntactically valid `.proto` files
- Follow proto3 naming conventions (snake_case fields, PascalCase messages/enums)
- Acceptance: Generated content is syntactically valid proto3

**REQ-010: Library API**
- Provide library (not CLI tool) that can be imported
- Accept package name as parameter
- Accept OpenAPI document bytes as input
- Return generated proto3 content as bytes
- Return single `.proto` file content (not multiple files)
- Return structured errors with context (schema name, field name, issue description)
- Acceptance: Can be imported and used from another Go project

**REQ-011: Handle Required/Optional Fields**
- Ignore OpenAPI `required` array (proto3 fields are optional by default)
- Ignore `nullable: true` directive (same handling as optional)
- Do not use proto3 `optional` keyword
- Do not use wrapper types (`google.protobuf.*Value`)
- Do not add comments distinguishing required vs optional
- Acceptance: All fields generated without required/optional distinction

**REQ-012: Handle Array Types**
- Convert OpenAPI arrays to proto3 `repeated` fields
- Resolve array item types:
  - If item is `$ref`: use referenced message type
  - If item is inline object: generate nested message for item type
  - If item is scalar: use proto3 scalar type
  - If item is enum: use enum type (hoisted to top-level)
- Nested arrays (array of arrays) should return an error (unsupported)
- Acceptance: Arrays correctly map to repeated fields with proper item types

**REQ-013: Error Handling for Unsupported Features**
- Return clear errors for unsupported OpenAPI constructs
- Unsupported features include:
  - `allOf`, `anyOf`, `oneOf`, `not` (schema composition)
  - Nested arrays (array items with type: array)
  - External `$ref` (different files)
  - Validation constraints (minimum, maximum, pattern, etc.) - ignored, not error
  - Top-level schemas that are primitives or arrays (not objects or enums)
  - Properties with no `type` specified and no `$ref`
  - Circular references (if libopenapi doesn't resolve them)
- Error messages should include schema name and specific issue
- Acceptance: Encountering unsupported features returns clear, actionable error messages

**REQ-014: Handle Naming Conflicts**
- Detect duplicate message/enum names after conversion to PascalCase
- When conflict detected, append numeric suffix (`_2`, `_3`, etc.) to later occurrence
- Track all generated names to prevent conflicts
- Acceptance: No duplicate message or enum names in generated proto

### Non-Functional Requirements

**Performance:**
- Convert OpenAPI specs with <100 top-level schemas in `components/schemas` in <1 second
- Memory efficient for typical use cases (no specific streaming requirement for v1)

**Reliability:**
- Idempotent: same input OpenAPI bytes always produce identical output content bytes
- Deterministic: field numbers and message ordering are consistent

**Usability:**
- Clear error messages for unsupported OpenAPI features
- Helpful error context (schema name, property name, issue description)
- Errors should guide user toward fixing OpenAPI spec or understanding limitation

**Maintainability:**
- Well-documented code
- Separation of concerns (parsing, mapping, generation)
- Testable architecture with unit tests for each component

## 5. Technical Approach

### Chosen Solution
Build a focused Go library using `github.com/pb33f/libopenapi` for parsing OpenAPI 3.0 documents, with custom conversion logic to map schemas to proto3 message definitions. Generate output using Go text templates.

### Rationale
- **libopenapi chosen over kin-openapi**: Already in use in another project (user requirement)
- **Custom conversion over existing tools**: Existing tools either lack `json_name` support (openapi2proto) or require JVM (openapi-generator)
- **Library over CLI**: Designed for integration into code generation pipelines
- **Template-based generation**: Simplifies proto3 output formatting and maintenance
- **Single file output**: Simplifies initial implementation; multi-file support can be future enhancement
- **No well-known types**: Avoid complexity of imports; use simple string types for dates/timestamps

### ADR Alignment
Not applicable (greenfield project with no existing ADRs)

### Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Public API Layer                        │
│  - Convert(openapi []byte, packageName string) ([]byte, err) │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                    Parser Component                          │
│  - Uses libopenapi to parse OpenAPI 3.0 YAML                │
│  - Extracts components/schemas using FromOldest()           │
│  - Iterates schemas in order                                │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                  Type Mapper Component                       │
│  - Maps OpenAPI types → proto3 types                        │
│  - Detects $ref and resolves to message types               │
│  - Handles nested objects (inline vs referenced)            │
│  - Assigns field numbers based on appearance order          │
│  - Detects unsupported features and returns errors          │
│  - Uses Naming Converter for all name transformations       │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                 Naming Converter Component                   │
│  - Converts camelCase → snake_case                          │
│  - Converts property names → PascalCase (for messages)      │
│  - Detects naming differences for json_name                 │
│  - Generates enum value names (UPPERCASE_SNAKE_CASE)        │
│  - Tracks names for conflict detection                      │
│  - Handles duplicate name resolution (numeric suffixes)     │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                  Proto Generator Component                   │
│  - Uses Go text/template to format .proto output            │
│  - Generates syntax, package, messages, enums               │
│  - Adds comments from descriptions                          │
│  - Formats json_name annotations                            │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                 Output: Single .proto file (bytes)           │
└─────────────────────────────────────────────────────────────┘
```

### Type Mapping Specification

| OpenAPI 3.0 Type | OpenAPI Format | Proto3 Type | Notes |
|------------------|----------------|-------------|-------|
| `integer` | `int32` | `int32` | Default for integer without format |
| `integer` | `int64` | `int64` | Explicit 64-bit integers |
| `integer` | (none) | `int32` | Default when no format specified |
| `number` | `float` | `float` | Single-precision floating point |
| `number` | `double` | `double` | Double-precision floating point |
| `number` | (none) | `double` | Default for number without format |
| `string` | (none) | `string` | UTF-8 encoded strings |
| `string` | `byte` | `bytes` | Base64-encoded in OpenAPI, bytes in proto |
| `string` | `binary` | `bytes` | Binary data |
| `string` | `date` | `string` | No native date (not using google.protobuf.Timestamp) [1] |
| `string` | `date-time` | `string` | No native timestamp (not using google.protobuf.Timestamp) [1] |
| `boolean` | - | `bool` | Boolean values |
| `array` | - | `repeated <type>` | See REQ-012 for item type resolution |
| `object` | - | `message` | Nested or top-level message; see REQ-007 |
| `enum` | - | `enum` | Top-level enum with UNSPECIFIED; see REQ-005 |

**[1] Rationale for string types:** Using `google.protobuf.Timestamp` would require imports and add complexity. For v1, simple string types maintain OpenAPI compatibility and simplify implementation.

**Unsupported types (return error):**
- `allOf`, `anyOf`, `oneOf`, `not`
- Properties with no `type` and no `$ref`
- Top-level schemas that are not objects or enums

**Ignored OpenAPI features (no error, not converted):**
- Validation constraints: `minimum`, `maximum`, `minLength`, `maxLength`, `pattern`, `multipleOf`, etc.
- `required` array
- `nullable` directive
- `readOnly`, `writeOnly`
- `deprecated`
- `example`, `examples`
- `default`
- `xml` metadata

### Field Numbering Strategy

1. **Iterate properties in order**: Use libopenapi's `FromOldest()` to maintain YAML insertion order
2. **Sequential assignment**: Start at 1, increment for each field within a message
3. **Deterministic**: Same schema always produces same field numbers
4. **No gaps**: Sequential numbering without skips
5. **No custom numbering**: Do not support OpenAPI extensions for custom field numbers (future enhancement)
6. **Independent numbering**: Each message has its own field numbering starting from 1

### Naming Convention Handling

**Message Names:**
- Top-level schemas: Use schema key from `components/schemas` as-is (assumed to be PascalCase)
- Nested messages: Convert property name to PascalCase (e.g., `shippingAddress` → `ShippingAddress`)
- Conflicts: Append `_2`, `_3`, etc. to later occurrences

**Field Names:**
- Convert OpenAPI property names to snake_case for proto field names
- Add `json_name` annotation only if original name differs from snake_case version
- Preserve exact original OpenAPI name in `json_name` annotation

**Examples:**
```
OpenAPI: userId     → Proto: user_id [json_name = "userId"]
OpenAPI: email      → Proto: email (no json_name annotation)
OpenAPI: user_id    → Proto: user_id (no json_name annotation)
OpenAPI: HTTPStatus → Proto: http_status [json_name = "HTTPStatus"]
```

**Enum Names:**
- Top-level schema enums: Use schema key from `components/schemas`
- Inline enums: Derive from property path (e.g., property `status` → enum `Status`)
- Values: UPPERCASE_SNAKE_CASE with enum name prefix
- First value: `{ENUM_NAME}_UNSPECIFIED = 0`

### Enum Handling Specification

**Input OpenAPI (top-level):**
```yaml
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - pending
```

**Output Proto3:**
```protobuf
enum Status {
  STATUS_UNSPECIFIED = 0;  // Auto-generated
  STATUS_ACTIVE = 1;       // Shifted from position 0
  STATUS_INACTIVE = 2;     // Shifted from position 1
  STATUS_PENDING = 3;      // Shifted from position 2
}
```

**Input OpenAPI (inline enum):**
```yaml
components:
  schemas:
    User:
      type: object
      properties:
        role:
          type: string
          enum:
            - admin
            - user
            - guest
```

**Output Proto3:**
```protobuf
// Hoisted to top-level
enum Role {
  ROLE_UNSPECIFIED = 0;
  ROLE_ADMIN = 1;
  ROLE_USER = 2;
  ROLE_GUEST = 3;
}

message User {
  Role role = 1;
}
```

**Rules:**
1. Always generate `{ENUM_NAME}_UNSPECIFIED = 0` as first value
2. Original enum values start at 1
3. Preserve original ordering (after UNSPECIFIED)
4. Convert values to UPPERCASE_SNAKE_CASE with enum name prefix
5. All enums are top-level (never nested in messages, always hoisted)
6. Inline enum name derived from property name in PascalCase

### Nested vs Top-Level Messages

**Top-Level Messages:**
- All object schemas in `components/schemas`
- All enum definitions (including those hoisted from inline properties)

**Nested Messages:**
- Inline object types within properties (no `$ref`)
- Only when `type: object` appears directly in property definition

**Error Cases:**
- Top-level schema in `components/schemas` that is primitive (string, integer, etc.): error
- Top-level schema that is array: error
- Nested arrays: error

**Example:**

OpenAPI:
```yaml
components:
  schemas:
    User:
      type: object
      properties:
        userId:
          type: string
        preferences:    # Inline object → nested message
          type: object
          properties:
            theme:
              type: string
            notifications:
              type: boolean
```

Proto3:
```protobuf
message User {
  string user_id = 1 [json_name = "userId"];

  message Preferences {
    string theme = 1;
    bool notifications = 2;
  }

  Preferences preferences = 2;
}
```

### Reference Resolution ($ref)

**Approach:**
- Use libopenapi's automatic reference resolution
- When encountering a property, check if it's a reference to another schema
- If internal reference: use the referenced schema name as the message type
- If external reference: return error
- If inline object: generate nested message

**Supported:**
- Internal references: `#/components/schemas/Address`

**Not Supported (returns error):**
- External file references: `./other-file.yaml#/components/schemas/Address`
- Relative references: `../common/schemas.yaml#/definitions/Address`
- URL references: `https://example.com/schemas.yaml#/Address`

**Example:**

OpenAPI:
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
        userId:
          type: string
        homeAddress:
          $ref: '#/components/schemas/Address'
        workAddress:
          $ref: '#/components/schemas/Address'
```

Proto3:
```protobuf
message Address {
  string street = 1;
  string city = 2;
}

message User {
  string user_id = 1 [json_name = "userId"];
  Address home_address = 2 [json_name = "homeAddress"];
  Address work_address = 3 [json_name = "workAddress"];
}
```

### Array Item Type Resolution

Arrays are converted to `repeated` fields. Item type resolution follows these rules:

**Scalar items:**
```yaml
tags:
  type: array
  items:
    type: string
```
→ `repeated string tags = 1;`

**Reference items:**
```yaml
addresses:
  type: array
  items:
    $ref: '#/components/schemas/Address'
```
→ `repeated Address addresses = 1;`

**Inline object items:**
```yaml
contacts:
  type: array
  items:
    type: object
    properties:
      name:
        type: string
      phone:
        type: string
```
→
```protobuf
message Contact {  // Generated nested message
  string name = 1;
  string phone = 2;
}

repeated Contact contacts = 1;
```

**Inline enum items:**
```yaml
statuses:
  type: array
  items:
    type: string
    enum: [active, inactive]
```
→
```protobuf
enum Status {  // Hoisted to top-level
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}

repeated Status statuses = 1;
```

**Nested arrays (unsupported - returns error):**
```yaml
matrix:
  type: array
  items:
    type: array
    items:
      type: integer
```
→ Error: "Nested arrays are not supported"

### Error Handling Specification

**Error Structure:**
Errors should be structured Go errors with clear messages including context:
```go
fmt.Errorf("schema '%s': unsupported feature '%s' in property '%s'", schemaName, feature, propertyName)
```

**Error Categories:**

1. **Parse Errors:**
   - Invalid YAML syntax
   - Not a valid OpenAPI 3.0 document
   - libopenapi parsing failures

2. **Unsupported Feature Errors:**
   - Schema composition (`allOf`, `anyOf`, `oneOf`, `not`)
   - Nested arrays
   - External references
   - Top-level non-object/non-enum schemas
   - Properties with no type and no $ref

3. **Validation Errors:**
   - Empty package name
   - Empty OpenAPI input

**Error Behavior:**
- Fail-fast: Return error on first unsupported feature encountered
- Do not accumulate multiple errors (v1 simplification)
- Provide actionable error messages that help user understand what to fix

**Example Error Messages:**
```
"schema 'User': property 'metadata' uses 'anyOf' which is not supported"
"schema 'Config': nested arrays are not supported in property 'matrix'"
"schema 'User': property 'address' references external file which is not supported"
"schema 'StringList': top-level array schemas are not supported, only objects and enums"
```

## 6. Dependencies and Impacts

### External Dependencies
- `github.com/pb33f/libopenapi` - OpenAPI 3.0/3.1 parser and validator
- Go standard library:
  - `text/template` - Proto file generation
  - `strings` - String manipulation for naming conversions
  - `fmt` - Error formatting
  - `bytes` - Output buffer

### Internal Dependencies
None (new greenfield project, no internal dependencies)

### Database Impacts
None (pure code generation, no persistence)

### Import Requirements
Generated proto files will NOT include any imports (no `google.protobuf.*` types). This is a v1 simplification.

## 7. Backward Compatibility

### Is this project in production?
- No - Breaking changes are permitted without migration

### Breaking Changes Allowed
- Yes - Breaking changes are permitted for this feature

### If Breaking Changes Are Allowed
Not applicable - new project with no existing users

## 8. Testing Strategy

### Testing Approach

All tests MUST be in the test package format: `package <package>_test` (not `package <package>`).
Test names should be in camelCase starting with a capital letter.
Use `github.com/stretchr/testify/require` for critical assertions and `github.com/stretchr/testify/assert` for non-critical assertions.

Tests should follow a **functional/end-to-end style**: Given an OpenAPI spec input, assert the exact proto file output generated.

### Functional Test Cases

Each test should follow the pattern:
```go
func TestScenarioName(t *testing.T) {
  for _, test := range []struct {
    name     string
    given    string
    expected string
  }{
    {
      name:     "simple string field",
      given:    `<OpenAPI YAML for string field>`,
      expected: `<Expected Proto3 output for string>`,
    },
    {
      name:     "integer field with validation",
      given:    `<OpenAPI YAML for integer field>`,
      expected: `<Expected Proto3 output for integer>`,
    },
  } {
    t.Run(test.name, func(t *testing.T) {
      result, err := Convert([]byte(test.given), "testpkg")
      require.NoError(t, err)
      assert.Equal(t, test.expected, string(result))
    })
  }
}
```

**Test 1: SimpleScalarTypes**
- **Given**: OpenAPI schema with string, integer (int32/int64), number (float/double), boolean fields
- **Expected**: Proto message with corresponding scalar types (string, int32, int64, float, double, bool)
- **Validates**: REQ-002 (basic type mapping)

**Test 2: FieldNamingWithJsonName**
- **Given**: OpenAPI schema with camelCase field names (userId, emailAddress, HTTPStatus)
- **Expected**: Proto message with snake_case fields and json_name annotations
- **Validates**: REQ-003 (json_name generation), naming conventions

**Test 3: FieldNamingAlreadySnakeCase**
- **Given**: OpenAPI schema with snake_case field names (user_id, email_address)
- **Expected**: Proto message with same field names, no json_name annotations
- **Validates**: REQ-003 (json_name omitted when not needed)

**Test 4: FieldOrderingPreserved**
- **Given**: OpenAPI schema with properties in specific order
- **Expected**: Proto message with sequential field numbers (1, 2, 3...) matching input order
- **Validates**: REQ-004 (field ordering and numbering)

**Test 5: TopLevelEnum**
- **Given**: OpenAPI schema defining an enum in components/schemas
- **Expected**: Proto enum with ENUM_NAME_UNSPECIFIED=0, values shifted +1 with prefix
- **Validates**: REQ-005 (enum conversion, UNSPECIFIED, value shifting)

**Test 6: InlineEnumHoisted**
- **Given**: OpenAPI schema with inline enum property
- **Expected**: Proto with top-level enum (hoisted) and message referencing it
- **Validates**: REQ-005 (inline enum hoisting)

**Test 7: DescriptionsAsComments**
- **Given**: OpenAPI schema with descriptions on message and fields
- **Expected**: Proto with // comments above message and fields
- **Validates**: REQ-006 (comment generation)

**Test 8: MultiLineDescriptions**
- **Given**: OpenAPI schema with multi-line description
- **Expected**: Proto with each line prefixed with //
- **Validates**: REQ-006 (multi-line comment formatting)

**Test 9: NestedObjectInline**
- **Given**: OpenAPI schema with inline object property
- **Expected**: Proto with nested message definition inside parent
- **Validates**: REQ-007 (nested message generation)

**Test 10: DeeplyNestedObjects**
- **Given**: OpenAPI schema with multiple levels of nested objects
- **Expected**: Proto with multiple levels of nested messages
- **Validates**: REQ-007 (deep nesting)

**Test 11: SchemaReference**
- **Given**: OpenAPI with two schemas, one referencing the other via $ref
- **Expected**: Proto with two top-level messages, one using the other as field type
- **Validates**: REQ-008 ($ref resolution)

**Test 12: MultipleReferencesToSameSchema**
- **Given**: OpenAPI schema with multiple fields referencing same schema
- **Expected**: Proto message with multiple fields of same message type
- **Validates**: REQ-008 (reusable references)

**Test 13: ArrayOfScalars**
- **Given**: OpenAPI schema with array of strings
- **Expected**: Proto with repeated string field
- **Validates**: REQ-012 (array → repeated with scalar items)

**Test 14: ArrayOfReferences**
- **Given**: OpenAPI schema with array of $ref to another schema
- **Expected**: Proto with repeated field of referenced message type
- **Validates**: REQ-012 (array with reference items)

**Test 15: ArrayOfInlineObjects**
- **Given**: OpenAPI schema with array of inline objects
- **Expected**: Proto with nested message for item type and repeated field
- **Validates**: REQ-012 (array with inline object items)

**Test 16: ArrayOfInlineEnums**
- **Given**: OpenAPI schema with array of inline enums
- **Expected**: Proto with top-level enum (hoisted) and repeated field
- **Validates**: REQ-012 (array with inline enum items)

**Test 17: PackageNameFromOption**
- **Given**: OpenAPI schema converted with package name "myapi"
- **Expected**: Proto with `package myapi;` declaration
- **Validates**: REQ-009, REQ-010 (package declaration)

**Test 18: RequiredFieldsIgnored**
- **Given**: OpenAPI schema with required array specifying mandatory fields
- **Expected**: Proto with all fields same (no optional keyword or distinction)
- **Validates**: REQ-011 (required ignored)

**Test 19: NullableFieldsIgnored**
- **Given**: OpenAPI schema with nullable: true on fields
- **Expected**: Proto with all fields same (no wrapper types)
- **Validates**: REQ-011 (nullable ignored)

**Test 20: DuplicateNamesResolved**
- **Given**: OpenAPI schemas that result in duplicate message names after conversion
- **Expected**: Proto with numeric suffixes (_2, _3) on later occurrences
- **Validates**: REQ-014 (naming conflict resolution)

**Test 21: CompleteRealWorldExample**
- **Given**: OpenAPI spec with mix of all supported features (enums, nested objects, arrays, refs, descriptions)
- **Expected**: Complete proto file with all features correctly converted
- **Validates**: End-to-end integration of all requirements

### Error Case Tests

**Test 22: ErrorOnAllOf**
- **Given**: OpenAPI schema using allOf
- **Expected**: Error with message indicating allOf is unsupported
- **Validates**: REQ-013 (unsupported schema composition)

**Test 23: ErrorOnAnyOf**
- **Given**: OpenAPI schema using anyOf
- **Expected**: Error with message indicating anyOf is unsupported
- **Validates**: REQ-013 (unsupported schema composition)

**Test 24: ErrorOnOneOf**
- **Given**: OpenAPI schema using oneOf
- **Expected**: Error with message indicating oneOf is unsupported
- **Validates**: REQ-013 (unsupported schema composition)

**Test 25: ErrorOnNestedArrays**
- **Given**: OpenAPI schema with array of arrays
- **Expected**: Error with message indicating nested arrays are unsupported
- **Validates**: REQ-012, REQ-013 (nested array error)

**Test 26: ErrorOnExternalRef**
- **Given**: OpenAPI schema with $ref to external file
- **Expected**: Error with message indicating external references are unsupported
- **Validates**: REQ-008, REQ-013 (external ref error)

**Test 27: ErrorOnTopLevelPrimitive**
- **Given**: OpenAPI with component/schema that is a primitive (string)
- **Expected**: Error with message indicating only objects and enums are supported at top level
- **Validates**: REQ-013 (top-level primitive error)

**Test 28: ErrorOnTopLevelArray**
- **Given**: OpenAPI with component/schema that is an array
- **Expected**: Error with message indicating only objects and enums are supported at top level
- **Validates**: REQ-013 (top-level array error)

**Test 29: ErrorOnPropertyWithNoType**
- **Given**: OpenAPI schema with property that has no type and no $ref
- **Expected**: Error with message indicating property must have type or $ref
- **Validates**: REQ-013 (missing type error)

**Test 30: ErrorContextIncludesSchemaName**
- **Given**: Any error scenario
- **Expected**: Error message includes schema name and property name for context
- **Validates**: REQ-010, REQ-013 (error context)

### Optional Validation Tests

**Test 31: ProtocCompilation (optional)**
- **Given**: Generated proto file from any successful test
- **When**: Compiled with `protoc` (if available in CI)
- **Expected**: Compilation succeeds with no errors
- **Validates**: REQ-009 (syntactically valid proto3)
- **Note**: Only run if `protoc` is available; not required for passing tests

### Test Organization

Group tests by feature area in separate test files:
- `scalar_types_test.go` - Tests 1-4 (basic types and naming)
- `enums_test.go` - Tests 5-6 (enum handling)
- `comments_test.go` - Tests 7-8 (description conversion)
- `nested_messages_test.go` - Tests 9-10 (nested objects)
- `references_test.go` - Tests 11-12 ($ref handling)
- `arrays_test.go` - Tests 13-16 (array conversions)
- `metadata_test.go` - Tests 17-20 (package, required/nullable, naming conflicts)
- `integration_test.go` - Test 21 (complete real-world example)
- `errors_test.go` - Tests 22-30 (error scenarios)

### User Acceptance Criteria

1. Generated `.proto` content is syntactically valid proto3
2. Field names with `json_name` annotations match original OpenAPI names exactly
3. Field numbers are sequential (1, 2, 3...) and deterministic
4. All enums include `{NAME}_UNSPECIFIED = 0` as first value
5. Inline objects appear as nested messages within their parent
6. Top-level schemas appear as top-level messages
7. Referenced schemas (via $ref) use correct message type names
8. Descriptions appear as `//` comments above definitions
9. Package name matches user-provided option
10. Required/optional/nullable directives are ignored (all fields same)
11. Unsupported features return clear error messages
12. Duplicate names are resolved with numeric suffixes

## 9. Implementation Notes

### Estimated Complexity
**Medium** - Well-defined problem with clear type mappings, leveraging existing parser library. Added complexity from comprehensive error handling and edge cases.

### Suggested Implementation Order

1. **Phase 1: Core Library Structure**
   - Define public API (`Convert` function signature)
   - Set up libopenapi integration
   - Parse OpenAPI documents and extract schemas from components/schemas
   - Handle basic parse errors

2. **Phase 2: Naming Converter Component**
   - Implement camelCase → snake_case conversion
   - Implement property name → PascalCase for messages
   - Implement json_name detection logic
   - Implement duplicate name tracking and suffix generation
   - Unit test thoroughly (foundation for other components)

3. **Phase 3: Simple Type Mapping**
   - Implement scalar type conversion (string, integer, number, boolean)
   - Detect unsupported type features (return errors)
   - Generate basic messages with simple fields
   - Implement field numbering (sequential from 1)

4. **Phase 4: Enums**
   - Implement enum type detection (top-level and inline)
   - Generate UNSPECIFIED values
   - Implement value shifting and naming (with prefix)
   - Implement inline enum hoisting to top-level

5. **Phase 5: Complex Types - Arrays**
   - Implement array → repeated conversion
   - Implement item type resolution (scalar, ref, inline object, inline enum)
   - Detect and error on nested arrays

6. **Phase 6: Complex Types - Objects**
   - Implement object → message conversion
   - Implement nested message generation for inline objects
   - Distinguish inline vs top-level schemas

7. **Phase 7: References**
   - Implement $ref detection
   - Use libopenapi's resolution for internal refs
   - Map to message types
   - Detect and error on external refs

8. **Phase 8: Comments and Documentation**
   - Extract descriptions from schemas and properties
   - Generate proto3 `//` comments
   - Format multi-line descriptions (each line prefixed with `//`)

9. **Phase 9: Output Generation**
   - Create proto3 template
   - Generate complete .proto file content
   - Add syntax and package declarations
   - Format json_name annotations
   - Return as bytes

10. **Phase 10: Error Handling Polish**
    - Ensure all error paths have clear messages
    - Add schema/property context to all errors
    - Test error scenarios comprehensively

### Code Style Considerations

Follow project CLAUDE.md guidelines:
- Use `const` for constants used more than once
- Prefer short variable names (1-2 words)
- Inline single-use variables into function calls
- Use `lo.ToPtr()` from `github.com/samber/lo` for pointer creation
- Format struct literals with visual tapering (longest lines first)
- Use `require` for critical assertions that halt tests, `assert` for non-critical
- Tests MUST be in `package <package>_test` format
- Test names in camelCase starting with capital letter
- Avoid test logging; use comments instead
- Be explicit in tests, don't apply DRY principle

### Rollback Strategy

Not applicable for library (users control integration via go.mod). If issues arise:
- Users can pin to previous version in `go.mod`
- Can revert to manual proto file creation
- No data migration needed (pure code generation)
- No file system state (library returns bytes, doesn't write files)

### Output Specification

The library returns generated proto3 content as `[]byte` (or `string`). It does NOT write files to disk.
The output is a single `.proto` file containing:
- `syntax = "proto3";` declaration
- `package <user-provided-name>;` declaration
- All top-level enum definitions (including hoisted inline enums)
- All top-level message definitions (from components/schemas)
- Nested messages defined within their parent messages
- Comments from OpenAPI descriptions

Example output structure:
```protobuf
syntax = "proto3";

package myapi;

// Description from OpenAPI if present
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}

// User represents a user account
message User {
  string user_id = 1 [json_name = "userId"];
  string email = 2;

  // Nested message for preferences
  message Preferences {
    string theme = 1;
    bool notifications = 2;
  }

  Preferences preferences = 3;
  Status status = 4;
}
```

## 10. ADR Recommendation

**Should a new ADR be created?**

Not immediately necessary since this is a new greenfield project. However, consider creating an ADR after initial implementation if:
- The library becomes critical to other services
- Significant architectural decisions emerge during implementation
- Multiple conversion strategies are evaluated and one is chosen

Potential future ADR topics:
- **ADR-001: Field Numbering Strategy** - Sequential vs alphabetical vs custom
- **ADR-002: Enum UNSPECIFIED Handling** - Always adding UNSPECIFIED=0
- **ADR-003: Nested vs Top-Level Message Placement** - Inline object strategy
- **ADR-004: Optional/Nullable Field Handling** - Decision to ignore vs use optional keyword
- **ADR-005: Date/Time Type Mapping** - String vs google.protobuf.Timestamp

## 11. Out of Scope / Limitations

Explicitly out of scope for v1:

**OpenAPI Features:**
- Schema composition: `allOf`, `anyOf`, `oneOf`, `not`
- Polymorphism and discriminators
- External references to other files
- Nested arrays (array of arrays)
- Validation constraints (preserved in future as proto options or comments)
- Links, callbacks, security schemes
- Servers, paths, operations (only schemas converted)
- Top-level schemas that are not objects or enums

**Proto3 Features:**
- Multiple file output (single file only)
- Imports (no `google.protobuf.*` or custom imports)
- Proto options (file-level, message-level, field-level custom options)
- Service definitions (gRPC services)
- Custom field numbering (sequential only)
- Reserved fields/numbers
- Map types (OpenAPI `additionalProperties` not supported)

**Future Enhancements:**
- Support for `additionalProperties` → proto3 `map<string, T>`
- Support for validation constraints as proto options
- Multi-file output (one file per schema or by namespace)
- Configurable field numbering strategy
- Support for `allOf` composition (merge properties)
- CLI wrapper around library
- Configuration file support

## 12. Open Questions

All questions from research phase have been resolved. No open questions remaining for implementation.
