# JSON Example Generation Implementation Plan

## Overview

Add the ability to generate JSON examples from OpenAPI schemas for use in documentation or injection into OpenAPI `example:` or `examples:` fields. The implementation will create a new `ConvertToExamples()` function that leverages existing parsing and schema traversal infrastructure to generate realistic JSON examples that honor schema constraints.

## Current State Analysis

**Existing Infrastructure:**
- OpenAPI parsing via `parser.ParseDocument()` at parser/parser.go:24
- Schema extraction via `doc.Schemas()` at parser/parser.go:44 (preserves YAML order)
- Type detection patterns in `internal/mapper.go:14` (handles refs, arrays, objects, enums, scalars)
- Property iteration via `schema.Properties.FromOldest()` at internal/builder.go:163
- Reference resolution via `propProxy.IsReference()` and `propProxy.Schema()`
- Schema constraints accessible via `base.Schema` fields:
  - `Minimum`, `Maximum`, `MinLength`, `MaxLength`
  - `Pattern`, `Enum`, `Default`, `Example`
  - `MinItems`, `MaxItems`, `Required`

**Key Discoveries:**
- All schema parsing logic is reusable (parser package)
- Type detection patterns from `ProtoType()` can guide example generation
- Schema traversal respects YAML order via `FromOldest()`
- Constraints are already accessible, just need interpretation

**Current Limitations:**
- No JSON example generation capability
- No constraint-aware value generation
- No circular reference detection for examples

## Desired End State

A complete JSON example generation system that:
- Provides `ConvertToExamples()` function returning `map[string]json.RawMessage`
- Accepts `ExampleOptions` to specify which schemas and generation settings
- Honors all schema constraints (min/max, patterns, enums, required fields)
- Handles circular references via detection + depth limiting
- Generates realistic values:
  - Random numbers within min/max range
  - Strings matching common patterns (email, UUID, URL, date)
  - Enums picking from available values
  - Arrays with minItems count
- Follows existing project testing patterns (functional testing via public API)

### Validation Commands

After each phase:
- `go test ./...` - All tests pass
- `go build ./...` - Code compiles without errors

## What We're NOT Doing

To maintain focused scope and manageable complexity:
- ✗ Complex regex pattern generation (no external regex-to-string libraries)
- ✗ Injecting examples back into OpenAPI documents (separate concern)
- ✗ Generating multiple example variations per schema
- ✗ Supporting OpenAPI 3.1+ `examples` field (multiple examples per schema)
- ✗ Custom example generation via callbacks or plugins
- ✗ Statistical or AI-based realistic data generation
- ✗ Validation of generated examples against schema (assume correct implementation)

## Implementation Approach

**High-Level Strategy:**
1. Create new public API `ConvertToExamples()` in convert.go
2. Build example generator in `internal/examplegenerator.go` following existing patterns
3. Reuse schema parsing, type detection, and traversal logic
4. Implement bottom-up generation (scalars → objects → arrays → refs)
5. Use visited map for circular detection + depth counter for safety
6. Test incrementally via functional tests

**Pattern Matching:**
- Follow `BuildMessages()` structure: parse → traverse → generate
- Mirror `ProtoType()` type detection order: ref → array → object → enum → scalar
- Use same error handling patterns as `builder.go` and `mapper.go`

## Phase 1: Core Infrastructure and Scalar Types

### Overview
Establish the public API, options struct, and core example generation infrastructure. Implement scalar type generation with constraint support (min/max, minLength/maxLength, enums). This phase delivers basic example generation for simple schemas.

### Changes Required

#### 1. Public API
**File**: `convert.go`
**Changes**: Add public function and result type

```go
// ExampleResult contains generated JSON examples for schemas
type ExampleResult struct {
    Examples map[string]json.RawMessage // schema name → JSON example
}

// ExampleOptions configures JSON example generation
type ExampleOptions struct {
    SchemaNames  []string // Specific schemas to generate (ignored if IncludeAll is true)
    MaxDepth     int      // Maximum nesting depth (default 5)
    IncludeAll   bool     // If true, generate examples for all schemas (takes precedence over SchemaNames)
    Seed         int64    // Random seed for deterministic generation (0 = use time-based seed)
}

// ConvertToExamples generates JSON examples from OpenAPI schemas
func ConvertToExamples(openapi []byte, opts ExampleOptions) (*ExampleResult, error)
```

