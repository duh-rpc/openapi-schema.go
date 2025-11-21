> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# OneOf Union Support Implementation Plan

## Overview

This plan implements support for OpenAPI `oneOf` discriminated unions by generating Go structs with custom JSON marshaling instead of Protocol Buffer messages for schemas containing unions. The library uses the **oapi-codegen pattern** where union types contain pointer fields to each variant, with custom marshaling to match the flat OpenAPI JSON format. A transitive closure approach determines which types must be generated as Go code, ensuring clean separation between proto and Go outputs.

## Current State Analysis

The library currently rejects any schema containing unions (`oneOf`, `anyOf`, `allOf`, `not`) because these cannot be represented in protobuf without breaking JSON compatibility (see docs/discriminated-unions.md).

### Key Discoveries:
- Validation occurs in two places:
  - `internal/mapper.go:224-248`: `validateSchema()` for properties
  - `internal/builder.go:308-332`: `validateTopLevelSchema()` for top-level schemas
- Current architecture: Parse OpenAPI → Build proto-specific IR → Generate proto3
- Single output path with data structures (ProtoMessage, ProtoField, ProtoEnum) tightly coupled to protobuf
- No existing mechanism for alternative code generation

## Desired End State

### Specification

The `Convert()` function returns a `ConvertResult` struct containing:
1. **Protobuf bytes**: Proto3 definitions for union-free types
2. **Golang bytes**: Go struct definitions for types containing or referencing unions, with custom JSON marshaling
3. **TypeMap**: Metadata indicating which types are in proto vs Go and why

**Union Detection Strategy:**
- Detect `oneOf` with discriminator in schemas (property-level or top-level)
- Reject `oneOf` without discriminator (require discriminator for Phase 1)
- Mark schema as "Go-only"
- Compute transitive closure: mark all schemas that reference union types as "Go-only"
- Mark union variant types (referenced in oneOf) as "Go-only"
- Generate proto for remaining types

**Go Code Format (oapi-codegen pattern):**
```go
// Union type with pointer fields to variants
type Pet struct {
    Dog *Dog
    Cat *Cat
}

// Custom marshaling to match flat OpenAPI JSON
func (p *Pet) MarshalJSON() ([]byte, error) { /* ... */ }
func (p *Pet) UnmarshalJSON(data []byte) error { /* ... */ }

// Variant types as separate structs
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

**ConvertOptions:**
- `GoPackagePath` defaults to `PackagePath` if not provided
- No backwards compatibility concerns (breaking API change is acceptable)

### Verification

**Success Criteria:**
```go
openapi := []byte(`
openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Address:
      type: object
      properties:
        street:
          type: string

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

    Owner:
      type: object
      properties:
        name:
          type: string
        pet:
          $ref: '#/components/schemas/Pet'
`)

result, err := Convert(openapi, ConvertOptions{
    PackageName:   "testpkg",
    PackagePath:   "github.com/example/proto/v1",
    GoPackagePath: "github.com/example/types/v1", // optional, defaults to PackagePath
})

require.NoError(t, err)

// Protobuf contains: Address message only
// Golang contains: Pet (union wrapper), Dog, Cat, Owner structs with MarshalJSON/UnmarshalJSON
// TypeMap contains:
//   "Address" -> TypeInfo{Location: "proto", Reason: ""}
//   "Pet" -> TypeInfo{Location: "golang", Reason: "contains oneOf"}
//   "Dog" -> TypeInfo{Location: "golang", Reason: "variant of union type Pet"}
//   "Cat" -> TypeInfo{Location: "golang", Reason: "variant of union type Pet"}
//   "Owner" -> TypeInfo{Location: "golang", Reason: "references union type Pet"}

// Verify generated Go code compiles and marshals correctly
var owner Owner
err = json.Unmarshal([]byte(`{"name":"Alice","pet":{"petType":"dog","bark":"woof"}}`), &owner)
require.NoError(t, err)
assert.Equal(t, "Alice", owner.Name)
assert.NotNil(t, owner.Pet.Dog)
assert.Equal(t, "woof", owner.Pet.Dog.Bark)
```

**Validation Commands:**
- `go test ./...` - All tests pass
- `make build` - Build succeeds

## What We're NOT Doing

- Support for `anyOf`, `allOf`, `not` (future phases)
- Support for `oneOf` without discriminators (requires discriminator in Phase 1)
- Support for inline oneOf variants (only `$ref`-based variants in Phase 1)
- Nested unions (unions within unions) in phase 1
- Cross-language references (proto referencing Go types)
- File writing (library returns bytes only)
- Intelligent pluralization for field names
- Backwards compatibility (breaking API change is acceptable)
- Custom validation types (email, uuid formats map to string in Phase 1)
- External dependencies (no openapi_types package)

## Implementation Approach

Use transitive closure to partition schemas into proto-compatible and Go-only sets. Build separate intermediate representations (IR) for each, then generate output in parallel. The existing proto generation pipeline remains unchanged; we add a parallel Go generation pipeline with custom JSON marshaling.

**Key Strategy:**
1. Change public API first (break everything intentionally in controlled way)
2. Fix all existing tests to use new API (establish working baseline)
3. Detect unions during schema processing (don't reject oneOf with discriminator)
4. Build dependency graph of schema references
5. Mark union variants as Go-only (Dog, Cat in the Pet example)
6. Compute transitive closure to identify all Go-only types
7. Split schemas into proto set and Go set
8. Generate proto output for proto set (existing pipeline)
9. Generate Go output for Go set with custom marshaling (new pipeline)

**Critical Design Decisions:**
- Only `$ref`-based oneOf variants supported (reject inline schemas)
- Discriminator required for Phase 1
- Case-insensitive matching between discriminator values and schema names
- Explicit mappings in discriminator.mapping override default matching

## Phase 1: Public API Changes

### Overview
Break the public API intentionally by changing the return type of `Convert()` to return a struct with separate proto/Go outputs and metadata. Initially implement with only proto output populated (existing behavior preserved), leaving Go output empty. This establishes the new interface that all subsequent phases will build upon.

### Changes Required:

#### 1. Define New Result Types
**File**: `convert.go`
**Changes**: Add new types for API

```go
type ConvertResult struct {
    Protobuf []byte
    Golang   []byte
    TypeMap  map[string]*TypeInfo
}

type TypeInfo struct {
    Location TypeLocation
    Reason   string
}

type TypeLocation string

