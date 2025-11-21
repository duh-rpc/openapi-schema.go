> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# OpenAPI 3.0 to Protobuf 3 Conversion Library - Implementation Plan

## Overview

This implementation plan defines the step-by-step construction of a Go library that converts OpenAPI 3.0 schema definitions to Protocol Buffer 3 (proto3) format. The library will parse OpenAPI YAML specifications using `github.com/pb33f/libopenapi` and generate corresponding `.proto` files with proper type mappings, JSON field name annotations, and protobuf conventions.

**Module**: `github.com/duh-rpc/openapi-proto`

**Primary API**:
```go
func Convert(openapi []byte, packageName string) ([]byte, error)
```

## Current State Analysis

**What exists now:**
- Empty Go module at `github.com/duh-rpc/openapi-proto`
- `go.mod` with Go 1.24.4
- Comprehensive technical specification (approved, ready for implementation)
- No existing code, tests, or dependencies

**What's missing:**
- All implementation code
- All tests
- Dependencies (libopenapi, testify)
- Build automation (Makefile)
- Example usage
- Documentation (README)

**Key constraints discovered:**
- Tests must be functional/end-to-end style: given OpenAPI YAML → assert exact proto output
- Tests in `package <pkg>_test` format (external tests only - no unit tests of internal functions)
- Naming: Simple letter-by-letter snake_case conversion (no acronym detection)
- Naming: No singularization logic - error if array item property name ends with 's'/'es'
- External reference detection via libopenapi build errors
- Enum values: preserve numbers, convert `-` to `_`
- Proto3 keyword conflicts: let protoc catch them (fail-fast)
- Multi-type properties (`type: [string, null]`): error as unsupported

## Desired End State

A production-ready Go library that:
- Converts OpenAPI 3.0 `components/schemas` to valid proto3 format
- Preserves field ordering from YAML insertion order
- Generates proper `json_name` annotations for camelCase fields
- Hoists inline enums to top-level with `UNSPECIFIED` values
- Handles nested objects, arrays, and `$ref` references
- Returns clear, actionable errors for unsupported features
- Achieves >80% test coverage through functional tests only
- Passes `go test ./...` and `golangci-lint run ./...`
- Includes README with usage examples and naming conventions documentation

**Verification**:
```bash
make test          # all tests pass
make lint          # lint all passes
make coverage      # Generates coverage report showing >80% coverage
```

DO NOT RUN `make ci`, as it requires files to be committed, you MUST NOT commit phases unless explicitly asked.

### Key Discoveries:

**libopenapi API patterns** (from research):
- Parse: `libopenapi.NewDocument(bytes)` → `BuildV3Model()` → `docModel.Model`
- Schemas: `docModel.Model.Components.Schemas.FromOldest()` preserves YAML order
- Iteration: `for name, proxy := range schemas.FromOldest()` pattern
- References: `schemaProxy.IsReference()`, `schemaProxy.GetReference()`, `schemaProxy.Schema()` auto-resolves
- Properties: `schema.Properties.FromOldest()` maintains insertion order
- Build errors: `schemaProxy.GetBuildError()` returns resolution failures (including external refs)
- Type access: `schema.Type` is `[]string`, use first element

**Testing pattern**:
- CLAUDE.md shows correct pattern: `conv.Convert()` with OpenAPI bytes → assert exact proto output
- All tests in `package conv_test` (external)
- Table-driven tests with `given` (OpenAPI YAML) and `expected` (proto3 output)
- No unit tests of internal functions - test only through public API

**Makefile pattern** (from `duh-cli/Makefile`):
- Standard targets: `test`, `lint`, `tidy`, `ci`, `coverage`, `clean`
- CI target runs: `tidy && lint && test`
- For libraries: skip `build` and `install` targets (no binary)

**Design Decisions Made**:
1. **Schema output order**: YAML insertion order (same as input)
2. **Nested message naming**: Error if property ends with 's'/'es' (no singularization)
3. **Snake case algorithm**: Simple letter-by-letter (each uppercase → lowercase + underscore)
4. **Testing strategy**: Functional tests only (no internal unit tests)
5. **Output ordering**: Enums and messages mixed in processing order
6. **Multi-type support**: Error as unsupported feature
7. **Proto3 keywords**: Pass through, let protoc validate (fail-fast)

## What We're NOT Doing

Explicitly out of scope for initial implementation:

**OpenAPI Features:**
- Schema composition: `allOf`, `anyOf`, `oneOf`, `not` (return errors)
- External references to other files (return errors via libopenapi)
- Nested arrays (return errors)
- Validation constraints (ignored silently)
- Singularization logic for property names (error on plural names)
- Multi-type properties: `type: [string, null]` (return errors)
- Polymorphism, discriminators, links, callbacks

**Proto3 Features:**
- Multiple file output (single file only)
- Import statements (no `google.protobuf.*` types)
- Proto options beyond `json_name`
- Service definitions
- Custom field numbering
- Map types (`additionalProperties` not supported)
- Reserved keyword detection/handling (let protoc catch it)

**Build/Tooling:**
- CLI wrapper around library (library only)
- Configuration file support
- Auto-formatting of generated proto (basic template output only)
- Acronym detection in snake_case conversion (simple algorithm only)

## Implementation Approach

**Strategy**: Build incrementally in phases, each adding a layer of functionality while maintaining all tests passing. Follow TDD approach: write tests first, implement to make them pass.