**Function Responsibilities:**
- Validate inputs (openapi not empty)
- Default MaxDepth to 5 if not specified or <= 0
- Validate options: if IncludeAll is false and len(SchemaNames) == 0, return error "must specify SchemaNames or set IncludeAll"
- Parse OpenAPI document via `parser.ParseDocument()`
- Extract schemas via `doc.Schemas()`
- Determine which schemas to generate:
  - If IncludeAll is true: generate all schemas (ignore SchemaNames)
  - Otherwise: generate only schemas in SchemaNames list
- Default Seed to time.Now().UnixNano() if opts.Seed == 0
- Call `internal.GenerateExamples()` with filtered schemas and seed
- Return `ExampleResult` with generated examples map

**Testing Requirements:**
```go
func TestConvertToExamples(t *testing.T)
func TestConvertToExamplesValidation(t *testing.T)
```

**Test Objectives:**
- Validate empty openapi input returns error
- Validate successful example generation for simple schemas
- Validate SchemaNames filtering works correctly
- Validate IncludeAll generates all schemas
- Follow pattern from convert_test.go:12-75

**Context for implementation:**
- Follow input validation pattern from `Convert()` at convert.go:89-104
- Follow parsing pattern from `Convert()` at convert.go:106-114
- Return error early if parsing fails

#### 2. Example Generator Core
**File**: `internal/examplegenerator.go`
**Changes**: Create new file with core generator

```go
// ExampleContext holds state during example generation
type ExampleContext struct {
    schemas  map[string]*parser.SchemaEntry // All available schemas (name + proxy)
    path     []string                       // Current path for circular detection (e.g., ["User", "Address"])
    depth    int                            // Current nesting depth
    maxDepth int                            // Maximum allowed depth
    rand     *rand.Rand                     // Random number generator (seeded for determinism)
}

// GenerateExamples generates JSON examples for specified schemas
func GenerateExamples(entries []*parser.SchemaEntry, schemaNames []string, maxDepth int, seed int64) (map[string]json.RawMessage, error)

// generateExample generates a JSON example for a single schema
func generateExample(name string, proxy *base.SchemaProxy, ctx *ExampleContext) (interface{}, error)

// generateScalarValue generates a value for a scalar type with constraints
func generateScalarValue(schema *base.Schema, typ, format string, ctx *ExampleContext) (interface{}, error)
```

**Function Responsibilities:**

**GenerateExamples:**
- Build schema map from entries for reference resolution (map[name]*parser.SchemaEntry)
- Create rand.Rand with seed: `rand.New(rand.NewSource(seed))`
- Initialize ExampleContext with rand.Rand, empty path slice, depth=0, maxDepth
- Filter entries to requested schemaNames (if not empty)
- For each schema, call generateExample()
- Marshal results to json.RawMessage via json.Marshal()
- Handle marshal errors and return error if any fail
- Return map[string]json.RawMessage

**generateExample:**
- Check if name is in ctx.path (circular reference detected)
- If circular, return nil (will be omitted from parent object/array)
- Check depth limit via ctx.depth >= ctx.maxDepth
- If at depth limit, return nil
- Append name to ctx.path (track current traversal path)
- Defer removal of name from ctx.path (ensures cleanup on return/error)
- Resolve schema via proxy.Schema()
- Detect type (ref, array, object, enum, scalar) following ProtoType() pattern from mapper.go:14-108
- Delegate to appropriate generator
- Return generated value (path cleanup handled by defer)

**generateScalarValue:**
- Check precedence: Example > Default > Generated value
- If schema.Example != nil, parse and return Example value
- If schema.Default != nil, parse and return Default value
- Otherwise generate value based on type:
  - Integer with min/max: `ctx.rand.Intn(max-min+1) + min`
  - Integer without constraints: return 0
  - Number with min/max: `ctx.rand.Float64() * (max - min) + min`
  - Number without constraints: return 0.0
  - String: call generateStringValue() (Phase 3)
  - Boolean: `ctx.rand.Intn(2) == 1` (random true/false)
- Handle enum types: pick first value from schema.Enum for determinism
- Validate constraints before generation (error if min > max)
- Follow scalar type mapping from MapScalarType() at mapper.go:111-141

**Testing Requirements:**
```go
func TestConvertToExamplesScalarTypes(t *testing.T)
func TestConvertToExamplesConstraints(t *testing.T)
func TestConvertToExamplesEnums(t *testing.T)
func TestConvertToExamplesDeterministic(t *testing.T)
```

