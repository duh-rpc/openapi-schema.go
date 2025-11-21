> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# Fix Field Name Suffix Bug Implementation Plan

## Overview

Fix incorrect field name suffixing when field names are reused across different messages. Currently, field names are tracked globally causing fields like `email` in `CreateUserRequest` and `CreateUserResponse` to get suffixed (`email`, `email_2`), when they should both be `email` since they exist in separate message scopes.

## Current State Analysis

### Bug Description
The `NameTracker` in `internal/naming.go` is used globally for all name collision detection, including:
- Message names (correct - global scope)
- Enum names (correct - global scope)
- Field names (incorrect - should be message scope)

### Problematic Code Locations
- `internal/builder.go:123` - Top-level message field generation uses global tracker
- `internal/builder.go:266` - Nested message field generation uses global tracker

Both locations call:
```go
protoFieldName := ctx.Tracker.UniqueName(sanitizedName)
```

This adds the field name to the **global** tracker shared across all messages, causing false collisions.

### Current Behavior Example
```yaml
CreateUserRequest:
  properties:
    email: string
    name: string

CreateUserResponse:
  properties:
    email: string
    name: string
```

Generates (INCORRECT):
```protobuf
message CreateUserRequest {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

message CreateUserResponse {
  string email_2 = 1 [json_name = "email"];  // ✗ Wrong suffix
  string name_2 = 2 [json_name = "name"];    // ✗ Wrong suffix
}
```

### Key Discoveries

1. **Global scope collision handling works correctly**:
   - `internal/conflicts_test.go` verifies message/enum name collisions
   - Message names use `ctx.Tracker.UniqueName()` at `builder.go:104` ✓
   - Enum names use `ctx.Tracker.UniqueName()` at `builder.go:188` ✓
   - Nested message names use `ctx.Tracker.UniqueName()` at `builder.go:244` ✓

2. **No existing test for field name scoping**:
   - Tests verify message/enum collisions, not field collisions
   - No test verifies same field names across different messages

3. **Per-message tracking needed for edge cases**:
   - Handles sanitization collisions within same message
   - Example: `user-name` and `user_name` both sanitize to `user_name`
   - First becomes `user_name`, second becomes `user_name_2` (correct within message)

## Desired End State

### Expected Behavior
Field names are unique only within their parent message:

```protobuf
message CreateUserRequest {
  string email = 1 [json_name = "email"];
  string name = 2 [json_name = "name"];
}

message CreateUserResponse {
  string email = 1 [json_name = "email"];  // ✓ No suffix
  string name = 2 [json_name = "name"];    // ✓ No suffix
}
```

### Name Scoping Rules
- **Global scope**: Message names, Enum names (use `ctx.Tracker`)
- **Message scope**: Field names (use per-message tracker)
- **Not affected**: Enum values use `ToEnumValueName()` which generates predictable prefixed names, no tracker needed

### Verification
Run existing tests to ensure no regressions:
```bash
go test ./internal/... -v
```

New test should verify field names work across messages:
```bash
go test ./internal/... -v -run TestFieldNamesAcrossMessages
```

## What We're NOT Doing

- Not changing global scope collision handling for messages/enums
- Not modifying the `NameTracker` implementation itself
- Not changing field numbering logic
- Not altering sanitization rules
- Not affecting nested message name collision handling

## Implementation Approach

Create a per-message `NameTracker` instance for field name collision detection while preserving the global `ctx.Tracker` for message and enum names. This minimal change isolates field name scoping to each message without affecting the proven global collision handling.

---

## Phase 1: Fix Field Name Scoping in Top-Level Messages ✓

### Overview
Update `buildMessage()` to use a per-message `NameTracker` for field names instead of the global tracker, ensuring fields in different messages don't collide.

### Changes Required

#### 1. Field Name Tracking in buildMessage() ✓
**File**: `internal/builder.go`
**Changes**: Update field name generation logic

```go
func buildMessage(name string, proxy *base.SchemaProxy, ctx *Context) (*ProtoMessage, error)
```