**Testing Philosophy**:
- ONLY functional/end-to-end tests through public `Convert()` API
- NO unit tests of internal functions
- Tests prove the implementation works for real-world OpenAPI specs
- Coverage achieved through comprehensive test scenarios, not unit tests

**Architecture** (from spec lines 192-238):
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
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                  Type Mapper Component                       │
│  - Maps OpenAPI types → proto3 types                        │
│  - Handles nested objects, arrays, refs                     │
│  - Assigns field numbers based on order                     │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                 Naming Converter Component                   │
│  - Converts camelCase → snake_case (simple algorithm)       │
│  - Converts property names → PascalCase (messages)          │
│  - Generates enum value names (UPPERCASE_SNAKE_CASE)        │
│  - Tracks names for conflict detection                      │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                  Proto Generator Component                   │
│  - Uses Go text/template to format output                   │
│  - Generates syntax, package, messages, enums               │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                 Output: Single .proto file (bytes)           │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Project Setup & Core Library Structure

### Overview
Establish the project foundation with dependencies, build automation, public API structure, and basic OpenAPI parsing. This phase creates the scaffolding for all future phases.

### Acceptance Criteria:
- [x] `go.mod` includes all required dependencies
- [x] `Makefile` with standard targets (`test`, `lint`, `tidy`, `ci`, `coverage`, `clean`)
- [x] Public `Convert()` function exists and can parse valid OpenAPI 3.0 YAML
- [x] Basic input validation (empty bytes, empty package name) returns errors
- [x] Can extract schemas from `components/schemas` section in YAML order
- [x] All tests pass with `make test`

### Changes Required:

#### 1. Dependency Setup ✅
**File**: `go.mod`
**Changes**: Add required dependencies

```bash
go get github.com/pb33f/libopenapi@latest
go get github.com/stretchr/testify@latest
go mod tidy
```

**Dependencies**:
- `github.com/pb33f/libopenapi` - OpenAPI 3.0/3.1 parser
- `github.com/stretchr/testify` - Testing assertions (require, assert)

#### 2. Build Automation ✅
**File**: `Makefile`
**Changes**: Create Makefile based on duh-cli pattern

```makefile
.PHONY: test lint tidy coverage ci clean

test:
	go test -v ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy && git diff --exit-code

ci: tidy lint test
	@echo
	@echo "\033[32mEVERYTHING PASSED!\033[0m"

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean:
	rm -f coverage.out coverage.html
	go clean
```

#### 3. Public API Layer ✅
**File**: `convert.go`
**Changes**: Define public API with basic structure

```go
// Convert converts OpenAPI 3.0 schemas to Protocol Buffer 3 format
func Convert(openapi []byte, packageName string) ([]byte, error)
```

**Function Responsibilities**:
- Validate inputs: non-empty openapi bytes, non-empty packageName
- Call parseDocument() to get OpenAPI model
- Call extractSchemas() to get ordered schema list
- Build internal representation (messages, enums)
- Call generate() to produce proto3 output
- Return structured errors with context

**Testing Requirements**:
```go
func TestConvertBasics(t *testing.T)
```

**Test Objectives**:
- Empty openapi bytes → error "openapi input cannot be empty"
- Empty package name → error "package name cannot be empty"
- Invalid YAML → error mentioning YAML syntax
- Valid minimal OpenAPI 3.0 → success with basic output
- OpenAPI 2.0 (Swagger) → error "only OpenAPI 3.x is supported"
- Valid JSON OpenAPI → success (libopenapi handles both YAML and JSON)

#### 4. Parser Component ✅
**File**: `internal/parser/parser.go`
**Changes**: Create parser using libopenapi

```go
// document wraps the libopenapi v3 document model
type document struct {
	model *v3.Document
}

// parseDocument parses OpenAPI bytes and returns the document
func parseDocument(openapi []byte) (*document, error)

// schemas returns schemas from components/schemas in insertion order
func (d *document) schemas() ([]*schemaEntry, error)

// schemaEntry represents a schema with its name and proxy
type schemaEntry struct {
	Name  string
	Proxy *base.SchemaProxy
}
```

**Function Responsibilities**:
- `parseDocument`: Use `libopenapi.NewDocument(openapi)` to parse
- Call `doc.BuildV3Model()` to get OpenAPI 3.x model
- Check for build errors, return wrapped error with context
- Validate document is OpenAPI 3.x (check version)
- Return wrapped document struct
- `schemas`: Access `d.model.Components.Schemas`
- Return error if Components is nil or Schemas is nil
- Iterate using `Schemas.FromOldest()` to preserve YAML order
- For each: create schemaEntry with name and proxy
- Do NOT call `proxy.Schema()` yet - defer until needed
- Return ordered slice of schemaEntry

**Testing Requirements**:
```go
func TestConvertParseDocument(t *testing.T)
func TestConvertExtractSchemas(t *testing.T)
```

**Test Objectives** (all via Convert() API):
- Parse valid OpenAPI 3.0 YAML successfully
- Parse valid OpenAPI 3.0 JSON successfully
- Invalid YAML syntax → error
- Non-OpenAPI document → error
- Extract schemas in YAML insertion order (test with 3+ schemas)
- Document with no `components` section → success (no schemas)
- Document with empty `components/schemas` → success (no schemas)