**Test Objectives:**
- All tests use `ConvertToExamples()` (functional testing, not internal functions)
- Use table-driven tests with OpenAPI YAML input and expected JSON assertions
- Validate integer with min/max generates value in range (unmarshal and check)
- Validate number with min/max generates value in range
- Validate string with minLength/maxLength generates correct length
- Validate enum picks first value consistently (deterministic)
- Validate schema.Default is used when present
- Validate schema.Example is used when present
- Validate same seed produces identical output (determinism test)
- Follow functional testing pattern from CLAUDE.md and convert_test.go

**Context for implementation:**
- Reference type detection pattern from ProtoType() at mapper.go:14-108
- Reference enum handling from isEnumSchema() at builder.go:265
- Reference scalar mapping from MapScalarType() at mapper.go:111-141
- Use math/rand for random value generation
- Create seeded rand.Rand: `rand.New(rand.NewSource(seed))` for determinism
- For enums: pick `schema.Enum[0]` (first value) for consistent output
- Import requirements: `encoding/json`, `math/rand`, `time`, `github.com/pb33f/libopenapi/datamodel/high/base`, `github.com/duh-rpc/openapi-proto.go/internal/parser`

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All tests pass
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors

## Phase 2: Complex Types (Objects, Arrays, References)

### Overview
Extend example generation to handle complex types: objects (with property iteration), arrays (with items), and schema references. Implement circular reference detection and depth limiting. This phase delivers complete example generation for nested structures.

### Changes Required

#### 1. Object Example Generation
**File**: `internal/examplegenerator.go`
**Changes**: Add object handling functions

```go
// generateObjectExample generates example for object schema
func generateObjectExample(schema *base.Schema, name string, ctx *ExampleContext) (map[string]interface{}, error)
```

**Function Responsibilities:**
- Check depth limit before recursing (ctx.depth >= ctx.maxDepth → return nil)
- Create result map[string]interface{}
- Iterate properties via schema.Properties.FromOldest() (preserves YAML order)
- Increment ctx.depth before recursing
- For each property, call generatePropertyValue()
- If generatePropertyValue() returns nil (depth limit or circular ref), omit from map
- If generatePropertyValue() returns error, propagate error
- Decrement ctx.depth after all properties processed
- Return map[string]interface{} (may be empty if all fields hit limits)
- Follow property iteration pattern from buildMessage() at builder.go:163-247
- Generate ALL properties (required and optional) - do not filter by schema.Required

**Testing Requirements:**
```go
func TestConvertToExamplesObjects(t *testing.T)
func TestConvertToExamplesNestedObjects(t *testing.T)
func TestConvertToExamplesDepthLimit(t *testing.T)
```

**Test Objectives:**
- All tests use `ConvertToExamples()` (functional testing)
- Validate simple object with scalar properties generates correct JSON structure
- Validate nested objects generate correctly up to depth limit
- Validate depth limit prevents deep nesting (fields omitted beyond limit)
- Validate property order matches YAML order
- Validate all properties (required and optional) are generated
- Follow functional testing pattern from convert_test.go with table-driven tests

**Context for implementation:**
- Reference property iteration from builder.go:163-247
- Use FromOldest() to preserve YAML order
- Handle nil schema.Properties (empty object)

#### 2. Array Example Generation
**File**: `internal/examplegenerator.go`
**Changes**: Add array handling

```go
// generateArrayExample generates example for array schema
func generateArrayExample(schema *base.Schema, propertyName string, ctx *ExampleContext) ([]interface{}, error)
```

**Function Responsibilities:**
- Validate schema.Items and schema.Items.A exist (return error if missing)
- Validate constraints: if MinItems > MaxItems, return error "invalid schema: minItems > maxItems"
- Determine item count:
  - Start with schema.MinItems if specified and > 0, else 1
  - Cap at schema.MaxItems if specified and MaxItems < item count
- Check depth limit before recursing (ctx.depth >= ctx.maxDepth → return empty array [])
- Increment ctx.depth
- Generate item count examples by calling generateItemValue()
- Filter out nil values (from circular refs or depth limits)
- Decrement ctx.depth
- Return []interface{} (may have fewer items than requested if some were nil)
- Follow array handling pattern from ResolveArrayItemType() at mapper.go:146-235

**Testing Requirements:**
```go
func TestConvertToExamplesArrays(t *testing.T)
func TestConvertToExamplesArrayConstraints(t *testing.T)
func TestConvertToExamplesArrayOfObjects(t *testing.T)
func TestConvertToExamplesInvalidArraySchema(t *testing.T)
```

