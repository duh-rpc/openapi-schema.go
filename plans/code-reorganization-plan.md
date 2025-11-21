# Code Reorganization Implementation Plan

## Overview

This plan reorganizes the openapi-schema.go codebase from a flat internal package structure into capability-based packages. The reorganization improves code maintainability by grouping related functionality together while keeping the public API unchanged.

## Current State Analysis

**Current Package Structure:**
- Root: `convert.go` (public API with 4 main functions)
- `internal/` - All implementation code in single package (35+ files)
- `internal/parser/` - OpenAPI document parsing (already separated)

**Main Capabilities:**
1. **Proto Generation** - Converts OpenAPI to Protocol Buffer 3 format
2. **Go Struct Generation** - Generates Go structs (especially for union types)
3. **Example Generation** - Creates JSON examples from OpenAPI schemas
4. **Example Validation** - Validates examples against OpenAPI schemas

**Key Files in internal/:**
- `builder.go` - Builds proto messages (BuildMessages, buildMessage, buildEnum)
- `generator.go` - Generates proto3 source code
- `mapper.go` - Maps OpenAPI types to proto types
- `golang.go` - Builds Go structs (BuildGoStructs, buildGoStruct)
- `gogen.go` - Generates Go source code
- `examplegenerator.go` - Generates JSON examples
- `examplevalidator.go` - Validates examples
- `dependencies.go` - Dependency graph for transitive closure
- `naming.go` - Naming utilities (PascalCase, snake_case, sanitization)
- `errors.go` - Error helper functions
- 25+ test files

## Desired End State

**New Package Structure:**
```
.
├── convert.go                           # Public API (unchanged)
├── convert_test.go                      # Public API tests (unchanged)
├── convert_examples_test.go             # Public API tests (unchanged)
├── convert_examples_heuristics_test.go  # Public API tests (unchanged)
├── convert_examples_phase3_test.go      # Public API tests (unchanged)
├── convert_validate_test.go             # Public API tests (unchanged)
└── internal/
    ├── naming.go                        # Shared: naming utilities
    ├── naming_test.go                   # Shared: naming tests
    ├── errors.go                        # Shared: error helpers
    ├── errors_test.go                   # Shared: error tests
    ├── dependencies.go                  # Shared: dependency graph
    ├── dependencies_test.go             # Shared: dependency tests
    ├── utils.go                         # Shared: common utilities (NEW)
    ├── parser/
    │   └── parser.go                    # OpenAPI parsing (unchanged)
    ├── proto/
    │   ├── builder.go                   # Proto message building
    │   ├── generator.go                 # Proto3 code generation
    │   ├── mapper.go                    # OpenAPI → Proto type mapping
    │   ├── builder_test.go
    │   ├── generator_test.go
    │   ├── arrays_test.go
    │   ├── comments_test.go
    │   ├── conflicts_test.go
    │   ├── enums_test.go
    │   ├── field_numbers_test.go
    │   ├── field_scoping_test.go
    │   ├── format_enum_test.go
    │   ├── integration_test.go
    │   ├── nested_test.go
    │   ├── nullable_test.go
    │   ├── refs_test.go
    │   ├── required_test.go
    │   ├── scalars_test.go
    │   └── version_test.go
    ├── golang/
    │   ├── golang.go                    # Go struct building
    │   ├── gogen.go                     # Go code generation
    │   ├── golang_scalars_test.go
    │   ├── golang_structures_test.go
    │   ├── unions_test.go
    │   └── unions_runtime_test.go
    ├── example/
    │   ├── generator.go                 # Example generation (renamed from examplegenerator.go)
    │   └── integration_test.go          # Example integration tests (renamed)
    └── validate/
        └── validator.go                 # Example validation (renamed from examplevalidator.go)
```

**Import Changes:**
- `convert.go` will import: `internal/proto`, `internal/golang`, `internal/example`, `internal/validate`, `internal/parser`
- Each subpackage will import: `internal` (for shared utilities), `internal/parser`
- Cross-package dependencies: `internal/golang` imports `internal` (for DependencyGraph)

**Verification:**
- All tests pass: `go test ./...`
- Public API unchanged: existing client code continues to work
- No circular dependencies between new packages