**Context for Implementation**:
- libopenapi patterns: `NewDocument()` → check error → `BuildV3Model()` → check error
- `FromOldest()` returns iterator: `for name, proxy := range schemas.FromOldest()`
- Don't call `proxy.Schema()` in parser - let type mapper do that
- Reference spec REQ-001 (lines 45-50): Parse OpenAPI 3.0 YAML
- Reference spec REQ-004 (lines 67-72): Use FromOldest() for YAML order

---

## Phase 2: Naming Converter & Simple Type Mapping

### Overview
Implement naming conversion logic and basic scalar type mapping. This establishes the foundation for generating proto3 identifiers and handling simple types.

### Acceptance Criteria:
- [x] camelCase → snake_case conversion works (simple algorithm: each uppercase → lowercase + underscore)
- [x] Property names → PascalCase for message names works
- [x] Enum value naming (UPPERCASE_SNAKE_CASE with prefix) works
- [x] Duplicate name detection and suffix generation works
- [x] `json_name` detection logic works
- [x] OpenAPI scalar types map to proto3 types correctly
- [x] Basic proto3 messages with scalar fields generate successfully

### Changes Required:

#### 1. Naming Converter ✅
**File**: `internal/naming/naming.go`
**Changes**: Implement name conversion utilities

```go
// toSnakeCase converts camelCase/PascalCase to snake_case
// Algorithm: Each uppercase letter becomes lowercase with underscore prefix (except first char)
// Examples: userId → user_id, HTTPStatus → h_t_t_p_status, email → email
func toSnakeCase(s string) string

// toPascalCase converts snake_case/camelCase to PascalCase
// Examples: user_id → UserId, shippingAddress → ShippingAddress
func toPascalCase(s string) string

// toEnumValueName converts a value to ENUM_PREFIX_VALUE_NAME format
// Examples: (Status, in-progress) → STATUS_IN_PROGRESS, (Code, 401) → CODE_401
func toEnumValueName(enumName, value string) string

// needsJSONName returns true if proto field name differs from original
func needsJSONName(original, protoField string) bool

// nameTracker tracks used names and generates unique names when conflicts occur
type nameTracker struct {
	used map[string]int
}

func newNameTracker() *nameTracker

// uniqueName returns a unique name, adding numeric suffix if needed (_2, _3, etc.)
func (nt *nameTracker) uniqueName(name string) string
```

**Function Responsibilities**:
- `toSnakeCase`: Iterate each character, if uppercase: emit underscore (skip if first char) + lowercase
- `toSnakeCase`: Preserve existing underscores, numbers, and other characters as-is
- `toSnakeCase`: No special handling for acronyms (HTTPStatus → h_t_t_p_status)
- `toPascalCase`: Capitalize first letter and letters after underscores, remove underscores
- `toEnumValueName`: Convert value to UPPERCASE, convert '-' to '_', preserve numbers
- `toEnumValueName`: Prefix with enum name + underscore: `<ENUM>_<VALUE>`
- `needsJSONName`: Return `toSnakeCase(original) != original`
- `nameTracker.uniqueName`: First occurrence returns as-is, duplicates get `_2`, `_3`, etc.

**Testing Requirements**:
```go
func TestConvertSnakeCase(t *testing.T)
func TestConvertPascalCase(t *testing.T)
func TestConvertEnumValueName(t *testing.T)
func TestConvertJSONName(t *testing.T)
func TestConvertNameConflicts(t *testing.T)
```

**Test Objectives** (all via Convert() with targeted OpenAPI specs):
- toSnakeCase: `userId` → `user_id`, `HTTPStatus` → `h_t_t_p_status`, `email` → `email`
- toSnakeCase: preserve numbers: `user2Id` → `user2_id`, `http2Protocol` → `http2_protocol`
- toPascalCase: `user_id` → `UserId`, `shippingAddress` → `ShippingAddress`
- toEnumValueName: `(Status, active)` → `STATUS_ACTIVE`
- toEnumValueName: `(Status, in-progress)` → `STATUS_IN_PROGRESS`
- toEnumValueName: preserve numbers: `(Code, 401)` → `CODE_401`
- needsJSONName: `userId` field gets `[json_name = "userId"]`, `email` does not
- nameTracker: two schemas named `User` → `User` and `User_2`

**Context for Implementation**:
- Reference spec REQ-003 (lines 60-65): json_name annotation logic
- Reference spec REQ-014 (lines 150-154): Naming conflict resolution
- Reference spec naming examples (lines 298-304)
- Design decision: Simple letter-by-letter snake_case (no acronym detection)
- **IMPORTANT**: Document this naming behavior in README/godoc

#### 2. Type Mapper ✅
**File**: `internal/mapper/mapper.go`
**Changes**: Implement scalar type mapping

```go
// protoType returns the proto3 type for an OpenAPI schema
// Returns type name and error
func protoType(schema *base.Schema, propertyName string) (string, error)

// mapScalarType maps OpenAPI type+format to proto3 scalar type
func mapScalarType(typ, format string) (string, error)
```

**Function Responsibilities**:
- `protoType`: Get type from `schema.Type` (is `[]string`, use index 0)
- If `len(schema.Type) > 1`: return error "multi-type properties not supported"
- If `len(schema.Type) == 0`: return error "property must have type or $ref"
- Call `mapScalarType()` with type and format
- Return type name or error
- `mapScalarType`: Use spec table for type mapping (lines 240-258)

