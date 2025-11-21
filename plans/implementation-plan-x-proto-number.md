> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# x-proto-number Annotation Support Implementation Plan

## Overview

This plan adds support for the `x-proto-number` OpenAPI extension annotation that allows explicit control over protobuf field numbers during schema-to-protobuf conversion. Currently, field numbers are assigned sequentially (1, 2, 3...) based on YAML property order. The x-proto-number annotation enables users to specify exact field numbers, which is critical for maintaining backward compatibility when evolving APIs or migrating existing protobuf schemas to OpenAPI.

## Current State Analysis

### Field Number Assignment
- Sequential assignment implemented at `internal/builder.go:157-228` (buildMessage) and `internal/builder.go:397-439` (buildNestedMessage)
- Field numbers start at 1 and increment for each property
- Property iteration uses `schema.Properties.FromOldest()` to preserve YAML order
- No custom numbering support exists

### Extension Handling
- **No OpenAPI extension handling currently exists** in the codebase
- The `libopenapi` library provides `schema.Extensions` as `*orderedmap.Map[string, *yaml.Node]`
- Extensions can be accessed from `*base.Schema` objects during processing

### Validation Patterns
- Two-pass algorithm in `BuildMessages()` (line 65-128):
  - Pass 1: Schema registration and top-level validation
  - Pass 2: Type-specific validation before building
- Dedicated validation functions called before processing:
  - `validateTopLevelSchema()` at line 452 - first pass validation
  - `validateEnumSchema()` at line 284 - second pass, before buildEnum
- Error helpers in `internal/errors.go`: `SchemaError()`, `PropertyError()`, `UnsupportedSchemaError()`

### Protobuf Field Number Constraints (from research)
- Valid range: 1 to 536,870,911 (2^29-1)
- Reserved range: 19000-19999 (FieldDescriptor::kFirstReservedNumber to kLastReservedNumber)
- Field number 0 is invalid
- Best practice: Use 1-15 for frequently-set fields (1 byte encoding vs 2 bytes for 16-2047)

### Key Discoveries
- Prior art: OpenAPI Generator uses `x-protobuf-index`, NYTimes openapi2proto uses `x-proto-tag`
- Our choice of `x-proto-number` doesn't conflict with existing tools
- Validation should follow `validateEnumSchema` pattern (separate function before processing)
- Both `buildMessage()` and `buildNestedMessage()` need modification

## Desired End State

### Specification

Users can add `x-proto-number` to any schema property to control its protobuf field number:

```yaml
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          x-proto-number: 1
        email:
          type: string
          x-proto-number: 2
        name:
          type: string
          x-proto-number: 5
```

Generates:

```protobuf
message User {
  string id = 1 [json_name = "id"];
  string email = 2 [json_name = "email"];
  string name = 5 [json_name = "name"];
}
```

### Constraints Enforced

1. **All-or-nothing**: If ANY field has `x-proto-number`, ALL fields in that schema must have it
2. **No duplicates**: Each field number must be unique within a schema
3. **Valid range**: Must be between 1 and 536,870,911
4. **Reserved range**: Cannot use 19000-19999
5. **Hard errors**: Any violation fails conversion immediately with descriptive error

### Verification

Run existing test suite to ensure no regressions:
```bash
go test ./...
```

New tests will validate x-proto-number behavior in isolation and integration scenarios.

## What We're NOT Doing

1. **Enum value numbering**: x-proto-number only applies to message fields, not enum values (future enhancement)
2. **Automatic gap filling**: If user specifies 1, 2, 5, we don't auto-assign 3 and 4 to other fields
3. **Smart conflict resolution**: Duplicate numbers cause hard errors, no automatic renumbering
4. **Warnings for best practices**: Numbers outside 1-15 range are allowed (valid but less efficient)
5. **Migration tooling**: No automatic conversion of existing schemas to use x-proto-number
6. **Cross-schema validation**: Field numbers only validated within their own schema, not across schemas

## Implementation Approach

### Strategy

Follow the established validation pattern from `validateEnumSchema`:
1. Create dedicated validation function to check all constraints
2. Call validation before property processing in both `buildMessage` and `buildNestedMessage`
3. Extract x-proto-number during field creation, falling back to auto-increment if not present
4. Use existing error helpers for consistent error messaging

