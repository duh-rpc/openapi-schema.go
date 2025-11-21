> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# Timestamp and Options Implementation Plan

## Overview

This plan implements three related features to enhance the protobuf generation:
1. Refactor `Convert()` function to accept a `ConvertOptions` struct for extensibility
2. Map OpenAPI `date` and `date-time` formats to `google.protobuf.Timestamp`
3. Add `import` and `option go_package` to generated proto files

## Current State Analysis

### Current Implementation
- **convert.go:31**: `Convert(openapi []byte, packageName string)` - Simple two-parameter signature
- **internal/mapper.go:102-106**: Date/time formats map to `string`
- **internal/generator.go:10-13**: Template has no import or option sections
- **No timestamp tracking**: Context doesn't track well-known type usage

### Testing Patterns
- 11 test files call `conv.Convert([]byte, string)`
- Table-driven tests with `given`, `expected`, `wantErr` structure
- Tests in `_test` packages (e.g., `package conv_test`, `package internal_test`)
- Pattern: `conv.Convert([]byte(yaml), "packageName")`

### Key Constraints
- **Breaking change acceptable** (not in production yet)
- Only import google/protobuf/timestamp.proto when actually used
- PackagePath is **required** - must be provided by caller (e.g., "github.com/myorg/myrepo/proto/v1")
- This will require a **major version bump** if following semantic versioning
- Users upgrading will need to update all `Convert()` calls to use `ConvertOptions{}`

## Desired End State

### Generated Proto File Example

**When timestamp IS used:**
```protobuf
syntax = "proto3";

package myapi;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/myorg/myrepo/proto/v1;myapi";

message Event {
  string name = 1 [json_name = "name"];
  google.protobuf.Timestamp createdAt = 2 [json_name = "createdAt"];
  google.protobuf.Timestamp eventDate = 3 [json_name = "eventDate"];
}
```

**When timestamp is NOT used:**
```protobuf
syntax = "proto3";

package myapi;

option go_package = "github.com/myorg/myrepo/proto/v1;myapi";

message User {
  string name = 1 [json_name = "name"];
  int32 age = 2 [json_name = "age"];
}
```

### API Usage

```go
import conv "github.com/duh-rpc/openapi-proto.go"

result, err := conv.Convert(openapi, conv.ConvertOptions{
    PackageName: "myapi",
    PackagePath: "github.com/myorg/myrepo/proto/v1",
})
```

### Verification
Run `make ci` to verify:
- All tests pass with new signature
- Timestamp mapping works correctly
- Imports are conditionally included
- go_package option is formatted correctly

DO NOT COMMIT CHANGES, ONLY STAGE THE CODE so `make ci` will run successfully.

## What We're NOT Doing

- Not implementing other well-known types (Duration, Any, Struct, etc.)
- Not supporting timestamp validation or constraints
- Not adding configuration for import paths (hardcoded to google/protobuf/timestamp.proto)
- Not providing backward compatibility for old signature (this is a breaking change)
- Not generating separate import statements for nested types
- Not validating PackagePath format (assumed to be valid Go import path)
- Not handling multiple import statements (only google/protobuf/timestamp.proto for now)
- Not testing generated .proto files with `protoc` (manual verification recommended)

## Implementation Approach

The implementation uses a phased approach:
1. Add ConvertOptions struct and refactor Convert() signature
2. Add timestamp tracking to Context
3. Update type mapper to return google.protobuf.Timestamp
4. Update generator to include imports and go_package option
5. Update all tests to use new signature

This ensures each phase is testable and the conversion logic remains clear.

## Phase 1: Add ConvertOptions Struct

### Overview
Create `ConvertOptions` struct and refactor `Convert()` function signature for extensibility.

### Changes Required

#### 1. convert.go

**File**: `convert.go`
**Changes**: Add ConvertOptions struct and refactor Convert function