**Type Mapping Table**:
- `integer` + `int32`/empty → `int32`
- `integer` + `int64` → `int64`
- `number` + `float` → `float`
- `number` + `double`/empty → `double`
- `string` + empty → `string`
- `string` + `byte`/`binary` → `bytes`
- `string` + `date`/`date-time` → `string`
- `boolean` + any → `bool`

**Testing Requirements**:
```go
func TestConvertScalarTypes(t *testing.T)
```

**Test Objectives** (via Convert()):
- All scalar type mappings per table above
- Default integer (no format) → int32
- Default number (no format) → double
- Date/datetime → string (not timestamp)
- Multi-type property → error "multi-type properties not supported"

**Context for Implementation**:
- Reference spec type mapping table (lines 240-258)
- Design decision: Multi-type not supported (error)

#### 3. Message Builder ✅
**File**: `internal/builder/builder.go`
**Changes**: Build proto3 message definitions

```go
// context holds state during conversion
type context struct {
	tracker  *nameTracker
	messages []*protoMessage
	enums    []*protoEnum
}

func newContext() *context

// protoMessage represents a proto3 message definition
type protoMessage struct {
	Name        string
	Description string
	Fields      []*protoField
	Nested      []*protoMessage
}

// protoField represents a proto3 field
type protoField struct {
	Name        string
	Type        string
	Number      int
	JSONName    string // Empty if not needed
	Description string
	Repeated    bool
}

// buildMessages processes all schemas and returns messages
func buildMessages(entries []*schemaEntry, ctx *context) error

// buildMessage creates a protoMessage from an OpenAPI schema
func buildMessage(name string, proxy *base.SchemaProxy, ctx *context) (*protoMessage, error)
```

**Function Responsibilities**:
- `buildMessages`: Iterate schema entries, call buildMessage for each
- `buildMessage`: Call `proxy.Schema()` to resolve schema
- Check `proxy.GetBuildError()` for resolution failures
- Validate schema.Type contains "object" (not primitive or array at top-level)
- Extract description from `schema.Description`
- Iterate `schema.Properties.FromOldest()` to maintain order
- For each property: assign sequential field number (1, 2, 3, ...)
- For each property: convert name via toSnakeCase, check needsJSONName
- For each property: determine type via protoType
- Store in message.Fields
- Add message to ctx.messages
- Return message

**Testing Requirements**:
```go
func TestConvertSimpleMessage(t *testing.T)
func TestConvertFieldOrdering(t *testing.T)
func TestConvertJSONNameAnnotation(t *testing.T)
```

**Test Objectives** (via Convert()):
- Object schema with multiple scalar fields → message with all fields
- Field numbers are sequential (1, 2, 3, ...)
- Field names in snake_case
- json_name present only when needed
- Descriptions extracted and appear as comments
- Top-level primitive schema → error "only objects and enums supported at top level"
- Top-level array schema → error "only objects and enums supported at top level"

**Context for Implementation**:
- Reference spec REQ-002 (lines 52-58): Type mapping
- Reference spec REQ-004 (lines 67-72): Field ordering
- Use `schema.Properties.FromOldest()` for iteration
- `schema.Type` is `[]string`, check if it contains "object"

#### 4. Proto Generator ✅
**File**: `internal/generator/generator.go`
**Changes**: Generate proto3 output

```go
// generate creates proto3 output from messages and enums
func generate(packageName string, messages []*protoMessage, enums []*protoEnum) ([]byte, error)
```

**Function Responsibilities**:
- Use `text/template` to format output
- Generate `syntax = "proto3";` on first line
- Generate blank line
- Generate `package <packageName>;` declaration
- Generate blank line
- For each message/enum: generate definition with blank line separator
- Format message fields with proper indentation (2 spaces)
- Format json_name as `[json_name = "original"]`
- Format comments as `// description` above definitions
- Return bytes

**Template Structure**:
```
syntax = "proto3";

package <packageName>;

// <message description>
message <MessageName> {
  // <field description>
  <type> <field_name> = <number> [json_name = "<original>"];
}
```

**Testing Requirements**:
Already covered by other tests

**Test Objectives**:
- Output starts with `syntax = "proto3";`
- Package declaration correct
- Messages in YAML order
- Fields numbered sequentially
- json_name formatted correctly

**Context for Implementation**:
- Reference spec output example (lines 949-976)
- Blank line spacing: 1 blank line between top-level definitions
- Comments: `//` prefix, preserve line breaks

---

## Phase 3: Enum Support

### Overview
Implement enum type detection and conversion with proto3 conventions: top-level definitions, UNSPECIFIED value at 0, value shifting, and inline enum hoisting.

### Acceptance Criteria:
- [x] Top-level enum schemas converted to proto3 enums
- [x] All enums include `<NAME>_UNSPECIFIED = 0` as first value
- [x] Original enum values shifted to start at 1
- [x] Enum values use UPPERCASE_SNAKE_CASE with enum name prefix
- [x] Inline enums hoisted to top-level
- [x] Enum ordering: UNSPECIFIED, then original values in order
- [x] Enums and messages mixed in processing order in output

### Changes Required:

#### 1. Enum Builder
**File**: `internal/builder/builder.go`
**Changes**: Add enum building to builder

