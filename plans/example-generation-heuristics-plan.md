> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# Example Generation Heuristics Implementation Plan

## Overview

Add intelligent heuristics to the JSON example generation system to produce more realistic and context-aware examples. This includes field-name-based heuristics for common patterns (cursors, error messages), field value overrides, and better default values for unconstrained numbers.

## Current State Analysis

**Existing Implementation:**
- Example generation in `ConvertToExamples()` at convert.go:342-383
- Core generation logic in `internal/examplegenerator.go`
- Scalar value generation at examplegenerator.go:131-188
- String value generation at examplegenerator.go:191-254
- Property value generation at examplegenerator.go:332-388

**Current Behavior:**
- Integers without min/max constraints return `0` (examplegenerator.go:158)
- Numbers without min/max constraints return `0.0` (examplegenerator.go:177)
- Strings use format-based templates or random alphanumeric (examplegenerator.go:191-254)
- No field-name-based heuristics
- No way to override specific field values

**Key Discoveries:**
- `generatePropertyValue()` already has access to `propertyName` (line 332)
- `generateScalarValue()` does NOT currently receive field name context
- Field name needs to be threaded through to enable heuristics
- ExampleContext already passes through all generation functions

## Desired End State

An enhanced example generation system that:
- Supports field value overrides via `ExampleOptions.FieldOverrides map[string]interface{}`
- Generates realistic cursor-like strings for fields named "cursor", "first", "after"
- Generates human-readable messages for fields named "error", "message"
- Uses random values (1-100 for integers, 1.0-100.0 for numbers) instead of zero for unconstrained numbers
- Maintains backward compatibility with existing tests
- All changes tested via functional tests using public `ConvertToExamples()` API

### Validation Commands

After each phase:
- `go test ./...` - All tests pass
- `go build ./...` - Code compiles without errors

## What We're NOT Doing

To maintain focused scope:
- ✗ Complex pattern matching beyond simple field name equality
- ✗ Context-aware heuristics based on parent object types
- ✗ Locale-specific message generation
- ✗ Learning from existing examples in the OpenAPI spec
- ✗ Multiple override strategies (only exact field name matching)

## Implementation Approach

**High-Level Strategy:**
1. Add `FieldOverrides` to `ExampleOptions` and thread through to context
2. Update integer/number defaults to use random values (1-100)
3. Add field name parameter to `generateScalarValue()`
4. Implement field name heuristics in `generateStringValue()`
5. Implement field override checking in `generateScalarValue()`

**Pattern Matching:**
- Follow existing constraint checking patterns from examplegenerator.go:144-177
- Use simple string equality for field name matching
- Priority: Example > Default > FieldOverride > Heuristics > Generated
- FieldOverrides use case-sensitive matching (to match exact JSON field names)
- Heuristics use case-insensitive matching (for flexibility with naming conventions)

## Phase 1: Field Overrides and Non-Zero Defaults

### Overview
Add FieldOverrides support to ExampleOptions and change default integer/number generation to use random values instead of zero. This phase delivers the foundation for field-specific customization.

### Changes Required

#### 1. Update ExampleOptions
**File**: `convert.go`
**Changes**: Add FieldOverrides field

```go
type ExampleOptions struct {
    SchemaNames   []string               // Specific schemas to generate (ignored if IncludeAll is true)
    MaxDepth      int                    // Maximum nesting depth (default 5)
    IncludeAll    bool                   // If true, generate examples for all schemas
    Seed          int64                  // Random seed for deterministic generation (0 = use time-based seed)
    FieldOverrides map[string]interface{} // Override values for specific field names (e.g., {"code": 400})
}
```

**Function Responsibilities:**
- Add new field to struct
- Pass FieldOverrides to `internal.GenerateExamples()` call at convert.go:375

**Testing Requirements:**
```go
func TestConvertToExamplesFieldOverrides(t *testing.T)
```

**Test Objectives:**
- Validate field override for integer field (e.g., {"code": 400} sets code field to 400)
- Validate field override for string field (e.g., {"status": "error"})
- Validate field override takes precedence over default and generated values
- Validate field override does not apply to fields with different names
- Use table-driven tests with OpenAPI YAML input
- Tests should be added to `convert_examples_test.go`