```go
// ConvertOptions configures the conversion from OpenAPI to Protocol Buffers
type ConvertOptions struct {
	PackageName string
	PackagePath string
}

// Convert converts OpenAPI 3.0 schemas to Protocol Buffer 3 format
func Convert(openapi []byte, opts ConvertOptions) ([]byte, error)
```

**Function Responsibilities:**
- `Convert()`: Validate options internally, parse document, build messages, generate output
- Validation logic should be inlined in Convert() to check PackageName and PackagePath are non-empty
- Pass `opts.PackagePath` through to generator

**Context for implementation:**
- Follow error handling pattern from existing Convert function (convert.go:32-38)
- Validation should happen before any processing
- PackageName validation already exists at line 36-38, adapt for struct field

#### 2. All Test Files

**Files to Update:**
- `convert_test.go`
- `internal/arrays_test.go`
- `internal/comments_test.go`
- `internal/conflicts_test.go`
- `internal/enums_test.go`
- `internal/errors_test.go`
- `internal/naming_test.go`
- `internal/nested_test.go`
- `internal/refs_test.go`
- `internal/required_test.go`
- `internal/scalars_test.go`

**Changes**: Update all `conv.Convert()` calls to use new signature

**Pattern to replace:**
```go
// OLD
result, err := conv.Convert([]byte(test.given), "testpkg")

// NEW
result, err := conv.Convert([]byte(test.given), conv.ConvertOptions{
    PackageName: "testpkg",
    PackagePath: "github.com/example/proto/v1",
})
```

**Test Objectives:**
- Verify all existing tests pass with new signature
- Verify validation errors for empty PackageName
- Verify validation errors for empty PackagePath

**New tests to add to convert_test.go TestConvertBasics:**

```go
{
    name:    "empty package path",
    given:   []byte("openapi: 3.0.0"),
    opts:    conv.ConvertOptions{PackageName: "test", PackagePath: ""},
    wantErr: "package path cannot be empty",
},
{
    name:    "both empty",
    given:   []byte("openapi: 3.0.0"),
    opts:    conv.ConvertOptions{},
    wantErr: "package name cannot be empty",
},
```

**Context for implementation:**
- Use consistent PackagePath in all tests: "github.com/example/proto/v1"
- Follow existing test structure (table-driven with given/expected/wantErr)
- Tests for validation should be added to convert_test.go TestConvertBasics

### Validation Commands
```bash
make test
```

## Phase 2: Track Timestamp Usage in Context

### Overview
Add timestamp usage tracking to Context so we know when to include the import.

### Changes Required

#### 1. internal/builder.go

**File**: `internal/builder.go`
**Changes**: Add timestamp tracking field to Context struct

```go
// Context holds state during conversion
type Context struct {
	Tracker          *NameTracker
	Messages         []*ProtoMessage
	Enums            []*ProtoEnum
	Definitions      []interface{}
	UsesTimestamp    bool
}
```

**Function Responsibilities:**
- `NewContext()`: Initialize UsesTimestamp to false
- Context is passed to all type mapping functions for tracking

**Context for implementation:**
- Follow existing pattern for other Context fields (builder.go:11-17)
- This field will be set by mapper.go when date/date-time types are encountered
- No changes to function signatures needed - Context already passed everywhere

### Validation Commands
```bash
make test
```

## Phase 3: Map Date/Time to Timestamp

### Overview
Update type mapper to return `google.protobuf.Timestamp` for date and date-time formats, and set timestamp tracking flag.

### Changes Required

#### 1. internal/mapper.go

**File**: `internal/mapper.go`
**Changes**: Modify MapScalarType to handle date/date-time formats

```go
// MapScalarType maps OpenAPI type+format to proto3 scalar type
// Updated signature to accept Context for timestamp tracking
func MapScalarType(ctx *Context, typ, format string) (string, error)
```