```go
// protoEnum represents a proto3 enum definition
type protoEnum struct {
	Name        string
	Description string
	Values      []*protoEnumValue
}

// protoEnumValue represents an enum value
type protoEnumValue struct {
	Name   string
	Number int
}

// isEnumSchema returns true if schema defines an enum
func isEnumSchema(schema *base.Schema) bool

// buildEnum creates a protoEnum from an OpenAPI schema
func buildEnum(name string, proxy *base.SchemaProxy, ctx *context) (*protoEnum, error)
```

**Function Responsibilities**:
- `isEnumSchema`: Check if `schema.Enum` is non-nil and non-empty
- `buildEnum`: Call `proxy.Schema()` to resolve
- Extract `schema.Description`
- Create first value: `<ENUM_NAME>_UNSPECIFIED = 0`
- Iterate `schema.Enum` (original values)
- For each value (starting at number 1): convert via toEnumValueName
- Store in enum.Values preserving order
- Add enum to ctx.enums
- Return enum

**Testing Requirements**:
```go
func TestConvertTopLevelEnum(t *testing.T)
func TestConvertEnumValueNaming(t *testing.T)
```

**Test Objectives** (via Convert()):
- Top-level enum schema → enum with UNSPECIFIED at 0
- Original values start at 1, preserve order
- Values use UPPERCASE_SNAKE_CASE with prefix
- Enum value with `-` → converted to `_`
- Enum value with numbers → preserved

**Context for Implementation**:
- Reference spec REQ-005 (lines 74-81): Enum conversion
- Reference spec enum examples (lines 312-364)
- `schema.Enum` is `[]any`, convert to string for naming

#### 2. Type Mapper Enhancement
**File**: `internal/mapper/mapper.go`
**Changes**: Handle enum types in protoType

**Function Responsibilities**:
- In `protoType`: Check `isEnumSchema(schema)` before checking scalar types
- If enum: return enum type name (from property name via toPascalCase)
- For inline enums: derive name from property name
- Track inline enum for hoisting (add to ctx.enums via buildEnum)

**Testing Requirements**:
```go
func TestConvertInlineEnum(t *testing.T)
```

**Test Objectives** (via Convert()):
- Inline enum property → enum hoisted to top-level
- Field uses enum type name
- Enum name derived from property name

**Context for Implementation**:
- Reference spec inline enum example (lines 336-364)

---

## Phase 4: Array Support

### Overview
Implement conversion of OpenAPI arrays to proto3 `repeated` fields with proper item type resolution. This phase includes validation for plural property names when inline objects/enums are used.

### Acceptance Criteria:
- [x] Arrays of scalars → `repeated <scalar>` fields
- [x] Arrays of `$ref` → `repeated <MessageType>` fields
- [x] Arrays of inline objects → error if property name ends with 's'/'es'
- [x] Arrays of inline enums → error if property name ends with 's'/'es'
- [x] Nested arrays → error

### Changes Required:

#### 1. Type Mapper Enhancement
**File**: `internal/mapper/mapper.go`
**Changes**: Handle array types

```go
// resolveArrayItemType determines the proto3 type for array items
// Returns type name and error
// For inline objects/enums: validates property name is not plural
func resolveArrayItemType(items *base.Schema, propertyName string, ctx *context) (string, error)
```

**Function Responsibilities**:
- Check if `schema.Type` contains "array"
- Access `schema.Items` (is `*DynamicValue[*SchemaProxy, bool]`)
- If Items is nil: return error "array must have items defined"
- Get items schema via `schema.Items.A` (the SchemaProxy)
- If items.Schema() is nil: return error with context
- If items is scalar: return scalar type via mapScalarType
- If items is `$ref`: extract reference name, return message type
- If items is inline object: **validate property name does NOT end with 's' or 'es'**
- If plural: return error "cannot derive message name from plural array property '<name>'; use singular form or $ref"
- If items is inline enum: **validate property name does NOT end with 's' or 'es'**
- If plural: return error "cannot derive enum name from plural array property '<name>'; use singular form or $ref"
- If items is array: return error "nested arrays not supported"
- Generate nested message/enum from items, return type name

**Testing Requirements**:
```go
func TestConvertArrayOfScalars(t *testing.T)
func TestConvertArrayOfReferences(t *testing.T)
func TestConvertArrayOfInlineObjects(t *testing.T)
func TestConvertArrayOfInlineEnums(t *testing.T)
func TestConvertArrayPluralName(t *testing.T)
```

**Test Objectives** (via Convert()):
- Array of strings → `repeated string field_name`
- Array of $ref → `repeated MessageType field_name`
- Array with inline object, property `contact` → nested message `Contact`, `repeated Contact contact`
- Array with inline object, property `contacts` → error about plural name
- Array with inline enum, property `status` → enum `Status`, `repeated Status status`
- Array with inline enum, property `statuses` → error about plural name
- Nested arrays → error "nested arrays not supported"

**Context for Implementation**:
- Reference spec REQ-012 (lines 127-135): Array handling
- Reference spec array examples (lines 479-551)
- Design decision: No singularization, error on plural property names
- Check for plural: `strings.HasSuffix(name, "s") || strings.HasSuffix(name, "es")`

#### 2. Message Builder Enhancement
**File**: `internal/builder/builder.go`
**Changes**: Handle array fields