**Function Responsibilities:**
- Create a new `NameTracker` instance at start: `fieldTracker := NewNameTracker()`
- Continue using `ctx.Tracker.UniqueName()` for message name (line 104) - no change
- Replace `ctx.Tracker.UniqueName(sanitizedName)` with `fieldTracker.UniqueName(sanitizedName)` at line 123
- Preserve all other logic (sanitization, field numbering, descriptions)

**Context for Implementation:**
- The global tracker `ctx.Tracker` is accessed via the Context parameter and used ONLY for message names
- Create local field tracker: `fieldTracker := NewNameTracker()`
- Field name generation occurs in the properties loop at lines 112-150
- Sanitization happens via `SanitizeFieldName()` before uniqueness check
- Current pattern at line 123: `protoFieldName := ctx.Tracker.UniqueName(sanitizedName)`
- New pattern: `protoFieldName := fieldTracker.UniqueName(sanitizedName)`
- Note: `ProtoType()` may call `buildNestedMessage()` recursively, which will create its own independent field tracker

#### 2. Testing Requirements ✓

```go
func TestFieldNamesAcrossMessages(t *testing.T)
```

**Test Objectives:**
- Verify fields with same names in different messages don't get suffixes
- Verify field name sanitization collisions within same message still get suffixes
- Verify field numbering starts at 1 for each message independently
- Verify the exact bug scenario (CreateUserRequest/CreateUserResponse) is fixed

**Test Scenarios:**
1. **Common field names across messages**: Multiple messages (CreateUserRequest, CreateUserResponse, UpdateUserRequest) all have `email`, `name` fields without suffixes
2. **Field numbering independence**: Verify each message has `email = 1, name = 2` not continuing field numbers across messages
3. **Sanitization collisions within message**: Message with `user-name` and `user_name` → both sanitize to `user_name`, become `user_name` and `user_name_2`
4. **Sanitization collisions across messages**: Two different messages each with `user-name` and `user_name` → each message independently has `user_name` and `user_name_2`
5. **Original bug scenario**: CreateUserRequest and CreateUserResponse with same field names

**Context for Implementation:**
- Create new file `internal/field_scoping_test.go`
- Follow table-driven test pattern from `internal/conflicts_test.go:11-150`
- Use correct API: `conv.Convert([]byte(test.given), conv.ConvertOptions{PackageName: "testpkg", PackagePath: "github.com/example/proto/v1"})`
- Use `require.NoError(t, err)` for error checking, `assert.Equal(t, test.expected, string(result))` for output comparison
- Pattern: `for _, test := range []struct { name, given, expected string }`
- Phase 1 creates this file with test cases 1, 2, 3, 5 (top-level messages only)
- Phase 2 adds test case 4 and nested message scenarios to this same test function

**Validation Commands:**
```bash
go test ./internal/... -v -run TestFieldNamesAcrossMessages
go test ./internal/... -v  # Ensure no regressions
```

---

## Phase 2: Fix Field Name Scoping in Nested Messages ✓

### Overview
Update `buildNestedMessage()` to use a per-message `NameTracker` for field names, ensuring consistency with top-level message behavior.

### Changes Required

#### 1. Field Name Tracking in buildNestedMessage() ✓
**File**: `internal/builder.go`
**Changes**: Update nested message field name generation logic

```go
func buildNestedMessage(propertyName string, proxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (*ProtoMessage, error)
```

**Function Responsibilities:**
- Create a new `NameTracker` instance at start: `fieldTracker := NewNameTracker()`
- Continue using `ctx.Tracker.UniqueName()` for nested message name (line 244) - no change
- Replace `ctx.Tracker.UniqueName(sanitizedName)` with `fieldTracker.UniqueName(sanitizedName)` at line 266
- Preserve all other logic (name derivation, sanitization, field numbering)

**Context for Implementation:**
- Same pattern as Phase 1 but for nested messages
- The global tracker `ctx.Tracker` is used ONLY for the nested message name itself
- Create local field tracker: `fieldTracker := NewNameTracker()`
- Nested message name generation at line 244: `msgName = ctx.Tracker.UniqueName(msgName)` - keep this
- Field name generation occurs in the properties loop at lines 254-293
- Current pattern at line 266: `protoFieldName := ctx.Tracker.UniqueName(sanitizedName)`
- New pattern: `protoFieldName := fieldTracker.UniqueName(sanitizedName)`
- Important: `buildNestedMessage()` can be called recursively for deeply nested structures; each call creates its own independent field tracker
- This is called from `ProtoType()` in `mapper.go` when processing inline object properties

