> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.

# Functional Test Coverage Expansion Implementation Plan

## Overview

This plan addresses critical testing gaps in the openapi-proto.go codebase by adding comprehensive functional tests for:
1. **Go code generation** - All scalar type mappings, array types, nested structures through the public API
2. **Dependency graph scenarios** - Complex type classification patterns including diamond dependencies and deep transitive closures
3. **Union/oneOf edge cases** - Discriminator validation, multiple variants, nested unions, and error conditions

All tests will follow the functional testing pattern established in the project, using the public `conv.Convert()` interface exclusively.

## Current State Analysis

### Existing Test Coverage
- **Strong areas**: Proto3 conversion pipeline (~1,408 lines in convert_test.go), scalar types (186 lines), arrays (437 lines), enums (326 lines), naming (634 lines)
- **Test pattern**: Table-driven tests with `conv.Convert()` public API, assertion with testify/require and testify/assert
- **Validation**: `go test ./...` (Makefile:4), compilation tests for generated Go code

### Coverage Gaps Identified
1. **Go Code Generation**: Existing tests validate Go output (convert_test.go:914-1349) but don't systematically cover all scalar format mappings (int8, uint8, uint16, uint32, uint64, float32). Missing: dedicated tests for array types in Go output, nested objects in Go structs
2. **Dependency Graph**: Limited testing beyond basic transitive closure (convert_test.go:851-912). Missing: diamond dependencies, complex reference patterns
3. **Union/OneOf**: 9 existing union tests in convert_test.go cover basic scenarios, but missing edge cases for 3+ variants, nested unions, discriminator conflicts, missing discriminator properties, arrays of unions

### Key Discoveries
- **Pattern reference**: convert_test.go:27-100 shows table-driven test structure with `conv.Convert()`
- **Go compilation tests**: convert_test.go:1200-1240 demonstrates compilation validation pattern
- **JSON round-trip tests**: convert_test.go:1242-1340 shows runtime validation approach
- **Type classification**: DependencyGraph.ComputeTransitiveClosure() (dependencies.go:53-119) implements BFS-based type classification
- **Go type mapping**: mapGoScalarType() (golang.go:284-340) contains all scalar mappings including int8, uint8, uint16, uint32, uint64

## Desired End State

### Success Criteria
1. Comprehensive test coverage for all Go scalar type mappings through functional tests
2. Complex dependency graph scenarios validated (diamond, deep transitive, cyclic references)
3. All union/oneOf edge cases tested with clear error messages
4. All tests pass with `go test ./...`
5. Generated Go code compiles and executes correctly for all scenarios
6. Test coverage report shows improvement in golang.go, dependencies.go coverage

### Verification Commands
```bash
make test           # Run all tests
make coverage       # Generate coverage report
go test -v ./...    # Verbose test output
```

## What We're NOT Doing

- NOT modifying implementation code (focus is purely on tests)
- NOT testing internal functions directly (only public `conv.Convert()` API)
- NOT creating unit tests for internal packages (maintaining functional test approach)
- NOT adding tests for features explicitly marked as unsupported (allOf, anyOf, not, oneOf without discriminator)
- NOT performance/benchmarking tests (out of scope)

## Implementation Approach

Use **Test-Driven approach** with functional tests that:
1. Define OpenAPI schemas with specific type combinations
2. Call `conv.Convert()` with appropriate options
3. Assert both Proto3 and Go output correctness
4. Validate generated Go code compiles and runs (where applicable)
5. Follow existing test patterns from convert_test.go, scalars_test.go, arrays_test.go

Each phase delivers working, passing tests that can be validated independently.

---

## Phase 1: Go Code Generation - Scalar Types

- [x] Implemented `internal/golang_scalars_test.go` with 7 test functions
- [x] All tests passing with `go test ./internal -run TestGo`
- [x] Full test suite still passing with `make test`

