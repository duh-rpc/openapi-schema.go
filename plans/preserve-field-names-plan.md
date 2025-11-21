> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# Preserve Original Field Names Implementation Plan

## Overview

Change the field name conversion behavior to preserve the original OpenAPI field names in the generated proto3 output. Instead of converting `HTTPStatus` to `h_t_t_p_status`, the library will use `HTTPStatus` directly when it's valid proto3 syntax, only performing minimal conversions when necessary to comply with proto3 syntax requirements.

## Current State Analysis

### What Exists Now

**Current behavior** (internal/naming.go:12-32):
- Simple letter-by-letter snake_case conversion
- Each uppercase letter becomes lowercase with underscore prefix
- No acronym detection
- `HTTPStatus` → `h_t_t_p_status`
- `userId` → `user_id`
- `http_status` → `http_status`

**Current usage** (internal/builder.go:117):
```go
protoFieldName := ctx.Tracker.UniqueName(ToSnakeCase(propName))
```

**Current tests**:
- `internal/naming_test.go` validates current snake_case behavior
- Tests explicitly check for `h_t_t_p_status` output
- Tests validate name conflict resolution with snake_case

**Documentation**:
- README.md:159-172 documents the letter-by-letter algorithm
- Explicitly states `HTTPStatus` → `h_t_t_p_status`
- Notes this is intentional, not a bug

### Proto3 Syntax Requirements

From Protocol Buffer specification research:

**Field name syntax** (proto3-spec):
- Grammar: `ident = letter { letter | decimalDigit | "_" }`
- Must start with a letter (A-Z or a-z)
- Can contain: letters, digits (0-9), underscores (_)
- Cannot contain: hyphens (-), spaces, dots, or other special characters
- **Uppercase letters ARE allowed**

**Valid examples**:
- `HTTPStatus` ✅
- `userId` ✅
- `user_id` ✅
- `http2Protocol` ✅

**Invalid examples**:
- `status-code` ❌ (hyphen not allowed)
- `user.name` ❌ (dot not allowed)
- `2ndValue` ❌ (starts with digit)
- `_private` ❌ (starts with underscore)

**Style guide recommendations** (not requirements):
- Recommends snake_case for fields
- Treats abbreviations as words: `DnsRequest` not `DNSRequest`
- These are conventions, not syntax constraints

### Key Discoveries

1. **Proto3 is more flexible than current implementation assumes**
   - Current conversion to snake_case is stylistic, not required
   - `HTTPStatus` is perfectly valid proto3 syntax

2. **json_name annotation still needed**
   - Current practice: always include json_name for consistency
   - Will continue this practice even when field name matches original

3. **Name conflict resolution still applies**
   - NameTracker logic for handling duplicates remains necessary
   - Example: `user_id` and `userId` might both be in OpenAPI
   - Currently both become `user_id` and `user_id_2`
   - New behavior: stay as `user_id` and `userId` (no conflict)

## Desired End State

A library that:
- Preserves original OpenAPI field names when they're valid proto3 syntax
- Only modifies names when proto3 syntax requires it
- Handles invalid characters by replacing with underscores
- Maintains name conflict detection and resolution
- Provides clear error messages for edge cases
- Updates all tests to validate new behavior
- Updates documentation to reflect new naming approach

**Verification**:
```bash
make test     # all tests pass with new behavior
make lint     # no lint errors
make coverage # >80% coverage maintained
```

## What We're NOT Doing

**Out of scope:**
- Configuration options for choosing naming modes
- Backwards compatibility mode or migration path
- Detecting and warning about style guide violations
- Advanced acronym detection or smart casing
- Converting to snake_case for "consistency"
- Handling proto3 reserved keywords (let protoc catch them)

**Explicit non-goals:**
- This is a breaking change - not providing backwards compatibility
- Not adding complexity for configurable naming strategies
- Not trying to be "smarter" than the user about naming

## Implementation Approach

**Strategy**: Modify the naming conversion logic to preserve original names, handling only syntax-level constraints. Update all tests to expect preserved names. Update documentation to reflect new behavior.

**Testing Philosophy**:
- Update existing tests to expect preserved names
- Add new tests for invalid character handling
- Ensure all functional tests still pass
- Validate name conflict resolution still works

**Breaking Change Management**:
- Update README with new naming behavior and breaking change notice
- Update all test expectations
- Document in code comments that this is a breaking change from previous behavior

## Phase 1: Sanitize Field Names (Minimal Conversion)