**Function Responsibilities**:
- When building field, check if type is array
- Call resolveArrayItemType to get item type
- Set field.Repeated = true
- Use item type as field type

#### 3. Generator Enhancement
**File**: `internal/generator/generator.go`
**Changes**: Format repeated fields

**Function Responsibilities**:
- When field.Repeated is true: prefix type with `repeated `
- Format: `repeated <type> <name> = <number>;`

**Context for Implementation**:
- Reference spec array examples show `repeated` keyword prefix
- Design decision: Error if property name suggests plural (ends with s/es)

---

## Phase 5: Nested Object Support

### Overview
Implement generation of nested message definitions for inline object properties. This phase includes plural name validation similar to arrays.

### Acceptance Criteria:
- [x] Inline object properties generate nested message definitions
- [x] Nested messages named via property name → PascalCase
- [x] Error if property name ends with 's'/'es' (potential plural)
- [x] Deep nesting works correctly
- [x] Nested message field numbering independent (starts at 1)

### Changes Required:

#### 1. Message Builder Enhancement ✅
**File**: `internal/builder/builder.go`
**Changes**: Handle inline objects

```go
// buildNestedMessage creates nested message from inline object property
func buildNestedMessage(propertyName string, proxy *base.SchemaProxy, ctx *context) (*protoMessage, error)
```

**Function Responsibilities**:
- Detect inline object: `schema.Type` contains "object" and NOT a reference
- **Validate property name does NOT end with 's' or 'es'**
- If plural: return error "cannot derive message name from property '<name>'; use singular form or $ref"
- Derive nested message name via toPascalCase(propertyName)
- Get unique name via ctx.tracker.uniqueName()
- Recursively call buildMessage for nested properties
- Add nested message to parent message.Nested collection
- Return nested message

**Testing Requirements**:
```go
func TestConvertNestedObject(t *testing.T)
func TestConvertDeeplyNested(t *testing.T)
func TestConvertNestedPluralName(t *testing.T)
```

**Test Objectives** (via Convert()):
- Inline object property → nested message inside parent
- Nested message name from property name (PascalCase)
- Multiple nesting levels work
- Field numbering independent in each message (nested starts at 1)
- Inline object with property ending in 's' → error about plural name

**Context for Implementation**:
- Reference spec REQ-007 (lines 90-94): Nested objects
- Reference spec nested example (lines 389-421)
- Design decision: Error on plural property names (same as arrays)

#### 2. Generator Enhancement ✅
**File**: `internal/generator/generator.go`
**Changes**: Format nested messages

**Function Responsibilities**:
- Render nested messages indented (2 spaces) within parent
- Nested messages appear before fields that reference them
- Blank line after nested message definition

**Template Enhancement**:
```
message ParentMessage {
  message NestedMessage {
    string field = 1;
  }

  NestedMessage nested_field = 1;
}
```

---

## Phase 6: Reference Resolution

### Overview
Implement `$ref` reference detection and resolution using libopenapi's automatic resolution. External references will fail during schema resolution with build errors.

### Acceptance Criteria:
- [x] Internal references (`#/components/schemas/Type`) use referenced type name
- [x] Multiple fields can reference same schema
- [x] External file references cause build error → wrapped with clear message
- [x] Reference resolution happens automatically via libopenapi

### Changes Required:

#### 1. Type Mapper Enhancement
**File**: `internal/mapper/mapper.go`
**Changes**: Handle `$ref` references

```go
// extractReferenceName extracts the schema name from a reference string
// Example: "#/components/schemas/Address" → "Address"
func extractReferenceName(ref string) (string, error)
```

**Function Responsibilities**:
- In `protoType`: Check `proxy.IsReference()` before processing
- If reference: get reference string via `proxy.GetReference()`
- Call `proxy.Schema()` to auto-resolve (libopenapi handles internal refs)
- Check `proxy.GetBuildError()` for resolution failures
- If build error: wrap with context "property '<prop>' references external file or unresolvable reference"
- Extract schema name from reference via extractReferenceName
- Return schema name as type

**Testing Requirements**:
```go
func TestConvertSchemaReference(t *testing.T)
func TestConvertMultipleReferences(t *testing.T)
func TestConvertExternalReference(t *testing.T)
```

**Test Objectives** (via Convert()):
- $ref to another schema → field uses referenced type name
- Multiple fields with same $ref → all use same type
- Schemas appear in YAML order in output
- External file reference → error indicating external refs not supported

**Context for Implementation**:
- Reference spec REQ-008 (lines 96-101): Reference resolution
- Reference spec examples (lines 423-477)
- libopenapi: `IsReference()`, `GetReference()`, `Schema()`, `GetBuildError()`
- Design decision: Rely on libopenapi build errors for external refs
- Extract name from ref: split by '/' and take last segment

---

## Phase 7: Description Comments

### Overview
Extract OpenAPI `description` fields and generate proto3 comments above corresponding messages and fields.

### Acceptance Criteria:
- [x] Schema descriptions → `//` comments above messages
- [x] Property descriptions → `//` comments above fields
- [x] Multi-line descriptions have each line prefixed with `//`
- [x] Empty descriptions result in no comment
- [x] Blank lines in descriptions preserved as `//`

### Changes Required:

#### 1. Comment Formatter
**File**: `internal/generator/generator.go`
**Changes**: Add comment formatting