## What We're NOT Doing

- NOT changing the public API in convert.go
- NOT changing test behavior or coverage
- NOT refactoring internal logic (only moving files)
- NOT changing function signatures (except capitalizing shared utilities)
- NOT reorganizing the documentation files

## Implementation Approach

This is a pure refactoring task - we're moving code without changing behavior. The approach is to:
1. Create new package structure
2. Move files and update package declarations
3. Extract shared utilities
4. Update imports throughout codebase
5. Verify all tests pass after each phase

**Rollback Strategy:**
- Create a git commit after each phase completes successfully
- If phase fails, use `git reset --hard` to rollback to previous phase
- Recommended: work in feature branch with commits per phase (`phase-1-utils`, `phase-2-proto`, etc.)

## Phase 1: Create Shared Utilities

### Overview
Extract commonly-used functions from `builder.go` and `mapper.go` into a new `internal/utils.go` file to avoid duplication across new packages.

**Why this is needed:** Currently, functions like `contains`, `isEnumSchema`, and `extractReferenceName` are accessible throughout `internal/` because all files share `package internal`. After reorganization, files will be in separate packages (`internal/proto`, `internal/golang`, `internal/example`). Without extraction, each package would need duplicate implementations of these utilities. By extracting them to `internal/utils.go` with exported names (Capital first letter), all subpackages can import and use them.

### Changes Required:

#### 1. Create `internal/utils.go`
**File**: `internal/utils.go`
**Changes**: Create new file with shared utility functions

```go
package internal

import (
    "fmt"
    "strings"
    "github.com/pb33f/libopenapi/datamodel/high/base"
)

// Contains checks if a slice contains a string (case-insensitive)
func Contains(slice []string, item string) bool

// ExtractReferenceName extracts the schema name from a reference string
// Example: "#/components/schemas/Address" → "Address"
func ExtractReferenceName(ref string) (string, error)

// IsEnumSchema returns true if schema defines an enum
func IsEnumSchema(schema *base.Schema) bool
```

**Function Responsibilities:**
- `Contains`: Case-insensitive string slice membership check
- `ExtractReferenceName`: Parse and validate OpenAPI schema references
- `IsEnumSchema`: Determine if a schema is an enum type
- Copy implementations from `builder.go` (contains, isEnumSchema) and `mapper.go` (extractReferenceName)
- Change to exported names (Capital first letter)

**Context for implementation:**
- These functions are currently used by proto, golang, and example packages
- Must be exported (capitalized) since they'll be called from subpackages
- Source: `contains` and `isEnumSchema` from `internal/builder.go:254-267`, `extractReferenceName` from `internal/mapper.go:239-256`

### Validation
- [ ] Run: `go build ./internal`
- [ ] Verify: No compilation errors
- [ ] Run: `go test ./internal`
- [ ] Verify: Utils tests pass

## Phase 2: Create Proto Package

### Overview
Move proto3 generation code into `internal/proto/` package, including message building, type mapping, and code generation.

### Changes Required:

#### 1. Create Package Directory
**File**: `internal/proto/` (directory)
**Changes**: Create new directory

#### 2. Move and Update `builder.go`
**File**: `internal/builder.go` → `internal/proto/builder.go`
**Changes**:
- Change package declaration from `package internal` to `package proto`
- Update import to add: `"github.com/duh-rpc/openapi-schema.go/internal"`
- Replace `contains` with `internal.Contains`
- Replace `isEnumSchema` with `internal.IsEnumSchema`
- Remove local definitions of `contains` and `isEnumSchema` functions
- Keep `isStringEnum`, `isIntegerEnum`, `extractEnumValues` as proto-specific helpers (used only within proto package)
- Note: `NameTracker` is accessed via `internal.NameTracker` (defined in naming.go)

