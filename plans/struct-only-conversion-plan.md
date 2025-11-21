> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# Go Struct-Only Conversion Implementation Plan

## Overview

Add a new `ConvertToStruct()` function that converts all OpenAPI schemas to Go structs only, without generating Protocol Buffer definitions. This provides a pure Go struct generation path for users who need Go types but not protobuf.

## Current State Analysis

The existing `Convert()` function uses a hybrid approach:
- **Proto generation**: Default for all schemas (convert.go:115-129)
- **Go generation**: Only for schemas containing or referencing oneOf unions (convert.go:132-142)
- **Classification**: Dependency graph with transitive closure determines which types need Go (internal/dependencies.go:50-120)

### Key Existing Components:
- `BuildGoStructs()` at internal/golang.go:51 - Builds Go struct IR for schemas
- `buildGoStruct()` at internal/golang.go:71 - Handles both union and regular struct generation
- `GenerateGo()` at internal/gogen.go:11 - Renders Go source from struct IR
- `BuildMessages()` at internal/builder.go:65 - Builds dependency graph and proto IR
- Union types get custom MarshalJSON/UnmarshalJSON (gogen.go:108-178)
- Regular structs get simple field mapping with JSON tags (golang.go:112-142)

### Key Discovery:
The Go struct generation code is already isolated from proto logic and can be reused directly. The dependency graph is needed for discriminator validation in union types but not for type classification.

## Desired End State

A new public API function `ConvertToStruct()` that:
- Converts ALL OpenAPI schemas to Go structs (no filtering by transitive closure)
- Reuses existing Go generation code (`BuildGoStructs`, `GenerateGo`)
- Union types retain custom JSON marshaling behavior
- Regular types become simple structs with JSON tags
- Returns `StructResult` with Go source and type metadata

### Verification:
```bash
go test ./... -run TestConvertToStruct
```
Verify that:
- All schemas convert to Go structs
- Union types have custom marshaling
- Regular types have simple struct definitions
- Generated code compiles successfully

## What We're NOT Doing

- Not modifying existing `Convert()` behavior
- Not changing proto generation logic
- Not adding configuration flags to `ConvertOptions`
- Not altering existing Go struct generation code
- Not creating a separate code path for struct generation (reuse existing)

## Implementation Approach

Reuse the existing Go struct generation pipeline by:
1. Building dependency graph (for discriminator validation)
2. Marking ALL schemas as "Go types" instead of filtering via transitive closure
3. Calling existing `BuildGoStructs()` and `GenerateGo()` functions
4. Returning new `StructResult` type with Go output only

## Phase 1: Add StructResult Type and ConvertToStruct Function

### Overview
Create the new result type and main conversion function that generates Go structs for all schemas.

### Changes Required:

#### 1. Public API Types
**File**: `convert.go`
**Changes**: Add StructResult type and ConvertToStruct function

```go
// StructResult contains Go struct output and type metadata
type StructResult struct {
	Golang  []byte
	TypeMap map[string]*TypeInfo
}

// ConvertToStruct converts all OpenAPI schemas to Go structs only
func ConvertToStruct(openapi []byte, opts ConvertOptions) (*StructResult, error)
```

**ConvertToStruct Responsibilities:**
- Validate inputs (openapi non-empty, opts.GoPackagePath non-empty)
- Default opts.PackageName to "main" if empty (needed by BuildMessages but not used for Go output)
- Parse OpenAPI document via `parser.ParseDocument()`
- Extract schemas via `doc.Schemas()`
- Create proto context via `internal.NewContext()` (needed by BuildMessages)
- Build dependency graph via `internal.BuildMessages(schemas, ctx)`
- Compute transitive closure via `graph.ComputeTransitiveClosure()` to get reasons map
- Create `goTypes` map with ALL schema names set to true (mark all for Go generation)
- Create `GoContext` via `internal.NewGoContext(internal.ExtractPackageName(opts.GoPackagePath))`
- Call `internal.BuildGoStructs(schemas, goTypes, graph, goCtx)`
- Generate Go source via `internal.GenerateGo(goCtx)`
- Build TypeMap via `buildStructTypeMap(schemas, reasons)` helper
- Return `&StructResult{Golang: goBytes, TypeMap: typeMap}`