### Overview
Replace the ToSnakeCase conversion with a sanitization function that only modifies field names when proto3 syntax requires it. The new function preserves the original name structure and only replaces invalid characters.

### Acceptance Criteria:
- New `SanitizeFieldName()` function replaces invalid characters with underscores
- Validates field names start with a letter
- Handles invalid leading characters (digits, underscores)
- Original casing preserved when valid
- Invalid characters (-, ., spaces) replaced with underscores
- `HTTPStatus` → `HTTPStatus` (unchanged)
- `status-code` → `status_code` (hyphen replaced)
- `userId` → `userId` (unchanged)

### Changes Required:

#### 1. Field Name Sanitizer - ✅ COMPLETE
**File**: `internal/naming.go`
**Changes**: Add new sanitization function

```go
// SanitizeFieldName sanitizes an OpenAPI field name for proto3 syntax.
// Preserves original name when valid, only modifies to meet proto3 requirements.
// Returns sanitized name and error if name is invalid beyond repair.
func SanitizeFieldName(name string) (string, error)

// isValidProtoFieldChar returns true if character is valid in proto3 field name
func isValidProtoFieldChar(r rune) bool
```

**Function Responsibilities**:
- `SanitizeFieldName`:
  - If name is empty: return error "field name cannot be empty"
  - Check first character: must be ASCII letter (a-z, A-Z)
  - If starts with digit: return error "field name must start with a letter, got '%s'"
  - If starts with underscore: return error "field name cannot start with underscore, got '%s'"
  - Use strings.Builder for efficient string construction
  - Iterate through characters in single pass:
    - If valid (ASCII letter, digit, or underscore): write as-is
    - If invalid: replace with single underscore
    - Track last written char to avoid consecutive underscores from replacements
  - After loop: trim trailing underscore only if it was added by sanitization (check if original ended with underscore)
  - If sanitized result is empty (e.g., input was "---"): return error "field name contains no valid characters"
  - Return sanitized name
