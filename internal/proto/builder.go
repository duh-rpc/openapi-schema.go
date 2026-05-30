package proto

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/duh-rpc/openapi-schema.go/internal"
	"github.com/duh-rpc/openapi-schema.go/internal/parser"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// Context holds state during conversion
type Context struct {
	Tracker       *internal.NameTracker
	Messages      []*ProtoMessage
	Enums         []*ProtoEnum
	Definitions   []interface{} // Mixed enums and messages in processing order
	FieldNumbers  *FieldNumbers // nil → positional numbering
	UsesTimestamp bool
}

// NewContext creates a new conversion context
func NewContext() *Context {
	return &Context{
		Tracker:       internal.NewNameTracker(),
		Messages:      []*ProtoMessage{},
		Enums:         []*ProtoEnum{},
		Definitions:   []interface{}{},
		UsesTimestamp: false,
	}
}

// ProtoMessage represents a proto3 message definition
type ProtoMessage struct {
	Name           string
	Description    string
	Fields         []*ProtoField
	Nested         []*ProtoMessage
	Reserved       []int  // proto field numbers retired via removal (rendered as `reserved N, M;`)
	OriginalSchema string // Original schema name before name tracker renaming
}

// ProtoField represents a proto3 field
type ProtoField struct {
	Name        string
	Type        string
	Number      int
	JSONName    string
	Description string
	Repeated    bool
	EnumValues  []string
}

// ProtoEnum represents a proto3 enum definition
type ProtoEnum struct {
	Name        string
	Description string
	Values      []*ProtoEnumValue
	Reserved    []int // proto numbers retired via removal (rendered as `reserved N, M;`)
}

// ProtoEnumValue represents an enum value
type ProtoEnumValue struct {
	Name   string
	Number int
}

// BuildMessages processes all schemas and returns messages and dependency graph
func BuildMessages(entries []*parser.SchemaEntry, ctx *Context) (*internal.DependencyGraph, error) {
	graph := internal.NewDependencyGraph()

	// First pass: Add all schemas to graph and detect unions
	for _, entry := range entries {
		if err := graph.AddSchema(entry.Name, entry.Proxy); err != nil {
			return nil, err
		}

		schema := entry.Proxy.Schema()
		if schema == nil {
			continue
		}

		// Validate schema first
		if err := validateTopLevelSchema(schema, entry.Name); err != nil {
			return nil, err
		}

		// Detect oneOf and mark as union
		if len(schema.OneOf) > 0 {
			variants := internal.ExtractVariantNames(schema.OneOf)
			graph.MarkUnion(entry.Name, "contains oneOf", variants)
		}
	}

	// Second pass: Build messages and track dependencies
	for _, entry := range entries {
		schema := entry.Proxy.Schema()
		if schema == nil {
			continue
		}

		// Skip oneOf schemas for now (will be handled as Go code in later phases)
		if len(schema.OneOf) > 0 {
			continue
		}

		// Check if it's an enum schema
		if internal.IsEnumSchema(schema) {
			// Validate enum schema first
			if err := validateEnumSchema(schema, entry.Name); err != nil {
				return nil, err
			}

			// Check if it's a string enum - skip building protobuf enum
			if isStringEnum(schema) {
				continue
			}
			// Only build enum for integer enums
			_, err := buildEnum(entry.Name, entry.Proxy, ctx)
			if err != nil {
				return nil, err
			}
			continue
		}

		_, err := buildMessage(entry.Name, entry.Proxy, ctx, graph)
		if err != nil {
			return nil, err
		}
	}
	return graph, nil
}

