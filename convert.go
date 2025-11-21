package schema

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/duh-rpc/openapi-schema.go/internal"
	"github.com/duh-rpc/openapi-schema.go/internal/golang"
	"github.com/duh-rpc/openapi-schema.go/internal/parser"
	"github.com/duh-rpc/openapi-schema.go/internal/proto"
)

// ConvertResult contains the outputs from converting OpenAPI to proto3 and Go code.
//
// When schemas contain oneOf with discriminators, a hybrid approach is used:
//   - Protobuf contains types that don't use unions (may be empty if all types use unions)
//   - Golang contains union types, their variants, and any types that reference them
//   - TypeMap provides metadata about where each type was generated and why
//
// Output field behavior:
//   - Protobuf is empty when all schemas are union-related (use unions or reference union types)
//   - Golang is empty when no schemas contain or reference oneOf unions
//   - Both may contain content when schemas are mixed (some use unions, some don't)
type ConvertResult struct {
	Protobuf []byte
	Golang   []byte
	TypeMap  map[string]*TypeInfo
}

// StructResult contains the output from converting OpenAPI to Go structs only.
//
// This is the result type for ConvertToStruct(), which generates Go structs for
// all schemas without producing Protocol Buffer definitions.
//
// Field behavior:
//   - Golang contains Go source code with all schema types as structs
//   - TypeMap provides metadata about each type (all marked as TypeLocationGolang)
//   - Union types include custom MarshalJSON/UnmarshalJSON methods
//   - Regular types are simple structs with JSON tags
type StructResult struct {
	Golang  []byte
	TypeMap map[string]*TypeInfo
}

// ExampleResult contains generated JSON examples for schemas
type ExampleResult struct {
	Examples map[string]json.RawMessage // schema name → JSON example
}

// ValidationResult contains the validation status for all examples in an OpenAPI spec
type ValidationResult struct {
	Schemas map[string]*SchemaValidationResult
}

// SchemaValidationResult contains validation details for a single schema
type SchemaValidationResult struct {
	SchemaPath  string
	HasExamples bool
	Valid       bool
	Issues      []ValidationIssue
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
	SchemaNames []string // Specific schemas to validate (ignored if IncludeAll is true)
	IncludeAll  bool     // If true, validate all schemas (takes precedence over SchemaNames)
}

// ExampleOptions configures JSON example generation
type ExampleOptions struct {
	SchemaNames []string // Specific schemas to generate (ignored if IncludeAll is true)
	MaxDepth    int      // Maximum nesting depth (default 5)
	IncludeAll  bool     // If true, generate examples for all schemas (takes precedence over SchemaNames)
	Seed        int64    // Random seed for deterministic generation (0 = use time-based seed)
	// FieldOverrides allows overriding generated values for specific field names (e.g., {"code": 400, "status": "error"}).
	// - Applies to any field with matching name (case-sensitive) across all schemas
	// - Takes precedence over heuristics and generated values
	// - Does NOT override schema.Example or schema.Default (those have higher precedence)
	// - Type must match schema type or error is returned
	FieldOverrides map[string]interface{}
}

// TypeInfo contains metadata about where a type is generated and why
type TypeInfo struct {
	Location TypeLocation
	Reason   string
}

// TypeLocation indicates whether a type is generated as proto or golang
type TypeLocation string

const (
	TypeLocationProto  TypeLocation = "proto"
	TypeLocationGolang TypeLocation = "golang"
)

// ConvertOptions configures the conversion from OpenAPI to Protocol Buffers
type ConvertOptions struct {
	// PackageName is the name of the generated proto3 package (e.g. "api")
	PackageName string
	// PackagePath is the path of the generated proto3 package (e.g. "github.com/myorg/proto/v1/api")
	PackagePath string
	// GoPackagePath is the path for generated Go code (defaults to PackagePath if empty)
	GoPackagePath string
}

