> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# String Enum Handling Implementation Plan

## Overview

Implement proper handling of OpenAPI string enums to preserve JSON wire format compatibility. Currently, string enums are converted to protobuf enum types (integer-based), which breaks JSON API compatibility. This plan changes string enums to map to protobuf `string` type with enum constraints documented in comments.

## Current State Analysis

### How Enums Work Currently

1. **Detection**: `isEnumSchema()` in `internal/builder.go:232` returns true for any schema with `enum` values
2. **Building**: `buildEnum()` in `internal/builder.go:237` creates protobuf enum definitions for ALL enums (both string and integer)
3. **Inline enums**: Hoisted to top-level enum definitions via `ProtoType()` in `internal/mapper.go:62-69`
4. **Referenced enums**: When a property uses `$ref` to an enum schema, the enum type name is returned from `ProtoType()`

### Key Files

- `internal/builder.go` - Enum detection and building logic, Context struct, ProtoField struct
- `internal/mapper.go` - Type mapping, ProtoType() function, ResolveArrayItemType()
- `internal/generator.go` - Template rendering for protobuf output
- `internal/enums_test.go` - Enum-specific test cases
- `convert_test.go` - Integration tests including enum references

### Existing Test Cases

**String Enum Tests** (need updating):
- `TestTopLevelEnum` - Top-level Status enum (string)
- `TestEnumWithDashes` - Status with dashed values (string)
- `TestEnumWithDescription` - Status with description (string)
- `TestInlineEnum` - User.status inline enum (string)
- `TestMultipleInlineEnums` - User with status and role inline enums (string)
- `TestEnumAndMessageMixed` - Mix of enums and messages (string)
- `TestConvertCompleteExample` - OrderStatus (top-level, referenced) and Category (inline) - both string

**Integer Enum Tests** (should NOT change):
- `TestEnumWithNumbers` - Code enum with type: integer, enum: [200, 401, 404, 500]

## Desired End State

### Behavior Changes

**String Enums** (type: string + enum):
- Top-level schemas: Do NOT generate standalone protobuf definitions
- Inline in properties: Field type is `string` with comment annotation
- Referenced by $ref: Field type is `string` with comment annotation
- Comment format: `// enum: [value1, value2, value3]`
- Combine with description if present

**Integer Enums** (type: integer + enum):
- No changes - continue generating protobuf enum types

### Example Transformations

**Before:**
```yaml
OrderStatus:
  type: string
  description: Status of an order
  enum: [pending, confirmed, shipped]

Order:
  type: object
  properties:
    status:
      $ref: '#/components/schemas/OrderStatus'
```

```proto
enum OrderStatus {
  ORDER_STATUS_UNSPECIFIED = 0;
  ORDER_STATUS_PENDING = 1;
  ORDER_STATUS_CONFIRMED = 2;
  ORDER_STATUS_SHIPPED = 3;
}

message Order {
  OrderStatus status = 1 [json_name = "status"];
}
```

**After:**
```proto
message Order {
  // Status of an order
  // enum: [pending, confirmed, shipped]
  string status = 1 [json_name = "status"];
}
```

## What We're NOT Doing

- NOT changing integer enum handling (they remain protobuf enums)
- NOT adding validation logic (only documentation in comments)
- NOT supporting other enum types besides string and integer
- NOT adding backwards compatibility flags (breaking change accepted)
- NOT generating validation code or runtime checks

## Implementation Approach

### Core Strategy

1. Add helper functions to distinguish string vs integer enums
2. Modify function signatures to return enum values alongside type information
3. Skip protobuf enum generation for string enum schemas
4. When processing fields (inline or referenced), check if string enum and return "string" + enum values
5. Store enum values in ProtoField struct
6. Modify generator to include enum comments

### Technical Approach