```go
// formatComment converts a description to proto3 comment lines
// Returns slice of comment lines, each prefixed with "//"
func formatComment(description string) []string
```

**Function Responsibilities**:
- If description empty or whitespace only: return empty slice
- Split description by newlines
- For each line: trim trailing whitespace
- Prefix each line with `// `
- Blank lines become `//` (no space after)
- Return slice of comment lines

**Testing Requirements**:
```go
func TestConvertDescriptionComments(t *testing.T)
func TestConvertMultiLineDescription(t *testing.T)
```

**Test Objectives** (via Convert()):
- Schema with description → comment above message
- Field with description → comment above field
- Multi-line description → multiple comment lines
- No description → no comment
- Blank line in description → `//` line

**Context for Implementation**:
- Reference spec REQ-006 (lines 83-88): Description conversion
- Preserve description formatting, prefix each line with `//`

#### 2. Builder Enhancement
**File**: `internal/builder/builder.go`
**Changes**: Extract descriptions

**Function Responsibilities**:
- When building message: extract `schema.Description`
- When building field: extract property `schema.Description`
- Store in message/field Description fields

#### 3. Generator Enhancement
**File**: `internal/generator/generator.go`
**Changes**: Render comments

**Function Responsibilities**:
- Call formatComment on message/field descriptions
- Render comment lines above definitions
- Maintain proper indentation (nested comments indented with parent)

---

## Phase 8: Required/Nullable Handling & Output Polish

### Overview
Finalize output generation by explicitly ignoring OpenAPI `required` and `nullable` directives, and polish the proto3 formatting.

### Acceptance Criteria:
- [x] `required` array ignored (not read or processed)
- [x] `nullable: true` ignored (not read or processed)
- [x] No `optional` keyword in output
- [x] No wrapper types
- [x] Clean, well-formatted proto3 output

### Changes Required:

#### 1. Builder Enhancement ✅
**File**: `internal/builder/builder.go`
**Changes**: Document ignored fields

**Function Responsibilities**:
- DO NOT read `schema.Required` array
- DO NOT check `schema.Nullable` field
- All fields generated uniformly

**Testing Requirements**:
```go
func TestConvertRequiredIgnored(t *testing.T)
func TestConvertNullableIgnored(t *testing.T)
```

**Test Objectives** (via Convert()):
- Schema with `required: [field1, field2]` → all fields identical
- Schema with `nullable: true` → field same as non-nullable
- No `optional` keyword in output
- No wrapper types in output

**Context for Implementation**:
- Reference spec REQ-011 (lines 119-125): Required/nullable ignored

#### 2. Generator Polish ✅
**File**: `internal/generator/generator.go`
**Changes**: Final formatting

**Function Responsibilities**:
- Consistent indentation: 2 spaces for nested content
- Blank line spacing: 1 blank line between top-level definitions
- No trailing whitespace on lines
- Consistent comment formatting

---

## Phase 9: Error Handling & Edge Cases

### Overview
Implement comprehensive error handling for all unsupported OpenAPI features with clear, contextual error messages.

### Acceptance Criteria:
- [x] All unsupported features return clear errors with schema/property context
- [x] Input validation errors are actionable
- [x] Error messages use consistent format with single quotes
- [x] All error test scenarios pass

### Changes Required:

#### 1. Error Utilities
**File**: `internal/errors/errors.go`
**Changes**: Structured error helpers

```go
// schemaError creates error with schema context
func schemaError(schemaName, message string) error

// propertyError creates error with schema and property context
func propertyError(schemaName, propertyName, message string) error

// unsupportedError creates error for unsupported features
func unsupportedError(schemaName, propertyName, feature string) error
```

**Function Responsibilities**:
- Format with consistent structure
- Include context (schema name, property name)
- Use single quotes around names
- Clear, actionable messages

**Example Formats**:
- `schema '<name>': <message>`
- `schema '<schema>': property '<prop>' <message>`
- `schema '<schema>': property '<prop>' uses '<feature>' which is not supported`

**Testing Requirements**:
```go
func TestConvertUnsupportedAllOf(t *testing.T)
func TestConvertUnsupportedAnyOf(t *testing.T)
func TestConvertUnsupportedOneOf(t *testing.T)
func TestConvertUnsupportedNot(t *testing.T)
func TestConvertPropertyNoType(t *testing.T)
func TestConvertTopLevelPrimitive(t *testing.T)
func TestConvertTopLevelArray(t *testing.T)
func TestConvertErrorContext(t *testing.T)
```

**Test Objectives** (via Convert()):
- allOf → error mentioning allOf not supported
- anyOf → error mentioning anyOf not supported
- oneOf → error mentioning oneOf not supported
- not → error mentioning not not supported
- Property with no type and no $ref → error
- Top-level primitive → error "only objects and enums supported"
- Top-level array → error "only objects and enums supported"
- All errors include schema name in format `schema 'Name': ...`
- Property errors include property name

**Context for Implementation**:
- Reference spec REQ-013 (lines 137-148): Unsupported features
- Reference spec error examples (lines 584-590)

#### 2. Type Mapper Enhancement
**File**: `internal/mapper/mapper.go`
**Changes**: Detect unsupported features

```go
// validateSchema checks for unsupported features
func validateSchema(schema *base.Schema, schemaName, propertyName string) error
```