### Phased Delivery

**Phase 1**: Core validation infrastructure - validation function and unit tests
**Phase 2**: Integration with message building - modify buildMessage and buildNestedMessage
**Phase 3**: All-or-nothing validation - ensure consistency within schemas
**Phase 4**: Comprehensive integration tests - end-to-end scenarios

Each phase builds on the previous and delivers working, tested functionality.

## Phase 1: Extension Extraction and Field Number Validation

### Overview
Implement the core validation logic for x-proto-number constraints. This phase creates the foundation by adding helper functions to extract field numbers from extensions and validate them against protobuf constraints. No integration with the main conversion flow yet - just isolated, testable validation logic.

### Changes Required

- [x] Extension Helper Functions
- [x] Field Number Validation Function
- [x] Integration Tests
- [x] All tests passing

#### 1. Extension Helper Functions
**File**: `internal/builder.go`
**Changes**: Add helper functions after `validateEnumSchema` (after line 315)

```go
// extractFieldNumber extracts x-proto-number from schema proxy extensions
// Returns (number, true, nil) if found and valid
// Returns (0, false, nil) if not present
// Returns (0, false, error) if present but invalid format
func extractFieldNumber(proxy *base.SchemaProxy) (int, bool, error)

// hasFieldNumber checks if a schema proxy has x-proto-number extension
func hasFieldNumber(proxy *base.SchemaProxy) bool
```

**Function Responsibilities:**
- `extractFieldNumber`:
  - Call `proxy.Schema()` to get schema
  - Return (0, false, nil) if schema is nil or schema.Extensions is nil
  - Use `schema.Extensions.Get("x-proto-number")` to retrieve the yaml.Node
  - Return (0, false, nil) if node is nil (extension not present)
  - Parse node.Value as integer using strconv.Atoi or similar
  - Return (0, false, error) for invalid values (not a number, decimals, strings) with descriptive error
  - Return (number, true, nil) if successfully parsed
- `hasFieldNumber`: Call extractFieldNumber and return true if found (ignoring errors)

**Implementation Example:**
```go
func extractFieldNumber(proxy *base.SchemaProxy) (int, bool, error) {
    schema := proxy.Schema()
    if schema == nil || schema.Extensions == nil {
        return 0, false, nil
    }

    node := schema.Extensions.Get("x-proto-number")
    if node == nil {
        return 0, false, nil
    }

    // Parse yaml.Node.Value string to int
    num, err := strconv.Atoi(node.Value)
    if err != nil {
        return 0, false, fmt.Errorf("x-proto-number must be a valid integer, got: %s", node.Value)
    }

    return num, true, nil
}

func hasFieldNumber(proxy *base.SchemaProxy) bool {
    _, found, _ := extractFieldNumber(proxy)
    return found
}
```

**Testing Requirements:**
```go
// Integration tests in package internal_test (per project guidelines)
func TestExtractFieldNumberValidation(t *testing.T)
```

**Test Objectives:**
Since these are private helper functions, they will be tested indirectly through integration tests that exercise the full conversion flow:
- Valid x-proto-number values parse correctly (verified by successful conversion)
- Invalid format values (non-numeric strings, decimals like "3.14") produce clear error messages
- Missing x-proto-number extensions work correctly (auto-increment behavior)
- Schema or extensions nil cases handled gracefully
- NOTE: These validation tests will be simple integration tests; comprehensive functional tests in Phase 4

**Context for implementation:**
- Required imports: `"github.com/pb33f/libopenapi/datamodel/high/base"`, `"strconv"`, `"fmt"`
- Extensions are `*orderedmap.Map[string, *yaml.Node]` from libopenapi
- Use `Extensions.Get(key)` method to retrieve specific extension by name
- yaml.Node.Value is a string that needs parsing to int
- Invalid formats should return descriptive errors (e.g., "x-proto-number must be a valid integer, got: abc")
- This allows validation to provide clear error messages to users
- Reference web research: Extensions field available on Schema objects
- Look at property iteration pattern at `internal/builder.go:158` for accessing proxies