- **Detection**: Create `isStringEnum()` and `isIntegerEnum()` helper functions
- **Signature changes**: Modify `ProtoType()` and `ResolveArrayItemType()` to return enum values
- **Field metadata**: Add `EnumValues []string` to ProtoField struct
- **Comment generation**: Update field rendering template to include enum comments

## Phase 1: Add String Enum Detection and Validation

### Overview
Add helper functions to distinguish string enums from integer enums, validate enum schemas, and extract enum values safely.

### Changes Required

#### 1. `internal/builder.go`

**File**: `internal/builder.go`

**Changes**: Add helper functions

```go
// isStringEnum returns true if schema is a string enum
func isStringEnum(schema *base.Schema) bool

// isIntegerEnum returns true if schema is an integer enum
func isIntegerEnum(schema *base.Schema) bool

// extractEnumValues extracts enum values as strings from schema
func extractEnumValues(schema *base.Schema) []string

// validateEnumSchema validates enum schema and returns error for unsupported cases
func validateEnumSchema(schema *base.Schema, schemaName string) error
```

**Function Responsibilities:**
- `isStringEnum()`: Check if schema has `type: string` and non-empty `enum` array
- `isIntegerEnum()`: Check if schema has `type: integer` and non-empty `enum` array
- `extractEnumValues()`: Extract enum values from schema.Enum, handling yaml.Node values, return string slice. For empty enum arrays, return empty slice (caller will skip generating enum comments).
- `validateEnumSchema()`: Validate enum schema and return errors for:
  - Enum with no type field → error "schema 'X': enum must have explicit type field"
  - Enum with null values → error "schema 'X': enum cannot contain null values"
  - Enum with mixed types → error "schema 'X': enum contains mixed types (string and integer)"
  - Empty enum array → return nil (no error, will be handled by empty EnumValues downstream)
  - Duplicate enum values → return nil (allowed, no deduplication)

**Testing Requirements:**
```go
func TestIsStringEnum(t *testing.T)
func TestIsIntegerEnum(t *testing.T)
func TestExtractEnumValues(t *testing.T)
func TestValidateEnumSchema(t *testing.T)
```

**Test Objectives:**
- Verify string enum detection for type: string + enum
- Verify integer enum detection for type: integer + enum
- Verify extractEnumValues correctly extracts values from yaml.Node
- Verify empty enum array returns empty slice (not error)
- Verify validateEnumSchema errors with proper messages for:
  - Enum without type field: "schema 'Status': enum must have explicit type field"
  - Enum with null values: "schema 'Status': enum cannot contain null values"
  - Enum with mixed types: "schema 'Code': enum contains mixed types (string and integer)"
- Verify validateEnumSchema passes for valid string and integer enums
- Verify validateEnumSchema passes for empty enum (returns nil)
- Verify validateEnumSchema passes for duplicate enum values (no deduplication)

**Context for Implementation:**
- Reference existing `isEnumSchema()` function at line 232
- Schema.Type is a string slice, use `contains()` helper
- Schema.Enum is a slice of yaml.Node pointers
- Look at `buildEnum()` lines 262-274 for enum value extraction pattern

#### 2. `internal/builder.go` - Update ProtoField

**File**: `internal/builder.go`

**Changes**: Add EnumValues field to ProtoField struct

```go
type ProtoField struct {
    Name        string
    Type        string
    Number      int
    JSONName    string
    Description string
    Repeated    bool
    EnumValues  []string  // NEW: Enum values for string enum fields
}
```

**Function Responsibilities:**
- Store enum values for string enum fields
- Empty slice for non-enum fields

**Context for Implementation:**
- ProtoField struct defined at lines 40-48
- Field will be populated by buildMessage when processing properties

## Phase 2: Modify Type Resolution

### Overview
Update ProtoType() and ResolveArrayItemType() to handle string enums differently from integer enums, returning enum values alongside type information.

### Changes Required

#### 1. `internal/mapper.go` - Update ProtoType signature

**File**: `internal/mapper.go`