### Overview
Add comprehensive functional tests for all Go scalar type mappings including integer variants (int8, int16, uint8, uint16, uint32, uint64), float32, and special string formats (email, uuid, password). Tests validate that OpenAPI schemas with these types generate correct Go code through the public API.

### Changes Required

#### 1. New Test File: Go Scalar Type Mappings
**File**: `internal/golang_scalars_test.go` ✅
**Purpose**: Test all Go scalar type mappings end-to-end through `conv.Convert()`

```go
package internal_test

// TestGoScalarTypeMappings validates all scalar type conversions to Go types
func TestGoScalarTypeMappings(t *testing.T)

// TestGoIntegerFormats validates integer format mappings (int8, int16, int32, int64, uint variants)
func TestGoIntegerFormats(t *testing.T)

// TestGoNumberFormats validates number format mappings (float32, float64/double)
func TestGoNumberFormats(t *testing.T)

// TestGoStringFormats validates string format mappings (email, uuid, password, date, date-time)
func TestGoStringFormats(t *testing.T)

// TestGoArrayTypes validates array type generation in Go ([]int32, []string, []*CustomType)
func TestGoArrayTypes(t *testing.T)

// TestGoTimestampGeneration validates time.Time generation and import
func TestGoTimestampGeneration(t *testing.T)
```

**Test Objectives:**
- Verify `mapGoScalarType()` correctly maps all integer formats: int8→int8, int16→int16, uint8→uint8, uint16→uint16, uint32→uint32, uint64→uint64
- Verify default integer format (no format specified) maps to int32
- Verify number formats: float→float32, double→float64, default→float64
- Verify string formats: date→time.Time, date-time→time.Time, byte→[]byte, binary→[]byte
- Verify special string formats (email, uuid, password) map to string
- Verify array types generate correct Go slice syntax: []int32, []string, []*Message
- Verify time.Time usage triggers import of "time" package in Go output
- Verify generated Go output compiles successfully

**Context for Implementation:**
- Follow test pattern from `internal/scalars_test.go:11-186`
- Reference Go type mapping logic in `internal/golang.go:234-340`
- Use table-driven tests with OpenAPI YAML in `given` field, expected Go output snippets
- Assert both `result.Golang` content and compilation success
- Validation: `go test ./internal -run TestGo`

---

## Phase 2: Go Code Generation - Complex Structures

- [x] Implemented `internal/golang_structures_test.go` with 6 test functions
- [x] All tests passing with `go test ./internal -run TestGoNested\|TestGoMixed\|TestGoPackage\|TestGoField\|TestGoPointer\|TestGoMultiple`
- [x] Full test suite still passing with `make test`
- [x] Phase 2 completed and verified

### Overview
Test Go code generation for complex structures including nested objects, inline enums, mixed field types, and package name extraction from paths. Validates structural correctness of generated Go code.

### Changes Required

#### 1. New Test File: Go Complex Structures
**File**: `internal/golang_structures_test.go` ✅
**Purpose**: Test Go struct generation for nested objects, inline enums, and complex field combinations

```go
package internal_test

// TestGoNestedObjects validates nested object generation as separate Go structs
func TestGoNestedObjects(t *testing.T)

// TestGoInlineEnums validates inline enum handling in Go structs
func TestGoInlineEnums(t *testing.T)

// TestGoMixedFields validates structs with scalars, objects, arrays, enums mixed
func TestGoMixedFields(t *testing.T)

// TestGoPackageNameExtraction validates package name extraction from GoPackagePath
func TestGoPackageNameExtraction(t *testing.T)

// TestGoFieldJSONTags validates JSON tags match OpenAPI property names
func TestGoFieldJSONTags(t *testing.T)

// TestGoPointerFields validates pointer usage for objects and optional fields
func TestGoPointerFields(t *testing.T)
```

