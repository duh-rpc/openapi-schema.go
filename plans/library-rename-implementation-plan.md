# Library Rename: openapi-proto.go → openapi-schema.go Implementation Plan

## Overview

This plan renames the library from `openapi-proto.go` to `openapi-schema.go` to better reflect its current scope. The library has evolved from a simple OpenAPI-to-Proto converter into a multi-purpose OpenAPI schema processing toolkit with 4 main capabilities: Convert (Proto3 + Go), ConvertToStruct (Go only), ConvertToExamples (JSON generation), and ValidateExamples (validation).

## Current State Analysis

**Module Information:**
- Current module path: `github.com/duh-rpc/openapi-proto.go`
- Current package name: `conv`
- Current import alias: `conv`
- GitHub repository: `duh-rpc/openapi-proto.go`

**Files Requiring Changes:**
- 1 module declaration (go.mod)
- 6 package declarations (convert.go + 5 test files at root)
- 10 import path declarations across 8 files
- Approximately 146 type/function references (`conv.Convert()`, `conv.ConvertOptions`, etc.)
- 27 total Go files affected (1 main + 5 root tests + ~22 internal tests + convert.go)
- 4 documentation files (README.md, CLAUDE.md, docs/*)
- 2 planning documents (historical reference)
- Badge URLs and external links

**Key Discoveries:**
- Only 1 of 4 public functions generates protobuf (25% of functionality)
- User is the only external consumer, so backward compatibility is not required
- No existing git tags or releases to worry about
- GitHub will automatically redirect old repository URLs to new name

## Desired End State

**New Module Information:**
- New module path: `github.com/duh-rpc/openapi-schema.go`
- New package name: `schema`
- New import pattern: `import schema "github.com/duh-rpc/openapi-schema.go"`
- GitHub repository: `duh-rpc/openapi-schema.go` (renamed)

**Verification:**
- All tests pass: `make test`
- All linting passes: `make lint`
- Module is tidy: `make tidy`
- Formatting is correct: `make fmt`
- CI pipeline passes: `make ci`

## What We're NOT Doing

- Creating git tags or version numbers (no existing releases)
- Maintaining backward compatibility (user is sole consumer)
- Creating deprecation notices or migration guides
- Updating external package registries (go.dev will auto-update)
- Modifying internal package structure or logic
- Changing functionality or behavior

## Implementation Approach

This is a pure rename operation with no logic changes. We'll update all references systematically:
1. Module and package declarations first
2. Import statements second
3. Documentation third
4. Verify everything works

The changes are mechanical string replacements - no code logic is modified.

## Phase 0: Pre-flight Check

### Overview
Verify the current state of the codebase and establish baseline metrics before making changes.

### Commands to Run

```bash
# Count files that will be affected
echo "=== Files with openapi-proto references ==="
grep -rl "openapi-proto" . --include="*.go" --include="*.md" --exclude-dir=".git" | wc -l
grep -rl "openapi-proto" . --include="*.go" --include="*.md" --exclude-dir=".git"

echo -e "\n=== Package declarations to update ==="
grep -n "^package conv" *.go internal/*.go 2>/dev/null

echo -e "\n=== Count of conv. references ==="
grep -r "conv\." . --include="*.go" --exclude-dir=".git" | wc -l

echo -e "\n=== Files with conv. references ==="
grep -l "conv\." . --include="*.go" --exclude-dir=".git"

echo -e "\n=== Current module path ==="
head -1 go.mod

echo -e "\n=== Git remote URL ==="
git remote -v

echo -e "\n=== Current tests status ==="
make test
```

### Expected Baseline
- ~27 Go files with references
- 6 package declarations (convert.go + 5 root test files)
- ~146 conv. references across test files
- Module: `github.com/duh-rpc/openapi-proto.go`
- All tests should pass before starting

### Validation
- [ ] Run all pre-flight commands
- [ ] Verify: All tests pass in current state
- [ ] Document: Save baseline metrics for comparison after changes

## Phase 1: Module and Package Declarations

### Overview
Update the module path and package names in all Go files to use the new `openapi-schema.go` name and `schema` package.

### Changes Required

#### 1. Module Declaration
**File**: `go.mod`
**Changes**: Update module path to new repository name

**Current:**
```go
module github.com/duh-rpc/openapi-proto.go
```

**New:**
```go
module github.com/duh-rpc/openapi-schema.go
```

#### 2. Main Package Declaration
**File**: `convert.go`
**Changes**: Change package from `conv` to `schema`

**Current:**
```go
package conv
```

**New:**
```go
package schema
```

#### 3. Test Package Declarations
**Files**:
- `convert_test.go`
- `convert_examples_phase3_test.go`
- `convert_examples_heuristics_test.go`
- `convert_examples_test.go`
- `convert_validate_test.go`

**Changes**: Change package from `conv_test` to `schema_test` in each file

**Current:**
```go
package conv_test
```

**New:**
```go
package schema_test
```

**Context for implementation:**
- All package declarations are on line 1 of their respective files
- Simple find-and-replace operation across 6 files
- No other code changes needed in this phase

### Validation
- [ ] Run: `go mod tidy`
- [ ] Verify: No errors, go.mod shows new module path
- [ ] Run: `go build ./...`
- [ ] Verify: All packages compile successfully

## Phase 2: Update Import Statements

### Overview
Update all import statements to use the new module path and package alias.

**Scope of Changes:**
This phase updates approximately:
- 10 import path declarations
- 146 function/type reference changes across test files
- 27 total files (1 main package + 5 root tests + ~22 internal tests + convert.go)

**Types requiring rename:**
- `conv.Convert()` → `schema.Convert()`
- `conv.ConvertToExamples()` → `schema.ConvertToExamples()`
- `conv.ConvertToStruct()` → `schema.ConvertToStruct()`
- `conv.ValidateExamples()` → `schema.ValidateExamples()`
- `conv.ConvertOptions` → `schema.ConvertOptions`
- `conv.ConvertResult` → `schema.ConvertResult`
- `conv.StructResult` → `schema.StructResult`
- `conv.ExampleOptions` → `schema.ExampleOptions`
- `conv.ExampleResult` → `schema.ExampleResult`
- `conv.ValidateOptions` → `schema.ValidateOptions`
- `conv.ValidationResult` → `schema.ValidationResult`

### Changes Required

#### 1. Root Package Import Statements
**File**: `convert.go`
**Changes**: Update internal package imports to new module path

**Current (Lines 8-9):**
```go
"github.com/duh-rpc/openapi-proto.go/internal"
"github.com/duh-rpc/openapi-proto.go/internal/parser"
```

**New:**
```go
"github.com/duh-rpc/openapi-schema.go/internal"
"github.com/duh-rpc/openapi-schema.go/internal/parser"
```

#### 2. Test File Imports - Root Level
**Files** (all 5 root-level test files):
- `convert_test.go` (line 9)
- `convert_examples_test.go` (line 7)
- `convert_examples_phase3_test.go` (line 7)
- `convert_examples_heuristics_test.go` (line 7)
- `convert_validate_test.go` (line 6)

**Changes**: Update package import alias from `conv` to `schema`

**Current:**
```go
conv "github.com/duh-rpc/openapi-proto.go"
```

**New:**
```go
schema "github.com/duh-rpc/openapi-schema.go"
```

**Note**: These test files also need all references to `conv.` changed to `schema.` throughout the file (e.g., `conv.Convert()` → `schema.Convert()`, `conv.ConvertOptions` → `schema.ConvertOptions`, etc.)

#### 3. Internal Package Imports
**File**: `internal/examplevalidator.go` (line 8)
**Changes**: Update parser import path

**Current:**
```go
"github.com/duh-rpc/openapi-proto.go/internal/parser"
```

**New:**
```go
"github.com/duh-rpc/openapi-schema.go/internal/parser"
```

#### 4. Internal Test File Imports
**Files** (all ~22 internal test files that import the main package):
- `internal/generator_test.go`
- `internal/integration_test.go`
- `internal/dependencies_test.go`
- `internal/enums_test.go`
- `internal/field_numbers_test.go`
- `internal/naming_test.go`
- `internal/unions_test.go`
- `internal/unions_runtime_test.go`
- `internal/golang_structures_test.go`
- `internal/golang_scalars_test.go`
- `internal/version_test.go`
- `internal/nullable_test.go`
- `internal/scalars_test.go`
- `internal/required_test.go`
- `internal/nested_test.go`
- `internal/conflicts_test.go`
- `internal/arrays_test.go`
- `internal/errors_test.go`
- `internal/builder_test.go`
- `internal/comments_test.go`
- `internal/refs_test.go`
- `internal/field_scoping_test.go`
- `internal/integration_examples_test.go`

**Changes**: Update package import alias from `conv` to `schema` where present

**Current:**
```go
conv "github.com/duh-rpc/openapi-proto.go"
```

**New:**
```go
schema "github.com/duh-rpc/openapi-schema.go"
```

**Note**: Not all internal test files import the main package. Only update files that have this import. These test files also need all references to `conv.` changed to `schema.` throughout the file (e.g., `conv.ConvertOptions`, `conv.ConvertResult`, etc.)

**Context for implementation:**
- Use find-replace functionality to update import paths systematically
- After updating imports, use find-replace to change all `conv.` → `schema.` across Go files
- Approximately 146 occurrences of `conv.` references need updating
- Some internal test files may not import the main package - only update files that have the import
- Verify with grep after changes to ensure no `conv.` references remain

### Validation
- [ ] Run: `grep -r "conv\." . --include="*.go" | wc -l`
- [ ] Verify: Returns 0 (no conv. references remain)
- [ ] Run: `grep -r "package conv" . --include="*.go"`
- [ ] Verify: Returns 0 results (all package declarations updated)
- [ ] Run: `go build ./...`
- [ ] Verify: No import errors
- [ ] Run: `make test`
- [ ] Verify: All tests pass

## Phase 3: Update Documentation

### Overview
Update all documentation files to reflect the new library name, module path, and package alias.

### Changes Required

#### 1. README.md
**File**: `README.md`
**Changes**: Update title, badges, installation instructions, and all code examples

**Locations to update:**
- Line 1: Title "OpenAPI to Protobuf Converter" → "OpenAPI Schema Processor"
- Line 5: Badge URL `go-mod/go-version/duh-rpc/openapi-proto.go` → `go-mod/go-version/duh-rpc/openapi-schema.go`
- Line 6: CI badge URL `duh-rpc/openapi-proto.go/workflows` → `duh-rpc/openapi-schema.go/workflows`
- Line 8: Go Report Card `github.com/duh-rpc/openapi-proto.go` → `github.com/duh-rpc/openapi-schema.go`
- Line 17: Installation `go get github.com/duh-rpc/openapi-proto.go` → `go get github.com/duh-rpc/openapi-schema.go`
- Lines 31, 79, 138: Import examples `conv "github.com/duh-rpc/openapi-proto.go"` → `schema "github.com/duh-rpc/openapi-schema.go"`
- Line 12: Description text update to emphasize schema processing capabilities

**Additional Changes:**
- Update all code examples that use `conv.Convert()` to `schema.Convert()`
- Update all code examples that use `conv.ConvertOptions` to `schema.ConvertOptions`
- Update all code examples that use `conv.ConvertResult` to `schema.ConvertResult`
- Update references to variable names like `conv` to `schema` in example code

#### 2. CLAUDE.md
**File**: `CLAUDE.md`
**Changes**: Update import statement in functional testing example

**Location**: Line 26
**Current:**
```go
import conv "github.com/duh-rpc/openapi-proto.go"
```

**New:**
```go
import schema "github.com/duh-rpc/openapi-schema.go"
```

**Additional Changes:**
- Update example code that uses `conv.Convert()` to `schema.Convert()`
- Update example code that uses `conv.ConvertOptions` to `schema.ConvertOptions`

#### 3. Documentation Files
**Files**: `docs/*.md` (check all markdown files in docs directory)
**Changes**: Search for references to "openapi-proto" and update to "openapi-schema", update `conv` references to `schema`

**Known references:**
- `docs/examples.md` (line 23): Import statement and code examples
- `docs/objects.md` (line 3): Text reference to library name
- `docs/discriminated-unions.md`: Import statements and code examples with `conv.` usage

**Pattern to search:**
- Any reference to "openapi-proto"
- Import paths with old module name
- Code examples using `conv.` prefix

#### 4. Planning Documents (Historical Reference)
**Files**: `plans/*.md` (excluding this plan)
**Changes**: Add a historical note at the top of existing plan documents, but do not update internal references

**Known files:**
- `plans/implementation-plan.md`
- `plans/implementation-plan-x-proto-number.md`
- `plans/example-validation-plan.md` (if it exists)

**Add this note at the top of each historical plan:**
```markdown
> **Historical Note**: This plan was created when the library was named `openapi-proto.go`.
> The library has since been renamed to `openapi-schema.go`. Import paths and package
> names in this document reflect the old naming for historical accuracy.
```

**Do NOT update**: Internal code references or import paths within these documents - preserve them as historical artifacts

**Context for implementation:**
- README.md is the most critical file - it's the first thing users see
- Code examples must be updated to use `schema` package alias
- Badge URLs are used by GitHub to display status
- Search all .md files for "openapi-proto" to catch any missed references

### Validation
- [ ] Run: `grep -r "openapi-proto" README.md CLAUDE.md docs/`
- [ ] Verify: No matches found (except in historical planning docs with notes)
- [ ] Run: `grep -r "conv \"github.com" README.md CLAUDE.md docs/`
- [ ] Verify: No matches found
- [ ] Manual: Review README.md in a markdown viewer
- [ ] Verify: All examples display correctly with new import paths

## Phase 4: Final Verification and GitHub Rename

### Overview
Run the complete test suite, verify all changes are correct, commit changes, and rename the GitHub repository.

### Changes Required

#### 1. Run Full Test Suite
**Commands to run:**
```bash
make ci
```

**This executes:**
- `go mod tidy` - Ensures module dependencies are clean
- `go fmt ./...` - Verifies formatting
- `golangci-lint run ./...` - Runs linters
- `go test -v ./...` - Runs all tests

**Expected output:**
- All commands pass with no errors
- Tests show new package name `schema` in output
- No references to old `conv` package

#### 2. Verify Import Paths
**Command:**
```bash
grep -r "openapi-proto" . --include="*.go" --include="*.md" --exclude-dir=".git"
```

**Expected output:**
- No matches in .go files
- Only matches in plans/*.md files with historical notes
- No matches in README.md, CLAUDE.md, or docs/

#### 3. Commit Changes
**Commands:**
```bash
git add .
git commit -m "Rename library from openapi-proto.go to openapi-schema.go

- Update module path to github.com/duh-rpc/openapi-schema.go
- Change package name from conv to schema
- Update all import statements and references
- Update documentation and examples
- Update badge URLs

This rename better reflects the library's current scope as a
general-purpose OpenAPI schema processing toolkit rather than
just a protobuf converter."
```

**Context for implementation:**
- Follow CLAUDE.md guidelines for commit messages (no Co-Authored-By)
- Use descriptive commit message explaining the rename
- Ensure all files are staged before committing

#### 4. Rename GitHub Repository
**Steps:**
1. Go to repository settings on GitHub
2. Navigate to "Repository name" section
3. Change name from `openapi-proto.go` to `openapi-schema.go`
4. Confirm the rename
5. Wait for confirmation message that rename is complete

**GitHub behavior:**
- Automatically redirects old URLs to new repository
- Updates all GitHub-hosted references (issues, PRs, etc.)
- CI badges will automatically update
- Go module proxy will recognize the new path on next fetch

**Note:** This step must be done manually through the GitHub web interface. It cannot be automated via git commands.

#### 5. Update Git Remote and Clear Cache
**Commands:**
```bash
# Verify current remote URL
git remote -v

# Update remote URL to new repository name (GitHub redirects work, but explicit is better)
git remote set-url origin git@github.com:duh-rpc/openapi-schema.go.git

# Verify the update
git remote -v

# Clear local module cache to force fresh fetch
go clean -modcache
```

**Note:** The remote URL update is optional since GitHub redirects work, but it's cleaner to have the correct URL.

#### 6. Push Changes
**Commands:**
```bash
git push origin main
```

**Expected outcome:**
- Changes pushed successfully
- CI workflow triggers and passes
- Repository is now accessible at new URL

**Context for implementation:**
- Push after GitHub rename is complete
- Verify CI pipeline runs successfully with new name
- Old import paths will break (intentional, no backward compatibility needed)

### Validation
- [ ] Run: `make ci`
- [ ] Verify: All checks pass, output shows `schema` package
- [ ] Run: `grep -r "openapi-proto" . --include="*.go" --include="*.md" --exclude-dir=".git"`
- [ ] Verify: Only matches in plans/*.md with historical notes
- [ ] Run: `grep -r "conv\." . --include="*.go" --exclude-dir=".git" | wc -l`
- [ ] Verify: Returns 0 (no conv. references remain in Go files)
- [ ] Run: `git status`
- [ ] Verify: All changes committed
- [ ] Manual: Check GitHub repository page
- [ ] Verify: Repository name is `duh-rpc/openapi-schema.go`
- [ ] Manual: Check CI pipeline on GitHub
- [ ] Verify: Latest commit shows passing checks
- [ ] Manual: Visit `https://github.com/duh-rpc/openapi-proto.go`
- [ ] Verify: Redirects to `https://github.com/duh-rpc/openapi-schema.go`

## Rollback Procedure

If issues arise during implementation, use these rollback procedures:

### Before Commit (During Phases 1-3)
```bash
# Discard all uncommitted changes
git reset --hard HEAD

# Verify rollback
git status  # Should show "working tree clean"
make test   # Should pass with old naming
```

### After Commit, Before Push (After Phase 3)
```bash
# Revert the rename commit
git log -1  # Note the commit hash
git revert <commit-hash>

# Or reset to previous commit (harder reset)
git reset --hard HEAD~1

# Verify rollback
make test
```

### After Push, Before GitHub Rename
```bash
# Revert on remote
git revert <commit-hash>
git push origin main

# Alternative: Force push previous state (use with caution)
git reset --hard HEAD~1
git push --force origin main
```

### After GitHub Rename
```bash
# Use GitHub UI to rename repository back to openapi-proto.go
# Then:
git remote set-url origin git@github.com:duh-rpc/openapi-proto.go.git
git pull origin main
```

**Note:** Since you're the only user, rollback is low-risk. However, after GitHub rename and push, it's simpler to fix forward than rollback.