**Error Handling:**
- Return error if openapi is empty: "openapi input cannot be empty"
- Return error if opts.GoPackagePath is empty: "GoPackagePath cannot be empty"
- Propagate errors from ParseDocument, doc.Schemas, BuildMessages, BuildGoStructs, GenerateGo

**Context for implementation:**
- Follow validation pattern from `Convert()` at convert.go:74-84
- Create proto context via `internal.NewContext()` at line 101 (required by BuildMessages even though we ignore proto output)
- Call `BuildMessages()` at convert.go:102 to create dependency graph and validate schemas
- Discard ctx.Messages and ctx.Enums (proto IR) - only use the returned graph
- Compute transitive closure at convert.go:108 to get reasons map for TypeMap
- Follow Go generation pattern from `Convert()` at convert.go:132-142
- Build TypeMap using helper function with reasons from transitive closure

**Note on PackageName**:
BuildMessages requires PackageName in the schema validation flow, even though it's not used for Go struct generation. If opts.PackageName is empty, default it to "main" to satisfy BuildMessages requirements.

#### 2. Helper Function for StructResult TypeMap
**File**: `convert.go`
**Changes**: Add helper to build TypeMap for struct-only conversion

```go
// buildStructTypeMap creates TypeMap marking all schemas as Golang location
func buildStructTypeMap(schemas []*parser.SchemaEntry, reasons map[string]string) map[string]*TypeInfo
```

**Function Responsibilities:**
- Iterate over all schema entries
- For each schema name, create TypeInfo with Location=TypeLocationGolang
- If schema name exists in reasons map, use that reason (e.g., "contains oneOf", "variant of union type X")
- If schema name not in reasons map, set Reason="" (regular struct, not union-related)
- Return complete TypeMap

**Context for implementation:**
- Reference existing `buildTypeMap()` at convert.go:153-173 for structure
- The reasons map comes from `graph.ComputeTransitiveClosure()` which returns (goTypes, protoTypes, reasons)
- Reasons map contains entries like: "contains oneOf", "variant of union type Pet", "references union type Pet"
- All types should have TypeLocationGolang (no proto types in struct-only conversion)
- NOTE: Cannot access graph.unionReasons directly as it's unexported - must use transitive closure results

### Validation
- [ ] Run: `go build ./...`
- [ ] Verify: Package compiles without errors
- [ ] Run: `go test ./... -run TestConvertToStructBasics`
- [ ] Verify: Basic conversion test passes

### Open Questions for Implementation:
- **Enum handling**: The plan does not specify enum behavior. Current `BuildMessages()` processes enums (builder.go:105-121) but they don't become Go structs. Should enums be skipped in struct-only conversion or generate Go const declarations?
- **Empty schemas**: Should empty components/schemas return empty Golang bytes or minimal package declaration?

## Phase 2: Add Comprehensive Tests

### Overview
Add functional tests covering various schema types and edge cases.

### Changes Required:

#### 1. Basic Functionality Tests
**File**: `convert_test.go`
**Changes**: Add tests for ConvertToStruct validation and basic conversion

**Test Signatures:**
```go
func TestConvertToStructValidation(t *testing.T)
func TestConvertToStructSimpleSchemas(t *testing.T)
func TestConvertToStructWithUnions(t *testing.T)
```

**Test Objectives:**
- Verify input validation (empty openapi, empty GoPackagePath)
- Verify simple schemas convert to Go structs with proper fields
- Verify union types generate custom marshaling
- Verify all schemas appear in TypeMap as TypeLocationGolang
- Verify Golang field is populated, not empty
- Verify generated package name matches GoPackagePath
- Verify empty PackageName defaults to "main" and doesn't cause errors

**Context for implementation:**
- Follow test pattern from TestConvertBasics at convert_test.go:27-119
- Use table-driven tests with struct{name, given, expected, wantErr}
- For Go output validation, use `assert.Contains()` to check for struct definitions (more flexible than exact match)
- Use `require.NoError`, `require.NotNil` for critical assertions
- Use `assert.Equal`, `assert.Contains` for content verification
- Reference TestOneOfBasicGeneration at convert_test.go:899-979 for union testing pattern