- `isValidProtoFieldChar`:
  - Return true if: (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
  - Return false otherwise
  - **Important**: ASCII-only, no unicode letters. This ensures protoc compatibility.

**Algorithm Details** (single-pass implementation):
```go
// Pseudocode for single-pass sanitization:
// 1. Validate first char is ASCII letter, error if not
// 2. Build result char by char:
//    - Valid char → append to result
//    - Invalid char → append '_' if last char wasn't '_'
// 3. If original ended with valid char but result ends with '_':
//    - Trim the trailing '_' (was added by sanitization)
// 4. Return result
```

**Consecutive Underscore Handling**:
- Underscores in original field name: preserved as-is (e.g., `user__id` → `user__id`)
- Underscores from sanitization: collapsed (e.g., `status--code` → `status_code`)
- Implementation: only write '_' if previous written char wasn't '_'

**Testing Requirements**:
```go
func TestSanitizeFieldName(t *testing.T)
func TestSanitizeInvalidCharacters(t *testing.T)
func TestSanitizeInvalidStart(t *testing.T)
```

**Test Objectives** (via Convert()):
- **Preservation cases**:
  - `HTTPStatus` → `HTTPStatus` (case preserved)
  - `userId` → `userId` (case preserved)
  - `user_id` → `user_id` (underscores preserved)
  - `user__id` → `user__id` (consecutive underscores in original preserved)
  - `UserID` → `UserID` (all caps preserved)
- **Sanitization cases**:
  - `status-code` → `status_code` (hyphen replaced)
  - `user.name` → `user_name` (dot replaced)
  - `first name` → `first_name` (space replaced)
  - `status--code` → `status_code` (consecutive hyphens collapsed)
  - `api..v2..endpoint` → `api_v2_endpoint` (consecutive dots collapsed)
  - `user---name` → `user_name` (multiple consecutive replaced)
  - `status-` → `status` (trailing underscore trimmed after sanitization)
  - `ñame` → `_ame` (non-ASCII 'ñ' replaced, but then errors because starts with underscore - see error cases)
- **Error cases**:
  - `2ndValue` → error "field name must start with a letter, got '2ndValue'"
  - `_private` → error "field name cannot start with underscore, got '_private'"
  - Empty string → error "field name cannot be empty"
  - `---` → error "field name contains no valid characters" (all chars invalid)
  - `___` → error "field name cannot start with underscore" (starts with underscore)
  - `ñame` → error "field name must start with a letter" (non-ASCII not allowed)
- **Edge cases**:
  - Very long name (1000 chars) → preserved if valid
  - Mixed valid/invalid: `user-ID` → `user_ID`

**Context for Implementation**:
- Replace ToSnakeCase usage in builder.go:117
- Proto3 allows uppercase, so preserve it
- Only modify when syntax requires it
- Reference proto3 spec for valid characters

#### 2. Update Builder Usage - ✅ COMPLETE
**File**: `internal/builder.go`
**Changes**: Replace ToSnakeCase with SanitizeFieldName

**Current code** (line 117):
```go
protoFieldName := ctx.Tracker.UniqueName(ToSnakeCase(propName))
```

**New code**:
```go
sanitizedName, err := SanitizeFieldName(propName)
if err != nil {
    return nil, PropertyError(name, propName, err.Error())
}
protoFieldName := ctx.Tracker.UniqueName(sanitizedName)
```

**Function Responsibilities**:
- Call SanitizeFieldName instead of ToSnakeCase
- Handle error from SanitizeFieldName
- Wrap with PropertyError for context
- Pass sanitized name to UniqueName for conflict resolution

**Testing Requirements**:
Already covered by test cases in Phase 1.1

**Context for Implementation**:
- This is in buildMessage function
- Also check buildAllOfMergedMessage (line 256) for same pattern
- Ensure both locations updated

#### 3. Remove NeedsJSONName Function - ✅ COMPLETE
**File**: `internal/naming.go`
**Changes**: Remove NeedsJSONName function (line 91-94)

**Rationale**:
- Current practice: always include json_name for consistency and clarity
- With preservation, the decision remains: always include json_name
- NeedsJSONName function becomes redundant
- Simplifies codebase

**Function Responsibilities**:
- Delete the NeedsJSONName function
- json_name will always be included in generated proto output
- This is already the current behavior (line 139 in builder.go always sets JSONName)

**Testing Requirements**:
None - function is not used in current codebase (builder always sets JSONName)

**Test Objectives**:
- Verify all fields have json_name annotations
- No change in behavior, just removing unused function

**Context for Implementation**:
- Check if function is referenced anywhere before removing
- If used, replace calls with `true` literal
- Generator already handles json_name output unconditionally

#### 4. Keep ToSnakeCase for Other Uses - ✅ COMPLETE
**File**: `internal/naming.go`
**Changes**: Keep ToSnakeCase for enum names and message names

**Function Responsibilities**:
- ToSnakeCase still used for:
  - Enum value name generation (line 85: `ToSnakeCase(enumName)`)
  - Converting to lowercase when needed
  - This function remains for internal uses
- SanitizeFieldName is specifically for field names from OpenAPI

**Testing Requirements**:
None - existing enum tests should still pass

**Test Objectives**:
- Enum names still converted properly
- Enum value prefixes use snake_case

**Context for Implementation**:
- Don't remove ToSnakeCase, it has other uses
- Only change field name handling
- ToPascalCase also remains for message names

---

## Phase 2: Update Existing Tests - ✅ COMPLETE

### Overview
Update all existing tests to expect preserved field names instead of snake_case conversions. Add new test cases for invalid character handling and edge cases.

### Acceptance Criteria:
- All tests in `internal/naming_test.go` updated with new expectations
- All tests in `internal/scalars_test.go` updated
- Name conflict tests updated to reflect new behavior
- New tests added for invalid character scenarios
- All tests pass with new naming behavior

### Changes Required:

#### 1. Update Naming Tests - ✅ COMPLETE
**File**: `internal/naming_test.go`
**Changes**: Update test expectations

**Existing tests that need updates**:
- `TestConvertJSONNameAnnotation`:
  - Line 37: Change expected from `user_id` to `userId`
  - Line 37: json_name stays `[json_name = "userId"]`
  - Line 61: Keep `email` as `email`
  - Line 85: Change from `h_t_t_p_status` to `HTTPStatus`
  - Line 85: json_name stays `[json_name = "HTTPStatus"]`
- `TestConvertAlwaysIncludesJsonName`:
  - Line 125: Change from `user_id` to `userId`
  - Line 126: Change from `user_id_2` to `user_id` (no conflict anymore!)
  - Line 127: Change from `h_t_t_p_status` to `HTTPStatus`
  - Line 128: Keep `status_code` as `status_code`
  - Line 129: Keep `email` as `email`

**New test cases to add**:
```go
func TestConvertFieldNameSanitization(t *testing.T)
```

**Test Objectives**:
- Field with hyphen: `status-code` → `status_code [json_name = "status-code"]`
- Field with dot: `user.name` → `user_name [json_name = "user.name"]`
- Field with space: `first name` → `first_name [json_name = "first name"]`
- Field with multiple invalid chars: `user--name..test` → `user_name_test`

```go
func TestConvertInvalidFieldNames(t *testing.T)
```

**Test Objectives**:
- Field starting with digit: `2ndValue` → error
- Field starting with underscore: `_private` → error
- Empty field name → error

**Context for Implementation**:
- These tests verify the new SanitizeFieldName behavior
- Update expected proto output strings
- Error test cases should check error message content

#### 2. Update Scalar Type Tests - ✅ COMPLETE
**File**: `internal/scalars_test.go`
**Changes**: Update field name expectations

**Pattern to change**:
- All fields currently expect snake_case
- Update to expect preserved names

**Example changes**:
- Line 70: `int32Field` → keep as `int32Field` (not `int32_field`)
- Similar updates for all camelCase field names in test cases

**Testing Requirements**:
None - updating existing tests

**Test Objectives**:
- All scalar type mappings still work correctly
- Field names preserved from OpenAPI
- json_name annotations still generated

**Context for Implementation**:
- Search for expected proto output in test cases
- Update field name expectations systematically
- Maintain json_name annotations in all expected output

#### 3. Update Name Conflict Tests - ✅ COMPLETE
**File**: `internal/naming_test.go`
**Changes**: Update TestConvertAlwaysIncludesJsonName

**Current behavior**:
- `userId` and `user_id` both convert to `user_id`
- Creates conflict: `user_id` and `user_id_2`

**New behavior**:
- `userId` stays as `userId`
- `user_id` stays as `user_id`
- No conflict! Each remains unique

**Test case update**:
```go
// Old expectation (lines 125-129):
string user_id = 1 [json_name = "userId"];
string user_id_2 = 2 [json_name = "user_id"];

// New expectation:
string userId = 1 [json_name = "userId"];
string user_id = 2 [json_name = "user_id"];
```

**Testing Requirements**:
Update the existing test case

**Test Objectives**:
- Demonstrate that name conflicts are reduced
- Show that different casings don't collide
- NameTracker still works for actual duplicates

**Context for Implementation**:
- This is a positive side effect of preservation
- Fewer conflicts means cleaner proto output
- Still need NameTracker for true duplicates

#### 4. Add New Edge Case Tests - ✅ COMPLETE
**File**: `internal/naming_test.go`
**Changes**: Add comprehensive edge case coverage

**New test cases**:

```go
func TestConvertMixedValidAndInvalidChars(t *testing.T)
```

**Test Objectives**:
- `user-ID` → `user_ID [json_name = "user-ID"]`
- `HTTP-Status` → `HTTP_Status [json_name = "HTTP-Status"]`
- `api.v2.endpoint` → `api_v2_endpoint [json_name = "api.v2.endpoint"]`

```go
func TestConvertConsecutiveInvalidChars(t *testing.T)
```

**Test Objectives**:
- `status---code` → `status_code` (collapse consecutive)
- `user...name` → `user_name` (collapse dots)
- `first  name` → `first_name` (collapse spaces)

```go
func TestConvertTrailingInvalidChars(t *testing.T)
```

**Test Objectives**:
- `status-` → `status` (trim trailing underscore)
- `user_` → `user_` (already valid, keep it)
- `name-_-` → `name` (sanitize and trim)

**Context for Implementation**:
- These tests ensure robust handling of edge cases
- Validate the sanitization logic thoroughly
- Ensure clean proto output

---

## Phase 3: Update AllOf Handling - ✅ COMPLETE (N/A)

### Overview
Update the `buildAllOfMergedMessage` function to use the new field name sanitization. This function has a separate code path for processing fields from allOf merged schemas.

**STATUS**: This phase is not applicable. The codebase intentionally does not support `allOf` per the original implementation plan. The library returns clear error messages when `allOf` is encountered (both at top-level and in properties). Since `allOf` schemas are rejected before any field processing occurs, there is no field name handling to update. The existing error tests verify proper error messages are returned.

### Acceptance Criteria:
- ✅ AllOf error handling verified - returns "uses 'allOf' which is not supported"
- ✅ Existing allOf tests pass (TestUnsupportedAllOf)
- ✅ No regressions in error handling

### Changes Required:

#### 1. Update AllOf Builder - ✅ N/A
**File**: `internal/builder.go`
**Changes**: Update field name handling in buildAllOfMergedMessage

**Current code** (line 256):
```go
protoFieldName := ctx.Tracker.UniqueName(ToSnakeCase(propName))
```

**New code**:
```go
sanitizedName, err := SanitizeFieldName(propName)
if err != nil {
    return nil, PropertyError(name, propName, err.Error())
}
protoFieldName := ctx.Tracker.UniqueName(sanitizedName)
```

**Function Responsibilities**:
- Same pattern as regular buildMessage
- Call SanitizeFieldName
- Handle errors with PropertyError
- Pass to UniqueName for conflict resolution

**Testing Requirements**:
```go
func TestConvertAllOfFieldNames(t *testing.T)
```

**Test Objectives** (via Convert()):
- AllOf schema with camelCase field → preserved in merged message
- AllOf schema with invalid characters → sanitized
- AllOf with field name conflicts → UniqueName resolution

**Context for Implementation**:
- Line 256 is the location to update
- Mirror the pattern from regular buildMessage
- Ensure consistent behavior across both code paths

#### 2. Update AllOf Tests - ✅ N/A
**File**: `internal/errors_test.go`
**Changes**: Verified allOf error tests

**Verification**:
- TestUnsupportedAllOf tests both top-level and property-level allOf usage
- Both test cases verify proper error message: "uses 'allOf' which is not supported"
- Tests pass successfully with current implementation
- No field name updates needed since allOf is not processed

---

## Phase 4: Update Documentation - ✅ COMPLETE

### Overview
Update all documentation to reflect the new field name preservation behavior. This includes README, godoc comments, and any supplementary documentation.

### Acceptance Criteria:
- ✅ README.md updated with new naming behavior
- ✅ README includes breaking change notice and migration guidance
- ✅ Godoc comments updated
- ✅ Examples show preserved field names
- ✅ Supplementary documentation updated

### Changes Required:

#### 1. Update README Naming Section - ✅ COMPLETE
**File**: `README.md`
**Changes**: Rewrite naming conventions section

**Current section** (lines 159-172):
```markdown
### Field Names: camelCase → snake_case

The library uses a simple letter-by-letter algorithm...
- `HTTPStatus` → `h_t_t_p_status` (not `http_status`)
```

**New section**:
```markdown
### Field Names: Preservation

The library preserves original OpenAPI field names when they're valid proto3 syntax:
- `HTTPStatus` → `HTTPStatus` (preserved)
- `userId` → `userId` (preserved)
- `user_id` → `user_id` (preserved)

Invalid characters are replaced with underscores:
- `status-code` → `status_code` (hyphen → underscore)
- `user.name` → `user_name` (dot → underscore)
- `first name` → `first_name` (space → underscore)

All fields include a `json_name` annotation to explicitly map to the original OpenAPI field name.

#### Proto3 Field Name Requirements

Field names must:
- Start with an ASCII letter (A-Z or a-z) - non-ASCII letters like `ñ` are not allowed
- Contain only ASCII letters, digits (0-9), and underscores (_)
- Field names starting with digits or underscores will cause errors
- Field names that are proto3 reserved keywords (like `message`, `enum`, `package`) will cause protoc compilation errors - the library does not detect or prevent these

**Note on Reserved Keywords:** Proto3 has reserved keywords like `message`, `enum`, `service`, `package`, `import`, `option`, etc. If your OpenAPI schema has field names that match these keywords, the generated proto file will fail to compile with protoc. This is intentional - the library lets protoc handle keyword validation rather than maintaining a keyword list that might change across proto versions.

#### Best Practices

While proto3 syntax allows mixed-case field names, the [Protocol Buffers style guide](https://protobuf.dev/programming-guides/style/) recommends snake_case for consistency across languages. If you control your OpenAPI schema, consider using snake_case field names to align with proto3 conventions.

#### BREAKING CHANGE Notice

**This represents a breaking change from previous library behavior.**

**Previous behavior:**
- Field names were converted to snake_case: `HTTPStatus` → `h_t_t_p_status`
- Simple letter-by-letter conversion with no acronym detection

**New behavior:**
- Field names are preserved when valid: `HTTPStatus` → `HTTPStatus`
- Only invalid characters are replaced: `status-code` → `status_code`

**Migration:**
If you have existing code that references generated proto field names, you will need to update those references. For example:
- Proto references: `message.h_t_t_p_status` → `message.HTTPStatus`
- Any tooling parsing .proto files needs adjustment for new field names

**Rationale:**
Preserving original names provides more intuitive mapping between OpenAPI and proto, respects your naming choices, and avoids surprising transformations like `HTTPStatus` → `h_t_t_p_status`.
```

**Testing Requirements**:
None - documentation only

**Context for Implementation**:
- Emphasize preservation as primary behavior
- Document sanitization as fallback for invalid chars
- Include comprehensive breaking change notice with migration steps
- Reference proto3 spec and style guide
- Explain both technical and usability rationale

#### 2. Update Godoc Comments - ✅ COMPLETE
**File**: `convert.go` and `internal/naming.go`
**Changes**: Update function documentation

**convert.go**:
```go
// Convert converts OpenAPI 3.0 schemas to Protocol Buffer 3 format.
//
// Field names are preserved from the OpenAPI schema when they meet proto3 syntax
// requirements. Invalid characters (hyphens, dots, spaces) are replaced with
// underscores. All fields include json_name annotations for correct JSON mapping.
//
// Examples:
//   - HTTPStatus → HTTPStatus [json_name = "HTTPStatus"]
//   - userId → userId [json_name = "userId"]
//   - status-code → status_code [json_name = "status-code"]
```

**internal/naming.go**:
```go
// SanitizeFieldName sanitizes an OpenAPI field name for proto3 syntax.
// Preserves the original name structure when valid, only modifying to meet
// proto3 requirements:
//   - Must start with a letter (A-Z, a-z)
//   - Can contain letters, digits, underscores
//   - Invalid characters replaced with underscores
//
// Returns error if name cannot be sanitized (e.g., starts with digit).
```

**Testing Requirements**:
None - documentation only

**Context for Implementation**:
- Update public API documentation
- Include examples in godoc
- Document error conditions
- Keep concise but informative

#### 3. Update Supplementary Docs - ✅ COMPLETE
**File**: `docs/scalar.md` and other docs files
**Changes**: Update field name examples

**Function Responsibilities**:
- Search for field name examples in docs/
- Update to show preserved names
- Update any snake_case references
- Ensure consistency across all documentation

**Testing Requirements**:
None - documentation only

**Context for Implementation**:
- Check all files in docs/ directory
- Update systematically
- Verify no outdated examples remain

**Completed Changes**:
- Updated README.md with comprehensive field name preservation section
- Added breaking change notice with migration guidance
- Updated convert.go godoc with examples
- SanitizeFieldName godoc already correct
- Updated docs/scalar.md with preserved field name examples
- Updated docs/objects.md naming reference
- All tests still pass

---

## Phase 5: Enum and Message Name Handling

### Overview
Verify that enum value names and message names continue to work correctly with the new field name handling. These should still use case conversion logic as they have different requirements than field names.

### Acceptance Criteria:
- Enum value names still use UPPERCASE_SNAKE_CASE
- Message names still use PascalCase
- Only field names use preservation logic
- Existing enum and message tests still pass
- No regression in enum/message naming

### Changes Required:

#### 1. Verify Enum Value Naming
**File**: `internal/naming.go`
**Changes**: No changes needed, verification only

**Function Responsibilities**:
- `ToEnumValueName` (line 84-89) should continue using ToSnakeCase
- Enum values: `(Status, in-progress)` → `STATUS_IN_PROGRESS`
- Enum prefix uses snake_case of enum name
- This behavior should NOT change

**Testing Requirements**:
```go
// Verify existing tests still pass
func TestConvertTopLevelEnum(t *testing.T)
func TestConvertEnumValueNaming(t *testing.T)
```

**Test Objectives**:
- Enum values still uppercase with underscores
- Enum name prefix still snake_cased
- No changes to enum value generation

**Context for Implementation**:
- Enums have different naming requirements than fields
- Keep existing ToSnakeCase usage for enums
- Only field names use new SanitizeFieldName

#### 2. Verify Message Name Handling
**File**: `internal/naming.go`
**Changes**: No changes needed, verification only

**Function Responsibilities**:
- `ToPascalCase` (line 36-80) should continue working
- Message names: `user_account` → `UserAccount`
- Property-to-message: `shippingAddress` → `ShippingAddress`
- This behavior should NOT change

**Testing Requirements**:
```go
// Verify existing tests still pass
func TestConvertNestedObject(t *testing.T)
func TestConvertMessageNaming(t *testing.T)
```

**Test Objectives**:
- Message names still PascalCase
- Nested message names derived correctly
- No changes to message name generation

**Context for Implementation**:
- Messages use PascalCase convention
- Different from field name handling
- Keep existing ToPascalCase logic

#### 3. Document Naming Distinctions
**File**: `README.md`
**Changes**: Clarify different naming rules

**Add section**:
```markdown
### Naming Conventions Summary

Different proto3 elements use different naming conventions:

**Field Names**: Preserved from OpenAPI
- `HTTPStatus` → `HTTPStatus`
- `userId` → `userId`
- Invalid chars replaced with underscores

**Message Names**: PascalCase
- `user_account` schema → `UserAccount` message
- `shipping-address` schema → `ShippingAddress` message

**Enum Values**: UPPERCASE_SNAKE_CASE with prefix
- `Status` enum, `active` value → `STATUS_ACTIVE`
- `Status` enum, `in-progress` value → `STATUS_IN_PROGRESS`

This follows proto3 conventions while preserving field names from your OpenAPI schema.
```

**Testing Requirements**:
None - documentation only

**Context for Implementation**:
- Clarify that only fields use preservation
- Messages and enums have different rules
- Help users understand the complete picture

---

## Phase 6: Integration Testing & Validation

### Overview
Run comprehensive integration tests to ensure the field name preservation works correctly across all features (nested objects, arrays, references, allOf, etc.). Verify no regressions.

### Acceptance Criteria:
- All existing tests pass with updated expectations
- New integration tests cover complex scenarios
- Coverage remains >80%
- No performance regressions
- make test, make lint, make coverage all pass

### Changes Required:

#### 1. Run Full Test Suite
**File**: All test files
**Changes**: Verify all tests pass

**Function Responsibilities**:
- Run `make test`
- Verify all updated tests pass
- Check for any unexpected failures
- Fix any issues found

**Testing Requirements**:
```bash
make test
```

**Test Objectives**:
- All unit tests pass
- All integration tests pass
- No test failures or errors
- Clean test output

**Context for Implementation**:
- This validates all previous phases
- Catch any missed test updates
- Ensure complete coverage

#### 2. Add Complex Integration Test
**File**: `internal/integration_test.go` (new or existing)
**Changes**: Add comprehensive test case

**New test**:
```go
func TestConvertComplexSchemaWithPreservedNames(t *testing.T)
```

**Test Input** (OpenAPI YAML):
```yaml
openapi: 3.0.0
info:
  title: Complex Integration Test API
  version: 1.0.0
paths: {}
components:
  schemas:
    APIResponse:
      type: object
      properties:
        HTTPStatus:
          type: integer
          description: HTTP status code
        userId:
          type: string
        status-code:
          type: string
          description: Custom status with hyphen
        api.version:
          type: string
        nested:
          type: object
          properties:
            innerField:
              type: string
            inner-hyphen:
              type: string
        tagList:
          type: array
          items:
            $ref: '#/components/schemas/Tag'
    Tag:
      type: object
      properties:
        tagName:
          type: string
        tag-id:
          type: integer
```

**Expected Output** (Proto3):
```protobuf
syntax = "proto3";

package testpkg;

message APIResponse {
  message Nested {
    string innerField = 1 [json_name = "innerField"];
    string inner_hyphen = 2 [json_name = "inner-hyphen"];
  }

  // HTTP status code
  int32 HTTPStatus = 1 [json_name = "HTTPStatus"];
  string userId = 2 [json_name = "userId"];
  // Custom status with hyphen
  string status_code = 3 [json_name = "status-code"];
  string api_version = 4 [json_name = "api.version"];
  Nested nested = 5 [json_name = "nested"];
  repeated Tag tagList = 6 [json_name = "tagList"];
}

message Tag {
  string tagName = 1 [json_name = "tagName"];
  int32 tag_id = 2 [json_name = "tag-id"];
}
```

**Test Objectives**:
- HTTPStatus preserved (mixed case)
- userId preserved (camelCase)
- status-code → status_code (hyphen sanitized)
- api.version → api_version (dot sanitized)
- Nested object with both preserved and sanitized names
- Array of references with preserved field names
- All json_name annotations correct
- Field numbering correct
- Comments preserved

**Context for Implementation**:
- This test validates complete end-to-end behavior
- Covers all major features with field name preservation
- Ensures no regressions across the codebase
- Provides concrete example of expected behavior

#### 3. Verify Coverage
**File**: N/A
**Changes**: Check test coverage

**Function Responsibilities**:
- Run `make coverage`
- Verify >80% coverage maintained
- Identify any uncovered code paths
- Add tests if coverage dropped

**Testing Requirements**:
```bash
make coverage
```

**Test Objectives**:
- Coverage >80% overall
- New SanitizeFieldName function well covered
- All error paths tested
- No significant coverage drops

**Context for Implementation**:
- Coverage requirement from original plan
- Ensure quality standards maintained
- Guide additional test creation if needed

#### 4. Run Linting
**File**: N/A
**Changes**: Verify code quality

**Function Responsibilities**:
- Run `make lint`
- Fix any linting errors
- Ensure clean code

**Testing Requirements**:
```bash
make lint
```

**Test Objectives**:
- No linting errors
- Code style consistent
- No new warnings

**Context for Implementation**:
- Maintain code quality standards
- Clean up any issues introduced
- Follow project conventions

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

**Breaking Change Management**:
- This is an intentional breaking change
- Document in README with clear "BREAKING CHANGE" notice
- Include before/after examples and migration steps
- No CHANGELOG.md (not part of this project)
- Update all test expectations to match new behavior

**Design Decisions Made**:
1. **Invalid leading digits**: Error with message (don't auto-prefix with 'x_' or 'field_')
2. **Unicode characters**: Reject non-ASCII letters (ASCII-only for protoc compatibility)
3. **Consecutive underscores in original**: Preserve as-is (e.g., `user__id` stays `user__id`)
4. **Consecutive underscores from sanitization**: Collapse to single (e.g., `status--code` → `status_code`)
5. **Breaking change compatibility**: No configuration option, just make the change
6. **json_name**: Always include for all fields (remove NeedsJSONName function)

**Testing Strategy**:
- Update existing tests systematically
- Add comprehensive edge case tests (empty after sanitization, consecutive chars, etc.)
- Add concrete integration test with full OpenAPI YAML and expected proto output
- Ensure comprehensive integration testing
- Maintain >80% coverage requirement

**Error Handling**:
- Clear errors for fields starting with digits: "field name must start with a letter, got '<name>'"
- Clear errors for fields starting with underscores: "field name cannot start with underscore, got '<name>'"
- Clear errors for non-ASCII: "field name must start with a letter, got '<name>'"
- Error for empty after sanitization: "field name contains no valid characters"
- Contextual errors with schema and property names using PropertyError pattern

**Edge Cases Addressed**:
- Consecutive invalid characters → collapse to single underscore (algorithm: don't write '_' if last char was '_')
- Trailing underscore from sanitization → trim if original didn't end with valid char
- Underscores in original field → preserve (e.g., `user__id` → `user__id`)
- Field names that are proto3 keywords → let protoc catch them (document in README)
- Unicode/non-ASCII characters → reject with error (ASCII-only validation)
- Empty string or only invalid chars → error
- Very long field names → preserve if valid (no length limit)

**Performance Considerations**:
- SanitizeFieldName uses single-pass algorithm for efficiency
- Uses strings.Builder for efficient string construction
- No regex needed (manual character iteration)
- Name conflict resolution (NameTracker) unchanged
- No performance degradation expected
- Actually improves performance: fewer name collisions means less NameTracker work

**Code Organization**:
- Keep ToSnakeCase for enum value naming (still needed for ENUM_VALUE_NAME format)
- Add SanitizeFieldName as internal function (not exported)
- Remove NeedsJSONName function (no longer needed)
- Update builder.go in two locations: buildMessage (line 117) and buildAllOfMergedMessage (line 256)
- Maintain clean separation of concerns

**Documentation Standards**:
- Clear examples in README with both preservation and sanitization cases
- Complete godoc comments with examples
- Breaking change notice directly in README (no CHANGELOG.md)
- Consistent terminology throughout
- Document proto3 reserved keyword limitation

## Reference Materials

**Proto3 Specification**:
- Field identifier grammar: `ident = letter { letter | decimalDigit | "_" }`
- Valid characters: letters (A-Z, a-z), digits (0-9), underscores (_)
- Must start with letter

**Proto3 Style Guide**:
- Recommends snake_case for fields (not required)
- Treats abbreviations as words: `DnsRequest` not `DNSRequest`
- Our approach prioritizes user intent over style guide

**Current Implementation**:
- internal/naming.go - naming conversion functions
- internal/builder.go:117 - field name conversion for properties
- internal/builder.go:256 - field name conversion for allOf
- internal/naming_test.go - existing naming tests

**Dependencies**:
- No new dependencies required
- Uses existing unicode package for character validation
- Uses existing string manipulation from standard library