#### 2. Field Number Validation Function
**File**: `internal/builder.go`
**Changes**: Add validation function after extension helpers

```go
// validateFieldNumbers validates x-proto-number extensions on schema properties
// Returns error if:
// - Field numbers are duplicated
// - Field numbers are out of valid range (1 to 536,870,911)
// - Field numbers use reserved range (19000-19999)
// - Field number is 0 (invalid)
func validateFieldNumbers(schema *base.Schema, schemaName string) error
```

**Function Responsibilities:**
- Return nil if schema.Properties is nil (no properties to validate)
- Return nil if schema has 0 properties (empty schema - nothing to validate)
- Iterate properties using `schema.Properties.FromOldest()` pattern (like line 158)
- For each property: extract field number using `extractFieldNumber()`
- If extraction returns an error (invalid format), return `PropertyError()` with the error message
- Skip properties without x-proto-number (mixed mode allowed in this phase; Phase 3 will enforce all-or-nothing)
- Validate each field number against constraints:
  - Must be >= 1 (reject 0 and negative numbers)
  - Must be <= 536,870,911 (2^29-1)
  - Must NOT be in range 19000-19999 (reserved)
- Track seen numbers in a map[int]string to detect duplicates (map value = property name)
- Return `PropertyError()` for property-specific violations (invalid format, invalid range, reserved)
- Return `SchemaError()` for schema-wide violations (duplicates - include both property names)
- Include helpful context: field number value, property names in errors

**Testing Requirements:**
```go
// Integration tests in package internal_test (per project guidelines)
func TestFieldNumberValidation(t *testing.T)
```

**Test Objectives:**
Since validateFieldNumbers is a private function, it will be tested indirectly through integration tests:
- Valid field numbers pass validation (1, 15, 100, 536870911)
- Invalid format values (strings, decimals) produce clear error messages
- Field number 0 rejected with descriptive error
- Negative field numbers rejected (test -1)
- Field numbers > 536,870,911 rejected (test 536870912)
- Reserved range boundaries tested:
  - 18999 passes (just below reserved range)
  - 19000 rejected (start of reserved range)
  - 19500 rejected (middle of reserved range)
  - 19999 rejected (end of reserved range)
  - 20000 passes (just above reserved range)
- Duplicate field numbers detected with error listing both properties
- Schemas with no x-proto-number annotations pass (auto-increment works)
- Empty schemas (0 properties) pass (no error)
- Mixed schemas (some with, some without) pass in this phase (Phase 3 will reject)

**Context for implementation:**
- Follow pattern from `validateEnumSchema()` at line 284
- Use `PropertyError(schemaName, propName, message)` from `internal/errors.go:13`
- Use `SchemaError(schemaName, message)` from `internal/errors.go:7`
- Error message examples:
  - `PropertyError(schemaName, "userId", "x-proto-number must be between 1 and 536870911")`
  - `PropertyError(schemaName, "status", "x-proto-number 19500 is in reserved range 19000-19999")`
  - `SchemaError(schemaName, "duplicate x-proto-number 5 used by properties 'id' and 'email'")`

### Validation Commands
```bash
go test ./internal -run TestExtractFieldNumberValidation
go test ./internal -run TestFieldNumberValidation
go test ./internal
```

## Phase 2: Integration with Message Building

### Overview
Integrate the validation and extraction logic into the message building flow. This phase modifies `buildMessage()` and `buildNestedMessage()` to validate field numbers before processing and use x-proto-number values when creating fields. After this phase, users can specify custom field numbers in their OpenAPI schemas.

**Note on Mixed Mode**: This phase temporarily allows mixed schemas (some fields with x-proto-number, some without) to enable incremental development and testing. Phase 3 will enforce the all-or-nothing rule to prevent this in production use.

### Changes Required

- [x] Integrate field extraction into buildMessage
- [x] Integrate field extraction into buildNestedMessage
- [x] Integration tests for field extraction
- [x] All tests passing

#### 1. Integrate Validation into buildMessage
**File**: `internal/builder.go`
**Changes**: Modify `buildMessage()` function to call validation before field processing