**Test Objectives:**
- Verify nested objects generate separate struct definitions with pointer fields
- Verify inline enums in properties are hoisted to top-level enum types
- Verify mixed field types (scalars, refs, arrays, nested) generate correct struct
- Verify package name extraction: "github.com/example/types/v1" → "package v1"
- Verify JSON tags preserve original OpenAPI names: `Field string \`json:"field"\``
- Verify object references and nested objects use pointer types: `*Message`
- Verify generated structs compile and can be instantiated

**Context for Implementation:**
- Follow pattern from `internal/nested_test.go:1-401` for nested objects
- Reference `buildGoStruct()` in `internal/golang.go:71-174`
- Reference `goType()` for type resolution in `internal/golang.go:234-281`
- Use compilation tests like `convert_test.go:1200-1240`
- Test package extraction with various paths: v1, v2, types, api
- Validation: `go test ./internal -run TestGo`

---

## Phase 3: Dependency Graph - Diamond Dependencies

- [x] Implemented `internal/dependencies_test.go` with 5 test functions
- [x] All tests passing with `go test ./internal -run TestDependencyGraph`
- [x] Full test suite still passing with `make test`
- [x] Phase 3 completed and verified

### Overview
Test complex dependency scenarios where multiple types reference the same union type, creating diamond-shaped dependency graphs. Validates transitive closure computation correctly marks all referencing types as Go-only.

### Changes Required

#### 1. New Test File: Dependency Graph Scenarios
**File**: `internal/dependencies_test.go` ✅
**Purpose**: Test complex type classification scenarios through dependency graph

```go
package internal_test

// TestDependencyGraphDiamond validates diamond dependency pattern classification
func TestDependencyGraphDiamond(t *testing.T)

// TestDependencyGraphDeepTransitive validates deep transitive closure (A→B→C→D→Union)
func TestDependencyGraphDeepTransitive(t *testing.T)

// TestDependencyGraphMultipleUnions validates schemas referencing multiple union types
func TestDependencyGraphMultipleUnions(t *testing.T)

// TestDependencyGraphOrphanedTypes validates types with no dependencies stay proto-only
func TestDependencyGraphOrphanedTypes(t *testing.T)

// TestDependencyGraphSiblingReferences validates siblings of union variants
func TestDependencyGraphSiblingReferences(t *testing.T)
```

**Test Objectives:**
- Verify diamond pattern: Two paths to same union type (Owner→Pet, Home→Pet where Pet is union) marks both Owner and Home as Go-only
- Verify deep chains: A→B→C→D where D is union marks all A,B,C,D as Go
- Verify multiple unions: Schema references Union1 and Union2, correctly marked for both
- Verify orphaned schemas (no refs to unions) remain proto-only in TypeMap
- Verify sibling types that don't reference union stay proto-only
- Verify TypeMap.Reason explains classification: "references union type X"
- Verify generated output: proto-only types in .proto, Go-only types in .go

**Context for Implementation:**
- Extend pattern from `convert_test.go:800-912` for TypeMap validation
- Reference `DependencyGraph.ComputeTransitiveClosure()` in `internal/dependencies.go:53-119`
- Use `assertProtoOnlyTypeMap()` helper for proto validation (convert_test.go:14-25)
- Create OpenAPI schemas with explicit reference chains
- Diamond pattern example from union support plan (plans/union-support-implementation-plan.md:676-694):
  - Owner references Pet, Home references Pet, Pet is union → Owner and Home both become Go-only
- BFS algorithm handles circular dependencies correctly (visited set prevents infinite loops)
- Assert TypeMap classification and reasons for each type
- Validation: `go test ./... -run TestDependencyGraph`

---

## Phase 4: Union/OneOf - Variant Edge Cases

- [x] Implemented `internal/unions_test.go` with 7 test functions
- [x] All tests passing with `go test ./internal -run TestUnion`
- [x] Full test suite still passing with `make test`
- [x] Phase 4 completed and verified

### Overview
Test union/oneOf scenarios with 3+ variants, nested unions (unions containing unions), arrays of unions, and discriminator edge cases. Validates error handling and correct Go marshaling generation.