**Changes**: Update function signature and return values

```go
// ProtoType returns the proto3 type for an OpenAPI schema.
// Returns type name, whether it's repeated, enum values (for string enums), and error.
func ProtoType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, bool, []string, error)
```

**Function Responsibilities:**
- Check if reference points to string enum schema → return "string" + enum values
- Check if inline string enum → return "string" + enum values
- Check if integer enum → call buildEnum() as before, return enum type name + nil
- For all other types → return type + nil enum values

**Reference handling** (lines 21-39):
- After resolving schema at line 25, check `isStringEnum(resolvedSchema)`
- If true, extract enum values and return ("string", false, enumValues, nil)
- Otherwise, extract type name and return (typeName, false, nil, nil)

**Inline enum handling** (lines 61-69):
- Check `isStringEnum(schema)` vs `isIntegerEnum(schema)`
- String enum: return ("string", false, extractEnumValues(schema), nil)
- Integer enum: call buildEnum() and return (enumName, false, nil, nil)

**Context for Implementation:**
- Current signature at line 14 returns (string, bool, error)
- Reference resolution at line 25: `resolvedSchema := propProxy.Schema()`
- Inline enum check at line 62: `if isEnumSchema(schema)`
- All callers need updating (in buildMessage, buildNestedMessage)

#### 2. `internal/mapper.go` - Update ResolveArrayItemType

**File**: `internal/mapper.go`

**Changes**: Update function signature and return values

```go
// ResolveArrayItemType determines the proto3 type for array items.
// Returns type name, enum values (for string enums), and error.
func ResolveArrayItemType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, []string, error)
```

**Function Responsibilities:**
- Handle array items that are string enums (inline and referenced)
- Handle array items that are integer enums
- Return enum values for string enum items

**Reference handling in arrays** (around line 144):
- If `itemProxy.IsReference()`: resolve reference and check if string enum
- Extract reference name, then resolve schema
- If resolved schema is string enum: return ("string", extractEnumValues(resolvedSchema), nil)
- Otherwise: return (refName, nil, nil) as before

**Inline enum in array handling** (lines 157-172):
- Check `isStringEnum(itemsSchema)` vs `isIntegerEnum(itemsSchema)`
- String enum: return ("string", extractEnumValues(itemsSchema), nil)
- Integer enum: validate property name is not plural, call buildEnum(), return (enumName, nil, nil)

**Context for Implementation:**
- Current signature at line 123 returns (string, error)
- Called from ProtoType() at line 44
- Array reference check at line 144: `if itemProxy.IsReference()`
- Array item enum check at line 157: `if isEnumSchema(itemsSchema)`

#### 3. `internal/builder.go` - Update buildMessage callers

**File**: `internal/builder.go`

**Changes**: Update ProtoType() call sites to handle enum values

```go
// In buildMessage() around line 187
protoType, repeated, enumValues, err := ProtoType(propSchema, propName, propProxy, ctx, msg)

// Update field description handling (lines 196-200)
// Clear description for inline objects and integer enums (hoisted)
// Keep description for string enums (not hoisted) and scalar types
fieldDescription := propSchema.Description
if len(propSchema.Type) > 0 && contains(propSchema.Type, "object") {
    fieldDescription = ""
}
// Integer enums are hoisted to top-level, clear description from field
// String enums stay as string fields, keep description on field
if isIntegerEnum(propSchema) {
    fieldDescription = ""
}

// When creating field around line 202
field := &ProtoField{
    Name:        protoFieldName,
    Type:        protoType,
    Number:      fieldNumber,
    Description: fieldDescription,
    Repeated:    repeated,
    JSONName:    propName,
    EnumValues:  enumValues,  // NEW
}
```

**Function Responsibilities:**
- Capture enum values from ProtoType()
- Pass enum values to ProtoField
- Keep description on field for string enums (no longer hoisted)