// Convert converts OpenAPI 3.x schemas (3.0, 3.1, 3.2) to Protocol Buffer 3 format.
// It takes OpenAPI specification bytes (YAML or JSON) and conversion options,
// and returns a ConvertResult containing proto3 output, Go output, and type metadata.
//
// Field names are preserved from the OpenAPI schema when they meet proto3 syntax
// requirements. Invalid characters (hyphens, dots, spaces) are replaced with
// underscores. All fields include json_name annotations for correct JSON mapping.
//
// Examples:
//   - HTTPStatus → HTTPStatus [json_name = "HTTPStatus"]
//   - userId → userId [json_name = "userId"]
//   - status-code → status_code [json_name = "status-code"]
//
// The function validates inputs, parses the OpenAPI document, extracts schemas,
// and generates corresponding proto3 message definitions.
//
// Returns an error if:
//   - openapi is empty
//   - opts.PackageName is empty
//   - opts.PackagePath is empty
//   - the OpenAPI document is invalid or not version 3.x
//   - any schema contains unsupported features
func Convert(openapi []byte, opts ConvertOptions) (*ConvertResult, error) {
	if len(openapi) == 0 {
		return nil, fmt.Errorf("openapi input cannot be empty")
	}

	if opts.PackageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	if opts.PackagePath == "" {
		return nil, fmt.Errorf("package path cannot be empty")
	}

	// Default GoPackagePath to PackagePath if not provided
	if opts.GoPackagePath == "" {
		opts.GoPackagePath = opts.PackagePath
	}

	doc, err := parser.ParseDocument(openapi)
	if err != nil {
		return nil, err
	}

	schemas, err := doc.Schemas()
	if err != nil {
		return nil, err
	}

	ctx := proto.NewContext()
	graph, err := proto.BuildMessages(schemas, ctx)
	if err != nil {
		return nil, err
	}

	// Compute transitive closure to classify types
	goTypes, protoTypes, reasons := graph.ComputeTransitiveClosure()

	// Build TypeMap using classification results
	typeMap := buildTypeMap(goTypes, protoTypes, reasons)

	// Generate proto for proto-only types
	// Skip proto generation only if there are Go types but no proto types
	var protoBytes []byte
	if len(protoTypes) > 0 || len(goTypes) == 0 {
		protoMessages := filterProtoMessages(ctx.Messages, protoTypes)
		// Create new context with filtered messages
		protoCtx := proto.NewContext()
		protoCtx.Messages = protoMessages
		protoCtx.Enums = ctx.Enums
		protoCtx.Definitions = filterProtoDefinitions(ctx.Definitions, protoTypes)
		protoCtx.UsesTimestamp = ctx.UsesTimestamp

		protoBytes, err = proto.Generate(opts.PackageName, opts.PackagePath, protoCtx)
		if err != nil {
			return nil, err
		}
	}

	// Generate Go for Go-only types
	var goBytes []byte
	if len(goTypes) > 0 {
		goCtx := golang.NewGoContext(golang.ExtractPackageName(opts.GoPackagePath))
		err := golang.BuildGoStructs(schemas, goTypes, graph, goCtx)
		if err != nil {
			return nil, err
		}
		goBytes, err = golang.GenerateGo(goCtx)
		if err != nil {
			return nil, err
		}
	}

	return &ConvertResult{
		Protobuf: protoBytes,
		Golang:   goBytes,
		TypeMap:  typeMap,
	}, nil
}