#### 2. Union Type Tests
**File**: `convert_test.go`
**Changes**: Add tests verifying union types get custom marshaling

**Test Signatures:**
```go
func TestConvertToStructUnionMarshaling(t *testing.T)
func TestConvertToStructMultipleUnions(t *testing.T)
```

**Test Objectives:**
- Verify union types have MarshalJSON method
- Verify union types have UnmarshalJSON method
- Verify discriminator-based unmarshaling works
- Verify multiple unions in same schema work correctly
- Verify TypeMap marks unions with "contains oneOf" reason
- Verify variants are included in output

**Context for implementation:**
- Follow pattern from TestOneOfBasicGeneration at convert_test.go:899-979
- Check for `func (u *TypeName) MarshalJSON()` in output
- Check for `func (u *TypeName) UnmarshalJSON()` in output
- Verify discriminator switch cases exist

#### 3. Mixed Schema Tests
**File**: `convert_test.go`
**Changes**: Add tests with mix of regular and union schemas

**Test Signatures:**
```go
func TestConvertToStructMixedTypes(t *testing.T)
func TestConvertToStructReferencingTypes(t *testing.T)
```

**Test Objectives:**
- Verify schemas with both regular structs and unions convert correctly
- Verify types referencing unions are included in output
- Verify all types appear in TypeMap
- Verify no proto output is generated (not applicable for StructResult)
- Verify schema order is preserved

**Context for implementation:**
- Follow pattern from TestOneOfMultipleUnionFields at convert_test.go:1077-1167
- Use mixed OpenAPI with both regular and union schemas
- Verify all expected struct definitions exist in output
- Check TypeMap has all schemas

#### 4. Compilation and Runtime Tests
**File**: `convert_test.go`
**Changes**: Add tests verifying generated code compiles and runs

**Test Signatures:**
```go
func TestConvertToStructCodeCompiles(t *testing.T)
func TestConvertToStructJSONRoundTrip(t *testing.T)
```

**Test Objectives:**
- Verify generated Go code compiles without errors
- Verify JSON marshaling/unmarshaling works for regular structs
- Verify JSON marshaling/unmarshaling works for union types
- Verify discriminator-based routing in unions works at runtime

**Context for implementation:**
- Follow pattern from TestGeneratedGoCodeCompiles at convert_test.go:1169-1225
- Create temp directory with generated code
- Write go.mod file
- Run `go build ./...` to verify compilation
- Follow pattern from TestOneOfJSONRoundTrip at convert_test.go:1227-1334 for runtime tests
- Create test program that exercises marshaling/unmarshaling

#### 5. Edge Case Tests
**File**: `convert_test.go`
**Changes**: Add tests for edge cases and error conditions

**Test Signatures:**
```go
func TestConvertToStructEmptySchemas(t *testing.T)
func TestConvertToStructTypeMapCompleteness(t *testing.T)
func TestConvertToStructUnionOnlySchemas(t *testing.T)
```

**Test Objectives:**
- Verify empty components/schemas behavior (determine: empty bytes or package declaration?)
- Verify TypeMap contains all schemas
- Verify TypeMap reasons are correct for unions vs regular types
- Verify package name extraction works correctly (from GoPackagePath)
- Verify schemas containing only unions work correctly (no regular structs)

**Context for implementation:**
- Follow pattern from TestConvertExtractSchemas at convert_test.go:178-254
- Test with empty schemas section (decide expected behavior)
- Verify TypeMap has expected number of entries
- Check ExtractPackageName handles versioned paths (v1, v2) correctly at golang.go:396-405
- Test union-only schema case (Pet + Dog + Cat with no other types)

### Validation
- [ ] Run: `go test ./... -run TestConvertToStruct`
- [ ] Verify: All ConvertToStruct tests pass
- [ ] Run: `go test ./... -race`
- [ ] Verify: No race conditions detected
- [ ] Run: `go test ./...`
- [ ] Verify: All existing tests still pass (no regression)