// buildMessage creates a protoMessage from an OpenAPI schema
func buildMessage(name string, proxy *base.SchemaProxy, ctx *Context, graph *internal.DependencyGraph) (*ProtoMessage, error) {
	schema := proxy.Schema()
	if schema == nil {
		if err := proxy.GetBuildError(); err != nil {
			return nil, internal.SchemaError(name, fmt.Sprintf("failed to resolve schema: %v", err))
		}
		return nil, internal.SchemaError(name, "schema is nil")
	}

	// Check if it's an object type
	if len(schema.Type) == 0 || !internal.Contains(schema.Type, "object") {
		return nil, internal.SchemaError(name, "only objects and enums supported at top level")
	}

	// Validate field numbers before processing
	if err := validateFieldNumbers(schema, name); err != nil {
		return nil, err
	}

	msg := &ProtoMessage{
		Name:           ctx.Tracker.UniqueName(internal.ToPascalCase(name)),
		Description:    schema.Description,
		Fields:         []*ProtoField{},
		Nested:         []*ProtoMessage{},
		OriginalSchema: name,
	}

	// When explicit field numbers are supplied for this message, they fully drive
	// numbering and the reserved list; otherwise numbering stays positional.
	msgNums := messageNumbersFor(ctx, name)
	if msgNums != nil {
		msg.Reserved = msgNums.Reserved
	}

	fieldTracker := internal.NewNameTracker()

	// Process properties in YAML order
	if schema.Properties != nil {
		fieldNumber := 1
		for propName, propProxy := range schema.Properties.FromOldest() {
			propSchema := propProxy.Schema()
			if propSchema == nil {
				return nil, internal.PropertyError(name, propName, "has nil schema")
			}

			// Track dependency if property references another schema
			if propProxy.IsReference() {
				ref := propProxy.GetReference()
				parts := strings.Split(ref, "/")
				if len(parts) > 0 {
					refName := parts[len(parts)-1]
					if refName != "" {
						graph.AddDependency(name, refName)
					}
				}
			}

			// Track dependencies in array items
			if len(propSchema.Type) > 0 && internal.Contains(propSchema.Type, "array") {
				if propSchema.Items != nil && propSchema.Items.A != nil {
					itemProxy := propSchema.Items.A
					if itemProxy.IsReference() {
						ref := itemProxy.GetReference()
						parts := strings.Split(ref, "/")
						if len(parts) > 0 {
							refName := parts[len(parts)-1]
							if refName != "" {
								graph.AddDependency(name, refName)
							}
						}
					}
				}
			}

			sanitizedName, err := internal.SanitizeFieldName(propName)
			if err != nil {
				return nil, internal.PropertyError(name, propName, err.Error())
			}
			protoFieldName := fieldTracker.UniqueName(sanitizedName)
			protoType, repeated, enumValues, err := ProtoType(propSchema, propName, propProxy, ctx, msg)
			if err != nil {
				// Don't wrap with PropertyError if the error already contains the property name
				if strings.Contains(err.Error(), fmt.Sprintf("property '%s'", propName)) {
					return nil, fmt.Errorf("schema '%s': %w", name, err)
				}
				return nil, internal.PropertyError(name, propName, err.Error())
			}

			// For inline objects and integer enums, description goes to the nested type, not the field
			// For string enums, keep description on field (not hoisted)
			fieldDescription := propSchema.Description
			if len(propSchema.Type) > 0 && internal.Contains(propSchema.Type, "object") {
				fieldDescription = ""
			}
			if isIntegerEnum(propSchema) {
				fieldDescription = ""
			}

			// Field number priority: supplied FieldNumbers (by JSON name) override
			// everything; otherwise the x-proto-number extension; otherwise positional.
			customFieldNum, hasCustomNum, _ := extractFieldNumber(propProxy)
			actualFieldNumber := fieldNumber
			if msgNums != nil {
				num, ok := msgNums.Fields[propName]
				if !ok {
					return nil, internal.PropertyError(name, propName, "no proto field number mapped in FieldNumbers")
				}
				actualFieldNumber = num
			} else if hasCustomNum {
				actualFieldNumber = customFieldNum
			}

			field := &ProtoField{
				Name:        protoFieldName,
				Type:        protoType,
				Number:      actualFieldNumber,
				Description: fieldDescription,
				Repeated:    repeated,
				JSONName:    propName,
				EnumValues:  enumValues,
			}

			msg.Fields = append(msg.Fields, field)

			// Only advance the positional counter when positional numbering is active.
			if msgNums == nil && !hasCustomNum {
				fieldNumber++
			}
		}
	}

	// With supplied numbers, emit fields in number order so the proto is byte-identical
	// regardless of OpenAPI declaration order (a pure reorder is a no-op).
	if msgNums != nil {
		sortFieldsByNumber(msg.Fields)
	}

	ctx.Messages = append(ctx.Messages, msg)
	ctx.Definitions = append(ctx.Definitions, msg)
	return msg, nil
}

func sortFieldsByNumber(fields []*ProtoField) {
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].Number < fields[j].Number })
}