**Function Signatures:**
```go
// BuildMessages processes all schemas and returns messages and dependency graph
func BuildMessages(entries []*parser.SchemaEntry, ctx *Context) (*internal.DependencyGraph, error)

// buildMessage creates a ProtoMessage from an OpenAPI schema
func buildMessage(name string, proxy *base.SchemaProxy, ctx *Context, graph *internal.DependencyGraph) (*ProtoMessage, error)

// buildEnum creates a ProtoEnum from an OpenAPI schema
func buildEnum(name string, proxy *base.SchemaProxy, ctx *Context) (*ProtoEnum, error)

// buildNestedMessage creates nested message from inline object property
func buildNestedMessage(propertyName string, proxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (*ProtoMessage, error)
```

**Type Definitions** (keep in this file):
- `Context` - Proto generation state
- `ProtoMessage` - Proto message definition
- `ProtoField` - Proto field definition
- `ProtoEnum` - Proto enum definition
- `ProtoEnumValue` - Proto enum value definition

**Context for implementation:**
- Uses `internal.DependencyGraph` (imported from parent)
- Uses `internal.Contains` and `internal.IsEnumSchema` from utils
- Uses `mapper.go` functions (ProtoType) - will be in same package after move
- Pattern reference: Keep transitive closure logic at `builder.go:66-89` for union detection

#### 3. Move and Update `generator.go`
**File**: `internal/generator.go` → `internal/proto/generator.go`
**Changes**:
- Change package declaration to `package proto`
- No other changes needed (self-contained)

**Function Signatures:**
```go
// Generate creates proto3 output from messages and enums
func Generate(packageName string, packagePath string, ctx *Context) ([]byte, error)
```

**Context for implementation:**
- Uses template-based generation
- Handles enum and message rendering
- Comment formatting for proto3 syntax

#### 4. Move and Update `mapper.go`
**File**: `internal/mapper.go` → `internal/proto/mapper.go`
**Changes**:
- Change package declaration to `package proto`
- Add import: `"github.com/duh-rpc/openapi-schema.go/internal"`
- Replace `extractReferenceName` with `internal.ExtractReferenceName`
- Replace `contains` with `internal.Contains`
- Replace `isEnumSchema` with `internal.IsEnumSchema`
- Remove local `extractReferenceName` function definition
- `isStringEnum` will be accessible from builder.go (both in same proto package)

**Function Signatures:**
```go
// ProtoType returns the proto3 type for an OpenAPI schema
func ProtoType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, bool, []string, error)

// MapScalarType maps OpenAPI type+format to proto3 scalar type
func MapScalarType(ctx *Context, typ, format string) (string, error)

// ResolveArrayItemType determines the proto3 type for array items
func ResolveArrayItemType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, []string, error)
```

**Context for implementation:**
- Calls `buildNestedMessage` and `buildEnum` from builder.go (same package)
- Uses type mapping table at `mapper.go:110-141`
- Handles inline objects, enums, and references

#### 5. Move Proto Test Files
**Files**: Move the following from `internal/` to `internal/proto/`:
- `builder_test.go` → `internal/proto/builder_test.go`
- `generator_test.go` → `internal/proto/generator_test.go`
- `arrays_test.go` → `internal/proto/arrays_test.go`
- `comments_test.go` → `internal/proto/comments_test.go`
- `conflicts_test.go` → `internal/proto/conflicts_test.go`
- `enums_test.go` → `internal/proto/enums_test.go`
- `field_numbers_test.go` → `internal/proto/field_numbers_test.go`
- `field_scoping_test.go` → `internal/proto/field_scoping_test.go`
- `format_enum_test.go` → `internal/proto/format_enum_test.go`
- `integration_test.go` → `internal/proto/integration_test.go`
- `nested_test.go` → `internal/proto/nested_test.go`
- `nullable_test.go` → `internal/proto/nullable_test.go`
- `refs_test.go` → `internal/proto/refs_test.go`
- `required_test.go` → `internal/proto/required_test.go`
- `scalars_test.go` → `internal/proto/scalars_test.go`
- `version_test.go` → `internal/proto/version_test.go`

**Changes**: For each test file:
- Update package declaration:
  - If currently `package internal` → change to `package proto`
  - If currently `package internal_test` → change to `package proto_test`
- Update imports:
  - For `package proto` tests: Keep same-package access, no proto import needed
  - For `package proto_test` tests: Add `"github.com/duh-rpc/openapi-schema.go/internal/proto"`
  - All tests need: `"github.com/duh-rpc/openapi-schema.go/internal/parser"` (if using parser types)