**Test Objectives:**
- All tests use `ConvertToExamples()` (functional testing)
- Validate array with scalar items generates minItems count (or 1 if unspecified)
- Validate minItems constraint is honored
- Validate maxItems constraint caps generation
- Validate array of objects works correctly
- Validate invalid schema (minItems > maxItems) returns error
- Validate missing Items returns error
- Follow functional testing pattern from convert_test.go with table-driven tests

**Context for implementation:**
- Reference ResolveArrayItemType() at mapper.go:146-235
- Access items via schema.Items.A
- Handle MinItems and MaxItems from schema

#### 3. Reference Example Generation
**File**: `internal/examplegenerator.go`
**Changes**: Add reference handling

```go
// generateReferenceExample generates example for $ref schema
func generateReferenceExample(ref string, propProxy *base.SchemaProxy, ctx *ExampleContext) (interface{}, error)
```

**Function Responsibilities:**
- Extract schema name from ref using extractReferenceName() pattern
- Check if schemaName is in ctx.path (circular reference detection)
- If circular, return nil (field will be omitted from parent)
- Retrieve schema entry from ctx.schemas[schemaName]
- If not found in map, return error "schema '%s' not found"
- Call generateExample() with schema name and entry.Proxy
- generateExample() will handle path tracking
- Follow reference resolution pattern from ProtoType() at mapper.go:20-46

**Testing Requirements:**
```go
func TestConvertToExamplesReferences(t *testing.T)
func TestConvertToExamplesCircularReferences(t *testing.T)
```

**Test Objectives:**
- All tests use `ConvertToExamples()` (functional testing)
- Validate $ref to simple schema resolves correctly
- Validate circular reference detection breaks cycles (field omitted)
- Validate $ref to object schema works
- Validate User → Address → User circular scenario
- Test with fixed Seed to ensure deterministic circular handling
- Follow functional testing pattern from convert_test.go with table-driven tests

**Context for implementation:**
- Reference extractReferenceName() pattern from mapper.go:239-256
- Use ctx.schemas map for schema lookup (stores SchemaEntry, not just Proxy)
- Use ctx.path slice for circular detection (check if name in slice)
- Return nil for circular refs to omit field from parent object
- Circular detection is handled by generateExample() which manages path

#### 4. Property Value Generation Dispatcher
**File**: `internal/examplegenerator.go`
**Changes**: Add property value dispatcher

```go
// generatePropertyValue generates example value for object property
func generatePropertyValue(propertyName string, propProxy *base.SchemaProxy, ctx *ExampleContext) (interface{}, error)
```

**Function Responsibilities:**
- Resolve schema via propProxy.Schema()
- Check if reference via propProxy.IsReference()
- If reference, call generateReferenceExample()
- Detect type: array, object, enum, scalar (follow ProtoType() pattern)
- Delegate to appropriate generator
- Return interface{} value
- Follow type detection pattern from ProtoType() at mapper.go:14-108

**Testing Requirements:**
- Covered by object and array tests above

**Context for implementation:**
- Mirror ProtoType() structure from mapper.go:14-108
- Check ref first, then array, then object, then enum, then scalar

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All tests pass including circular reference tests
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors

## Phase 3: Pattern and Constraint Handling

### Overview
Add best-effort string pattern matching for common formats (email, UUID, URL, date, date-time) and handle string length constraints (minLength/maxLength). This phase delivers realistic example values for constrained string types.

### Changes Required

#### 1. Pattern Template Generator
**File**: `internal/examplegenerator.go`
**Changes**: Add pattern matching and template generation

```go
// generateStringValue generates string value honoring format and length constraints
func generateStringValue(schema *base.Schema, format string, ctx *ExampleContext) (string, error)
```

**Function Responsibilities:**

**generateStringWithPattern:**
- Validate constraints: if minLength > maxLength, return error "invalid schema: minLength > maxLength"
- Check schema.Format for common formats (email, uuid, uri, date, date-time, hostname)
- If format matches, return appropriate template
- Apply minLength/maxLength constraints to template:
  - If template shorter than minLength, pad with 'x' characters
  - If template longer than maxLength, truncate
- If no format, generate random alphanumeric string:
  - Length: random between minLength and maxLength (or default 10 if no constraints)
  - Use ctx.rand for random character selection
  - Character set: [a-zA-Z0-9]
- REMOVED: schema.Pattern matching (too complex, removed from scope)

