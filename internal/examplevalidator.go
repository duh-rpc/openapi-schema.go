package internal

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duh-rpc/openapi-schema.go/internal/parser"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	yaml "go.yaml.in/yaml/v4"
)

// ExampleValidationResult wraps validation results for export to public API
type ExampleValidationResult struct {
	Schemas map[string]*SchemaValidation
}

// SchemaValidation contains validation details for a single schema
type SchemaValidation struct {
	SchemaPath  string
	HasExamples bool
	Valid       bool
	Issues      []Issue
}

// Issue represents a single validation error or warning
type Issue struct {
	Severity     Severity
	ExampleField string
	Message      string
	Line         int
}

// Severity indicates whether an issue is an error or warning
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// ValidateExamples validates examples in OpenAPI spec against schemas
func ValidateExamples(openapi []byte, schemaNames []string) (*ExampleValidationResult, error) {
	// Parse raw document for version detection
	document, err := libopenapi.NewDocument(openapi)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}

	version := document.GetVersion()
	isOpenAPI30 := strings.HasPrefix(version, "3.0")

	// Parse schemas via existing parser
	parsedDoc, err := parser.ParseDocument(openapi)
	if err != nil {
		return nil, err
	}

	schemas, err := parsedDoc.Schemas()
	if err != nil {
		return nil, err
	}

	// Build schema map for filtering
	schemaMap := make(map[string]*parser.SchemaEntry)
	for _, entry := range schemas {
		schemaMap[entry.Name] = entry
	}

	// Determine which schemas to validate
	targetSchemas := schemas
	if len(schemaNames) > 0 {
		targetSchemas = make([]*parser.SchemaEntry, 0, len(schemaNames))
		for _, name := range schemaNames {
			if entry, ok := schemaMap[name]; ok {
				targetSchemas = append(targetSchemas, entry)
			}
		}
	}

	// Create validator
	validator := schema_validation.NewSchemaValidator()

	// Validate examples for each schema
	results := make(map[string]*SchemaValidation)
	for _, schemaEntry := range targetSchemas {
		schema := schemaEntry.Proxy.Schema()
		schemaName := schemaEntry.Name

		result := &SchemaValidation{
			SchemaPath:  schemaName,
			HasExamples: false,
			Valid:       true,
			Issues:      []Issue{},
		}

		// Add OpenAPI 3.0 warning if applicable (only once, added to first schema)
		if isOpenAPI30 && len(results) == 0 {
			result.Issues = append(result.Issues, Issue{
				Severity:     SeverityWarning,
				ExampleField: "",
				Message:      "OpenAPI 3.0 detected: validation may have limitations due to JSON Schema divergence. OpenAPI 3.1+ recommended for full JSON Schema compliance.",
				Line:         0,
			})
		}

		// Validate singular 'example' field
		if schema.Example != nil {
			result.HasExamples = true
			issues := validateExample(schema, schema.Example, "example", validator, isOpenAPI30)
			result.Issues = append(result.Issues, issues...)
			if hasErrors(issues) {
				result.Valid = false
			}
		}

		// Validate plural 'examples' field (OpenAPI 3.1+)
		if len(schema.Examples) > 0 {
			result.HasExamples = true
			for i, exampleNode := range schema.Examples {
				exampleField := fmt.Sprintf("examples[%d]", i)
				issues := validateExample(schema, exampleNode, exampleField, validator, isOpenAPI30)
				result.Issues = append(result.Issues, issues...)
				if hasErrors(issues) {
					result.Valid = false
				}
			}
		}

		results[schemaName] = result
	}

	return &ExampleValidationResult{
		Schemas: results,
	}, nil
}

// validateExample validates a single example against a schema
func validateExample(schema *base.Schema, exampleNode *yaml.Node, exampleField string, validator schema_validation.SchemaValidator, isOpenAPI30 bool) []Issue {
	var issues []Issue

	// Convert yaml.Node to interface{}
	var exampleData interface{}
	err := exampleNode.Decode(&exampleData)
	if err != nil {
		issues = append(issues, Issue{
			Severity:     SeverityError,
			ExampleField: exampleField,
			Message:      fmt.Sprintf("failed to decode example: %v", err),
			Line:         exampleNode.Line,
		})
		return issues
	}

	// Convert to JSON
	exampleJSON, err := json.Marshal(exampleData)
	if err != nil {
		issues = append(issues, Issue{
			Severity:     SeverityError,
			ExampleField: exampleField,
			Message:      fmt.Sprintf("failed to marshal example to JSON: %v", err),
			Line:         exampleNode.Line,
		})
		return issues
	}

	// Validate using version-aware method
	var valid bool
	var validationErrors []*errors.ValidationError
	if isOpenAPI30 {
		valid, validationErrors = validator.ValidateSchemaStringWithVersion(schema, string(exampleJSON), 3.0)
	} else {
		valid, validationErrors = validator.ValidateSchemaString(schema, string(exampleJSON))
	}

	if !valid {
		for _, validationError := range validationErrors {
			issues = append(issues, Issue{
				Severity:     SeverityError,
				ExampleField: exampleField,
				Message:      validationError.Message,
				Line:         exampleNode.Line,
			})
		}
	}

	return issues
}

// hasErrors checks if any issues are errors (not warnings)
func hasErrors(issues []Issue) bool {
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}