- Update function calls to use `proto.` prefix only in `package proto_test` files

**Test Objectives:**
- Verify proto message building works correctly
- Verify type mapping (scalars, arrays, enums, objects)
- Verify field number assignment
- Verify reference resolution
- Verify edge cases (nested objects, nullable types, version compatibility)

**Context for implementation:**
- Follow existing test patterns from `internal/integration_test.go`
- Tests use table-driven approach with OpenAPI YAML fixtures
- Use `require.NoError` for critical checks, `assert.Equal` for value comparisons per CLAUDE.md

### Validation
- [ ] Run: `go build ./internal/proto`
- [ ] Verify: No compilation errors
- [ ] Run: `go test ./internal/proto`
- [ ] Verify: All proto tests pass

## Phase 3: Create Golang Package

### Overview
Move Go struct generation code into `internal/golang/` package, including struct building and code generation for union types.

### Changes Required:

#### 1. Create Package Directory
**File**: `internal/golang/` (directory)
**Changes**: Create new directory

#### 2. Move and Update `golang.go`
**File**: `internal/golang.go` → `internal/golang/golang.go`
**Changes**:
- Change package declaration to `package golang`
- Add imports:
  - `"github.com/duh-rpc/openapi-schema.go/internal"`
- Replace `extractReferenceName` with `internal.ExtractReferenceName`
- Replace `contains` with `internal.Contains`
- Note: `NameTracker` is accessed via `internal.NameTracker` (defined in naming.go)

**Function Signatures:**
```go
// NewGoContext initializes empty context with package name
func NewGoContext(packageName string) *GoContext

// BuildGoStructs processes schemas marked as Go-only, build GoStruct for each
func BuildGoStructs(entries []*parser.SchemaEntry, goTypes map[string]bool, graph *internal.DependencyGraph, ctx *GoContext) error

// buildGoStruct builds Go struct - if oneOf present, create union wrapper; otherwise regular struct
func buildGoStruct(name string, proxy *base.SchemaProxy, graph *internal.DependencyGraph, ctx *GoContext) (*GoStruct, error)

// ExtractPackageName extracts package name from full Go package path
func ExtractPackageName(packagePath string) string
```

**Type Definitions** (keep in this file):
- `GoStruct` - Go struct definition with union metadata
- `GoField` - Go struct field definition
- `GoContext` - Go generation state

**Context for implementation:**
- Uses `internal.DependencyGraph` for discriminator validation
- Uses discriminator map building at `golang.go:146-232`
- Go type mapping at `golang.go:296-352`

#### 3. Move and Update `gogen.go`
**File**: `internal/gogen.go` → `internal/golang/gogen.go`
**Changes**:
- Change package declaration to `package golang`
- No other changes needed (self-contained)

**Function Signatures:**
```go
// GenerateGo produces Go source code from GoStruct IR with custom JSON marshaling
func GenerateGo(ctx *GoContext) ([]byte, error)
```

**Context for implementation:**
- Template-based Go code generation
- Renders structs with custom MarshalJSON/UnmarshalJSON for unions
- Pattern at `gogen.go:56-202`

#### 4. Move Golang Test Files
**Files**: Move the following from `internal/` to `internal/golang/`:
- `golang_scalars_test.go` → `internal/golang/golang_scalars_test.go`
- `golang_structures_test.go` → `internal/golang/golang_structures_test.go`
- `unions_test.go` → `internal/golang/unions_test.go`
- `unions_runtime_test.go` → `internal/golang/unions_runtime_test.go`

**Changes**: For each test file:
- Update package declaration
- Update imports to reference `internal/golang`
- Update function calls to use `golang.` prefix where needed

**Test Objectives:**
- Verify Go scalar type mapping
- Verify Go struct generation
- Verify union type handling (discriminators, marshaling)
- Verify runtime union behavior (JSON round-trip)

**Context for implementation:**
- Tests use ConvertToStruct() from public API
- Union tests verify custom marshaling logic
- Follow patterns from `internal/unions_runtime_test.go:1-370`