### Changes Required

#### 1. New Test File: Union Edge Cases
**File**: `internal/unions_test.go` ✅
**Purpose**: Test oneOf/discriminator edge cases through functional API

```go
package internal_test

// TestUnionThreeVariants validates unions with 3+ variants
func TestUnionThreeVariants(t *testing.T)

// TestUnionNestedInProperty validates union type as object property
func TestUnionNestedInProperty(t *testing.T)

// TestUnionArrayOfUnions validates array containing union types
func TestUnionArrayOfUnions(t *testing.T)

// TestUnionMissingDiscriminatorProperty validates error when variant lacks discriminator
func TestUnionMissingDiscriminatorProperty(t *testing.T)

// TestUnionDiscriminatorConflict validates error when discriminator values conflict
func TestUnionDiscriminatorConflict(t *testing.T)

// TestUnionWithNestedObjects validates union variants containing nested objects
func TestUnionWithNestedObjects(t *testing.T)

// TestUnionMultipleUnionFields validates struct with multiple union-typed fields
func TestUnionMultipleUnionFields(t *testing.T)
```

**Test Objectives:**
- Verify 3+ variant unions generate correct Go struct with all pointer fields
- Verify union as property: `type Owner struct { Pet *Pet }` where Pet is union
- Verify arrays of unions: `type Container struct { Pets []*Pet }`
- Verify error when variant schema missing discriminator property
- Verify error when multiple variants map to same discriminator value
- Verify union variants with nested objects generate correctly
- Verify multiple union fields in one struct generate separate MarshalJSON logic
- Verify generated Go code compiles and marshals correctly

**Context for Implementation:**
- Extend pattern from `convert_test.go:914-1240` for union tests
- Reference discriminator validation in `internal/golang.go:166-232`
- Use JSON round-trip test pattern from `convert_test.go:1242-1340`
- Create test programs that marshal/unmarshal unions with multiple variants
- Assert error messages for validation failures
- Validation: `go test ./internal -run TestUnion`

---

## Phase 5: Union/OneOf - Runtime JSON Validation

- [x] Implemented `internal/unions_runtime_test.go` with 6 test functions
- [x] All tests passing with `go test ./internal -run TestUnionJSON`
- [x] Full test suite still passing with `make test`
- [x] Phase 5 completed and verified

### Overview
Create comprehensive runtime tests that compile and execute generated Go code to validate JSON marshaling/unmarshaling for unions with edge cases including case-insensitive discriminators, empty variants, and unknown discriminator values.

### Changes Required

#### 1. Extended Runtime Tests
**File**: `internal/unions_runtime_test.go` ✅
**Purpose**: Compile and execute generated Go code to validate runtime behavior

```go
package internal_test

// TestUnionJSONRoundTripMultipleVariants validates marshal/unmarshal with 3+ variants
func TestUnionJSONRoundTripMultipleVariants(t *testing.T)

// TestUnionJSONCaseInsensitive validates case-insensitive discriminator matching
func TestUnionJSONCaseInsensitive(t *testing.T)

// TestUnionJSONUnknownDiscriminator validates error on unknown discriminator value
func TestUnionJSONUnknownDiscriminator(t *testing.T)

// TestUnionJSONEmptyVariant validates error when no variant is set
func TestUnionJSONEmptyVariant(t *testing.T)

// TestUnionJSONNestedObjects validates unions with nested object variants
func TestUnionJSONNestedObjects(t *testing.T)

// TestUnionJSONMultipleFields validates struct with multiple union-typed fields
func TestUnionJSONMultipleFields(t *testing.T)
```

**Test Objectives:**
- Verify marshal/unmarshal round-trip for 3-variant union (Dog, Cat, Bird)
- Verify case-insensitive matching: "dog", "Dog", "DOG" all match Dog variant
- Verify unmarshal error with clear message for unknown discriminator value
- Verify marshal error when union struct has no variant set (all nil)
- Verify nested objects within union variants marshal correctly
- Verify multiple union fields in struct marshal independently
- Verify all generated code compiles without errors
- Verify JSON output format matches expected flat structure (not nested oneof)