**Context for Implementation:**
- buildMessage at line 120
- ProtoType call at line 187
- Description clearing logic at lines 196-200 needs updating
- Field creation at lines 202-210
- Similar updates needed in buildNestedMessage() at line 328

#### 4. `internal/builder.go` - Update BuildMessages

**File**: `internal/builder.go`

**Changes**: Validate and skip buildEnum for string enum schemas

```go
// In BuildMessages() around line 102-108
if isEnumSchema(schema) {
    // Validate enum schema first
    if err := validateEnumSchema(schema, entry.Name); err != nil {
        return nil, err
    }

    // Check if it's a string enum - skip building protobuf enum
    if isStringEnum(schema) {
        continue
    }
    // Only build enum for integer enums
    _, err := buildEnum(entry.Name, entry.Proxy, ctx)
    if err != nil {
        return nil, err
    }
    continue
}
```

**Function Responsibilities:**
- Validate enum schemas for unsupported cases
- Skip creating protobuf enum definitions for string enum schemas
- Continue creating protobuf enums for integer enum schemas

**Context for Implementation:**
- BuildMessages at line 64
- Enum check at line 103: `if isEnumSchema(schema)`
- buildEnum call at line 104

## Phase 3: Update Code Generation

### Overview
Modify the protobuf generator to include enum value comments for string enum fields.

### Changes Required

#### 1. `internal/generator.go` - Update field rendering

**File**: `internal/generator.go`

**Changes**: Add enum comment generation in renderMessageWithIndent

```go
// In renderMessageWithIndent(), around lines 114-129
for _, field := range msg.Fields {
    // Render field description
    if field.Description != "" {
        result.WriteString(formatComment(field.Description, indent+"  "))
    }

    // NEW: Render enum values comment if present
    if len(field.EnumValues) > 0 {
        enumComment := formatEnumComment(field.EnumValues, indent+"  ")
        result.WriteString(enumComment)
    }

    // Render field definition
    // ... existing field rendering code ...
}

// NEW helper function
func formatEnumComment(values []string, indent string) string
```

**Function Responsibilities:**
- `formatEnumComment()`: Format enum values as `// enum: [value1, value2, value3]`
- Format: Space after each comma, values in square brackets
- Combine description and enum comment (description first, then enum comment on next line)
- Use proper indentation matching field indentation
- Handle special characters in enum values:
  - Values with spaces: `["foo bar", "baz"]` → `// enum: [foo bar, baz]`
  - Values with quotes: No escaping needed in proto comments
  - Values with brackets: No escaping needed in proto comments
  - Example output: `// enum: [a"b, c[d], e f]`
- If EnumValues is empty slice, do NOT generate comment (skip silently)

**Testing Requirements:**
```go
func TestFormatEnumComment(t *testing.T)
```

**Test Objectives:**
- Verify comment format: `// enum: [value1, value2, value3]` with spaces after commas
- Verify proper indentation
- Verify JSON array format (comma-separated with spaces, square brackets)
- Verify special character handling:
  - Enum with quotes: `["a\"b", "c"]` → `// enum: [a"b, c]`
  - Enum with brackets: `["a[b]", "c"]` → `// enum: [a[b], c]`
  - Enum with spaces: `["foo bar", "baz"]` → `// enum: [foo bar, baz]`
- Verify empty EnumValues produces no comment

**Context for Implementation:**
- renderMessageWithIndent at line 94
- Field rendering loop at lines 114-129
- Existing formatComment function at line 143 for reference
- Field description rendering at lines 115-117

## Phase 4: Update Tests

### Overview
Update all test cases that involve string enums to expect the new output format.

### Changes Required

#### 1. Update String Enum Tests

**File**: `internal/enums_test.go`