### Validation
- [ ] Run: `go build ./internal/golang`
- [ ] Verify: No compilation errors
- [ ] Run: `go test ./internal/golang`
- [ ] Verify: All golang tests pass

## Phase 4: Create Example Package

### Overview
Move example generation code into `internal/example/` package.

### Changes Required:

#### 1. Create Package Directory
**File**: `internal/example/` (directory)
**Changes**: Create new directory

#### 2. Move and Update Example Generator
**File**: `internal/examplegenerator.go` → `internal/example/generator.go`
**Changes**:
- Rename file for consistency (examplegenerator.go → generator.go)
- Change package declaration to `package example`
- Add import: `"github.com/duh-rpc/openapi-schema.go/internal"`
- Replace `extractReferenceName` with `internal.ExtractReferenceName`
- Replace `contains` with `internal.Contains`
- Replace `isEnumSchema` with `internal.IsEnumSchema`

**Function Signatures:**
```go
// GenerateExamples generates JSON examples for specified schemas
func GenerateExamples(entries []*parser.SchemaEntry, schemaNames []string, maxDepth int, seed int64, fieldOverrides map[string]interface{}) (map[string]json.RawMessage, error)
```

**Type Definitions** (keep in this file):
- `ExampleContext` - Example generation state

**Context for implementation:**
- Uses random number generation with seed for deterministic output
- Honors schema constraints (min/max, minLength/maxLength, enum values)
- Field heuristics at `examplegenerator.go:236-269` for cursor/message fields
- Format handling at `examplegenerator.go:273-318`

#### 3. Move Example Test Files
**Files**: Move from `internal/` to `internal/example/`:
- `integration_examples_test.go` → `internal/example/integration_test.go`

**Changes**:
- Rename file for consistency
- Update package declaration
- Update imports to reference `internal/example`

**Test Objectives:**
- Verify example generation for various schema types
- Verify constraint handling (min/max, enums)
- Verify format-specific generation (email, uuid, date)
- Verify field heuristics (cursor, message fields)
- Verify circular reference handling

**Context for implementation:**
- Tests use ConvertToExamples() from public API
- Validate generated JSON structure and values
- Pattern from `internal/integration_examples_test.go`

### Validation
- [ ] Run: `go build ./internal/example`
- [ ] Verify: No compilation errors
- [ ] Run: `go test ./internal/example`
- [ ] Verify: All example tests pass

## Phase 5: Create Validate Package

### Overview
Move example validation code into `internal/validate/` package.

### Changes Required:

#### 1. Create Package Directory
**File**: `internal/validate/` (directory)
**Changes**: Create new directory

#### 2. Move and Update Example Validator
**File**: `internal/examplevalidator.go` → `internal/validate/validator.go`
**Changes**:
- Rename file for consistency (examplevalidator.go → validator.go)
- Change package declaration to `package validate`
- No utility imports needed (self-contained with parser)

**Function Signatures:**
```go
// ValidateExamples validates examples in OpenAPI spec against schemas
func ValidateExamples(openapi []byte, schemaNames []string) (*ExampleValidationResult, error)
```

**Type Definitions** (keep in this file):
- `ExampleValidationResult` - Validation results wrapper
- `SchemaValidation` - Per-schema validation details
- `Issue` - Validation error/warning
- `Severity` - Error severity enum

**Context for implementation:**
- Uses libopenapi-validator for schema validation
- Handles both singular 'example' and plural 'examples' fields
- OpenAPI 3.0 vs 3.1+ version-aware validation at `examplevalidator.go:46-54`
- Validation logic at `examplevalidator.go:142-191`

### Validation
- [ ] Run: `go build ./internal/validate`
- [ ] Verify: No compilation errors
- [ ] Run: `go test ./...`
- [ ] Verify: All tests pass (including root-level convert_validate_test.go)

**Note:** The validate package has no internal tests. It's a thin wrapper around libopenapi-validator and is tested exclusively through the public API via `convert_validate_test.go` at the root level.

## Phase 6: Update Root Package

### Overview
Update `convert.go` to import and use the new package structure. This is the final integration phase.

### Changes Required:

#### 1. Update convert.go Imports
**File**: `convert.go`
**Changes**: Replace single internal import with new packages