## Phase 3: Update Documentation (Optional)

### Overview
This phase is optional but recommended for user-facing documentation.

### Changes Required:

#### 1. Function Documentation
**File**: `convert.go`
**Changes**: Ensure ConvertToStruct has comprehensive godoc

**Documentation should include:**
- Purpose: Converting all OpenAPI schemas to Go structs
- Behavior: Union types get custom marshaling, regular types get simple structs
- Parameters: openapi (YAML/JSON bytes), opts (package configuration)
- Return value: StructResult with Golang source and TypeMap
- Error conditions: Invalid inputs, parsing errors, generation errors
- Example usage (optional)

**Context for implementation:**
- Follow documentation style from Convert() at convert.go:51-73
- Mention that this generates Go-only output (no proto)
- Note that ConvertOptions.PackagePath is not used (only GoPackagePath matters)

### Validation
- [ ] Run: `go doc conv.ConvertToStruct`
- [ ] Verify: Documentation is clear and complete

## Technical Notes

### Dependency Graph Requirement
The dependency graph is needed for multiple purposes:
1. **Schema lookup**: `buildGoStruct()` calls `buildDiscriminatorMap()` at golang.go:93, which validates discriminator properties exist in variant schemas via `graph.schemas` at golang.go:201
2. **Union metadata**: Graph tracks which types are unions (`graph.hasUnion`), their variants (`graph.unionVariants`), and reasons (`graph.unionReasons`)
3. **Classification reasons**: `ComputeTransitiveClosure()` returns reasons map needed for TypeMap construction
4. **Schema validation**: BuildMessages performs schema validation during graph construction (builder.go:81-89)

The graph is populated during first pass of `BuildMessages()` at builder.go:69-90. Note that graph fields are unexported, so `buildGoStruct()` accesses them indirectly via method calls and the schemas parameter.

### Code Reuse Strategy
Maximum code reuse by:
- Using `BuildMessages()` to create graph and validate schemas (ignore proto Context.Messages and Context.Enums output)
  - Create empty Context via `internal.NewContext()` and pass to BuildMessages
  - BuildMessages returns graph which contains all needed schema metadata
  - Discard the proto IR (Context.Messages, Context.Enums) - only use the DependencyGraph return value
- Using `ComputeTransitiveClosure()` to get reasons map for TypeMap (ignore goTypes/protoTypes classification)
- Using `BuildGoStructs()` without modification (just pass all schemas as goTypes)
- Using `GenerateGo()` without modification
- Only new code: ConvertToStruct orchestration, StructResult type, buildStructTypeMap helper

### Testing Strategy
Follow project's functional testing philosophy:
- Tests use public API only (`ConvertToStruct()`)
- No direct calls to internal functions
- Tests in `conv_test` package for true external testing
- Verify generated code compiles and runs (integration-style tests)
- Use `require` for critical assertions, `assert` for content validation

### Type Classification Difference

**Convert()**: Uses transitive closure to filter
```
Pet (union) → Golang
├─ Dog (variant) → Golang
├─ Cat (variant) → Golang
Owner (references Pet) → Golang
Product (independent) → Proto
```

**ConvertToStruct()**: Marks all as Golang
```
Pet (union) → Golang
├─ Dog (variant) → Golang
├─ Cat (variant) → Golang
Owner (references Pet) → Golang
Product (independent) → Golang ✓ (difference)
```

### ConvertOptions Usage
For `ConvertToStruct()`:
- **Required**: `GoPackagePath` (must be non-empty)
- **Optional**: `PackageName` (defaults to "main" if empty)
- **Ignored**: `PackagePath` (proto-specific, not used for Go generation)

**Why PackageName is needed**:
Although PackageName is proto-specific and not used in the final Go output, BuildMessages() requires it during schema validation. The implementation should default PackageName to "main" if not provided, ensuring BuildMessages succeeds while not forcing users to provide a proto package name for struct-only conversion.

This ensures users explicitly specify where Go code should be generated (GoPackagePath) while providing sensible defaults for proto-specific fields.