**Context for implementation:**
- Follow ExampleOptions pattern from convert.go:49-55
- Document that FieldOverrides applies to any field with matching name across all schemas

#### 2. Update ExampleContext
**File**: `internal/examplegenerator.go`
**Changes**: Add FieldOverrides to context

```go
type ExampleContext struct {
    schemas        map[string]*parser.SchemaEntry // All available schemas (name + proxy)
    path           []string                       // Current path for circular detection
    depth          int                            // Current nesting depth
    maxDepth       int                            // Maximum allowed depth
    rand           *rand.Rand                     // Random number generator
    fieldOverrides map[string]interface{}         // Field name to value overrides
}
```

**Function Responsibilities:**
- Add fieldOverrides field to struct
- Update GenerateExamples() to accept fieldOverrides parameter and populate ctx.fieldOverrides

**Testing Requirements:**
- Covered by Phase 1 test above

**Context for implementation:**
- Initialize in GenerateExamples() at line 30

#### 3. Change Integer/Number Defaults to Random
**File**: `internal/examplegenerator.go`
**Changes**: Update generateScalarValue integer and number cases

**Function Responsibilities:**
- For integer type (line 141-158):
  - If no min/max specified, generate random between 1-100: `ctx.rand.Intn(100) + 1` (produces readable non-zero values)
  - Remove return 0 at line 158
- For number type (line 160-177):
  - If no min/max specified, generate random between 1.0-100.0: `ctx.rand.Float64()*99.0 + 1.0` (produces readable non-zero values)
  - Remove return 0.0 at line 177

**Testing Requirements:**
```go
func TestConvertToExamplesRandomDefaults(t *testing.T)
```

**Test Objectives:**
- Validate integer without constraints generates value between 1-100
- Validate number without constraints generates value between 1.0-100.0
- Validate with fixed seed, values are deterministic and non-zero
- Use multiple test runs to ensure values vary (with different seeds)
- Tests should be added to `convert_examples_test.go`

**Context for implementation:**
- Modify existing generateScalarValue() function
- Keep existing constraint validation (min > max error)
- Random generation should happen when `schema.Minimum == nil && schema.Maximum == nil`

#### 4. Update GenerateExamples Signature
**File**: `internal/examplegenerator.go`
**Changes**: Add fieldOverrides parameter

```go
func GenerateExamples(entries []*parser.SchemaEntry, schemaNames []string, maxDepth int, seed int64, fieldOverrides map[string]interface{}) (map[string]json.RawMessage, error)
```

**Function Responsibilities:**
- Add fieldOverrides parameter
- Pass to ExampleContext initialization at line 30
- Handle nil fieldOverrides (treat as empty map)

**Testing Requirements:**
- Covered by existing tests (nil map should work)

**Context for implementation:**
- Update call site in convert.go:375
- Pass opts.FieldOverrides

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All tests pass (some existing tests may need seed adjustments for non-zero defaults)
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors

## Phase 2: Field Name Heuristics for Strings

### Overview
Add field-name-based heuristics to generate realistic values for common field patterns: cursor fields get base64-looking strings, and error/message fields get human-readable text.

### Changes Required

#### 1. Thread Field Name Through to generateScalarValue
**File**: `internal/examplegenerator.go`
**Changes**: Add fieldName parameter to generateScalarValue

```go
func generateScalarValue(fieldName string, schema *base.Schema, typ, format string, ctx *ExampleContext) (interface{}, error)
```

**Function Responsibilities:**
- Add fieldName as first parameter
- Update all call sites:
  - generateExample() at line 127
  - generatePropertyValue() at line 387
- Pass fieldName through to generateStringValue() at line 180

**Testing Requirements:**
- Covered by Phase 2 tests below

**Context for implementation:**
- Update function signature
- Update calls at lines 127 and 387 to pass appropriate field name
- In generateExample(), field name is the schema name
- In generatePropertyValue(), field name is propertyName