**Current imports:**
```go
import (
    "github.com/duh-rpc/openapi-schema.go/internal"
    "github.com/duh-rpc/openapi-schema.go/internal/parser"
)
```

**New imports:**
```go
import (
    "github.com/duh-rpc/openapi-schema.go/internal"
    "github.com/duh-rpc/openapi-schema.go/internal/proto"
    "github.com/duh-rpc/openapi-schema.go/internal/golang"
    "github.com/duh-rpc/openapi-schema.go/internal/example"
    "github.com/duh-rpc/openapi-schema.go/internal/validate"
    "github.com/duh-rpc/openapi-schema.go/internal/parser"
)
```

**Context for implementation:**
- Keep `internal` import for DependencyGraph
- Add subpackage imports for capability-specific functions

#### 2. Update Convert() Function Calls
**File**: `convert.go`
**Function**: `Convert()` at lines 144-221

**Changes**: Update function calls to use new package prefixes

**Before:**
```go
ctx := internal.NewContext()
graph, err := internal.BuildMessages(schemas, ctx)
protoBytes, err = internal.Generate(opts.PackageName, opts.PackagePath, protoCtx)
goCtx := internal.NewGoContext(internal.ExtractPackageName(opts.GoPackagePath))
err := internal.BuildGoStructs(schemas, goTypes, graph, goCtx)
goBytes, err = internal.GenerateGo(goCtx)
```

**After:**
```go
ctx := proto.NewContext()
graph, err := proto.BuildMessages(schemas, ctx)
protoBytes, err = proto.Generate(opts.PackageName, opts.PackagePath, protoCtx)
goCtx := golang.NewGoContext(golang.ExtractPackageName(opts.GoPackagePath))
err := golang.BuildGoStructs(schemas, goTypes, graph, goCtx)
goBytes, err = golang.GenerateGo(goCtx)
```

**Context for implementation:**
- Line 172: `ctx := proto.NewContext()` (first context creation)
- Line 173: `graph, err := proto.BuildMessages(schemas, ctx)`
- Line 190: `protoCtx := proto.NewContext()` (second context for filtered proto types)
- Line 197: `protoBytes, err = proto.Generate(opts.PackageName, opts.PackagePath, protoCtx)`
- Line 205: `goCtx := golang.NewGoContext(golang.ExtractPackageName(opts.GoPackagePath))`
- Line 206: `err := golang.BuildGoStructs(schemas, goTypes, graph, goCtx)`
- Line 211: `goBytes, err = golang.GenerateGo(goCtx)`

#### 3. Update ConvertToStruct() Function Calls
**File**: `convert.go`
**Function**: `ConvertToStruct()` at lines 248-307

**Changes**: Update function calls

**Before:**
```go
ctx := internal.NewContext()
graph, err := internal.BuildMessages(schemas, ctx)
goCtx := internal.NewGoContext(internal.ExtractPackageName(opts.GoPackagePath))
err = internal.BuildGoStructs(schemas, goTypes, graph, goCtx)
goBytes, err := internal.GenerateGo(goCtx)
```

**After:**
```go
ctx := proto.NewContext()
graph, err := proto.BuildMessages(schemas, ctx)
goCtx := golang.NewGoContext(golang.ExtractPackageName(opts.GoPackagePath))
err = golang.BuildGoStructs(schemas, goTypes, graph, goCtx)
goBytes, err := golang.GenerateGo(goCtx)
```

**Context for implementation:**
- Line 273: `proto.NewContext()`
- Line 274: `proto.BuildMessages()`
- Line 289: `golang.ExtractPackageName()`
- Line 289: `golang.NewGoContext()`
- Line 290: `golang.BuildGoStructs()`
- Line 295: `golang.GenerateGo()`

#### 4. Update ConvertToExamples() Function Calls
**File**: `convert.go`
**Function**: `ConvertToExamples()` at lines 383-424

**Changes**: Update function call

**Before:**
```go
examples, err := internal.GenerateExamples(schemas, schemaNames, opts.MaxDepth, opts.Seed, opts.FieldOverrides)
```