// messageNumbersFor returns the supplied number mapping for a message keyed by its
// OpenAPI schema name, or nil when none was supplied (positional numbering).
func messageNumbersFor(ctx *Context, schemaName string) *MessageNumbers {
	if ctx.FieldNumbers == nil {
		return nil
	}
	mn, ok := ctx.FieldNumbers.Messages[schemaName]
	if !ok {
		return nil
	}
	return &mn
}

// enumNumbersFor returns the supplied number mapping for an enum keyed by its
// OpenAPI schema name, or nil when none was supplied (positional numbering).
func enumNumbersFor(ctx *Context, schemaName string) *EnumNumbers {
	if ctx.FieldNumbers == nil {
		return nil
	}
	en, ok := ctx.FieldNumbers.Enums[schemaName]
	if !ok {
		return nil
	}
	return &en
}

// isStringEnum returns true if schema is a string enum
func isStringEnum(schema *base.Schema) bool {
	if schema == nil || len(schema.Enum) == 0 {
		return false
	}
	return len(schema.Type) > 0 && internal.Contains(schema.Type, "string")
}

// isIntegerEnum returns true if schema is an integer enum
func isIntegerEnum(schema *base.Schema) bool {
	if schema == nil || len(schema.Enum) == 0 {
		return false
	}
	return len(schema.Type) > 0 && internal.Contains(schema.Type, "integer")
}

// extractEnumValues extracts enum values as strings from schema
func extractEnumValues(schema *base.Schema) []string {
	if schema == nil || len(schema.Enum) == 0 {
		return []string{}
	}

	values := make([]string, 0, len(schema.Enum))
	for _, value := range schema.Enum {
		if value != nil {
			values = append(values, value.Value)
		}
	}
	return values
}

// validateEnumSchema validates enum schema and returns error for unsupported cases
func validateEnumSchema(schema *base.Schema, schemaName string) error {
	if schema == nil || len(schema.Enum) == 0 {
		return nil
	}

	// Check for explicit type field
	if len(schema.Type) == 0 {
		return fmt.Errorf("schema '%s': enum must have explicit type field", schemaName)
	}

	// Check for null values and mixed types
	var hasString, hasInteger bool
	for _, value := range schema.Enum {
		if value == nil || value.Value == "" {
			return fmt.Errorf("schema '%s': enum cannot contain null values", schemaName)
		}

		// Check if value looks like an integer
		if _, err := fmt.Sscanf(value.Value, "%d", new(int)); err == nil {
			hasInteger = true
		} else {
			hasString = true
		}
	}

	// Check for mixed types
	if hasString && hasInteger {
		return fmt.Errorf("schema '%s': enum contains mixed types (string and integer)", schemaName)
	}

	return nil
}

// extractFieldNumber extracts x-proto-number from schema proxy extensions
// Returns (number, true, nil) if found and valid
// Returns (0, false, nil) if not present
// Returns (0, false, error) if present but invalid format
func extractFieldNumber(proxy *base.SchemaProxy) (int, bool, error) {
	schema := proxy.Schema()
	if schema == nil || schema.Extensions == nil {
		return 0, false, nil
	}

	node, found := schema.Extensions.Get("x-proto-number")
	if !found || node == nil {
		return 0, false, nil
	}

	// Parse yaml.Node.Value string to int using strconv.Atoi
	// This properly rejects decimals like "3.14" unlike fmt.Sscanf
	num, err := strconv.Atoi(node.Value)
	if err != nil {
		return 0, false, fmt.Errorf("x-proto-number must be a valid integer, got: %s", node.Value)
	}

	return num, true, nil
}