const (
    TypeLocationProto  TypeLocation = "proto"
    TypeLocationGolang TypeLocation = "golang"
)
```

**Function Responsibilities:**
- ConvertResult: Container for both outputs plus metadata
- TypeInfo: Metadata about type location and reason
- TypeLocation: Enum for proto vs golang

**Context for implementation:**
- These types will be shared between convert.go and internal packages
- TypeInfo.Reason should be empty string for proto-only types initially
- TypeMap will initially contain all types as "proto" location

#### 2. Update ConvertOptions
**File**: `convert.go`
**Changes**: Add GoPackagePath field

```go
type ConvertOptions struct {
    PackageName   string
    PackagePath   string
    GoPackagePath string // New field, optional, defaults to PackagePath
}
```

**Function Responsibilities:**
- GoPackagePath: Optional package path for Go code generation
- If empty, defaults to PackagePath value
- Will be used in later phases for Go code generation

#### 3. Update Convert Function Signature
**File**: `convert.go`
**Changes**: Change return type

```go
func Convert(openapi []byte, opts ConvertOptions) (*ConvertResult, error)
```

**Function Responsibilities:**
- Change return type from `[]byte` to `*ConvertResult`
- Initially populate only Protobuf field (existing behavior)
- Set Golang field to empty/nil
- Populate TypeMap with all schemas as TypeLocationProto with empty reason
- Default GoPackagePath to PackagePath if not provided

**Implementation Details:**
```go
func Convert(openapi []byte, opts ConvertOptions) (*ConvertResult, error) {
    // Validate inputs
    if err := validateOptions(opts); err != nil {
        return nil, err
    }

    // Default GoPackagePath
    if opts.GoPackagePath == "" {
        opts.GoPackagePath = opts.PackagePath
    }

    // Parse OpenAPI document (existing logic from convert.go:49-57)
    doc, err := libopenapi.NewDocument(openapi)
    if err != nil {
        return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
    }

    v3Model, errs := doc.BuildV3Model()
    if len(errs) > 0 {
        return nil, fmt.Errorf("failed to build model: %v", errs)
    }

    // Extract schemas from document
    schemas := v3Model.Model.Components.Schemas
    if schemas == nil {
        schemas = orderedmap.New[string, *base.SchemaProxy]()
    }

    // Parse schemas into entries (existing logic)
    entries, err := parser.ParseSchemas(schemas)
    if err != nil {
        return nil, err
    }

    // Build proto messages (existing logic)
    ctx := builder.NewContext(opts.PackageName, opts.PackagePath)
    messages, err := builder.BuildMessages(entries, ctx)
    if err != nil {
        return nil, err
    }

    // Generate proto output (existing logic)
    protoBytes := generator.Generate(messages, ctx)

    // Build TypeMap (all types are proto for now)
    typeMap := buildInitialTypeMap(schemas)

    return &ConvertResult{
        Protobuf: protoBytes,
        Golang:   nil, // Empty for now
        TypeMap:  typeMap,
    }, nil
}

func buildInitialTypeMap(schemas *orderedmap.Map[string, *base.SchemaProxy]) map[string]*TypeInfo {
    typeMap := make(map[string]*TypeInfo)

    // Iterate over all schemas from OpenAPI document
    for pair := schemas.First(); pair != nil; pair = pair.Next() {
        name := pair.Key()
        typeMap[name] = &TypeInfo{
            Location: TypeLocationProto,
            Reason:   "", // Empty for proto-only types
        }
    }

    return typeMap
}
```

**Key Changes from Existing Code:**
- Existing `Convert()` at convert.go:40-70 returns `[]byte`
- New version returns `*ConvertResult` with same proto bytes in `Protobuf` field
- Added `buildInitialTypeMap()` to populate TypeMap from parsed schemas
- `schemas` comes from `v3Model.Model.Components.Schemas` (orderedmap from libopenapi)

**Context for implementation:**
- Modify existing Convert function at convert.go:40-70
- Extract schema names during parsing to populate TypeMap
- Existing proto generation logic remains unchanged
- This phase intentionally breaks all existing tests

### Validation Commands:
- `go build ./...` - Code compiles (tests will fail, expected)

## Phase 2: Update All Existing Tests

### Overview
Fix all existing tests to use the new `ConvertResult` API. This establishes a working baseline with the new interface before implementing union support. All tests should pass after this phase using `result.Protobuf` instead of direct bytes.

### Changes Required:

#### 1. Update Test Expectations
**File**: `convert_test.go`
**Changes**: Update all test cases for new return type

**Required Changes:**
- Change all `conv.Convert()` calls to expect `*ConvertResult, error`
- Use `result.Protobuf` instead of direct bytes
- Add assertion: `assert.Nil(t, result.Golang)` or `assert.Empty(t, result.Golang)`
- Add assertion: `assert.NotNil(t, result.TypeMap)`
- Verify TypeMap contains expected schema names as proto location

**Example transformation:**
```go
// Before
protoBytes, err := conv.Convert([]byte(openapi), conv.ConvertOptions{
    PackageName: "testpkg",
    PackagePath: "github.com/example/proto",
})
require.NoError(t, err)
assert.Contains(t, string(protoBytes), "message User")

// After
result, err := conv.Convert([]byte(openapi), conv.ConvertOptions{
    PackageName: "testpkg",
    PackagePath: "github.com/example/proto",
})
require.NoError(t, err)
require.NotNil(t, result)
assert.Empty(t, result.Golang) // No Go output yet
assert.NotNil(t, result.TypeMap)
assert.Equal(t, conv.TypeLocationProto, result.TypeMap["User"].Location)
assert.Empty(t, result.TypeMap["User"].Reason)
assert.Contains(t, string(result.Protobuf), "message User")
```

#### 2. Update Error Test Cases
**File**: `convert_test.go`
**Changes**: Ensure error cases still work

**Required Changes:**
- Verify error cases still return appropriate errors
- Confirm result is nil when error is returned
- Keep existing error message validation

#### 3. Add TypeMap Validation
**File**: `convert_test.go`
**Changes**: Add comprehensive TypeMap checks

**Required Checks:**
- TypeMap contains all top-level schemas
- All entries have Location = TypeLocationProto
- All entries have empty Reason (for proto-only types)
- TypeMap keys match schema names from OpenAPI spec

**Example helper function:**
```go
func assertProtoOnlyTypeMap(t *testing.T, result *conv.ConvertResult, expectedSchemas []string) {
    require.NotNil(t, result.TypeMap)
    assert.Len(t, result.TypeMap, len(expectedSchemas))

    for _, schema := range expectedSchemas {
        info, exists := result.TypeMap[schema]
        require.True(t, exists, "schema %s not in TypeMap", schema)
        assert.Equal(t, conv.TypeLocationProto, info.Location)
        assert.Empty(t, info.Reason)
    }
}
```

### Validation Commands:
- `go test ./...` - All tests pass with new API

## Phase 3: Union Detection & Validation

### Overview
Modify validation logic to detect (not reject) `oneOf` with discriminators. Schemas with unions should pass validation but still generate proto-only output at this phase. Add tests confirming oneOf schemas are accepted.

### Changes Required:

#### 1. Update Validation Logic
**File**: `internal/mapper.go`
**Changes**: Modify union validation to detect rather than reject oneOf with discriminator

```go
func validateSchema(schema *base.Schema, propertyName string) error
```

**Function Responsibilities:**
- Allow oneOf with discriminator to pass validation
- **Reject oneOf with inline variants** (only $ref variants supported in Phase 1)
- Reject oneOf without discriminator with clear error message
- Continue rejecting anyOf/allOf/not
- Keep existing validation for other unsupported features

**Implementation Details:**
```go
// Check for oneOf
if schema.OneOf != nil && len(schema.OneOf) > 0 {
    // Require discriminator
    if schema.Discriminator == nil || schema.Discriminator.PropertyName == "" {
        return fmt.Errorf("oneOf in property '%s' requires discriminator", propertyName)
    }

    // Require all variants to be $ref (no inline schemas)
    for i, variant := range schema.OneOf {
        if !variant.IsReference() {
            return fmt.Errorf("oneOf variant %d in property '%s' must use $ref, inline schemas not supported", i, propertyName)
        }
    }

    // Valid oneOf - will be handled as Go code
    return nil
}
```

**File**: `internal/builder.go`
**Changes**: Modify top-level validation

```go
func validateTopLevelSchema(schema *base.Schema, schemaName string) error
```

**Function Responsibilities:**
- Same logic as validateSchema but for top-level schemas
- Allow oneOf with discriminator and $ref variants
- Reject oneOf without discriminator or with inline variants

**Context for implementation:**
- Follow error handling patterns from internal/errors.go:11-27
- Reference existing validation in internal/mapper.go:224-248
- Check schema.Discriminator.PropertyName for discriminator existence
- Use SchemaProxy.IsReference() to detect $ref vs inline

#### 2. Add Basic Union Tests
**File**: `convert_test.go`
**Changes**: Add tests for oneOf acceptance

**Test Cases:**
```go
func TestOneOfWithDiscriminatorAccepted(t *testing.T) {
    // Test that oneOf with discriminator doesn't error
    // Still generates proto-only output at this phase
}