**After:**
```go
examples, err := example.GenerateExamples(schemas, schemaNames, opts.MaxDepth, opts.Seed, opts.FieldOverrides)
```

**Context for implementation:**
- Line 416: `example.GenerateExamples()`

#### 5. Update ValidateExamples() Function Calls
**File**: `convert.go`
**Function**: `ValidateExamples()` at lines 426-487

**Changes**: Update function call

**Before:**
```go
internalResult, err := internal.ValidateExamples(openapi, schemaNames)
```

**After:**
```go
internalResult, err := validate.ValidateExamples(openapi, schemaNames)
```

**Context for implementation:**
- Line 457: `validate.ValidateExamples()`

#### 6. Update Helper Functions
**File**: `convert.go`
**Functions**: `filterProtoMessages()` at line 350, `filterProtoDefinitions()` at line 364

**Changes**: Update to use proto package types

**Before:**
```go
func filterProtoMessages(messages []*internal.ProtoMessage, protoTypes map[string]bool) []*internal.ProtoMessage
func filterProtoDefinitions(definitions []interface{}, protoTypes map[string]bool) []interface{}
```

**After:**
```go
func filterProtoMessages(messages []*proto.ProtoMessage, protoTypes map[string]bool) []*proto.ProtoMessage
func filterProtoDefinitions(definitions []interface{}, protoTypes map[string]bool) []interface{}
```

**Context for implementation:**
- Update type references in function signatures
- Update type assertions from `*internal.ProtoMessage` to `*proto.ProtoMessage`
- Pattern at `convert.go:350-381`

### Validation
- [ ] Run: `go build ./...`
- [ ] Verify: No compilation errors
- [ ] Run: `go test ./...`
- [ ] Verify: All tests pass (root and internal)
- [ ] Verify: Public API unchanged (existing client code works)
- [ ] Run: `go test -race ./...` (optional: race detector)
- [ ] Run: `make coverage` (optional: check coverage maintained)
- [ ] Run: `make lint` (optional: ensure code quality)

## Testing Requirements

**After Each Phase:**
- All existing tests must pass
- No new tests required (this is refactoring only)
- Test coverage should remain the same

**Final Validation:**
- Run full test suite: `go test ./...`
- Run with race detector: `go test -race ./...`
- Verify coverage: `make coverage`
- Run linter: `make lint`

**Test Pattern Notes:**
- Root tests (`convert_test.go`, etc.) use `package schema_test` - stay at root
- Internal tests currently use `package internal` - after move, use package name (e.g., `package proto`, `package golang`)
- Note: CLAUDE.md specifies tests should be in `package XXX_test`, but current internal tests use same package to access unexported functions. Maintain current pattern for this refactoring.

## Implementation Notes

**Shared Utilities:**
- `Contains`, `ExtractReferenceName`, `IsEnumSchema` moved to `internal/utils.go`
- Must be exported (Capital) to be used by subpackages
- Proto-specific helpers (`isStringEnum`, `isIntegerEnum`, `extractEnumValues`) stay in proto package

**Dependencies:**
- `internal/dependencies.go` stays in root internal/ (used by both proto and golang)
- `internal/naming.go` stays in root internal/ (used by all packages)
- `internal/errors.go` stays in root internal/ (used by all packages)
- `internal/parser` unchanged (already separated)

**Type Locations:**
- Proto types (`Context`, `ProtoMessage`, etc.) in `internal/proto/builder.go`
- Go types (`GoContext`, `GoStruct`, etc.) in `internal/golang/golang.go`
- Shared types (`DependencyGraph`) in `internal/dependencies.go`

**Import Patterns After Reorganization:**
- Root `convert.go` imports: `internal`, `internal/proto`, `internal/golang`, `internal/example`, `internal/validate`, `internal/parser`
- `internal/proto` imports: `internal`, `internal/parser`
- `internal/golang` imports: `internal`, `internal/parser`
- `internal/example` imports: `internal`, `internal/parser`
- `internal/validate` imports: `internal/parser`

**Circular Dependency Prevention:**
- Proto and golang packages don't import each other
- Both import shared `internal` package
- `DependencyGraph` in `internal/` acts as coordination mechanism
- `convert.go` orchestrates calls between proto and golang packages