**Existing tests that need updates:**
- `TestTopLevelEnum` - Should NOT generate enum definition
- `TestEnumWithDashes` - Should generate string field with comment
- `TestEnumWithDescription` - Should combine description and enum comment (description first, then enum comment on separate line)
- `TestInlineEnum` - Should generate string field, not enum reference, with description preserved
- `TestMultipleInlineEnums` - Both enums should be string fields
- `TestEnumAndMessageMixed` - String enums removed, messages unchanged
- `TestEnumWithNumbers` - Integer enum should STILL work as before (protobuf enum generated)

**Testing Requirements:**
Each test should verify:
- No enum definition generated for string enums
- Fields use `string` type
- Comment includes `// enum: [...]` annotation with spaces after commas
- Description preserved when present (on separate line before enum comment)
- Integer enums still generate protobuf enum types (regression test)

**Context for Implementation:**
- Tests use table-driven pattern with given/expected strings
- Update expected protobuf output for each test
- Integer enum test (TestEnumWithNumbers) should NOT change

#### 2. Update Integration Tests

**File**: `convert_test.go`

**Changes**: Update `TestConvertCompleteExample`

Expected changes:
- Remove `OrderStatus` enum definition
- Remove `Category` enum definition
- `Order.status` field becomes `string` with enum comment
- `Product.category` field becomes `string` with enum comment

**Testing Requirements:**
- Verify complete e-commerce example generates correct output
- Verify both top-level referenced enum and inline enum are handled
- Verify other types (Address, Product, OrderItem, Order) unchanged

**Context for Implementation:**
- Test at line 430
- OrderStatus defined at line 438 (type: string, enum)
- Category inline enum at line 480 (type: string, enum)
- Expected output at line 525

#### 3. Add New Test Cases

**File**: `internal/enums_test.go`

**New test**: `TestStringEnumReference`

```go
func TestStringEnumReference(t *testing.T)
```

**Test Objectives:**
- Verify top-level string enum schema referenced via $ref
- Verify no standalone enum definition generated
- Verify referencing field uses string type with enum comment

**New test**: `TestMixedEnumTypes`

```go
func TestMixedEnumTypes(t *testing.T)
```

**Test Objectives:**
- Verify string enums and integer enums handled differently
- String enum → string field with comment
- Integer enum → protobuf enum type

**New test**: `TestStringEnumInArray`

```go
func TestStringEnumInArray(t *testing.T)
```

**Test Objectives:**
- Verify inline string enum in array generates `repeated string` with enum comment
- Example: `tags: [type: array, items: [type: string, enum: [draft, published]]]`

**New test**: `TestStringEnumArrayReference`

```go
func TestStringEnumArrayReference(t *testing.T)
```

**Test Objectives:**
- Verify array of referenced string enum generates `repeated string` with enum comment
- Example: `statuses: [type: array, items: [$ref: OrderStatus]]` where OrderStatus is string enum

**New test**: `TestEnumValidationErrors`

```go
func TestEnumValidationErrors(t *testing.T)
```

**Test Objectives:**
- Verify error for enum without type field with proper message
- Verify error for enum with null values with proper message
- Verify error for enum with mixed types with proper message
- Verify no error for empty enum array (returns empty EnumValues)
- Verify duplicate enum values allowed (no error, no deduplication)
- Verify enum values case-sensitive (can have both "Active" and "active")

**New test**: `TestStringEnumSpecialCharacters`

```go
func TestStringEnumSpecialCharacters(t *testing.T)
```

**Test Objectives:**
- Verify enum values with quotes, brackets, spaces are handled correctly
- Example: `enum: ["foo bar", "a\"b", "c[d]"]`
- Verify comment format: `// enum: [foo bar, a"b, c[d]]`

**New test**: `TestIntegerEnumDescriptionPreserved`

```go
func TestIntegerEnumDescriptionPreserved(t *testing.T)
```

**Test Objectives:**
- Verify integer enum field description is cleared (hoisted to enum definition)
- Regression test for existing behavior