Add validation call after object type check (after line 143), before field processing (before line 156):

```go
// buildMessage creates a protoMessage from an OpenAPI schema
func buildMessage(name string, proxy *base.SchemaProxy, ctx *Context, graph *DependencyGraph) (*ProtoMessage, error)
```

**Function Responsibilities:**
- After line 143 (object type validation): Add validation call
- Call `validateFieldNumbers(schema, name)` before property iteration
- Return error immediately if validation fails (fail-fast pattern)
- Continue with existing logic if validation passes
- During field creation loop (line 158-230):
  - After line 158: Extract field number from property proxy
  - Use extracted number if present, otherwise use auto-increment counter
  - Maintain backward compatibility: schemas without x-proto-number work as before

**Testing Requirements:**
```go
// Integration tests in package internal_test
func TestBuildMessageWithFieldNumbers(t *testing.T)
```

**Test Objectives:**
- Messages with valid x-proto-number annotations build correctly
- Field numbers from annotations appear in generated protobuf output
- Auto-increment still works for schemas without annotations
- Validation errors propagate correctly (duplicate numbers, invalid range, invalid format)
- Error includes schema name and property context

**Context for implementation:**
- Follow placement pattern from `BuildMessages()` line 106-108 (validateEnumSchema before buildEnum)
- Validation happens before any field processing
- No state changes if validation fails
- Reference existing buildMessage structure at lines 131-235

#### 2. Integrate Validation into buildNestedMessage
**File**: `internal/builder.go`
**Changes**: Modify `buildNestedMessage()` function similarly

Add validation call after message struct creation (after line 391), before property iteration (before line 396):

```go
// buildNestedMessage creates nested message from inline object property
func buildNestedMessage(propertyName string, proxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (*ProtoMessage, error)
```

**Function Responsibilities:**
- After line 391 (message struct creation): Add validation call
- Call `validateFieldNumbers(schema, propertyName)` before property iteration
- Return error immediately if validation fails
- During field creation loop (line 398-441):
  - Extract field number from property proxy
  - Use extracted number if present, otherwise use auto-increment
  - Maintain consistency with buildMessage implementation

**Testing Requirements:**
```go
// Integration tests in package internal_test
func TestBuildNestedMessageWithFieldNumbers(t *testing.T)
```

**Test Objectives:**
- Nested messages with x-proto-number build correctly
- Validation errors include nested context (property name)
- Both parent and nested messages can use x-proto-number independently
- Auto-increment works independently in parent vs nested messages
- Invalid format errors in nested messages include proper context

**Context for implementation:**
- Follow same pattern as buildMessage integration
- Use propertyName (not schema name) for error context in nested messages
- Reference existing buildNestedMessage structure at lines 363-449

#### 3. Field Number Extraction Logic
**File**: `internal/builder.go`
**Changes**: Update field creation in both buildMessage and buildNestedMessage

In `buildMessage()` property loop (around line 220):
- Before creating ProtoField, extract field number
- Use pattern: `fieldNum := extractFieldNumber(propProxy)` or similar
- Pass extracted number to ProtoField.Number field

In `buildNestedMessage()` property loop (around line 431):
- Same extraction and usage pattern

**Function Responsibilities:**
- Check if property has x-proto-number using `extractFieldNumber()`
- If present (found == true): use the extracted value for ProtoField.Number
- If not present (found == false): use current auto-increment counter value
- Increment counter after each field ONLY when x-proto-number NOT used
- Note: Phase 3 will enforce all-or-nothing, so mixed mode won't occur in practice
- When all fields have x-proto-number: counter starts at 1 but never increments (unused)
- When no fields have x-proto-number: counter provides sequential 1, 2, 3... (existing behavior)
- Maintain sequential numbering for non-annotated properties

**Testing Requirements:**
Integration tests cover this (TestBuildMessageWithFieldNumbers, TestBuildNestedMessageWithFieldNumbers)

**Test Objectives:**
- Extracted numbers appear in generated ProtoField structs
- Auto-increment counter not incremented for annotated fields
- Mixed schemas (some annotated, some not) work correctly