**Function Responsibilities:**
- When `typ == "string"` and `format == "date"`: set `ctx.UsesTimestamp = true` and return "google.protobuf.Timestamp"
- When `typ == "string"` and `format == "date-time"`: set `ctx.UsesTimestamp = true` and return "google.protobuf.Timestamp"
- When `typ == "string"` and `format` is anything else (or empty): return "string"
- Keep existing behavior for byte/binary formats (return "bytes")

**Context for implementation:**
- Modify the existing switch case at mapper.go:102-106
- Add Context as first parameter (Go convention for context-like parameters)
- Update all callers: ProtoType (line 83) and ResolveArrayItemType (line 196)
- Grep for "MapScalarType(" to verify no other callers exist

#### 2. internal/mapper.go (ProtoType function)

**File**: `internal/mapper.go`
**Changes**: Pass context to MapScalarType

```go
func ProtoType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, bool, error)
```

**Function Responsibilities:**
- Pass `ctx` to `MapScalarType` call at line 83
- No other changes needed - Context already available

#### 3. internal/mapper.go (ResolveArrayItemType function)

**File**: `internal/mapper.go`
**Changes**: Pass context to MapScalarType

```go
func ResolveArrayItemType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, error)
```

**Function Responsibilities:**
- Pass `ctx` to `MapScalarType` call at line 196
- No other changes needed - Context already available

#### 4. internal/scalars_test.go

**File**: `internal/scalars_test.go`
**Changes**: Update expected output for date/date-time fields

**Existing tests that need updates:**
- "all scalar type mappings" test (line 19-76)

**Expected output changes:**
```protobuf
// OLD
string dateField = 8 [json_name = "dateField"];
string dateTimeField = 9 [json_name = "dateTimeField"];

// NEW
google.protobuf.Timestamp dateField = 8 [json_name = "dateField"];
google.protobuf.Timestamp dateTimeField = 9 [json_name = "dateTimeField"];
```

**Test Objectives:**
- Verify date format maps to google.protobuf.Timestamp
- Verify date-time format maps to google.protobuf.Timestamp
- Verify other string formats (byte, binary) still work correctly

**Additional test case to add:**

```go
{
    name: "array of timestamps",
    given: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    Event:
      type: object
      properties:
        timestamp:
          type: array
          items:
            type: string
            format: date-time
`,
    expected: `syntax = "proto3";

package testpkg;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/example/proto/v1;testpkg";

message Event {
  repeated google.protobuf.Timestamp timestamp = 1 [json_name = "timestamp"];
}
`,
},
```

#### 5. convert_test.go

**File**: `convert_test.go`
**Changes**: Update expected output for date-time field in TestConvertCompleteExample

**Test at line 331** ("complete example with date-time field")

**Expected output changes:**
```protobuf
// OLD (line 485-486)
string createdAt = 7 [json_name = "createdAt"];

// NEW
google.protobuf.Timestamp createdAt = 7 [json_name = "createdAt"];
```

**Test Objectives:**
- Verify timestamp mapping works in complex real-world example
- Verify all other field types remain correct

**Context for implementation:**
- This is the only test in convert_test.go with a date-time field
- Tests in this file use more realistic schemas

### Validation Commands
```bash
make test
```

## Phase 4: Add Import and go_package Option

### Overview
Update generator template to conditionally include import statement and always include go_package option.

### Changes Required

#### 1. internal/generator.go

**File**: `internal/generator.go`
**Changes**: Update template and templateData struct

```go
const protoTemplate = `syntax = "proto3";

package {{.PackageName}};
{{if .UsesTimestamp}}
import "google/protobuf/timestamp.proto";
{{end}}
option go_package = "{{.GoPackage}}";
{{range .Definitions}}{{renderDefinition .}}{{end}}`

type templateData struct {
	PackageName   string
	Messages      []*ProtoMessage
	Enums         []*ProtoEnum
	Definitions   []interface{}
	UsesTimestamp bool
	GoPackage     string
}