**Context for Implementation:**
- Follow existing test patterns in enums_test.go
- Use conv.Convert() with ConvertOptions (note: lowercase conv, aliased import)
- Compare result.Protobuf against expected string
- For error tests, use require.Error() and assert.Contains() for error messages

## Phase 5: Update Documentation

### Overview
Update documentation to reflect the new string enum handling behavior.

### Changes Required

#### 1. Update README.md

**File**: `README.md`

**Changes**: Update Type Mapping section and enum example

Update lines 463-486 (Enum example):
- Show that string enums map to string fields with comments
- Add separate example for integer enums (which remain protobuf enums)
- Update "Supported Features" to clarify enum behavior

**Context for Implementation:**
- Current enum example at lines 463-486
- Type mapping table at lines 350-367
- Add note about string vs integer enum handling

#### 2. Update docs/enums.md

**File**: `docs/enums.md`

**Changes**: Complete rewrite to document new behavior

New sections:
- String Enums vs Integer Enums
- String Enum Mapping (string field + comment)
- Integer Enum Mapping (protobuf enum)
- Wire Format Compatibility
- When to Use Each Type
- Validation Rules and Linter Integration

New validation rules section should document:
- Enum must have explicit type field (no type inference) - enforced at build time
- Enum cannot contain null values - enforced at build time
- Enum cannot contain mixed types (all strings or all integers) - enforced at build time
- Empty enum arrays are allowed and generate no enum comment
- Duplicate enum values are allowed (no deduplication)
- Enum values are case-sensitive

Note: These rules are enforced by the converter during build. External linters can use these same validation rules to provide earlier feedback during development.

**Context for Implementation:**
- Current doc explains enum conversion limitations
- New doc should explain the fix and reasoning
- Include examples of both string and integer enums
- Explain JSON wire format preservation
- Document validation rules for external linter development

## Implementation Summary

### Key Changes
1. **String enums** → `string` fields with `// enum: [...]` comments
2. **Integer enums** → Protobuf enum types (unchanged behavior)
3. **Validation** → Enforce explicit type, reject nulls and mixed types
4. **Arrays** → Repeated string fields with enum comments for string enums
5. **Descriptions** → Preserved on string enum fields, cleared for integer enums

### Breaking Changes
- Top-level string enum schemas no longer generate standalone protobuf enum definitions
- Fields that previously referenced string enum types now use `string` type
- JSON wire format preserved (strings remain strings, not integers)

### Non-Breaking
- Integer enum behavior unchanged
- All existing tests for integer enums should pass without modification
- Field names, numbering, and other behaviors unchanged

## Validation Commands

After each phase, run:
```bash
go test ./internal -v -run TestIsStringEnum  # After Phase 1
go test ./internal -v -run TestProtoType     # After Phase 2
go test ./internal -v -run TestFormatEnum    # After Phase 3
go test ./internal -v                        # After Phase 4
```

After complete implementation, run:
```bash
make test          # Run all tests
make coverage      # Check test coverage
make lint          # Run linters
```

Verify:
- All tests pass
- No decrease in test coverage
- No new linter warnings

## Context for Implementation

### Key Patterns to Follow

1. **Error handling**: Use existing error types (SchemaError, PropertyError)
2. **Naming**: Follow existing conventions (isXxx for predicates, ToXxx for conversions)
3. **Testing**: Use table-driven tests with given/expected strings
4. **Comments**: Keep implementation comments minimal per guidelines

### Files to Reference

- `internal/naming.go` - ToEnumValueName() at line 84 for enum value formatting reference
- `internal/builder.go:232` - isEnumSchema() for enum detection pattern
- `internal/builder.go:262-274` - buildEnum() for enum value extraction
- `internal/generator.go:143` - formatComment() for comment formatting pattern

### Dependencies

- No new external dependencies required
- All changes use existing types and patterns
- Schema.Enum uses yaml.Node from existing libopenapi dependency