#### 2. Testing Requirements ✓

Add test cases to existing `TestFieldNamesAcrossMessages`:

```go
func TestFieldNamesAcrossMessages(t *testing.T) // Add nested message test cases
```

**Additional Test Objectives:**
- Verify nested messages have independent field name scopes from their parent
- Verify parent and nested messages can have same field names without conflicts
- Verify nested messages in different parent messages can have same field names
- Verify deeply nested structures (nested within nested) work correctly

**Test Scenarios:**
1. **Parent and nested with same field names**:
   - `User.email` and `User.Address.email` both exist without suffixes
2. **Multiple nested in different parents**:
   - `User.Address.street` and `Product.Warehouse.street` both exist without suffixes
3. **Mixed top-level and nested**:
   - `CreateUserRequest.email`, `CreateUserResponse.email`, `User.Contact.email`, `Product.Supplier.email` all exist without suffixes
4. **Deeply nested recursive structures**:
   - Each nested level has independent field scopes

**Context for Implementation:**
- Add these test cases to the table-driven test in `internal/field_scoping_test.go` created in Phase 1
- Follow nested message patterns from `internal/nested_test.go:11-150`
- Verify nested message field numbering remains independent (each nested message starts at 1)
- Ensure nested message definitions appear before parent fields in proto3 output
- Nested messages use PascalCase names derived from property name

**Validation Commands:**
```bash
go test ./internal/... -v -run TestFieldNamesAcrossMessages
go test ./internal/... -v  # Full regression test
```

---

## Phase 3: Verify and Document ✓

### Overview
Run full test suite, verify no regressions, and ensure the fix is complete.

### Changes Required

#### 1. Regression Testing ✓
**Commands**: Full test suite execution

**Testing Responsibilities:**
- Run all existing tests to ensure no regressions
- Verify global scope collision tests still pass (`internal/conflicts_test.go`)
- Verify field naming tests still pass (`internal/naming_test.go`)
- Verify all other feature tests pass (scalars, arrays, nested, enums, refs, etc.)

**Validation Commands:**
```bash
go test ./... -v
go test ./... -race  # Race condition check (general good practice, not specific to this fix)
```

#### 2. Verify Exact Bug Scenario is Fixed ✓
**Testing Responsibilities:**
- Validate the specific bug described in the issue is resolved
- CreateUserRequest and CreateUserResponse should both have `email` without suffixes

**Validation Command:**
```bash
# Run the specific test that reproduces the original bug
go test ./internal/... -v -run "TestFieldNamesAcrossMessages/original.*bug"
```

**Expected Output:**
```protobuf
message CreateUserRequest {
  string email = 1 [json_name = "email"];  // No suffix
  string name = 2 [json_name = "name"];
}

message CreateUserResponse {
  string email = 1 [json_name = "email"];  // No suffix (was email_2 before fix)
  string name = 2 [json_name = "name"];    // No suffix (was name_2 before fix)
}
```

#### 3. Manual Verification (Optional)
**Testing Responsibilities:**
- Ad-hoc verification with realistic scenario (not checked into codebase)
- Create a realistic OpenAPI spec with multiple messages sharing field names
- Convert to proto3 and manually inspect output
- Check that json_name annotations are preserved
- Verify field numbering remains sequential

**Example Verification:**
```bash
# Create temporary test YAML with User CRUD operations
# Use the conversion tool
# Manually inspect proto3 output for correct field names
```

#### 4. Existing Tests That May Need Updates

None expected, but verify these still pass:
- `internal/conflicts_test.go` - Message/enum collision tests
- `internal/naming_test.go` - Field name sanitization tests
- `internal/nested_test.go` - Nested message tests
- `internal/scalars_test.go` - Scalar type tests
- `convert_test.go` - Integration tests

**Context for Implementation:**
- These tests should NOT require changes
- If any fail, investigate why before modifying tests
- Follow the principle: "Prefer fixing implementation over changing tests"

**Validation Commands:**
```bash
go test ./... -v
```