// ConvertToStruct converts all OpenAPI schemas to Go structs only, without
// generating Protocol Buffer definitions. This provides a pure Go struct
// generation path for users who need Go types but not protobuf.
//
// Unlike Convert(), this function generates Go structs for ALL schemas:
//   - Union types (oneOf with discriminator) get custom MarshalJSON/UnmarshalJSON
//   - Regular types become simple structs with JSON tags
//   - All types appear in TypeMap with Location=TypeLocationGolang
//
// Field names follow the same conversion rules as Convert():
//   - Invalid proto characters (hyphens, dots, spaces) are replaced with underscores
//   - JSON tags preserve original field names from OpenAPI schema
//
// Parameters:
//   - openapi: OpenAPI specification bytes (YAML or JSON)
//   - opts: Conversion options (only GoPackagePath is required, PackageName defaults to "main")
//
// Returns:
//   - StructResult containing Go source code and type metadata
//
// Returns an error if:
//   - openapi is empty
//   - opts.GoPackagePath is empty
//   - the OpenAPI document is invalid or not version 3.x
//   - any schema contains unsupported features
func ConvertToStruct(openapi []byte, opts ConvertOptions) (*StructResult, error) {
	if len(openapi) == 0 {
		return nil, fmt.Errorf("openapi input cannot be empty")
	}

	if opts.GoPackagePath == "" {
		return nil, fmt.Errorf("GoPackagePath cannot be empty")
	}

	// Default PackageName to "main" if empty (needed by BuildMessages)
	if opts.PackageName == "" {
		opts.PackageName = "main"
	}

	doc, err := parser.ParseDocument(openapi)
	if err != nil {
		return nil, err
	}

	schemas, err := doc.Schemas()
	if err != nil {
		return nil, err
	}

	// Build dependency graph for schema validation and discriminator support
	ctx := proto.NewContext()
	graph, err := proto.BuildMessages(schemas, ctx)
	if err != nil {
		return nil, err
	}

	// Compute transitive closure to get reasons map for TypeMap
	_, _, reasons := graph.ComputeTransitiveClosure()

	// Mark ALL schemas for Go generation (not filtered by transitive closure)
	goTypes := make(map[string]bool)
	for _, schema := range schemas {
		goTypes[schema.Name] = true
	}

	// Generate Go structs for all schemas
	goCtx := golang.NewGoContext(golang.ExtractPackageName(opts.GoPackagePath))
	err = golang.BuildGoStructs(schemas, goTypes, graph, goCtx)
	if err != nil {
		return nil, err
	}

	goBytes, err := golang.GenerateGo(goCtx)
	if err != nil {
		return nil, err
	}

	// Build TypeMap marking all schemas as Golang location
	typeMap := buildStructTypeMap(schemas, reasons)

	return &StructResult{
		Golang:  goBytes,
		TypeMap: typeMap,
	}, nil
}

// buildTypeMap creates a TypeMap from dependency graph classification results
func buildTypeMap(goTypes, protoTypes map[string]bool, reasons map[string]string) map[string]*TypeInfo {
	typeMap := make(map[string]*TypeInfo)

	// Add Go types
	for name := range goTypes {
		typeMap[name] = &TypeInfo{
			Location: TypeLocationGolang,
			Reason:   reasons[name],
		}
	}

	// Add Proto types
	for name := range protoTypes {
		typeMap[name] = &TypeInfo{
			Location: TypeLocationProto,
			Reason:   "",
		}
	}

	return typeMap
}

// buildStructTypeMap creates TypeMap marking all schemas as Golang location
func buildStructTypeMap(schemas []*parser.SchemaEntry, reasons map[string]string) map[string]*TypeInfo {
	typeMap := make(map[string]*TypeInfo)

	for _, schema := range schemas {
		reason := ""
		if r, ok := reasons[schema.Name]; ok {
			reason = r
		}
		typeMap[schema.Name] = &TypeInfo{
			Location: TypeLocationGolang,
			Reason:   reason,
		}
	}

	return typeMap
}