// Generate creates proto3 output from messages and enums in order
func Generate(packageName string, packagePath string, messages []*ProtoMessage, enums []*ProtoEnum, definitions []interface{}, usesTimestamp bool) ([]byte, error)
```

**Function Responsibilities:**
- Build `GoPackage` string from `packagePath` and `packageName`: `"{packagePath};{packageName}"`
- Include timestamp import only when `usesTimestamp` is true
- Always include go_package option (packagePath is required by validation)
- Maintain existing template rendering logic
- Template blank line rules: blank line after import (if present), no blank line after option

**IMPORTANT Template Whitespace Rules:**
```protobuf
syntax = "proto3";
                           <- blank line
package name;
                           <- blank line (if import present)
import "google/...";       <- only if UsesTimestamp
                           <- blank line
option go_package = "..."; <- always present
                           <- blank line
message Foo {              <- first definition
```

**Context for implementation:**
- Follow existing template pattern from generator.go:10-13
- Template conditionals use Go template syntax: `{{if .UsesTimestamp}}...{{end}}`
- Line breaks matter - import should be on its own line
- go_package format: `"github.com/path/to/proto/v1;packagename"`

#### 2. convert.go

**File**: `convert.go`
**Changes**: Pass new parameters to Generate function

```go
func Convert(openapi []byte, opts ConvertOptions) ([]byte, error)
```

**Function Responsibilities:**
- Pass `opts.PackagePath` to Generate call at line 56
- Pass `ctx.UsesTimestamp` to Generate call
- Update Generate function call signature

**Context for implementation:**
- Generate is called at convert.go:56
- Context already has UsesTimestamp field from Phase 2
- No validation needed - already done in opts.Validate()

#### 3. internal/scalars_test.go

**File**: `internal/scalars_test.go`
**Changes**: Add import to expected output when timestamp is used

**Test: "all scalar type mappings" (line 19-76)**

**Expected output changes:**
```protobuf
syntax = "proto3";

package testpkg;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/example/proto/v1;testpkg";

message AllTypes {
  ...
  google.protobuf.Timestamp dateField = 8 [json_name = "dateField"];
  google.protobuf.Timestamp dateTimeField = 9 [json_name = "dateTimeField"];
  ...
}
```

**Test Objectives:**
- Verify import appears when timestamp is used
- Verify go_package option appears with correct format
- Verify import appears AFTER package, BEFORE option

#### 4. convert_test.go

**File**: `convert_test.go`
**Changes**: Add go_package option to all expected outputs

**Tests to update:**
- TestConvertBasics (line 11-85)
- TestConvertParseDocument (line 87-138)
- TestConvertExtractSchemas (line 140-210)
- TestConvertSimpleMessage (line 212-292)
- TestConvertFieldOrdering (line 294-329)
- TestConvertCompleteExample (line 331-492)

**Expected output pattern:**
```protobuf
syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1;testpkg";

// ... rest of definitions
```

**For TestConvertCompleteExample** (line 331):
```protobuf
syntax = "proto3";

package ecommerce;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/example/proto/v1;ecommerce";

// ... rest of definitions
```

**Test Objectives:**
- Verify go_package option appears in all generated protos
- Verify format: `"packagePath;packageName"`
- Verify import only appears when timestamp is used (TestConvertCompleteExample)

#### 5. All Other Test Files

**Files**:
- `internal/arrays_test.go`
- `internal/comments_test.go`
- `internal/conflicts_test.go`
- `internal/enums_test.go`
- `internal/errors_test.go`
- `internal/naming_test.go`
- `internal/nested_test.go`
- `internal/refs_test.go`
- `internal/required_test.go`

**Changes**: Add go_package option to all expected outputs

**Pattern:**
```protobuf
syntax = "proto3";

package testpkg;

option go_package = "github.com/example/proto/v1;testpkg";

// ... rest of definitions
```

**Test Objectives:**
- Verify all tests pass with go_package option added
- Verify no imports appear (these tests don't use timestamp)

### Validation Commands
```bash
make test
make ci
```

## Phase 5: Update Documentation

### Overview
Update README.md and documentation to reflect the new changes.

### Changes Required

#### 1. README.md

**File**: `README.md`
**Changes**: Update usage example, type mapping table, and features list

**Sections to Update:**

1. **Usage Example (line 24-53)**: Update to use ConvertOptions
```go
result, err := conv.Convert(openapi, conv.ConvertOptions{
    PackageName: "myapi",
    PackagePath: "github.com/myorg/myrepo/proto/v1",
})
```

2. **Output Example (line 82-95)**: Add import and go_package option
```protobuf
syntax = "proto3";

package myapi;

option go_package = "github.com/myorg/myrepo/proto/v1;myapi";

// A user account
message User {
  ...
}
```

3. **Supported Features (line 97-117)**: Update to include imports and options
- Change "❌ Import statements" to "✅ Import statements (google/protobuf/timestamp.proto)"
- Change "❌ Proto options beyond json_name" to "✅ go_package option"

4. **Type Mapping Table (line 143-161)**: Update date/date-time mappings
```markdown
| string       | date           | google.protobuf.Timestamp |
| string       | date-time      | google.protobuf.Timestamp |
```

5. **Unsupported Features (line 119-138)**: Remove from unsupported list
- Remove "Import statements" from Proto3 Features Not Generated section
- Remove "Proto options beyond json_name" from Proto3 Features Not Generated section

**Function Responsibilities:**
- Accurately document new API signature
- Show timestamp mapping in type table
- Update examples to include import and go_package
- Remove timestamp-related items from unsupported features

**Context for implementation:**
- Follow existing README formatting and structure
- Update line numbers may shift as content changes
- Keep examples consistent with test cases

#### 2. docs/scalar.md

**File**: `docs/scalar.md`
**Changes**: Update type mapping table and examples

**Sections to Update:**

1. **Type Mapping Table (line 9-23)**: Update date/date-time row
```markdown
| `string` | `date` or `date-time` | `google.protobuf.Timestamp` | RFC 3339 date-time |
```

2. **Notes on Type Mapping (line 24-30)**: Update date/datetime note
- Change: "Date/DateTime: These are converted to string rather than using google.protobuf.Timestamp for simplicity"
- To: "Date/DateTime: These are converted to google.protobuf.Timestamp per protobuf conventions"

3. **Complete Example (line 143-204)**: Update to show timestamp usage
- Add import when timestamp field is present
- Update generated proto example with go_package option

**Test Objectives:**
- Documentation accurately reflects implementation
- Examples are runnable and produce expected output
- Type mapping table is correct

### Validation Commands
```bash
# Manual verification - documentation doesn't affect tests
# Review generated output matches documented examples
make test
```

## Context for All Phases

### Key Discoveries
- **Context threading**: Context is already threaded through all type resolution functions (mapper.go)
- **Template-based generation**: Using Go text/template with custom functions (generator.go:24-27)
- **Ordered schemas**: libopenapi preserves YAML insertion order via FromOldest() (parser.go:54)
- **Error patterns**: Consistent error formatting using helper functions (errors.go)

### Integration Points
- **convert.go** calls **parser.ParseDocument()** → **builder.BuildMessages()** → **generator.Generate()**
- **builder.BuildMessages()** calls **mapper.ProtoType()** for each property
- **mapper.ProtoType()** calls **mapper.MapScalarType()** for scalar types
- **generator.Generate()** uses templates to render final output

### Testing Patterns
- All tests follow table-driven pattern with subtests using `t.Run()`
- Tests use raw YAML strings as input, compare string output
- Error tests check error message contains expected string
- Tests organized by feature: scalars, enums, arrays, refs, etc.

### Patterns to Follow
- **Struct field ordering**: Follow visual tapering (longest to shortest lines)
- **Error messages**: Use helper functions from errors.go for consistency
- **Naming**: Use ToPascalCase for message names, preserve field names
- **Comments**: No implementation comments about following guidelines