func TestOneOfWithoutDiscriminatorRejected(t *testing.T) {
    // Test that oneOf without discriminator returns clear error
}

func TestOneOfWithInlineVariantRejected(t *testing.T) {
    // Test that oneOf with inline schema (not $ref) returns error
}
```

**Test Objectives:**
- Verify oneOf with discriminator and $ref variants passes validation
- Verify oneOf without discriminator returns error containing "discriminator"
- Verify oneOf with inline variant returns error containing "inline" or "$ref"
- Output should still be proto-only (no Go code generated yet)

#### 3. Update Existing Error Tests
**File**: `internal/errors_test.go` or similar
**Changes**: Update error message expectations

**Required Updates:**
Existing tests at `internal/errors_test.go:134-191` (if they exist) expect error message:
```
"uses 'oneOf' which is not supported"
```

After Phase 3, oneOf WITH discriminator and $ref variants is accepted, so error messages change:
- oneOf without discriminator: `"oneOf requires discriminator"`
- oneOf with inline variant: `"oneOf variant must use $ref, inline schemas not supported"`
- anyOf/allOf/not: Keep existing error messages (still unsupported)

**Action Items:**
1. Find all tests asserting oneOf error messages
2. Update to new error messages or add discriminator to test schemas
3. Add new test cases for discriminator-related errors

**Example Update:**
```go
// Before
{
    name: "oneOf not supported",
    schema: `oneOf: [...]`,
    wantErr: "uses 'oneOf' which is not supported",
}

// After - Add discriminator to make it valid
{
    name: "oneOf with discriminator accepted",
    schema: `
      oneOf:
        - $ref: '#/components/schemas/Dog'
      discriminator:
        propertyName: petType`,
    wantErr: "", // No error
}

// New test for missing discriminator
{
    name: "oneOf requires discriminator",
    schema: `oneOf: [...]`,  // No discriminator
    wantErr: "oneOf requires discriminator",
}
```

### Validation Commands:
- `go test ./...` - All tests pass, oneOf schemas accepted but still generate proto

## Phase 4: Dependency Graph & Classification

### Overview
Build dependency graph during schema processing, compute transitive closure to identify which types must be generated as Go code, and populate TypeMap with correct locations and reasons. Types are still generated as proto-only at this phase, but TypeMap will show intended future state.

### Changes Required:

#### 1. Dependency Graph Infrastructure
**File**: `internal/dependencies.go` (new file)
**Changes**: Create dependency tracking infrastructure

```go
type DependencyGraph struct {
    schemas       map[string]*base.SchemaProxy
    edges         map[string][]string // from -> []to dependencies
    hasUnion      map[string]bool
    unionReasons  map[string]string
    unionVariants map[string][]string // union name -> variant names
}

func NewDependencyGraph() *DependencyGraph

func (g *DependencyGraph) AddSchema(name string, proxy *base.SchemaProxy) error

func (g *DependencyGraph) AddDependency(from, to string)

func (g *DependencyGraph) MarkUnion(schemaName, reason string, variants []string)

func (g *DependencyGraph) ComputeTransitiveClosure() (goTypes, protoTypes map[string]bool, reasons map[string]string)
```

**Function Responsibilities:**
- `NewDependencyGraph()`: Initialize empty graph with maps
- `AddSchema()`: Register schema in graph, detect if it contains oneOf
- `AddDependency()`: Record reference from one schema to another
- `MarkUnion()`: Mark schema as containing union with reason, track variant names
- `ComputeTransitiveClosure()`: Use BFS to find all schemas referencing union types, mark variants as Go-only, return reasons

**Implementation Details:**
```go
func (g *DependencyGraph) ComputeTransitiveClosure() (goTypes, protoTypes map[string]bool, reasons map[string]string) {
    goTypes = make(map[string]bool)
    reasons = make(map[string]string)
    visited := make(map[string]bool) // Track visited nodes to prevent infinite loops

    // Mark direct union types
    for name, reason := range g.unionReasons {
        goTypes[name] = true
        reasons[name] = reason
        visited[name] = true
    }

    // Mark union variants
    for unionName, variants := range g.unionVariants {
        for _, variant := range variants {
            if !goTypes[variant] { // Only set if not already marked
                goTypes[variant] = true
                reasons[variant] = fmt.Sprintf("variant of union type %s", unionName)
                visited[variant] = true
            }
        }
    }

    // BFS to find all types referencing Go-only types
    // Work backwards: find types that depend on (reference) Go-only types
    queue := make([]string, 0)
    for name := range goTypes {
        queue = append(queue, name)
    }

    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        // Find all types that depend on current (reference it)
        for from, deps := range g.edges {
            // Skip if already processed
            if visited[from] {
                continue
            }

            // Check if 'from' references 'current'
            for _, to := range deps {
                if to == current {
                    // Mark 'from' as Go-only because it references a Go-only type
                    goTypes[from] = true
                    reasons[from] = fmt.Sprintf("references union type %s", current)
                    visited[from] = true
                    queue = append(queue, from)
                    break // No need to check other dependencies
                }
            }
        }
    }

    // Proto types are everything else
    protoTypes = make(map[string]bool)
    for name := range g.schemas {
        if !goTypes[name] {
            protoTypes[name] = true
        }
    }

    return goTypes, protoTypes, reasons
}
```

**Algorithm Notes:**
- **Visited set**: Prevents processing the same node multiple times, avoiding infinite loops with circular dependencies
- **Edge direction**: `edges[A] = [B, C]` means "A depends on (references) B and C"
- **Transitive closure**: If A references B and B is Go-only, then A becomes Go-only
- **Circular dependencies**: Handled correctly - if A→B→A where B has union, both become Go-only
- **Reason tracking**: First reason assigned is kept (variants get more specific reason than "references union")

**Example with Circular Dependency:**
```yaml
Owner:
  properties:
    pet: {$ref: '#/components/schemas/Pet'}
Pet:
  oneOf: [Dog, Cat]
  discriminator: {propertyName: petType}
  properties:
    owner: {$ref: '#/components/schemas/Owner'}  # Circular!
Dog: {...}
Cat: {...}
```

**Processing:**
1. Pet marked as union (Go-only): `reasons[Pet] = "contains oneOf"`
2. Dog, Cat marked as variants (Go-only): `reasons[Dog] = "variant of union type Pet"`
3. BFS processes Pet: finds Owner references Pet → Owner becomes Go-only
4. BFS processes Owner: finds Pet references Owner → already visited, skip
5. Result: All four types are Go-only (correct)

**Context for implementation:**
- Use BFS (not DFS) to ensure all paths explored
- For oneOf schemas, extract variant names from $ref paths in schema.OneOf
- Handle $ref format: "#/components/schemas/TypeName" -> extract "TypeName"

#### 2. Integrate Graph into Builder
**File**: `internal/builder.go`
**Changes**: Build dependency graph during schema processing

**Modify BuildMessages:**
```go
func BuildMessages(entries []*parser.SchemaEntry, ctx *Context) ([]*ProtoMessage, *DependencyGraph, error)
```

**Function Responsibilities:**
- Create dependency graph
- For each schema, call AddSchema on graph
- Detect oneOf in schemas and mark them with variants using MarkUnion
- Track $ref dependencies using AddDependency
- Still build ProtoMessage for all types (Go generation comes in next phase)
- Return both proto messages and dependency graph

**Implementation Approach:**
```go
func BuildMessages(entries []*parser.SchemaEntry, ctx *Context) ([]*ProtoMessage, *DependencyGraph, error) {
    graph := NewDependencyGraph()

    // First pass: Add all schemas to graph and detect unions
    for _, entry := range entries {
        graph.AddSchema(entry.Name, entry.Proxy)

        schema := entry.Proxy.Schema()
        if schema == nil {
            continue
        }

        // Detect oneOf and mark as union
        if schema.OneOf != nil && len(schema.OneOf) > 0 {
            variants := extractVariantNames(schema.OneOf)
            graph.MarkUnion(entry.Name, "contains oneOf", variants)
        }
    }

    // Second pass: Build messages and track dependencies
    for _, entry := range entries {
        msg, err := buildMessage(entry, ctx, graph)
        if err != nil {
            return nil, nil, err
        }
        ctx.Messages = append(ctx.Messages, msg)
    }

    return ctx.Messages, graph, nil
}