#### 2. Add Cursor Field Heuristics
**File**: `internal/examplegenerator.go`
**Changes**: Update generateStringValue to detect cursor fields

**Function Responsibilities:**
- Add fieldName parameter: `func generateStringValue(fieldName string, schema *base.Schema, format string, ctx *ExampleContext) (string, error)`
- Before checking format, check if fieldName matches cursor patterns
- If fieldName is "cursor", "first", or "after" (case-insensitive):
  - Generate base64-looking string using charset: `[a-zA-Z0-9+/]`
  - Length: random between 16-32 characters
  - Example output: "dGhpc2lzYWN1cnNvcg" or "YWJjZGVmZ2hpamts"
- Cursor heuristic is applied BEFORE format checking
- If no cursor match, continue with existing format logic

**Testing Requirements:**
```go
func TestConvertToExamplesCursorHeuristics(t *testing.T)
```

**Test Objectives:**
- Validate field named "cursor" generates base64-looking string
- Validate field named "first" generates base64-looking string
- Validate field named "after" generates base64-looking string
- Validate field named "Cursor" (capitalized) also works (case-insensitive)
- Validate generated string length is 16-32 characters
- Validate generated string uses only [a-zA-Z0-9+/] characters
- Validate field named "other" does not trigger cursor heuristic

**Context for implementation:**
- Add case-insensitive comparison: `strings.ToLower(fieldName)`
- Use charset: `"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/"`
- Generate length: `ctx.rand.Intn(17) + 16` (produces 16-32 characters inclusive)
- Import: `strings` package
- Tests should be added to `convert_examples_test.go`

#### 3. Add Error/Message Field Heuristics
**File**: `internal/examplegenerator.go`
**Changes**: Update generateStringValue to detect message fields

**Function Responsibilities:**
- After cursor check, before format check, check if fieldName is "error" or "message" (case-insensitive)
- If fieldName is "error": return "An error occurred"
- If fieldName is "message": return "This is a message"
- Message heuristic is checked AFTER cursor but BEFORE format
- If no message match, continue with existing format logic

**Testing Requirements:**
```go
func TestConvertToExamplesMessageHeuristics(t *testing.T)
```

**Test Objectives:**
- Validate field named "error" generates "An error occurred"
- Validate field named "message" generates "This is a message"
- Validate field named "Error" (capitalized) also works (case-insensitive)
- Validate field named "description" does not trigger message heuristic

**Context for implementation:**
- Add case-insensitive comparison
- Use simple string constants for now
- Place check after cursor heuristic but before format check
- Tests should be added to `convert_examples_test.go`

#### 4. Update generateStringValue Call Sites
**File**: `internal/examplegenerator.go`
**Changes**: Pass fieldName to generateStringValue

**Function Responsibilities:**
- Update call at line 180 in generateScalarValue(): `return generateStringValue(fieldName, schema, format, ctx)`

**Testing Requirements:**
- Covered by heuristic tests above

**Context for implementation:**
- fieldName is already available in generateScalarValue() after Phase 2.1 changes

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All heuristic tests pass
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors

## Phase 3: Field Override Implementation

### Overview
Implement the field override checking logic in generateScalarValue to allow users to override specific field values via ExampleOptions.FieldOverrides.

### Changes Required

#### 1. Add Override Checking to generateScalarValue
**File**: `internal/examplegenerator.go`
**Changes**: Check fieldOverrides before generating values

**Function Responsibilities:**
- At the start of generateScalarValue(), after Example and Default checks, add override check
- Priority order: Example > Default > FieldOverride > Heuristics > Generated
- If ctx.fieldOverrides is not nil and contains fieldName (case-sensitive):
  - Retrieve override value
  - Validate type matches schema type:
    - integer type: ensure override is numeric (int or float64 from JSON)
    - number type: ensure override is numeric
    - string type: ensure override is string
    - boolean type: ensure override is bool
  - Return override value if type matches
  - Return error if type mismatch: "field override for '%s' has wrong type"
- If no override, continue with existing logic