**Context for implementation:**
- Current field number assignment at line 220: `Number: fieldNumber`
- Current counter increment at line 228: `fieldNumber++`
- Conditional logic: if has x-proto-number, use it; else use counter and increment
- Both functions follow identical pattern for consistency

### Validation Commands
```bash
go test ./internal -run TestBuildMessage
go test ./internal -run TestBuildNestedMessage
go test ./internal
```

## Phase 3: All-or-Nothing Validation

### Overview
Enforce the consistency rule: if ANY field in a schema has x-proto-number, then ALL fields must have it. This prevents ambiguous scenarios where some fields use custom numbers and others use auto-increment, which could lead to conflicts or confusion.

### Changes Required

- [x] Enhanced validateFieldNumbers with all-or-nothing check
- [x] Integration tests for all-or-nothing validation
- [x] All tests passing

#### 1. Enhance Field Number Validation
**File**: `internal/builder.go`
**Changes**: Modify `validateFieldNumbers()` function to enforce all-or-nothing rule

```go
// validateFieldNumbers validates x-proto-number extensions on schema properties
// Returns error if:
// - Field numbers are duplicated
// - Field numbers are out of valid range (1 to 536,870,911)
// - Field numbers use reserved range (19000-19999)
// - Field number is 0 (invalid)
// - Some but not all fields have x-proto-number (all-or-nothing violation)
func validateFieldNumbers(schema *base.Schema, schemaName string) error
```

**Function Responsibilities:**
- First pass: count total properties and properties with x-proto-number
- Check consistency: if count > 0 and count < total, return error
- Error message format: `"schema 'User': x-proto-number must be specified on all fields or none (found on 2 of 5 fields)"`
- If all-or-nothing check passes, proceed with existing validation (duplicates, range, reserved)
- Return SchemaError for all-or-nothing violations (schema-level issue, not property-level)
- Note: Each schema is validated independently (parent and nested have separate validation)
- Parent schema all-or-nothing rule is independent from nested message all-or-nothing rule

**Testing Requirements:**
```go
// Integration tests in package internal_test, add new test cases
func TestAllOrNothingValidation(t *testing.T)
```

**Test Objectives:**
- All fields with x-proto-number: passes validation
- No fields with x-proto-number: passes validation (uses auto-increment)
- 1 of 3 fields with x-proto-number: fails with all-or-nothing error
- 2 of 5 fields with x-proto-number: fails with count in error message
- Error message includes total count and annotated count

**Context for implementation:**
- First pass through properties: count annotated vs total
- Use pattern from duplicate detection: iterate once to gather stats, then validate
- Return early if all-or-nothing violated (no need to check other constraints)
- Error format: `SchemaError(schemaName, fmt.Sprintf("x-proto-number must be specified on all fields or none (found on %d of %d fields)", annotatedCount, totalCount))`

#### 2. Update Field Extraction Logic
**File**: `internal/builder.go`
**Changes**: Simplify field number extraction in buildMessage and buildNestedMessage

Since validation now enforces all-or-nothing, field creation logic can assume:
- Either ALL properties have x-proto-number (use extracted values)
- OR NO properties have x-proto-number (use auto-increment)
- No mixed scenarios to handle

**Function Responsibilities:**
- In property loops: attempt to extract x-proto-number
- If extracted: use it
- If not: use auto-increment (entire schema uses auto-increment)
- No special handling needed for mixed mode (validation prevents it)

**Testing Requirements:**
No new tests needed - existing integration tests cover both scenarios

**Test Objectives:**
- Verify simplified logic still works correctly
- Existing tests continue to pass

**Context for implementation:**
- Validation guarantees consistency before this code runs
- Simplifies conditional logic from Phase 2
- Reference buildMessage at lines 217-228 and buildNestedMessage at lines 428-439

### Validation Commands
```bash
go test ./internal -run TestAllOrNothingValidation
go test ./internal
```

## Phase 4: Comprehensive Integration Tests

### Overview
Add end-to-end integration tests that exercise the full conversion flow with x-proto-number annotations. These tests validate the complete feature from OpenAPI YAML input to protobuf output, ensuring all components work together correctly.