// Called during property processing in buildMessage or mapFieldType
func trackDependency(fromSchema string, propProxy *base.SchemaProxy, graph *DependencyGraph) {
    if propProxy.IsReference() {
        toSchema := extractReferenceName(propProxy.GetReference())
        graph.AddDependency(fromSchema, toSchema)
    }
}
```

**WHERE to call AddDependency:**

1. **In property processing** (when building proto field):
```go
// In buildMessage when processing schema.Properties
for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
    propName := pair.Key()
    propProxy := pair.Value()

    // Track dependency if property references another schema
    if propProxy.IsReference() {
        refName := extractReferenceName(propProxy.GetReference())
        graph.AddDependency(msgName, refName)
    }

    // Also track for object types with schema
    propSchema := propProxy.Schema()
    if propSchema != nil && propSchema.Type[0] == "object" {
        // Check nested refs
        trackNestedDependencies(msgName, propSchema, graph)
    }

    // Continue with field building...
}
```

2. **In array item processing**:
```go
// When processing array items
if schema.Type[0] == "array" && schema.Items != nil {
    itemProxy := schema.Items.A
    if itemProxy.IsReference() {
        refName := extractReferenceName(itemProxy.GetReference())
        graph.AddDependency(msgName, refName)
    }
}
```

3. **In oneOf variant processing**:
```go
// When detecting oneOf
if schema.OneOf != nil {
    variants := make([]string, 0)
    for _, variantProxy := range schema.OneOf {
        if variantProxy.IsReference() {
            variantName := extractReferenceName(variantProxy.GetReference())
            variants = append(variants, variantName)
            // Don't call AddDependency here - variants are special case
            // They become Go-only by MarkUnion, not by dependency
        }
    }
    graph.MarkUnion(msgName, "contains oneOf", variants)
}
```

**Helper Function:**
```go
func extractVariantNames(oneOf []*base.SchemaProxy) []string {
    variants := make([]string, 0, len(oneOf))
    for _, variant := range oneOf {
        if variant.IsReference() {
            name := extractReferenceName(variant.GetReference())
            variants = append(variants, name)
        }
    }
    return variants
}

// Reuse existing extractReferenceName from mapper.go:205-222
// Extracts "Dog" from "#/components/schemas/Dog"
```

**Context for implementation:**
- Modify existing BuildMessages function at internal/builder.go:63-81
- Use libopenapi SchemaProxy.IsReference() for detecting refs
- Extract variant names from oneOf $ref paths
- Continue generating proto for all types at this phase
- AddDependency is called during property traversal, NOT for oneOf variants

#### 3. Update Convert to Use Graph
**File**: `convert.go`
**Changes**: Use dependency graph to populate TypeMap correctly

**Update Convert function:**
- Call BuildMessages to get both proto messages and dependency graph
- Call ComputeTransitiveClosure on graph to get goTypes, protoTypes, reasons
- Populate TypeMap using the reasons map (types in goTypes get "golang" location, others "proto")
- Still generate only proto output (Go generation in next phase)

**Implementation Details:**
```go
func Convert(openapi []byte, opts ConvertOptions) (*ConvertResult, error) {
    // ... existing parsing ...

    // Build messages and dependency graph
    messages, graph, err := builder.BuildMessages(entries, ctx)
    if err != nil {
        return nil, err
    }

    // Compute transitive closure
    goTypes, protoTypes, reasons := graph.ComputeTransitiveClosure()

    // Build TypeMap
    typeMap := make(map[string]*TypeInfo)
    for name := range goTypes {
        typeMap[name] = &TypeInfo{
            Location: TypeLocationGolang,
            Reason:   reasons[name],
        }
    }
    for name := range protoTypes {
        typeMap[name] = &TypeInfo{
            Location: TypeLocationProto,
            Reason:   "",
        }
    }

    // Generate proto for ALL types (still) - Go generation in next phase
    protoBytes := generator.Generate(messages, ctx)

    return &ConvertResult{
        Protobuf: protoBytes,
        Golang:   nil, // Still empty
        TypeMap:  typeMap,
    }, nil
}
```

#### 4. Add TypeMap Classification Tests
**File**: `convert_test.go`
**Changes**: Add tests for TypeMap classification

**Test Cases:**
```go
func TestTypeMapClassifiesUnionTypes(t *testing.T) {
    // Test that union types are marked as "golang" in TypeMap
    // Verify reason is "contains oneOf"
}

func TestTypeMapClassifiesVariants(t *testing.T) {
    // Test that union variants (Dog, Cat) are marked as "golang"
    // Verify reason is "variant of union type Pet"
}

func TestTypeMapClassifiesReferencingTypes(t *testing.T) {
    // Test that Owner (references Pet union) is marked as "golang"
    // Verify reason is "references union type Pet"
}

func TestTypeMapTransitiveClosure(t *testing.T) {
    // Test A -> B -> C where C has union
    // Verify all three are marked as "golang"
}
```

### Validation Commands:
- `go test ./...` - All tests pass, TypeMap correctly classifies types

## Phase 5: Go Code IR & Type Mappings

### Overview
Create data structures to represent Go types with union support, mirroring the existing proto IR pattern. Document comprehensive type mappings from OpenAPI to Go native types.

### OpenAPI to Go Type Mappings

The following table shows how OpenAPI types/formats map to Go types (compared to proto3 and oapi-codegen):

| OpenAPI Type | Format | Proto3 Type | Our Go Type | oapi-codegen | Rationale |
|--------------|--------|-------------|-------------|--------------|-----------|
| integer | (none) | int32 | **int32** | int | Proto3 consistency, explicit size |
| integer | int8 | ❌ | **int8** | int8 | Standard OpenAPI format |
| integer | int16 | ❌ | **int16** | int16 | Standard OpenAPI format |
| integer | int32 | int32 | **int32** | int32 | Matches proto3 |
| integer | int64 | int64 | **int64** | int64 | Matches proto3 |
| integer | uint8 | ❌ | **uint8** | uint8 | Standard OpenAPI format |
| integer | uint16 | ❌ | **uint16** | uint16 | Standard OpenAPI format |
| integer | uint32 | ❌ | **uint32** | uint32 | Standard OpenAPI format |
| integer | uint64 | ❌ | **uint64** | uint64 | Standard OpenAPI format |
| number | (none) | double | **float64** | float32 | OpenAPI default is "double" precision |
| number | float | float | **float32** | float32 | Matches proto3 and oapi-codegen |
| number | double | double | **float64** | float64 | Explicit double precision |
| string | (none) | string | **string** | string | Basic type |
| string | byte | bytes | **[]byte** | []byte | Binary data |
| string | binary | bytes | **[]byte** | []byte | Binary data |
| string | date | google.protobuf.Timestamp | **time.Time** | openapi_types.Date | Simple, no dependencies |
| string | date-time | google.protobuf.Timestamp | **time.Time** | time.Time | Matches proto3 intent |
| string | email | ❌ | **string** | openapi_types.Email | Phase 1: no validation types |
| string | uuid | ❌ | **string** | openapi_types.UUID | Phase 1: no validation types |
| string | password | ❌ | **string** | string | No special handling |
| boolean | (none) | bool | **bool** | bool | Basic type |
| array | (none) | repeated ElementType | **[]ElementType** | []ElementType | Go slice |
| object | (none) | MessageName (embedded) | **\*TypeName** (pointer) | \*TypeName | Go pointer |
| $ref | (none) | MessageName (embedded) | **\*TypeName** (pointer) | \*TypeName | Go pointer |

**Design Decisions:**
- **int32 default**: Maintains proto3 consistency and explicit cross-platform size
- **float64 default**: OpenAPI spec default is "double" precision; Go float literals default to float64
- **time.Time for dates**: Simple, standard library, no external dependencies
- **string for email/uuid/password**: Phase 1 focuses on code generation, not validation; users can add validation later
- **Pointer types for objects**: Go idiom for optional/nullable complex types

**Key Differences from Proto3:**
- Proto uses `double`/`float` keywords; Go uses `float64`/`float32` native types
- Proto uses `bytes` keyword; Go uses `[]byte` slice type
- Proto uses `repeated` keyword; Go uses `[]` slice syntax
- Proto uses `google.protobuf.Timestamp`; Go uses `time.Time` from standard library
- Proto embeds messages directly; Go uses pointers to structs
- Proto doesn't support unsigned integers or int8/int16; Go does

### Changes Required:

#### 1. Go Type Structures ✓
**File**: `internal/golang.go` (new file)
**Changes**: Define Go-specific IR structures

```go
type GoStruct struct {
    Name               string
    Description        string
    Fields             []*GoField
    IsUnion            bool
    UnionVariants      []string
    Discriminator      string
    DiscriminatorMap   map[string]string // discriminator value -> type name
}