**Function Responsibilities**:
- Check for `allOf`, `anyOf`, `oneOf`, `not` (non-empty)
- If found: return unsupportedError with feature name
- Check for empty Type and no reference
- If found: return propertyError "must have type or $ref"

---

## Phase 10: Name Conflict Resolution & Final Integration

### Overview
Complete the implementation by ensuring name tracking works throughout, handle all edge cases, and add comprehensive integration tests.

### Acceptance Criteria:
- [x] Duplicate message/enum names get numeric suffixes (`_2`, `_3`)
- [x] Name tracking works across all generated types
- [x] Complete real-world integration test passes
- [x] All 30 test cases from spec pass
- [x] >80% test coverage

### Changes Required:

#### 1. Context Integration
**File**: `convert.go`, all builder files
**Changes**: Ensure nameTracker used throughout

**Function Responsibilities**:
- Create nameTracker in Convert()
- Pass context through all builder functions
- Call ctx.tracker.uniqueName() for all generated names (messages, enums)

**Testing Requirements**:
```go
func TestConvertNameConflicts(t *testing.T)
func TestConvertCompleteExample(t *testing.T)
```

**Test Objectives** (via Convert()):
- Duplicate schema names → numeric suffixes applied (_2, _3)
- Complex spec with all features → complete valid proto3 output
- Real-world example includes: enums, nested objects, arrays, refs, descriptions, all types

**Context for Implementation**:
- Reference spec REQ-014 (lines 150-154): Naming conflicts
- Reference spec Test 21 (lines 762-765): Complete integration

---

## Phase 11: Documentation & README

### Overview
Create comprehensive documentation including README with usage examples, API documentation, and important notes about naming conventions.

### Acceptance Criteria:
- [x] README.md with overview, installation, usage examples
- [x] Document snake_case algorithm (letter-by-letter, no acronym detection)
- [x] Document plural name validation for inline objects/enums
- [x] Godoc comments on all exported functions
- [x] Example OpenAPI → proto3 conversions

### Changes Required:

#### 1. README ✅
**File**: `README.md`
**Changes**: Create comprehensive README

**Contents**:
- Project overview and goals
- Installation instructions
- Basic usage example
- **Naming Conventions section**:
  - Explain snake_case conversion algorithm (each uppercase → underscore + lowercase)
  - Note: `HTTPStatus` → `h_t_t_p_status` (no special acronym handling)
  - Explain plural name validation for inline types
  - Best practices: use singular names or $ref for arrays
- Supported features list
- Unsupported features list
- Examples of OpenAPI → proto3 conversion
- Contributing guidelines

**Testing Requirements**:
None (documentation only)

**Context for Implementation**:
- Reference spec limitations (lines 994-1025)
- Design decisions about naming
- User must understand naming behavior to write compatible OpenAPI specs

---

## Validation Commands

After each phase:

```bash
# Run all tests
make test

# Check test coverage
make coverage

# Run linter
make lint
```

Final validation:

```bash
# All tests pass
make test

# High test coverage (>80%)
make coverage

# No lint errors
make lint
```

## Implementation Notes

**TDD Approach**:
- Write functional tests FIRST through `Convert()` API
- Design test cases to exercise internal code paths
- Implement features to make tests pass
- NO unit tests of internal functions

**Testing Strategy**:
- ONLY functional/end-to-end tests through public `Convert()` API
- NO unit tests of internal functions
- Table-driven tests with OpenAPI YAML → expected proto3 output
- Each test case is self-contained with complete OpenAPI spec
- Test objectives guide what internal code must do

**Test File Organization**:
- Split tests by feature area for maintainability
- Feature-specific tests live in `internal/` using `package internal_test`
- Integration tests live in project root using `package conv_test`
- All tests (both internal_test and conv_test) call only the public `conv.Convert()` API
- Test file structure:
  - `convert_integration_test.go` - Complete end-to-end scenarios (project root, `package conv_test`)
  - `internal/scalars_test.go` - Scalar type mappings (`package internal_test`)
  - `internal/enums_test.go` - Enum conversion (`package internal_test`)
  - `internal/arrays_test.go` - Array/repeated fields (`package internal_test`)
  - `internal/nested_test.go` - Nested objects (`package internal_test`)
  - `internal/refs_test.go` - Reference resolution (`package internal_test`)
  - `internal/naming_test.go` - Naming conventions (`package internal_test`)
  - `internal/errors_test.go` - Error handling (`package internal_test`)
- All tests import the public package: `conv "github.com/duh-rpc/openapi-proto"`

**Incremental Development**:
- Each phase builds on previous phases
- All tests from previous phases must continue passing
- Add new tests for new features

**Error Messages**:
- Always include context (schema name, property name)
- Use single quotes around names
- Be actionable - tell user how to fix

**Performance**:
- No optimization needed initially (spec requires <1s for <100 schemas)
- Focus on correctness and clarity

**Reference Materials**:
- Technical specification: `/Users/thrawn/Development/openapi-proto/plans/openapi-proto-research.md`
- libopenapi docs: https://pkg.go.dev/github.com/pb33f/libopenapi
- Proto3 spec: https://protobuf.dev/programming-guides/proto3/

**Dependencies**:
- `github.com/pb33f/libopenapi` - OpenAPI parser
- `github.com/stretchr/testify` - Test assertions
- Go standard library: `text/template`, `strings`, `fmt`, `bytes`