// validateFieldNumbers validates x-proto-number extensions on schema properties
// Returns error if:
// - Field numbers are duplicated
// - Field numbers are out of valid range (1 to 536,870,911)
// - Field numbers use reserved range (19000-19999)
// - Field number is 0 (invalid)
// - Some but not all fields have x-proto-number (all-or-nothing violation)
func validateFieldNumbers(schema *base.Schema, schemaName string) error {
	if schema == nil || schema.Properties == nil {
		return nil
	}

	// Return nil if schema has 0 properties
	if schema.Properties.Len() == 0 {
		return nil
	}

	// First pass: check all-or-nothing rule
	totalProps := schema.Properties.Len()
	annotatedCount := 0
	for _, propProxy := range schema.Properties.FromOldest() {
		_, found, _ := extractFieldNumber(propProxy)
		if found {
			annotatedCount++
		}
	}

	// Enforce all-or-nothing: if any field has x-proto-number, all must have it
	if annotatedCount > 0 && annotatedCount < totalProps {
		return internal.SchemaError(schemaName, fmt.Sprintf("x-proto-number must be specified on all fields or none (found on %d of %d fields)", annotatedCount, totalProps))
	}

	// Track seen field numbers to detect duplicates
	seen := make(map[int]string)

	// Second pass: validate field number constraints
	for propName, propProxy := range schema.Properties.FromOldest() {
		// Extract field number
		fieldNum, found, err := extractFieldNumber(propProxy)
		if err != nil {
			return internal.PropertyError(schemaName, propName, err.Error())
		}

		// Skip properties without x-proto-number (all fields have none if we reach here)
		if !found {
			continue
		}

		// Validate field number constraints
		if fieldNum < 1 {
			return internal.PropertyError(schemaName, propName, "x-proto-number must be between 1 and 536870911")
		}

		if fieldNum > 536870911 {
			return internal.PropertyError(schemaName, propName, "x-proto-number must be between 1 and 536870911")
		}

		// Check reserved range (19000-19999)
		if fieldNum >= 19000 && fieldNum <= 19999 {
			return internal.PropertyError(schemaName, propName, fmt.Sprintf("x-proto-number %d is in reserved range 19000-19999", fieldNum))
		}

		// Check for duplicates
		if existingProp, exists := seen[fieldNum]; exists {
			return internal.SchemaError(schemaName, fmt.Sprintf("duplicate x-proto-number %d used by properties '%s' and '%s'", fieldNum, existingProp, propName))
		}

		seen[fieldNum] = propName
	}

	return nil
}

// buildEnum creates a protoEnum from an OpenAPI schema
func buildEnum(name string, proxy *base.SchemaProxy, ctx *Context) (*ProtoEnum, error) {
	schema := proxy.Schema()
	if schema == nil {
		if err := proxy.GetBuildError(); err != nil {
			return nil, internal.SchemaError(name, fmt.Sprintf("failed to resolve schema: %v", err))
		}
		return nil, internal.SchemaError(name, "schema is nil")
	}

	enumName := ctx.Tracker.UniqueName(internal.ToPascalCase(name))

	enum := &ProtoEnum{
		Name:        enumName,
		Description: schema.Description,
		Values:      []*ProtoEnumValue{},
	}

	// Numbers come from the supplied mapping (keyed by literal enum value) when
	// present; otherwise declaration order from 0. The first declared value maps to
	// 0 with no special case, satisfying proto3's zero-value requirement: callers are
	// expected to declare an *_UNSPECIFIED sentinel first. The library no longer
	// synthesizes an UNSPECIFIED value.
	enumNums := enumNumbersFor(ctx, name)
	if enumNums != nil {
		enum.Reserved = enumNums.Reserved
	}

	for i, value := range schema.Enum {
		// Extract the actual value from yaml.Node; Value holds the string form.
		var strValue string
		if value != nil {
			strValue = value.Value
		}
		number := i
		if enumNums != nil {
			num, ok := enumNums.Variants[strValue]
			if !ok {
				return nil, internal.SchemaError(name, fmt.Sprintf("enum value %q has no proto number mapped in FieldNumbers", strValue))
			}
			number = num
		}
		enum.Values = append(enum.Values, &ProtoEnumValue{
			Name:   internal.ToEnumValueName(enumName, strValue),
			Number: number,
		})
	}

	// With supplied numbers, emit variants in number order for a deterministic,
	// reorder-invariant proto.
	if enumNums != nil {
		sort.SliceStable(enum.Values, func(i, j int) bool { return enum.Values[i].Number < enum.Values[j].Number })
	}

	ctx.Enums = append(ctx.Enums, enum)
	ctx.Definitions = append(ctx.Definitions, enum)
	return enum, nil
}