type GoField struct {
    Name        string
    Type        string
    JSONName    string
    Description string
    IsPointer   bool
}

type GoContext struct {
    Tracker       *NameTracker
    Structs       []*GoStruct
    PackageName   string
    NeedsTime     bool // Flag for time.Time import
}

func NewGoContext(packageName string) *GoContext
```

**Function Responsibilities:**
- GoStruct: Represents Go struct definition with union metadata
- GoField: Represents struct field with Go type, JSON tag, pointer flag
- GoContext: Holds state during Go code generation including package name
- DiscriminatorMap: Maps discriminator values to type names (case-insensitive)
- NewGoContext(): Initialize empty context with package name

**Context for implementation:**
- Mirror pattern from internal/builder.go:31-60 for ProtoMessage/ProtoField
- Use same NameTracker pattern for conflict resolution
- IsUnion flag indicates struct is union wrapper (has variant fields)
- UnionVariants lists variant type names for union structs
- DiscriminatorMap built from schema.Discriminator.Mapping or inferred

#### 2. Go Builder Functions ✓
**File**: `internal/golang.go`
**Changes**: Add builder functions for Go structs

```go
func BuildGoStructs(entries []*parser.SchemaEntry, goTypes map[string]bool, graph *DependencyGraph, ctx *GoContext) error

func buildGoStruct(name string, proxy *base.SchemaProxy, graph *DependencyGraph, ctx *GoContext) (*GoStruct, error)

func goType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *GoContext) (typeName string, isPointer bool, error)

func buildDiscriminatorMap(schema *base.Schema, variants []string) (map[string]string, error)
```

**Function Responsibilities:**
- `BuildGoStructs()`: Process schemas marked as Go-only, build GoStruct for each
- `buildGoStruct()`: Build Go struct - if oneOf present, create union wrapper; otherwise regular struct
- `goType()`: Map OpenAPI type to Go type using table above
- `buildDiscriminatorMap()`: Build map from discriminator values to type names

**Implementation Details for buildGoStruct:**
```go
func buildGoStruct(name string, proxy *base.SchemaProxy, graph *DependencyGraph, ctx *GoContext) (*GoStruct, error) {
    schema := proxy.Schema()
    if schema == nil {
        return nil, fmt.Errorf("schema for '%s' is nil", name)
    }

    goStruct := &GoStruct{
        Name:        name,
        Description: schema.Description,
        Fields:      make([]*GoField, 0),
    }

    // Check if this is a union type (schema-level oneOf)
    if schema.OneOf != nil && len(schema.OneOf) > 0 {
        // This is a union wrapper - create pointer fields for each variant
        goStruct.IsUnion = true
        goStruct.Discriminator = schema.Discriminator.PropertyName

        variants := extractVariantNames(schema.OneOf)
        goStruct.UnionVariants = variants

        // Build discriminator map with validation
        discriminatorMap, err := buildDiscriminatorMap(schema, variants, graph.schemas)
        if err != nil {
            return nil, err
        }
        goStruct.DiscriminatorMap = discriminatorMap

        // Create pointer field for each variant
        for _, variantName := range variants {
            goStruct.Fields = append(goStruct.Fields, &GoField{
                Name:      variantName,
                Type:      "*" + variantName, // Always pointer
                JSONName:  "-",               // Union types don't marshal fields directly
                IsPointer: false,             // Pointer already in Type string
            })
        }

        return goStruct, nil
    }

    // Regular struct - process properties
    if schema.Properties == nil {
        // Empty struct
        return goStruct, nil
    }

    for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
        propName := pair.Key()
        propProxy := pair.Value()

        // Get Go type for this property
        propSchema := propProxy.Schema()
        typeName, isPointer, err := goType(propSchema, propName, propProxy, ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to map type for property '%s': %w", propName, err)
        }

        // Convert property name to Go field name (PascalCase)
        fieldName := ToPascalCase(propName)

        goStruct.Fields = append(goStruct.Fields, &GoField{
            Name:        fieldName,
            Type:        typeName,
            JSONName:    propName,  // Original OpenAPI property name
            Description: propSchema.Description,
            IsPointer:   isPointer, // Not used if Type already has *
        })
    }

    return goStruct, nil
}
```

**Pointer Semantics Clarification:**

The `goType()` function returns `(typeName string, isPointer bool, error)` where:
- **For scalar types**: `typeName` does NOT include `*`, `isPointer` indicates if pointer needed
  - Example: `("string", false, nil)` → use as `string`
  - Example: `("int32", false, nil)` → use as `int32`

- **For object/ref types**: `typeName` INCLUDES `*`, `isPointer=false` (pointer already in string)
  - Example: `("*Dog", false, nil)` → use as `*Dog`
  - Example: `("*Owner", false, nil)` → use as `*Owner`

- **For array types**: `typeName` is complete slice syntax, `isPointer=false`
  - Example: `("[]string", false, nil)` → use as `[]string`
  - Example: `("[]*Dog", false, nil)` → use as `[]*Dog`

**Why this design?**
- Object types are ALWAYS pointers in Go (for optional/nullable semantics)
- The `*` is included in the type string to simplify usage
- `isPointer` flag is legacy/unused in current design
- Future: Could use `isPointer` flag if we want value semantics for some objects

**Implementation Details for buildDiscriminatorMap:**
```go
func buildDiscriminatorMap(schema *base.Schema, variants []string, schemas map[string]*base.SchemaProxy) (map[string]string, error) {
    mapping := make(map[string]string)
    discriminatorProp := schema.Discriminator.PropertyName

    // If explicit mapping exists, use it
    if schema.Discriminator != nil && len(schema.Discriminator.Mapping) > 0 {
        for value, ref := range schema.Discriminator.Mapping {
            typeName := extractTypeNameFromRef(ref) // Extract "Dog" from "#/components/schemas/Dog"

            // Check for conflicts (case-insensitive)
            lowerValue := strings.ToLower(value)
            if existing, exists := mapping[lowerValue]; exists && existing != typeName {
                return nil, fmt.Errorf("discriminator conflict: values '%s' and '%s' both map to lowercase '%s'",
                    existing, value, lowerValue)
            }

            mapping[lowerValue] = typeName // Store lowercase for case-insensitive lookup
        }

        // Validate that all variants are covered by mapping
        for _, variant := range variants {
            found := false
            for _, mappedType := range mapping {
                if mappedType == variant {
                    found = true
                    break
                }
            }
            if !found {
                return nil, fmt.Errorf("variant '%s' not covered by discriminator mapping", variant)
            }
        }

        return mapping, nil
    }

    // Otherwise, build case-insensitive mapping from variant names
    for _, variant := range variants {
        lowerVariant := strings.ToLower(variant)

        // Check for conflicts (e.g., "Dog" and "dog" both exist)
        if existing, exists := mapping[lowerVariant]; exists && existing != variant {
            return nil, fmt.Errorf("discriminator conflict: variants '%s' and '%s' both map to lowercase '%s'",
                existing, variant, lowerVariant)
        }

        mapping[lowerVariant] = variant // "dog" -> "Dog"
    }

    // Validate that discriminator property exists in all variant schemas
    for _, variant := range variants {
        variantProxy, exists := schemas[variant]
        if !exists {
            return nil, fmt.Errorf("variant '%s' not found in schemas", variant)
        }

        variantSchema := variantProxy.Schema()
        if variantSchema == nil {
            return nil, fmt.Errorf("variant '%s' has nil schema", variant)
        }

        // Check if discriminator property exists
        if variantSchema.Properties == nil {
            return nil, fmt.Errorf("discriminator property '%s' missing in variant '%s' (no properties)",
                discriminatorProp, variant)
        }

        hasDiscriminator := false
        for pair := variantSchema.Properties.First(); pair != nil; pair = pair.Next() {
            if pair.Key() == discriminatorProp {
                hasDiscriminator = true
                break
            }
        }

        if !hasDiscriminator {
            return nil, fmt.Errorf("discriminator property '%s' missing in variant '%s'",
                discriminatorProp, variant)
        }
    }

    return mapping, nil
}
```

**Validation Performed:**
1. **Conflict detection**: Two discriminator values with same lowercase form (e.g., "Dog" and "DOG")
2. **Discriminator property existence**: Verify each variant has the discriminator property
3. **Mapping completeness**: With explicit mapping, verify all variants are covered
4. **Schema existence**: Verify all variant names reference actual schemas

**Error Examples:**
- Conflict: `"discriminator conflict: variants 'Dog' and 'DOG' both map to lowercase 'dog'"`
- Missing property: `"discriminator property 'petType' missing in variant 'Dog'"`
- Missing variant: `"variant 'Cat' not found in schemas"`

**Context for implementation:**
- Follow builder pattern from internal/builder.go:83-158
- Reference type mapping logic from internal/mapper.go:88-118
- For union structs: create pointer field for each variant name
- For regular structs: iterate properties normally
- Use type mappings table from overview section
- Set ctx.NeedsTime flag when time.Time is used

#### 3. Type Mapping Functions ✓
**File**: `internal/golang.go`
**Changes**: Implement OpenAPI to Go type mapping

```go
func mapGoScalarType(typ, format string, ctx *GoContext) (string, error) {
    switch typ {
    case "integer":
        switch format {
        case "int8":
            return "int8", nil
        case "int16":
            return "int16", nil
        case "int32":
            return "int32", nil
        case "int64":
            return "int64", nil
        case "uint8":
            return "uint8", nil
        case "uint16":
            return "uint16", nil
        case "uint32":
            return "uint32", nil
        case "uint64":
            return "uint64", nil
        case "int", "":
            return "int32", nil // Default to int32 for proto3 consistency
        default:
            return "", fmt.Errorf("unsupported integer format: %s", format)
        }

    case "number":
        switch format {
        case "float":
            return "float32", nil
        case "double", "":
            return "float64", nil // Default to float64 (double precision)
        default:
            return "", fmt.Errorf("unsupported number format: %s", format)
        }

    case "string":
        switch format {
        case "date", "date-time":
            ctx.NeedsTime = true
            return "time.Time", nil
        case "byte", "binary":
            return "[]byte", nil
        case "email", "uuid", "password", "":
            // Phase 1: All string formats map to string
            // Future: Could add validation types
            return "string", nil
        default:
            // Unknown format defaults to string
            return "string", nil
        }

    case "boolean":
        return "bool", nil

    default:
        return "", fmt.Errorf("unsupported type: %s", typ)
    }
}