**Testing Requirements:**
```go
func TestConvertToExamplesFieldOverridePriority(t *testing.T)
func TestConvertToExamplesFieldOverrideTypeMismatch(t *testing.T)
```

**Test Objectives:**
- Validate FieldOverride takes precedence over heuristics ({"message": "custom"} overrides "This is a message")
- Validate FieldOverride does NOT override schema.Example (when schema has Example, that value is used and FieldOverride is ignored)
- Validate FieldOverride does NOT override schema.Default (when schema has Default, that value is used and FieldOverride is ignored)
- Validate type mismatch returns error (e.g., {"code": "string"} for integer field)
- Validate multiple field overrides work simultaneously

**Context for implementation:**
- Place check after Default check (line 137) but before type switch (line 140)
- For type checking, handle JSON unmarshaling types (float64 for all numbers)
- Convert float64 to int for integer fields if whole number
- Use `math.Mod(val, 1.0) == 0` to check if float64 is whole number
- Import: `math` package
- Tests should be added to `convert_examples_test.go`
- Note: FieldOverrides use case-sensitive matching to ensure exact JSON field name matches

#### 2. Update Documentation
**File**: `convert.go`
**Changes**: Add documentation for FieldOverrides

**Function Responsibilities:**
- Add godoc comment for FieldOverrides field explaining:
  - Applies to any field with matching name (case-sensitive) across all schemas
  - Takes precedence over heuristics and generated values
  - Does not override schema.Example or schema.Default
  - Type must match schema type or error is returned

**Testing Requirements:**
- No new tests (documentation only)

**Context for implementation:**
- Follow existing godoc comment style from ExampleOptions at convert.go:49-55

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All override tests pass
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors

## Phase 4: Integration Testing and Documentation

### Overview
Add comprehensive integration tests demonstrating all heuristics working together, and update documentation with examples.

### Changes Required

#### 1. Integration Tests
**File**: `convert_examples_heuristics_test.go` (new file in root directory)
**Changes**: Create comprehensive integration tests

**Note on Test Organization:**
- Phase 1-3 unit tests should be added to existing `convert_examples_test.go`
- Phase 4 integration tests create new `convert_examples_heuristics_test.go` file

```go
func TestConvertToExamplesAllHeuristicsTogether(t *testing.T)
func TestConvertToExamplesRealisticErrorResponse(t *testing.T)
func TestConvertToExamplesPaginationResponse(t *testing.T)
```

**Test Objectives:**
- Validate all heuristics work together in realistic schema
- Test error response schema with code field override, error/message fields
- Test pagination response with cursor, first, after fields
- Validate field overrides work with nested objects
- Validate random defaults produce non-zero values consistently

**Context for implementation:**
- Use realistic OpenAPI schemas (error responses, paginated lists)
- Combine multiple heuristics in single schema
- Validate JSON structure matches expectations
- Use fixed seed for deterministic testing

#### 2. Update README
**File**: `README.md`
**Changes**: Add section on field heuristics and overrides

**Content to add:**
- Add "Field Heuristics" subsection under "JSON Example Generation" (after line 216)
- Document cursor field heuristics with example
- Document message field heuristics with example
- Document FieldOverrides with example showing error response
- Show before/after examples

**Context for implementation:**
- Follow existing README structure
- Use code blocks with complete examples
- Place after Constraint Handling table (line 216)

#### 3. Update docs/examples.md
**File**: `docs/examples.md`
**Changes**: Add detailed heuristics documentation

**Content to add:**
- New section "Smart Field Heuristics"
- Document cursor fields (cursor, first, after)
- Document message fields (error, message)
- Document non-zero defaults for numbers
- New section "Field Overrides"
- Document FieldOverrides with complete example
- Document precedence order: Example > Default > Override > Heuristic > Generated

**Context for implementation:**
- Add after "Constraint Handling" section
- Include code examples and output samples
- Document limitations (case-sensitivity, exact match only)

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All tests including integration tests pass
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors
- [ ] Verify: README examples are accurate
- [ ] Verify: Documentation explains all heuristics clearly
