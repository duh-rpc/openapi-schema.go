> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# OpenAPI Example Validation Implementation Plan

## Review Status
- Review Cycles Completed: 2
- Final Approval: Pending
- Outstanding Concerns: None (all critical issues from review #1 have been addressed)
- Updates: Added concrete API usage, version detection, example extraction, and corrected options logic

## 1. Overview

Add a `ValidateExamples()` function to validate JSON examples defined in OpenAPI specifications against their corresponding schemas. This complements the existing `ConvertToExamples()` generation function by providing validation capabilities to ensure examples in OpenAPI specs are schema-compliant.

**Business Value:**
- Catch invalid examples in OpenAPI specs before they reach production
- Ensure API documentation examples are accurate and trustworthy
- Provide tooling for OpenAPI spec quality checks in CI/CD pipelines
- Support API design validation workflows

## 2. Current State Analysis

### Affected Modules
- `convert.go` - Public API surface (lines 1-390)
- `internal/parser/parser.go` - OpenAPI parsing infrastructure (lines 1-63)
- `go.mod` - Dependency management

### Current Behavior
- **Example Generation**: `ConvertToExamples()` generates JSON examples from schemas (convert.go:349-389)
- **Schema Parsing**: OpenAPI documents parsed via `parser.ParseDocument()` using pb33f/libopenapi v0.28.2
- **Schema Access**: Schemas extracted via `doc.Schemas()` from components/schemas
- **No Validation**: Currently no capability to validate existing examples against schemas

### Relevant ADRs Reviewed
No ADRs found in this project (checked `**/adr/**`, `**/decisions/**`, `**/.adr/**`).

### Technical Debt Identified
- No existing validation infrastructure
- No version detection for OpenAPI 3.0 vs 3.1/3.2 distinction
- Schema examples in inline definitions (parameters, request bodies, responses) not currently accessed (will be addressed in Phase 2)

## 3. Architectural Context

### Relevant ADRs
None (project does not use ADRs).

### Architectural Principles
Based on codebase analysis:
1. **Functional Testing**: All tests use public API, never call internal functions directly
2. **Phased Implementation**: Features delivered incrementally with validation at each phase
3. **Reuse Existing Patterns**: Follow established patterns from `Convert()` and `ConvertToExamples()`
4. **pb33f Ecosystem**: Already using pb33f/libopenapi for parsing

## 4. Requirements

### Functional Requirements

**REQ-001: Validate Schema Examples in components/schemas (Phase 1)**
- Phase 1: Function must validate `example` and `examples` fields in Schema Objects under `components/schemas`
- Phase 2: Extend to inline schemas in parameters, request bodies, and responses
- For schemas with `examples` map: validate ALL entries in the map
- If both `example` and `examples` exist on same schema: validate both
- Acceptance: Given OpenAPI spec with schema.example, validate returns pass/fail for each example

**REQ-002: OpenAPI Version Support**
- Primary support: OpenAPI 3.1 and 3.2 (JSON Schema compliant)
- For OpenAPI 3.0: Include warning in ValidationResult but attempt best-effort validation
- Return error if document is not OpenAPI 3.x
- Acceptance: Function detects version and adjusts validation behavior accordingly

**REQ-003: Validation Result Structure**
- Return `ValidationResult` struct with:
  - `Schemas map[string]*SchemaValidationResult` - Per-schema validation results
- `SchemaValidationResult` contains:
  - `SchemaPath string` - Schema identifier
  - `HasExamples bool` - Whether schema had any examples to validate
  - `Valid bool` - Overall validation status (meaningful only if HasExamples is true)
  - `Issues []ValidationIssue` - All validation issues (errors and warnings)
- `ValidationIssue` contains:
  - `Severity IssueSeverity` - "error" or "warning"
  - `ExampleField string` - Which example field (e.g., "example" or "examples.successCase")
  - `Message string` - Validation failure message
  - `Line int` - Line number in OpenAPI file (0 if unavailable, best effort)
- Acceptance: Result struct provides complete validation report with errors and warnings

**REQ-004: Missing Example Handling**
- Track schemas without examples via `HasExamples: false` in SchemaValidationResult
- Not an error condition, informational only
- All schemas checked are included in result (with or without examples)
- Acceptance: Result shows which schemas had no examples to validate via HasExamples field

**REQ-005: Validation Options**
- Accept `ValidateOptions` struct with:
  - `SchemaNames []string` - Filter to specific schemas (optional)
  - `IncludeAll bool` - Validate all schemas (default behavior if SchemaNames is empty)
- **Priority Logic** (follows existing `ConvertToExamples` pattern):
  - If `IncludeAll` is true: validate all schemas (ignore SchemaNames even if provided)
  - If `IncludeAll` is false and `SchemaNames` is non-empty: validate only those schemas
  - If `IncludeAll` is false and `SchemaNames` is empty: return error "must specify SchemaNames or set IncludeAll"
- Implementation: `if opts.IncludeAll { schemaNames = nil }` (same pattern as ConvertToExamples line 377-379)
- Acceptance: Can validate all schemas or filter to specific subset, IncludeAll takes precedence if both provided

**REQ-006: JSON Schema Validation**
- Use pb33f/libopenapi-validator v0.9.2 (latest stable as of Nov 2024) for schema validation
- Leverage version-aware validation (handles 3.0 vs 3.1 differences automatically)
- Validate example data against resolved JSON Schema (with $ref resolution)
- Collect ALL validation errors per schema (do not stop at first error)
- Acceptance: Validation uses industry-standard JSON Schema validator and reports all issues

**REQ-007: OpenAPI 3.0 Warning Handling**
- For OpenAPI 3.0 specs, include warning in ValidationResult
- Warning added as `ValidationIssue` with `Severity: IssueSeverityWarning`
- Warning message: "OpenAPI 3.0 detected: validation may have limitations due to JSON Schema divergence. OpenAPI 3.1+ recommended for full JSON Schema compliance."
- Validation still attempted (best effort)
- Acceptance: OpenAPI 3.0 specs include warning in result but validation proceeds

### Non-Functional Requirements

**Performance**
- Validation should complete in reasonable time for large specs
- No specific latency target (not real-time critical)
- Should handle specs with 100+ schemas without issue
- Collect all errors (comprehensive report) with no artificial limits

**Security**
- No security-critical operations (read-only validation)
- Standard input validation (empty checks, nil checks)

**Scalability**
- Function is synchronous (called per-spec, not per-request)
- No concurrent validation required in Phase 1

## 5. Technical Approach

### Chosen Solution

Add `ValidateExamples()` function following established patterns from `Convert()` and `ConvertToExamples()`:

1. **Public API** in `convert.go`:
   ```go
   // ValidationResult contains the validation status for all examples in an OpenAPI spec
   type ValidationResult struct {
       Schemas map[string]*SchemaValidationResult
   }

   // SchemaValidationResult contains validation details for a single schema
   type SchemaValidationResult struct {
       SchemaPath   string
       HasExamples  bool
       Valid        bool
       Issues       []ValidationIssue
   }

   // ValidationIssue represents a single validation error or warning
   type ValidationIssue struct {
       Severity     IssueSeverity
       ExampleField string
       Message      string
       Line         int
   }

   // IssueSeverity indicates whether an issue is an error or warning
   type IssueSeverity string

   const (
       IssueSeverityError   IssueSeverity = "error"
       IssueSeverityWarning IssueSeverity = "warning"
   )

   // ValidateOptions configures example validation
   type ValidateOptions struct {
       SchemaNames []string
       IncludeAll  bool
   }

   // ValidateExamples validates examples in OpenAPI spec against schemas
   func ValidateExamples(openapi []byte, opts ValidateOptions) (*ValidationResult, error)
   ```

2. **Validation Engine** in `internal/examplevalidator.go`:

   **Required imports:**
   ```go
   import (
       "encoding/json"
       "strings"
       "github.com/pb33f/libopenapi"
       "github.com/pb33f/libopenapi-validator/schema_validation"
       "github.com/duh-rpc/openapi-proto.go/internal/parser"
       "gopkg.in/yaml.v3"
   )
   ```

   **Implementation steps:**

   a. **Version Detection:**
   ```go
   // Parse raw document for version detection
   document, err := libopenapi.NewDocument(openapi)
   if err != nil {
       return nil, err
   }
   version := document.GetVersion() // Returns "3.0.0", "3.1.0", "3.2.0", etc.
   isOpenAPI30 := strings.HasPrefix(version, "3.0")
   ```

   b. **Parse schemas via existing parser:**
   ```go
   parsedDoc, err := parser.ParseDocument(openapi)
   if err != nil {
       return nil, err
   }
   schemas, err := parsedDoc.Schemas()
   ```

   c. **Create validator:**
   ```go
   validator := schema_validation.NewSchemaValidator()
   ```

   d. **For each schema, extract and validate examples:**
   ```go
   for _, schemaEntry := range schemas {
       schema := schemaEntry.Proxy.Schema()
       schemaName := schemaEntry.Name

       // Extract example field (singular)
       if schema.Example != nil {
           // Convert yaml.Node to JSON
           var exampleData interface{}
           err := schema.Example.Decode(&exampleData)
           if err == nil {
               exampleJSON, _ := json.Marshal(exampleData)

               // Validate using version-aware method
               var valid bool
               var errors []*schema_validation.ValidationError
               if isOpenAPI30 {
                   valid, errors = validator.ValidateSchemaStringWithVersion(
                       schema, string(exampleJSON), 3.0)
               } else {
                   valid, errors = validator.ValidateSchemaString(
                       schema, string(exampleJSON))
               }

               // Collect errors into Issues slice
           }
       }

       // Extract examples field (plural - OpenAPI 3.1+)
       if schema.Examples != nil {
           for _, exampleNode := range schema.Examples {
               // Same validation process as above
               // ExampleField = "examples.{index}"
           }
       }
   }
   ```

   e. **Line number extraction (best effort):**
   ```go
   // ValidationError from pb33f/libopenapi-validator contains:
   // - Message string
   // - Reason string
   // - SchemaLocation string
   // Line numbers may be available from SchemaLocation or context
   // If unavailable, set Line: 0
   ```

   f. **OpenAPI 3.0 warning injection:**
   ```go
   if isOpenAPI30 {
       // Add warning to first schema result or create synthetic entry
       warning := ValidationIssue{
           Severity:     IssueSeverityWarning,
           ExampleField: "",
           Message:      "OpenAPI 3.0 detected: validation may have limitations...",
           Line:         0,
       }
   }
   ```

3. **Dependency Addition**:
   - Add `github.com/pb33f/libopenapi-validator v0.9.2` to go.mod

### Rationale

**Why pb33f/libopenapi-validator:**
- Already using pb33f/libopenapi for parsing (ecosystem consistency)
- Handles OpenAPI 3.0 vs 3.1 validation differences automatically
- Uses santhosh-tekuri/jsonschema under the hood (mature, well-tested)
- Supports JSON Schema draft 2020-12 (aligns with OpenAPI 3.1+)
- Latest stable version: v0.9.2 (released Nov 2024)

**Why structured ValidationResult:**
- Follows existing ConvertResult pattern (consistency)
- Provides detailed error context for tooling and CI/CD integration
- Allows users to programmatically process validation failures
- Combines errors and warnings in single structure with clear distinction

**Why ValidateOptions with mutual exclusion:**
- Mirrors ExampleOptions pattern from ConvertToExamples()
- Prevents ambiguous configuration (can't filter and include all simultaneously)
- Clear error messaging for misconfiguration
- Future-proof for additional options

**Why collect all errors:**
- Users see complete picture of validation failures in one pass
- More efficient for large specs (no need to fix-and-rerun repeatedly)
- Better CI/CD integration (single validation run shows all issues)

### ADR Alignment
Not applicable (project does not use ADRs). Decision captured in this specification.

### Component Changes

**convert.go**:
- Add `ValidationResult`, `SchemaValidationResult`, `ValidationIssue`, `IssueSeverity` types
- Add `ValidateOptions` struct with SchemaNames and IncludeAll fields
- Add public `ValidateExamples()` function with:
  - Input validation (empty openapi check)
  - Options validation (SchemaNames and IncludeAll mutual exclusion)
  - OpenAPI parsing via `parser.ParseDocument()`
  - Delegation to internal validator

**internal/examplevalidator.go** (NEW):
- Create validation engine using pb33f/libopenapi-validator
- Extract examples from Schema Objects in components/schemas
- Validate each example (both `example` and `examples` entries) against schema
- Collect all validation errors per schema
- Detect OpenAPI version and add warning for 3.0
- Extract line numbers (best effort)
- Structure results into ValidationResult

**go.mod**:
- Add `github.com/pb33f/libopenapi-validator v0.9.2`

**Tests**:
- New file: `convert_validate_test.go` with functional tests for ValidateExamples()
- Test naming pattern: `TestValidateExamples*`

## 6. Dependencies and Impacts

### External Dependencies
- **New**: `github.com/pb33f/libopenapi-validator v0.9.2`
  - Transitively pulls in `github.com/santhosh-tekuri/jsonschema`
- **Existing**: `github.com/pb33f/libopenapi` v0.28.2 (no change)

### Internal Dependencies
- Reuses `parser.ParseDocument()` from `internal/parser/parser.go`
- Reuses `Document` and `SchemaEntry` types from parser package
- No dependencies on example generation code (independent feature)

### Database Impacts
None (no persistence layer).

## 7. Backward Compatibility

### Is this project in production?
- [x] Yes - Must maintain backward compatibility

### Breaking Changes Allowed
- [x] No - Must maintain backward compatibility

### Backward Compatibility Strategy

**This is a purely additive change:**
- New public function `ValidateExamples()` - does not affect existing APIs
- New struct types `ValidationResult`, `SchemaValidationResult`, `ValidationIssue`, `IssueSeverity`, `ValidateOptions` - additive
- New internal file `internal/examplevalidator.go` - no impact on existing internals
- New dependency `libopenapi-validator` v0.9.2 - does not conflict with existing dependencies

**No breaking changes:**
- Existing `Convert()`, `ConvertToStruct()`, and `ConvertToExamples()` remain unchanged
- Existing test suite remains unchanged (validation tests are additive)
- Existing go.mod dependencies remain at same versions
- No changes to existing struct fields or function signatures

## 8. Testing Strategy

### Unit Testing Approach
- All tests use public `ValidateExamples()` API (functional testing per CLAUDE.md)
- No direct testing of internal functions
- Table-driven tests with OpenAPI YAML input and expected validation results
- Test file: `convert_validate_test.go`

### Test Scenarios

**Phase 1 (components/schemas only):**
1. **Valid examples**: Schema with valid example returns success (`HasExamples: true, Valid: true, Issues: []`)
2. **Invalid examples**: Schema with type mismatch returns validation error
3. **Missing examples**: Schema without examples shows `HasExamples: false`
4. **Multiple examples**: Schema with `examples` map validates each entry
5. **Both example and examples**: Schema with both fields validates both
6. **Enum validation**: Example value not in enum returns error
7. **Constraint validation**: Example violating min/max, minLength/maxLength returns error
8. **Circular references**: Schemas with $ref cycles handled gracefully by validator
9. **OpenAPI 3.0**: Warning included in result, validation attempted
10. **OpenAPI 3.1+**: Full JSON Schema validation without warnings
11. **SchemaNames filtering**: Only specified schemas validated and reported (IncludeAll: false)
12. **IncludeAll priority**: IncludeAll: true validates all schemas even if SchemaNames is provided (IncludeAll takes precedence)
13. **Empty options error**: Neither SchemaNames nor IncludeAll returns error "must specify SchemaNames or set IncludeAll"
14. **Line numbers**: Best effort extraction (check that Line field exists, 0 acceptable)
15. **All errors collected**: Multiple validation errors per schema all included in Issues

### Integration Testing
- Test with real-world OpenAPI specs (PetStore example)
- Test with specs containing 50+ schemas (performance check)
- Test with deeply nested schemas ($ref resolution)

### User Acceptance Criteria
1. Given valid OpenAPI 3.1 spec with correct examples, ValidateExamples() returns all schemas with `Valid: true`
2. Given spec with invalid example (wrong type), ValidationResult shows specific error with schema path and message
3. Given spec with no examples, ValidationResult shows `HasExamples: false` for those schemas
4. Given SchemaNames filter, only specified schemas are validated and reported
5. Given OpenAPI 3.0 spec, warning appears in result and validation proceeds

## 9. Implementation Notes

### Estimated Complexity
**Medium** - Requires:
- New dependency integration (pb33f/libopenapi-validator)
- Version detection logic
- Error collection and structuring
- Options validation (mutual exclusion)

### Suggested Implementation Order

**Phase 1: Core Infrastructure and Components/Schemas Validation**
- Add ValidationResult, SchemaValidationResult, ValidationIssue, IssueSeverity, ValidateOptions types to convert.go
- Implement ValidateExamples() with basic input validation and options validation
- Add pb33f/libopenapi-validator v0.9.2 dependency to go.mod
- Implement internal/examplevalidator.go with:
  - Version detection (3.0 vs 3.1/3.2)
  - Example extraction from components/schemas schemas
  - Validation using pb33f/libopenapi-validator
  - Error collection (all errors per schema)
  - OpenAPI 3.0 warning injection
  - Line number extraction (best effort)
- Comprehensive functional tests in convert_validate_test.go
- Validation: All tests pass, handles all test scenarios

**Phase 2: Inline Schema Support** (Future work)
- Extend to validate examples in parameters (parameter.example, parameter.examples)
- Extend to validate examples in request/response bodies (mediaType.example, mediaType.examples)
- Handle inline schemas without explicit names (generate path identifiers)
- Update tests for inline schema validation

**Phase 3: Documentation**
- Update README.md with ValidateExamples() usage section
- Create docs/validation.md with detailed documentation
- Add code examples showing validation result processing
- Document OpenAPI 3.0 limitations and recommendations

### Code Style Considerations
- Follow existing patterns from convert.go (input validation, error handling)
- Use functional testing exclusively (per CLAUDE.md)
- Use `require` for critical assertions (nil checks, error checks), `assert` for non-critical (per CLAUDE.md)
- Avoid abbreviations in variable names: `validationResult` not `valResult` (per CLAUDE.md)
- Use `const` for values that don't change (per CLAUDE.md)
- Visual tapering for struct field ordering (per CLAUDE.md)
- Table-driven tests with `for _, test := range []struct` pattern

### Rollback Strategy
- Pure additive feature - simply don't call ValidateExamples()
- If critical issues found, can deprecate function in next release
- No data migration or state to roll back
- Dependency can be removed if validation feature is removed

## 10. ADR Recommendation

Not applicable - user indicated this change does not warrant an ADR.

## 11. Open Questions

None - all requirements clarified through user responses.

---

## Summary of User Decisions

All review questions resolved:
1. ✅ Phase 2 for inline schema validation (Phase 1: components/schemas only)
2. ✅ IncludeAll takes precedence over SchemaNames (follows existing ConvertToExamples pattern)
3. ✅ Errors and warnings combined in Issues array with IssueSeverity enum
4. ✅ Collect all errors per schema (comprehensive reporting)
5. ✅ Line numbers best effort (0 if unavailable)
6. ✅ ValidationResult structure finalized and approved
7. ✅ pb33f/libopenapi-validator v0.9.2 dependency confirmed
8. ✅ Concrete API usage and implementation details added to specification