**Context for Implementation:**
- Follow pattern from `convert_test.go:1242-1340` for round-trip tests
- Create temporary directory with go.mod and generated code
- Write test program that marshals/unmarshals various scenarios
- Execute test program with `exec.Command("go", "run", "main.go")`
- Assert compilation success and runtime output correctness
- Reference case-insensitive logic in `internal/gogen.go:1-186`
- Validation: `go test ./internal -run TestUnionJSON`

---

## Phase 6: Integration - Complex Scenarios

- [x] Implemented `internal/integration_test.go` with 4 test functions
- [x] All tests passing with `go test ./internal -run TestIntegration -v`
- [x] Full test suite still passing with `make test`
- [x] Phase 6 completed and verified

### Overview
Create end-to-end integration tests combining multiple features: schemas with unions + nested objects + arrays + enums all together, large schemas with 20+ types, and realistic domain models.

### Changes Required

#### 1. New Test File: Integration Tests
**File**: `internal/integration_test.go` ✅
**Purpose**: Test complex realistic scenarios combining all features

```go
package internal_test

// TestIntegrationEcommerce validates complete e-commerce domain model
func TestIntegrationEcommerce(t *testing.T)

// TestIntegrationLargeSchema validates schema with 25+ interconnected types
func TestIntegrationLargeSchema(t *testing.T)

// TestIntegrationUnionsWithNestedAndArrays validates unions + nested objects + arrays together
func TestIntegrationUnionsWithNestedAndArrays(t *testing.T)

// TestIntegrationMultipleUnionsWithReferences validates complex reference chains with unions
func TestIntegrationMultipleUnionsWithReferences(t *testing.T)
```

**Test Objectives:**
- Verify complete e-commerce schema (Product, Order, Payment union, Customer) generates correctly
- Verify large schema (25+ types) processes without error
- Verify combination of unions, nested objects, arrays, enums in one schema
- Verify complex reference chains with multiple unions maintain correct classification
- Verify all generated code (proto and Go) compiles successfully
- Verify realistic JSON payloads marshal/unmarshal correctly
- Verify performance remains acceptable for large schemas

**Context for Implementation:**
- Expand on e-commerce example from `convert_test.go:400-650`
- Create realistic domain models: payment systems, user profiles, content management
- Combine patterns from all previous phases
- Use both proto3 validation and Go compilation tests
- Assert TypeMap correctness for all types
- Measure and log processing time (informational only)
- Validation: `go test ./internal -run TestIntegration -v`

---

## Summary

### Test File Organization
```
internal/
├── golang_scalars_test.go      (Phase 1: 6 test functions)
├── golang_structures_test.go   (Phase 2: 6 test functions)
├── dependencies_test.go         (Phase 3: 5 test functions)
├── unions_test.go               (Phase 4: 7 test functions)
├── unions_runtime_test.go       (Phase 5: 6 test functions)
└── integration_test.go          (Phase 6: 4 test functions)
```

### Validation Progression
- **Phase 1-2**: `go test ./internal -run TestGo` validates Go generation
- **Phase 3**: `go test ./internal -run TestDependencyGraph` validates type classification
- **Phase 4-5**: `go test ./internal -run TestUnion` validates union handling
- **Phase 6**: `go test ./internal -run TestIntegration` validates complete scenarios
- **Final**: `make test && make coverage` validates all tests and coverage improvement

### Expected Outcomes
1. Test coverage increase for golang.go (currently untested → 80%+ coverage)
2. Test coverage increase for dependencies.go (currently untested → 75%+ coverage)
3. Union/oneOf edge cases comprehensively validated
4. All tests pass on CI with `make ci`
5. Clear error messages guide users when encountering unsupported patterns