// filterProtoMessages removes messages marked as Go-only from proto output
func filterProtoMessages(messages []*proto.ProtoMessage, protoTypes map[string]bool) []*proto.ProtoMessage {
	filtered := make([]*proto.ProtoMessage, 0, len(protoTypes))

	for _, msg := range messages {
		// Only include messages that are in protoTypes set (using original schema name)
		if protoTypes[msg.OriginalSchema] {
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

// filterProtoDefinitions removes definitions marked as Go-only from proto output
func filterProtoDefinitions(definitions []interface{}, protoTypes map[string]bool) []interface{} {
	filtered := make([]interface{}, 0)

	for _, def := range definitions {
		// Check if it's a ProtoMessage and filter accordingly
		if msg, ok := def.(*proto.ProtoMessage); ok {
			if protoTypes[msg.OriginalSchema] {
				filtered = append(filtered, def)
			}
		} else {
			// Keep enums and other definitions
			filtered = append(filtered, def)
		}
	}

	return filtered
}

// ConvertToExamples generates JSON examples from OpenAPI schemas
func ConvertToExamples(openapi []byte, opts ExampleOptions) (*ExampleResult, error) {
	if len(openapi) == 0 {
		return nil, fmt.Errorf("openapi input cannot be empty")
	}

	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 5
	}

	if !opts.IncludeAll && len(opts.SchemaNames) == 0 {
		return nil, fmt.Errorf("must specify SchemaNames or set IncludeAll")
	}

	if opts.Seed == 0 {
		opts.Seed = time.Now().UnixNano()
	}

	doc, err := parser.ParseDocument(openapi)
	if err != nil {
		return nil, err
	}

	schemas, err := doc.Schemas()
	if err != nil {
		return nil, err
	}

	schemaNames := opts.SchemaNames
	if opts.IncludeAll {
		schemaNames = nil
	}

	examples, err := internal.GenerateExamples(schemas, schemaNames, opts.MaxDepth, opts.Seed, opts.FieldOverrides)
	if err != nil {
		return nil, err
	}

	return &ExampleResult{
		Examples: examples,
	}, nil
}

// ValidateExamples validates examples in OpenAPI spec against schemas.
// It validates the 'example' and 'examples' fields in Schema Objects under components/schemas.
//
// For schemas with the 'examples' map, all entries are validated.
// If both 'example' and 'examples' exist on the same schema, both are validated.
//
// Parameters:
//   - openapi: OpenAPI specification bytes (YAML or JSON)
//   - opts: Validation options (SchemaNames to filter specific schemas, or IncludeAll to validate all)
//
// Returns:
//   - ValidationResult containing per-schema validation results with errors and warnings
//
// Returns an error if:
//   - openapi is empty
//   - opts.IncludeAll is false and opts.SchemaNames is empty
//   - the OpenAPI document is invalid or not version 3.x
func ValidateExamples(openapi []byte, opts ValidateOptions) (*ValidationResult, error) {
	if len(openapi) == 0 {
		return nil, fmt.Errorf("openapi input cannot be empty")
	}

	if !opts.IncludeAll && len(opts.SchemaNames) == 0 {
		return nil, fmt.Errorf("must specify SchemaNames or set IncludeAll")
	}

	schemaNames := opts.SchemaNames
	if opts.IncludeAll {
		schemaNames = nil
	}

	internalResult, err := internal.ValidateExamples(openapi, schemaNames)
	if err != nil {
		return nil, err
	}

	// Convert internal result to public API types
	result := &ValidationResult{
		Schemas: make(map[string]*SchemaValidationResult),
	}

	for schemaName, schemaValidation := range internalResult.Schemas {
		issues := make([]ValidationIssue, len(schemaValidation.Issues))
		for i, issue := range schemaValidation.Issues {
			issues[i] = ValidationIssue{
				Severity:     IssueSeverity(issue.Severity),
				ExampleField: issue.ExampleField,
				Message:      issue.Message,
				Line:         issue.Line,
			}
		}

		result.Schemas[schemaName] = &SchemaValidationResult{
			SchemaPath:  schemaValidation.SchemaPath,
			HasExamples: schemaValidation.HasExamples,
			Valid:       schemaValidation.Valid,
			Issues:      issues,
		}
	}

	return result, nil
}