func mapGoArrayType(schema *base.Schema, propProxy *base.SchemaProxy, ctx *GoContext) (string, error) {
    // Get element type
    elementType, isPointer, err := goType(schema.Items.A.Schema(), "item", schema.Items.A, ctx)
    if err != nil {
        return "", err
    }

    // Build slice type
    if isPointer {
        return "[]" + elementType, nil // Already has *
    }
    return "[]" + elementType, nil
}

func mapGoObjectType(propertyName string, propProxy *base.SchemaProxy, ctx *GoContext) (string, bool, error) {
    // Extract type name from $ref or inline schema
    typeName := extractTypeName(propProxy)

    // Objects/refs are always pointers in Go
    return "*" + typeName, false, nil // false because * already in type
}
```

**Function Responsibilities:**
- mapGoScalarType: Map OpenAPI scalars using type table (comprehensive integer/number format support)
- mapGoArrayType: Map arrays to Go slices
- mapGoObjectType: Map object references to pointer types
- Set ctx.NeedsTime when time.Time is used
- Default integer format (empty or "int") → int32 for proto3 consistency
- Default number format (empty) → float64 for double precision
- Unknown string formats → string (permissive, Phase 1)

#### 4. Package Name Extraction ✓
**File**: `internal/golang.go`
**Changes**: Add package name extraction from Go package path

```go
func extractPackageName(packagePath string) string {
    if packagePath == "" {
        return "main"
    }

    // Split by / to get path components
    parts := strings.Split(packagePath, "/")
    if len(parts) == 0 {
        return "main"
    }

    // Get last component
    last := parts[len(parts)-1]

    // Check if last component is version (v1, v2, etc.)
    if strings.HasPrefix(last, "v") && len(last) > 1 {
        // Try to parse as number
        if _, err := strconv.Atoi(last[1:]); err == nil {
            // It's a version, use second-to-last component if available
            if len(parts) > 1 {
                return parts[len(parts)-2]
            }
        }
    }

    return last
}
```

**Function Responsibilities:**
- Extract package name from full Go package path
- Handle version suffixes (e.g., "github.com/example/types/v1" → "types")
- Handle simple paths (e.g., "github.com/example/types" → "types")
- Handle edge cases (empty path → "main", single component → use as-is)

**Examples:**
```go
extractPackageName("github.com/example/proto/v1")      // "proto"
extractPackageName("github.com/example/types")          // "types"
extractPackageName("mypackage")                         // "mypackage"
extractPackageName("")                                  // "main"
extractPackageName("github.com/user/repo/v2")          // "repo"
extractPackageName("github.com/user/v1")               // "user"
```

### Validation Commands:
- `go test ./...` - All tests pass, no generation yet (just IR)

## Phase 6: Go Code Generation with Custom Marshaling

### Overview
Generate Go source code from GoStruct IR using text templates, including custom MarshalJSON/UnmarshalJSON methods for union types. Populate result.Golang field with generated code.

### Changes Required:

#### 1. Go Code Generator
**File**: `internal/gogen.go` (new file)
**Changes**: Create Go code generator with marshaling

```go
const goTemplate = `package {{.PackageName}}

import (
	"encoding/json"
	"fmt"
{{if .NeedsTime}}	"time"
{{end}}
)

{{range .Structs}}{{renderStruct .}}{{end}}
`

type goTemplateData struct {
    PackageName string
    Structs     []*GoStruct
    NeedsTime   bool
}

func GenerateGo(ctx *GoContext) ([]byte, error)

func renderStruct(s *GoStruct) string

func renderField(f *GoField, indent string) string

func renderUnionMarshal(s *GoStruct) string