### Changes Required

- [x] Integration Test File
- [x] Success Test Cases
- [x] Error Test Cases
- [x] Nested Message Test Cases
- [x] All tests passing

#### 1. Integration Test File
**File**: `internal/field_numbers_test.go` (new file)
**Changes**: Create comprehensive integration tests following existing test patterns

```go
func TestConvertWithFieldNumbers(t *testing.T)
```

**Test Objectives:**
- Basic field number specification (1, 2, 3)
- Non-sequential field numbers (1, 5, 10, 100)
- Maximum valid field number (536870911)
- Field numbers just below reserved range (18999)
- Field numbers just above reserved range (20000)
- Nested messages with field numbers
- Multiple schemas each with their own field numbers (no cross-schema conflicts)
- Error cases: duplicate numbers, invalid ranges, reserved range, all-or-nothing violations

**Testing Requirements:**
```go
func TestConvertWithFieldNumbers(t *testing.T)
func TestConvertFieldNumberErrors(t *testing.T)
func TestConvertNestedFieldNumbers(t *testing.T)
```

**Test Structure:**
Follow pattern from `internal/scalars_test.go:11-81`:
- Table-driven test with struct containing name, given (OpenAPI YAML), expected (Proto output), wantErr
- Call `conv.Convert()` with full OpenAPI document
- Assert generated protobuf matches expected output exactly
- For error cases: assert error contains expected message
- Use `require.NoError(t, err)` for success cases
- Use `require.ErrorContains(t, err, test.wantErr)` for error cases
- Use `assert.Equal(t, test.expected, string(result.Protobuf))` for output comparison

**Context for implementation:**
- Reference `internal/scalars_test.go` for test structure pattern
- Reference `internal/nested_test.go` for nested message test patterns
- Use package `conv "github.com/duh-rpc/openapi-proto.go"` for imports
- Full OpenAPI documents with info, paths, components sections
- Expected output includes full proto3 syntax with package, imports, options

#### 2. Success Test Cases
**File**: `internal/field_numbers_test.go`
**Changes**: Add test cases for valid x-proto-number usage

Test case examples:
```yaml
# Basic sequential
properties:
  id:
    type: string
    x-proto-number: 1
  name:
    type: string
    x-proto-number: 2
```

Expected:
```protobuf
message User {
  string id = 1 [json_name = "id"];
  string name = 2 [json_name = "name"];
}
```

Test case examples:
```yaml
# Non-sequential (gaps allowed)
properties:
  id:
    type: string
    x-proto-number: 1
  email:
    type: string
    x-proto-number: 5
  status:
    type: string
    x-proto-number: 10
```

Expected:
```protobuf
message User {
  string id = 1 [json_name = "id"];
  string email = 5 [json_name = "email"];
  string status = 10 [json_name = "status"];
}
```

**Test Objectives:**
- Sequential field numbers work
- Non-sequential field numbers work (gaps are allowed)
- Large field numbers work (test up to 536870911)
- Numbers avoiding reserved range work (18999, 20000)
- Property order in output matches YAML order (not field number order)
- **Explicit property ordering test**: Create a schema with properties ordered as (zebra: x-proto-number=3, apple: x-proto-number=1, banana: x-proto-number=2) in YAML. Verify output preserves YAML order (zebra, apple, banana), not field number order (apple, banana, zebra)

**Context for implementation:**
- Output order matches YAML property order, NOT field number order
- Reference property iteration: `schema.Properties.FromOldest()` preserves order
- Generated proto should maintain property order even with non-sequential numbers

#### 3. Error Test Cases
**File**: `internal/field_numbers_test.go`
**Changes**: Add test cases for validation errors

Test case examples:
```yaml
# Invalid format
properties:
  id:
    type: string
    x-proto-number: "abc"  # not a valid integer!
```

Expected error: `"x-proto-number must be a valid integer"`

Test case examples:
```yaml
# Duplicate field numbers
properties:
  id:
    type: string
    x-proto-number: 1
  name:
    type: string
    x-proto-number: 1  # duplicate!
```

Expected error: `"duplicate x-proto-number 1"`