**REMOVED: matchesCommonPattern function**
- Pattern matching removed from scope (too complex for Phase 3)
- Only schema.Format is supported with these templates:
  - email: "user@example.com"
  - uuid: "123e4567-e89b-12d3-a456-426614174000"
  - uri/url: "https://example.com"
  - date: "2024-01-15"
  - date-time: "2024-01-15T10:30:00Z"
  - hostname: "example.com"

**Testing Requirements:**
```go
func TestConvertToExamplesStringFormats(t *testing.T)
func TestConvertToExamplesStringLengthConstraints(t *testing.T)
```

**Test Objectives:**
- All tests use `ConvertToExamples()` (functional testing)
- Validate format:email generates email-like string
- Validate format:uuid generates UUID-like string
- Validate format:uri generates URL-like string
- Validate format:date generates date string
- Validate format:date-time generates ISO8601 timestamp
- Validate minLength constraint is honored (unmarshal and check length)
- Validate maxLength constraint is honored
- Validate invalid constraints (minLength > maxLength) return error
- Follow functional testing pattern from convert_test.go with table-driven tests
- Note: pattern matching removed from scope (too complex, use format only)

**Context for implementation:**
- Reference MapScalarType() format handling at mapper.go:125-133
- Only use schema.Format, ignore schema.Pattern (removed from scope)
- For random string generation, use charset "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
- Select random characters via ctx.rand.Intn(len(charset))

#### 2. Update Scalar Generator
**File**: `internal/examplegenerator.go`
**Changes**: Update generateScalarValue to use pattern generator

**Function Responsibilities:**
- For string type, call generateStringValue() instead of simple default
- Keep existing behavior for integer, number, boolean
- Preserve Default and Example value handling (Example > Default > Generated)

**Testing Requirements:**
- Covered by Phase 3 tests above

**Context for implementation:**
- Modify generateScalarValue() from Phase 1
- Add format parameter from schema.Format
- Call generateStringValue() for string type
- Validate constraints before generation (return error for invalid constraints)

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All pattern tests pass
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors

## Phase 4: Integration and Documentation

### Overview
Add comprehensive integration tests with real-world OpenAPI schemas, update README with usage examples, and ensure all edge cases are covered. This phase delivers production-ready JSON example generation.

### Changes Required

#### 1. Integration Tests
**File**: `/Users/thrawn/Development/openapi-proto.go/internal/integration_examples_test.go`
**Changes**: Create new integration test file

```go
func TestIntegrationComplexSchema(t *testing.T)
func TestIntegrationPetStore(t *testing.T)
func TestIntegrationCircularReferences(t *testing.T)
```

**Test Objectives:**
- Validate complete OpenAPI schema with mixed types generates valid JSON
- Validate PetStore-style schema (objects, arrays, refs, enums) generates correctly
- Validate circular reference schemas (User → Address → User) handle gracefully
- Validate depth limit prevents stack overflow
- Validate all constraints are honored in generated examples
- Follow integration test pattern from internal/integration_test.go

**Context for implementation:**
- Use realistic OpenAPI YAML snippets (from PetStore, Stripe, etc.)
- Test complete end-to-end flow via ConvertToExamples()
- Validate JSON can be unmarshaled successfully
- Validate values are within constraint ranges

#### 2. README Documentation
**File**: `/Users/thrawn/Development/openapi-proto.go/README.md`
**Changes**: Add JSON Example Generation section

**Content to add:**
- Overview of ConvertToExamples() function
- Basic usage example with ExampleOptions
- Example showing constraint handling (min/max, patterns, enums)
- Example showing SchemaNames filtering
- Example showing circular reference handling
- Add to table of contents

**Context for implementation:**
- Follow existing README structure and style
- Place section after "Go-Only Conversion" section (after line 125)
- Use code blocks with complete runnable examples
- Show both input OpenAPI YAML and output JSON

#### 3. Example Generator Documentation
**File**: `/Users/thrawn/Development/openapi-proto.go/docs/examples.md`
**Changes**: Create new documentation file

**Content to include:**
- Overview of example generation
- Constraint handling details (what's supported, what's not)
- Pattern/format support matrix
- Circular reference handling explanation
- Depth limit configuration
- Limitations and known issues

**Context for implementation:**
- Follow docs/enums.md and docs/discriminated-unions.md structure
- Include code examples
- Document all ExampleOptions fields

### Validation
- [ ] Run: `go test ./...`
- [ ] Verify: All tests including integration tests pass
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors
- [ ] Verify: README examples are accurate and runnable
- [ ] Verify: Generated JSON is valid and constraint-compliant
