## Commiting Code
- You MUST NOT include Co-Authored notification in commit message
- You MUST NOT include Claude attribution in commit or description messages

## Pull Requests
- You MUST format the description as follows
```
### Purpose
A paragraph describing what this change intends to acheive

### Implementation
- An explaination of each major code change made
```

## Testing Patterns

### Functional Testing Style
- **ALL TESTS MUST BE FUNCTIONAL** - Tests must start the server with d := NewDaemon(); d.Shutdown() 
    and interact with the service through the client returned by d.MustClient()
- Tests MUST NOT call internal package functions directly
- Test MUST always be in the test package `package XXX_test` and not `package XXX`
- Test names should be in camelCase and start with a capital letter (e.g., `TestGenerateClientWithDefaults`)

### Functional Testing Example
```go
import schema "github.com/duh-rpc/openapi-schema.go"

func TestScenarioName(t *testing.T) {
  for _, test := range []struct {
    name     string
    given    string
    expected string
  }{
    {
      name:     "simple string field",
      given:    `<OpenAPI YAML for string field>`,
      expected: `<Expected Proto3 output for string>`,
    },
    {
      name:     "integer field with validation",
      given:    `<OpenAPI YAML for integer field>`,
      expected: `<Expected Proto3 output for integer>`,
    },
  } {
    t.Run(test.name, func(t *testing.T) {
      result, err := schema.Convert([]byte(test.given), schema.ConvertOptions{
        PackageName: "testpkg",
        PackagePath: "github.com/example/proto/v1",
      })
      require.NoError(t, err)
      assert.Equal(t, test.expected, string(result.Protobuf))
    })
  }
}
```

### Additional Testing Guidelines
- Use integers to indicate multiples. For example, `listPageOne` becomes `listPage1` or simply `page1`.
- Avoid logging what the test is doing, instead prefer comments. For example, avoid `t.Log("Enrolling customers...")`
- Avoid the DRY (Don't Repeat Yourself) principle in tests, be explicit when testing repetitive behaviors.
- Table-driven tests MAY be used for sub-tests with `t.Run()` when testing multiple error cases (see `internal/lint/lint_test.go` for examples)
- Do NOT use `if (condition) { t.Error() }` for assertions. Use `github.com/stretchr/testify/require` and `github.com/stretchr/testify/assert`
- Do NOT use `require.Contains(t, err.Error(), test.wantErr)` use
  `require.ErrorContains(t, err, test.wantErr)` instead.
- Avoid placing explanations in require or assert statements. Do not include descriptive messages as the final parameter. For example:
    - Do NOT use: `require.NotNil(t, page1, "page1 result should not be nil")`
    - Do NOT use: `require.NoError(t, err, "Failed to enroll for %s", customerID)`
    - Do NOT use: `require.NotEmpty(t, endCursor, "EndCursor should not be empty")`
    - Instead use: `require.NotNil(t, page1)`, `require.NoError(t, err)`, `require.NotEmpty(t, endCursor)`
- Use `require` for critical assertions that should halt the test on failure, use `assert` for non-critical assertions that allow the test to continue. Import both packages when needed:
    - Use `require` for: error checking (`require.NoError`), nil checks that prevent further operations, setup/teardown operations
    - Use `assert` for: value comparisons (`assert.Equal`), boolean checks (`assert.True`/`assert.False`), length checks (`assert.Len`), existence checks where test can continue
    - Example:
    ```go
    // Critical operations - test cannot continue if these fail
    require.NoError(t, err)
    require.NotNil(t, result)

    // Non-critical assertions - test can continue to verify other aspects
    assert.Equal(t, expectedValue, result.Value)
    assert.True(t, result.IsValid)
    assert.Len(t, result.Items, 3)
    ```

## Code Guidelines

### Service Error Handling
- Service methods MUST return `duh.NewServiceError()` for all errors
- Use appropriate DUH error codes: `duh.CodeBadRequest`, `duh.CodeNotFound`, `duh.CodeInternalError`, etc.
- Example:
    ```go
    // BAD: Using NewClientError in service
    return duh.NewClientError("user not found", nil, nil)

    // GOOD: Using NewServiceError with proper code
    return duh.NewServiceError(duh.CodeNotFound, "user not found", nil, nil)
    return duh.NewServiceError(duh.CodeBadRequest, "email is required", nil, nil)
    ```

### General Guidelines
- Use `const` for variables that don't change and are used more than once
- Prefer one or two word variable names
- Avoid using local variables if the variable is only used once. Inline values directly into function calls instead.
    ```go
    // BAD: Creating variables that are used only once
    request := pkg.EnrollRequest{Thing: pkg.Thing1, Key: "value"}
    err := core.Enroll(ctx, request)

    // GOOD: Inline the struct directly
    err := core.Enroll(ctx, pkg.EnrollRequest{
        Thing:  pkg.Thing1,
        Key:    "value",
    })
    ```
- Prefer one or two word variables. `createdProductIDs` should be `created` or `createdIDs` if `created` is unclear in the current context
- Don't use abbreviations for variable names, use full words instead. For example `listP1` should be `listPage1`
- Use single letters for variable names only if the context is clear, and the scope is very small. For example, within a for loop where `i` is the index.
- Use `const` for variables that do not change. For example, `numEnrollments := 20` should be `const numEnrollments = 20`
- Do not comment in code that you are following guidelines
- Use `lo.ToPtr()` from the `github.com/samber/lo` package to create pointers to local variables

## Struct Field Formatting - Visual Tapering
When formatting struct literals, arrange fields to create a visual tapering effect:
- Order fields by the length of their complete line (field name + value)
- Place longer lines toward the top, shorter lines toward the bottom
- This creates a pleasing diagonal slope from long to short
- The goal is visual harmony and improved readability, not strict alphabetical or logical ordering

Example:
```go
req := pkg.CreateRequest{
    ThingType:      "THIS_THING_IS_WAY_TOO_DARN_LONG",  // longest line
    Quantity:       decimal.NewFromInt(100),            // medium length
    DriversID:      "DL-2134234122132",                 // medium length
    ThingID:        pkg.ThingTwenty,                    // medium length
    ThingName:      "thing-20",                         // shorter
    PackageID:      "pack",                             // shorter
    Msg:            "hi",                               // shortest
}

### Problem Solving
- Prefer fixing implementation over changing tests - When tests fail after code
  changes, the default assumption should be that the implementation broke
  expected behavior, not that the tests are wrong
- Tests encode business requirements - Test expectations often represent the
  correct business behavior that should be preserved. Changing tests to match
  broken implementation can mask real functional regressions
- Investigate test failures as potential bugs first - Before concluding "the
  tests need to be updated," thoroughly investigate whether the implementation
  correctly preserves the original business semantics