Test case examples:
```yaml
# Reserved range
properties:
  id:
    type: string
    x-proto-number: 19500  # in reserved range!
```

Expected error: `"19500 is in reserved range 19000-19999"`

Test case examples:
```yaml
# All-or-nothing violation
properties:
  id:
    type: string
    x-proto-number: 1
  name:
    type: string
    # missing x-proto-number!
```

Expected error: `"x-proto-number must be specified on all fields or none"`

**Test Objectives:**
- Invalid format values produce clear errors (test "abc", "3.14", "1.5")
- Duplicate field numbers produce clear error with both property names
- Reserved range violations detected (test 19000, 19500, 19999)
- Invalid ranges detected (0, negative, > 536870911)
- All-or-nothing violations show field counts
- Error messages include schema name and property context

**Context for implementation:**
- Use `require.ErrorContains(t, err, test.wantErr)` pattern
- Test partial error messages, not full text (more resilient)
- Error should prevent any protobuf output generation

#### 4. Nested Message Test Cases
**File**: `internal/field_numbers_test.go`
**Changes**: Add test cases for nested inline messages with field numbers

Test case examples:
```yaml
# Parent and nested both use x-proto-number
User:
  type: object
  properties:
    id:
      type: string
      x-proto-number: 1
    address:
      type: object
      x-proto-number: 2
      properties:
        street:
          type: string
          x-proto-number: 1
        city:
          type: string
          x-proto-number: 2
```

Expected:
```protobuf
message User {
  message Address {
    string street = 1 [json_name = "street"];
    string city = 2 [json_name = "city"];
  }

  string id = 1 [json_name = "id"];
  Address address = 2 [json_name = "address"];
}
```

**Test Objectives:**
- Nested messages can use x-proto-number independently
- Parent field numbers don't conflict with nested field numbers
- Nested field numbering is independent (can reuse number 1)
- All-or-nothing applies separately to parent and nested schemas
- Validation errors in nested messages include property context

**Context for implementation:**
- Reference `internal/nested_test.go` for nested message patterns
- Parent and nested have separate field number spaces (both can use 1, 2, 3)
- Nested messages appear before parent fields in output (existing behavior)

### Validation Commands
```bash
go test ./internal -run TestConvertWithFieldNumbers
go test ./internal -run TestConvertFieldNumberErrors
go test ./internal -run TestConvertNestedFieldNumbers
go test ./internal
go test ./...
```

All tests should pass, including existing tests (no regressions).

---

## Implementation Notes

### Testing Organization
- **All tests**: Use `package internal_test` (per project CLAUDE.md guidelines)
- **Phase 1-3**: Integration tests that exercise validation through the full conversion flow
- **Phase 4**: Comprehensive end-to-end integration tests with full OpenAPI documents
- Private helper functions (extractFieldNumber, validateFieldNumbers) tested indirectly through integration tests
- All tests follow functional testing style: use `conv.Convert()` API, not internal functions directly

### Key Technical Details
- Extension access: `schema.Extensions.Get("x-proto-number")` returns `*yaml.Node` or nil
- YAML node parsing: `node.Value` is a string, use `strconv.Atoi()` to convert to int
- Invalid format handling: `extractFieldNumber()` returns error for invalid formats (non-numeric, decimals), allowing clear validation error messages
- Auto-increment counter: starts at 1, increments only for fields without x-proto-number
- Field ordering: output preserves YAML property order, NOT field number order
- Validation scope: each schema validated independently (parent and nested separate)
- ProtoField.Number type: int (supports full range up to 536,870,911)

### Error Message Examples
```
schema 'User': property 'id' x-proto-number must be a valid integer, got: abc
schema 'User': property 'age' x-proto-number must be between 1 and 536870911
schema 'Product': property 'status' x-proto-number 19500 is in reserved range 19000-19999
schema 'Order': duplicate x-proto-number 5 used by properties 'id' and 'customerId'
schema 'Invoice': x-proto-number must be specified on all fields or none (found on 2 of 5 fields)
```

### Required Imports for Implementation
```go
import (
    "fmt"
    "strconv"
    "github.com/pb33f/libopenapi/datamodel/high/base"
)
```