// buildNestedMessage creates nested message from inline object property
func buildNestedMessage(propertyName string, proxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (*ProtoMessage, error) {
	schema := proxy.Schema()
	if schema == nil {
		if err := proxy.GetBuildError(); err != nil {
			return nil, fmt.Errorf("failed to resolve nested object: %w", err)
		}
		return nil, fmt.Errorf("nested object schema is nil")
	}

	// Validate property name is not plural
	// Simple check: error if ends with 's' or 'es' (no intelligent singularization)
	if strings.HasSuffix(propertyName, "es") {
		return nil, fmt.Errorf("cannot derive message name from property '%s'; use singular form or $ref", propertyName)
	}
	if strings.HasSuffix(propertyName, "s") {
		return nil, fmt.Errorf("cannot derive message name from property '%s'; use singular form or $ref", propertyName)
	}

	// Derive nested message name via PascalCase
	msgName := internal.ToPascalCase(propertyName)
	msgName = ctx.Tracker.UniqueName(msgName)

	// Validate field numbers before processing
	if err := validateFieldNumbers(schema, propertyName); err != nil {
		return nil, err
	}

	msg := &ProtoMessage{
		Name:           msgName,
		Description:    schema.Description,
		Fields:         []*ProtoField{},
		Nested:         []*ProtoMessage{},
		OriginalSchema: propertyName, // For nested messages, use property name
	}

	fieldTracker := internal.NewNameTracker()

	// Process properties in YAML order
	if schema.Properties != nil {
		fieldNumber := 1
		for propName, propProxy := range schema.Properties.FromOldest() {
			propSchema := propProxy.Schema()
			if propSchema == nil {
				return nil, fmt.Errorf("property '%s': has nil schema", propName)
			}

			sanitizedName, err := internal.SanitizeFieldName(propName)
			if err != nil {
				return nil, fmt.Errorf("property '%s': %w", propName, err)
			}
			protoFieldName := fieldTracker.UniqueName(sanitizedName)
			protoType, repeated, enumValues, err := ProtoType(propSchema, propName, propProxy, ctx, msg)
			if err != nil {
				// Don't wrap if the error already contains the property name
				if strings.Contains(err.Error(), fmt.Sprintf("property '%s'", propName)) {
					return nil, err
				}
				return nil, fmt.Errorf("property '%s': %w", propName, err)
			}

			// For inline objects and integer enums, description goes to the nested type, not the field
			// For string enums, keep description on field (not hoisted)
			fieldDescription := propSchema.Description
			if len(propSchema.Type) > 0 && internal.Contains(propSchema.Type, "object") {
				fieldDescription = ""
			}
			if isIntegerEnum(propSchema) {
				fieldDescription = ""
			}

			// Extract field number from x-proto-number extension if present
			customFieldNum, hasCustomNum, _ := extractFieldNumber(propProxy)
			actualFieldNumber := fieldNumber
			if hasCustomNum {
				actualFieldNumber = customFieldNum
			}

			field := &ProtoField{
				Name:        protoFieldName,
				Type:        protoType,
				Number:      actualFieldNumber,
				Description: fieldDescription,
				Repeated:    repeated,
				JSONName:    propName,
				EnumValues:  enumValues,
			}

			msg.Fields = append(msg.Fields, field)

			// Only increment auto-counter if we didn't use a custom number
			if !hasCustomNum {
				fieldNumber++
			}
		}
	}

	// Add to parent's nested messages
	if parentMsg != nil {
		parentMsg.Nested = append(parentMsg.Nested, msg)
	}

	return msg, nil
}

// validateTopLevelSchema checks for unsupported features at the schema level
func validateTopLevelSchema(schema *base.Schema, schemaName string) error {
	if schema == nil {
		return nil
	}

	// Check for schema composition features
	if len(schema.AllOf) > 0 {
		return internal.UnsupportedSchemaError(schemaName, "allOf")
	}

	if len(schema.AnyOf) > 0 {
		return internal.UnsupportedSchemaError(schemaName, "anyOf")
	}

	if len(schema.OneOf) > 0 {
		// Require at least 2 variants
		if len(schema.OneOf) < 2 {
			return fmt.Errorf("schema '%s': oneOf must have at least 2 variants", schemaName)
		}

		// Require discriminator
		if schema.Discriminator == nil || schema.Discriminator.PropertyName == "" {
			return fmt.Errorf("schema '%s': oneOf requires discriminator", schemaName)
		}

		// Require all variants to be $ref (no inline schemas)
		for i, variant := range schema.OneOf {
			if !variant.IsReference() {
				return fmt.Errorf("schema '%s': oneOf variant %d must use $ref, inline schemas not supported", schemaName, i)
			}
		}

		// Valid oneOf - will be handled as Go code
		return nil
	}

	if schema.Not != nil {
		return internal.UnsupportedSchemaError(schemaName, "not")
	}

	return nil
}