func renderUnionUnmarshal(s *GoStruct) string
```

**Function Responsibilities:**
- `GenerateGo()`: Execute template to produce Go source code with imports
- `renderStruct()`: Render struct definition with fields, add MarshalJSON/UnmarshalJSON for unions
- `renderField()`: Render individual field with JSON tag and pointer notation
- `renderUnionMarshal()`: Generate MarshalJSON for union - check which variant is non-nil, marshal that variant
- `renderUnionUnmarshal()`: Generate UnmarshalJSON for union - read discriminator, unmarshal into correct variant

**Context for implementation:**
- Follow template pattern from internal/generator.go:10-56
- Use strings.Builder for complex rendering like internal/generator.go:88-135
- Reference formatComment pattern from internal/generator.go:142-164
- Copy OpenAPI descriptions to Go doc comments

#### 2. Marshal/Unmarshal Code Generation
**File**: `internal/gogen.go`
**Changes**: Generate custom JSON marshaling for unions

**renderUnionMarshal pattern:**
```go
func (u *Pet) MarshalJSON() ([]byte, error) {
    if u.Dog != nil {
        return json.Marshal(u.Dog)
    }
    if u.Cat != nil {
        return json.Marshal(u.Cat)
    }
    return nil, fmt.Errorf("Pet: no variant set")
}
```

**renderUnionUnmarshal pattern (with case-insensitive lookup):**
```go
func (u *Pet) UnmarshalJSON(data []byte) error {
    var discriminator struct {
        PetType string ` + "`json:\"petType\"`" + `
    }
    if err := json.Unmarshal(data, &discriminator); err != nil {
        return err
    }

    switch strings.ToLower(discriminator.PetType) { // Case-insensitive
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
```

**Function Responsibilities:**
- Generate MarshalJSON checking each variant pointer, marshal non-nil one
- Generate UnmarshalJSON reading discriminator (case-insensitive), allocate/unmarshal variant
- Use DiscriminatorMap for value-to-type lookups
- Handle error cases (no variant set, unknown discriminator value)

**Context for implementation:**
- Use GoStruct.Discriminator for discriminator property name
- Use GoStruct.DiscriminatorMap for case-insensitive value-to-type mapping
- Add strings import for strings.ToLower
- Follow Go conventions for error messages

#### 3. Proto Message Filtering
**File**: `convert.go`
**Changes**: Add function to filter proto messages

```go
func filterProtoMessages(messages []*builder.ProtoMessage, protoTypes map[string]bool) []*builder.ProtoMessage {
    filtered := make([]*builder.ProtoMessage, 0, len(protoTypes))

    for _, msg := range messages {
        // Only include messages that are in protoTypes set
        if protoTypes[msg.Name] {
            filtered = append(filtered, msg)
        }
    }

    return filtered
}

func buildTypeMap(goTypes, protoTypes map[string]bool, reasons map[string]string) map[string]*TypeInfo {
    typeMap := make(map[string]*TypeInfo)

    // Add Go types
    for name := range goTypes {
        typeMap[name] = &TypeInfo{
            Location: TypeLocationGolang,
            Reason:   reasons[name],
        }
    }

    // Add Proto types
    for name := range protoTypes {
        typeMap[name] = &TypeInfo{
            Location: TypeLocationProto,
            Reason:   "", // Proto-only types have no reason
        }
    }

    return typeMap
}
```

**Function Responsibilities:**
- `filterProtoMessages()`: Remove messages marked as Go-only from proto output
- `buildTypeMap()`: Construct TypeInfo map from classification results
- Only top-level messages are filtered (nested messages stay with parent)

**Example:**
```
All messages: [Address, Pet, Dog, Cat, Owner]
protoTypes: {Address}
goTypes: {Pet, Dog, Cat, Owner}
→ filtered messages: [Address]
```

#### 4. Update Convert to Generate Go Code
**File**: `convert.go`
**Changes**: Generate Go output for Go-only types

**Update Convert function:**
```go
func Convert(openapi []byte, opts ConvertOptions) (*ConvertResult, error) {
    // ... existing parsing (from Phase 1) ...

    // Build messages and dependency graph
    messages, graph, err := builder.BuildMessages(entries, ctx)
    if err != nil {
        return nil, err
    }

    // Compute transitive closure
    goTypes, protoTypes, reasons := graph.ComputeTransitiveClosure()

    // Build TypeMap
    typeMap := buildTypeMap(goTypes, protoTypes, reasons)

    // Generate proto for proto-only types
    protoMessages := filterProtoMessages(messages, protoTypes)
    var protoBytes []byte
    if len(protoMessages) > 0 {
        protoBytes = generator.Generate(protoMessages, ctx)
    }

    // Generate Go for Go-only types
    var goBytes []byte
    if len(goTypes) > 0 {
        goCtx := golang.NewGoContext(extractPackageName(opts.GoPackagePath))
        err := golang.BuildGoStructs(entries, goTypes, graph, goCtx)
        if err != nil {
            return nil, err
        }
        goBytes, err = gogen.GenerateGo(goCtx)
        if err != nil {
            return nil, err
        }
    }

    return &ConvertResult{
        Protobuf: protoBytes,
        Golang:   goBytes,
        TypeMap:  typeMap,
    }, nil
}
```

**Key Points:**
- Filter messages before proto generation
- Handle empty proto output (all types are Go-only)
- Handle empty Go output (no unions, all types are proto)
- Both outputs can be populated simultaneously if some types are proto and others are Go

### Validation Commands:
- `go test ./...` - All tests pass (add generation tests)
- Verify generated Go code compiles

## Phase 7: Functional Testing

### Overview
Create comprehensive end-to-end functional tests validating the entire conversion pipeline with realistic OpenAPI specs containing unions. All tests use the public Convert() API per project guidelines.

### Changes Required:

#### 1. End-to-End Union Tests
**File**: `convert_test.go`
**Changes**: Add comprehensive union test cases using Convert()

**Test Cases:**
```go
func TestOneOfBasicGeneration(t *testing.T)
func TestOneOfWithDiscriminatorMapping(t *testing.T)
func TestOneOfTransitiveClosure(t *testing.T)
func TestOneOfVariantsMarkedAsGolang(t *testing.T)
func TestOneOfJSONRoundTrip(t *testing.T)
func TestOneOfCaseInsensitiveDiscriminator(t *testing.T)
func TestOneOfMultipleUnionFields(t *testing.T)
```

**Test Objectives:**
- Verify oneOf schemas generate correct Go structs via Convert()
- Verify union wrapper has pointer fields to variants
- Verify custom MarshalJSON/UnmarshalJSON are generated
- Verify discriminator case-insensitive matching ("dog" matches "Dog")
- Verify explicit discriminator.mapping is used when present
- Verify union variant types (Dog, Cat) marked as "golang" in TypeMap with correct reason
- Verify types referencing union types are marked as Go-only
- Verify JSON round-trip: marshal Go struct → OpenAPI JSON → unmarshal Go struct
- Verify generated Go code compiles
- Verify TypeMap accurately reflects type locations and reasons
- Verify multiple oneOf fields in same schema work

**Context for implementation:**
- Follow functional test pattern from existing convert_test.go
- Use realistic OpenAPI specs from docs/discriminated-unions.md examples
- Use `go/parser` to validate generated Go syntax
- Compile generated code in temporary directory with `go build`
- Use `json.Marshal`/`json.Unmarshal` to test custom marshaling
- Test case sensitivity: discriminator value "dog" should match schema "Dog"

#### 2. Generated Code Compilation Test
**File**: `convert_test.go`
**Changes**: Add concrete test to verify generated Go code compiles

```go
func TestGeneratedGoCodeCompiles(t *testing.T) {
    openapi := []byte(`
openapi: 3.0.0
info:
  title: Test
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`)

    result, err := conv.Convert(openapi, conv.ConvertOptions{
        PackageName:   "testpkg",
        PackagePath:   "github.com/example/proto",
        GoPackagePath: "github.com/example/types",
    })
    require.NoError(t, err)
    require.NotNil(t, result)
    require.NotEmpty(t, result.Golang)

    // Create temporary directory
    tmpDir := t.TempDir()

    // Write generated Go code
    goFile := filepath.Join(tmpDir, "types.go")
    err = os.WriteFile(goFile, result.Golang, 0644)
    require.NoError(t, err)

    // Create go.mod
    modContent := `module test
go 1.21
`
    modFile := filepath.Join(tmpDir, "go.mod")
    err = os.WriteFile(modFile, []byte(modContent), 0644)
    require.NoError(t, err)

    // Compile using go build
    cmd := exec.Command("go", "build", "./...")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "compilation failed:\n%s\nGenerated code:\n%s",
        string(output), string(result.Golang))
}
```

#### 3. JSON Round-Trip Test
**File**: `convert_test.go`
**Changes**: Add test for JSON marshaling/unmarshaling

```go
func TestOneOfJSONRoundTrip(t *testing.T) {
    openapi := []byte(` ... same as above ... `)

    result, err := conv.Convert(openapi, conv.ConvertOptions{
        PackageName:   "testpkg",
        PackagePath:   "github.com/example/proto",
        GoPackagePath: "github.com/example/types",
    })
    require.NoError(t, err)

    // Write and compile generated code
    tmpDir := t.TempDir()
    goFile := filepath.Join(tmpDir, "types.go")
    os.WriteFile(goFile, result.Golang, 0644)

    // Write test program that uses the generated types
    testProg := `package main

import (
    "encoding/json"
    "fmt"
    "os"
)

func main() {
    // Test Dog
    dogJSON := []byte(` + "`" + `{"petType":"dog","bark":"woof"}` + "`" + `)
    var pet Pet
    if err := json.Unmarshal(dogJSON, &pet); err != nil {
        fmt.Fprintf(os.Stderr, "unmarshal error: %v\n", err)
        os.Exit(1)
    }
    if pet.Dog == nil {
        fmt.Fprintf(os.Stderr, "expected Dog to be set\n")
        os.Exit(1)
    }
    if pet.Dog.Bark != "woof" {
        fmt.Fprintf(os.Stderr, "expected bark=woof, got %s\n", pet.Dog.Bark)
        os.Exit(1)
    }

    // Marshal back
    marshaled, err := json.Marshal(&pet)
    if err != nil {
        fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
        os.Exit(1)
    }

    // Verify JSON matches
    var original, remarshaled map[string]interface{}
    json.Unmarshal(dogJSON, &original)
    json.Unmarshal(marshaled, &remarshaled)

    if original["petType"] != remarshaled["petType"] {
        fmt.Fprintf(os.Stderr, "petType mismatch\n")
        os.Exit(1)
    }
    if original["bark"] != remarshaled["bark"] {
        fmt.Fprintf(os.Stderr, "bark mismatch\n")
        os.Exit(1)
    }

    fmt.Println("OK")
}
`

    testFile := filepath.Join(tmpDir, "main.go")
    os.WriteFile(testFile, []byte(testProg), 0644)

    // Create go.mod
    modFile := filepath.Join(tmpDir, "go.mod")
    os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)

    // Build and run test program
    cmd := exec.Command("go", "run", ".")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "test program failed:\n%s", string(output))
    assert.Contains(t, string(output), "OK")
}
```

**Test Strategy:**
1. **Syntax validation**: Use `go build` to verify generated code compiles
2. **Runtime validation**: Create test program that uses generated types
3. **JSON round-trip**: Unmarshal JSON → Marshal back → Compare
4. **Discriminator validation**: Verify correct variant is selected based on discriminator value
5. **Case insensitivity**: Test that "dog" (lowercase) matches Dog schema

**Why this approach:**
- Tests generated code in isolation (temporary directory)
- Verifies actual Go compilation, not just syntax checking
- Tests JSON marshaling/unmarshaling behavior at runtime
- Catches field name conflicts, import issues, type errors

### Validation Commands:
- `go test ./...` - All tests pass including JSON round-trip tests
- `go test -race ./...` - No race conditions
- `make build` - Build succeeds

## Phase 8: Documentation

### Overview
Update documentation to explain union support, type map usage, and API changes.

### Changes Required:

#### 1. Update README ✓
**File**: `README.md`
**Changes**: Add union support section

**Documentation Requirements:**
- Explain that oneOf schemas with discriminators generate Go structs
- Show example of ConvertResult usage
- Explain TypeMap and how to use it to determine type locations
- Show example of generated Go code (union wrapper + variants + marshaling)
- Link to discriminated-unions.md for design rationale
- Note that GoPackagePath defaults to PackagePath

#### 2. Update Discriminated Unions Doc ✓
**File**: `docs/discriminated-unions.md`
**Changes**: Update to reflect new support

**Documentation Requirements:**
- Update "Overview" section to indicate oneOf with discriminator is now supported via Go output
- Add section explaining the oapi-codegen pattern approach
- Show example of generated Go code for oneOf (union wrapper + variants + MarshalJSON/UnmarshalJSON)
- Explain transitive closure behavior (variants and referencing types become Go)
- Show JSON format and how marshaling maintains compatibility
- Explain discriminator requirement
- Keep existing explanation of why proto oneOf doesn't work
- Add "What's Supported" section listing oneOf with discriminator

#### 3. API Migration Guide ✓
**File**: `docs/api-migration.md` (new file)
**Changes**: Document API changes

**Documentation Requirements:**
- Explain change from `[]byte` to `*ConvertResult` return type
- Show before/after code examples:
  ```go
  // Before
  protoBytes, err := conv.Convert(openapi, opts)

  // After
  result, err := conv.Convert(openapi, opts)
  protoBytes := result.Protobuf
  goBytes := result.Golang
  typeMap := result.TypeMap
  ```
- Explain ConvertOptions.GoPackagePath field (optional, defaults to PackagePath)
- Explain TypeMap structure and usage
- Provide guidance on when proto vs Go output will be populated
- Note this is a breaking change (no backwards compatibility)

## Future Enhancements

The following features are explicitly out of scope for Phase 1 but could be added in future phases:

### Type System Enhancements
- **Custom validation types**: Support openapi_types.Email, openapi_types.UUID with validation
- **Additional formats**: Support for duration, hostname, ipv4, ipv6, uri, etc.
- **Nullable types**: Better handling of nullable vs optional fields (currently all use pointers)
- **Default values**: Generate code to set default values from OpenAPI schema

### Union Support Extensions
- **anyOf support**: Non-discriminated unions with multiple possible types
- **allOf support**: Type composition and inheritance
- **oneOf without discriminator**: Support non-discriminated exclusive unions
- **Inline oneOf variants**: Support inline schemas in oneOf (currently only $ref)
- **Nested unions**: Support unions within union types

### Code Generation Improvements
- **Go package name extraction**: Smarter extraction from GoPackagePath (handle version paths)
- **Field name customization**: Intelligent pluralization, custom naming strategies
- **Documentation generation**: Extract more OpenAPI metadata into Go doc comments
- **Validation generation**: Generate validation code from OpenAPI constraints (min, max, pattern, etc.)
- **Example generation**: Generate example values from OpenAPI examples field

### Testing & Tooling
- **Performance optimization**: Lazy parsing for large schemas
- **Better error messages**: Include schema paths in error messages for debugging
- **Circular dependency handling**: Detect and handle circular schema references
- **File writing utilities**: Helper functions to write generated code to filesystem
- **Multiple file output**: Split large generated code into multiple files

### Compatibility
- **Backwards compatibility mode**: Option to generate code compatible with previous versions
- **Proto3 optional support**: Better integration with proto3 optional fields
- **Cross-language references**: Allow proto types to reference Go types (complex)
