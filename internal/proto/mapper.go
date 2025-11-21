package proto

import (
	"fmt"
	"strings"

	"github.com/duh-rpc/openapi-schema.go/internal"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// ProtoType returns the proto3 type for an OpenAPI schema.
// Returns type name, whether it's repeated, enum values (for string enums), and error.
// For inline enums and objects, hoists them appropriately in the context.
// parentMsg is used for nested messages (can be nil for top-level).
func ProtoType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, bool, []string, error) {
	// Validate schema for unsupported features
	if err := validateSchema(schema, propertyName); err != nil {
		return "", false, nil, err
	}

	// Check if it's a reference first
	if propProxy.IsReference() {
		ref := propProxy.GetReference()

		// Try to resolve the reference (libopenapi handles internal refs automatically)
		resolvedSchema := propProxy.Schema()
		if resolvedSchema == nil {
			// Check if there's a build error (e.g., external reference)
			if err := propProxy.GetBuildError(); err != nil {
				return "", false, nil, fmt.Errorf("property '%s' references external file or unresolvable reference: %w", propertyName, err)
			}
			return "", false, nil, fmt.Errorf("property '%s' has unresolved reference", propertyName)
		}

		// Check if referenced schema is a string enum
		if isStringEnum(resolvedSchema) {
			enumValues := extractEnumValues(resolvedSchema)
			return "string", false, enumValues, nil
		}

		// Extract the schema name from the reference
		typeName, err := internal.ExtractReferenceName(ref)
		if err != nil {
			return "", false, nil, fmt.Errorf("property '%s': %w", propertyName, err)
		}
		return typeName, false, nil, nil
	}

	// Check if it's an array first
	if len(schema.Type) > 0 && internal.Contains(schema.Type, "array") {
		itemType, enumValues, err := ResolveArrayItemType(schema, propertyName, propProxy, ctx, parentMsg)
		if err != nil {
			return "", false, nil, err
		}
		return itemType, true, enumValues, nil
	}

	// Check if it's an inline object
	if len(schema.Type) > 0 && internal.Contains(schema.Type, "object") {
		// Build nested message
		nestedMsg, err := buildNestedMessage(propertyName, propProxy, ctx, parentMsg)
		if err != nil {
			return "", false, nil, err
		}
		return nestedMsg.Name, false, nil, nil
	}

	// Check if it's an enum
	if internal.IsEnumSchema(schema) {
		// Check if it's a string enum
		if isStringEnum(schema) {
			enumValues := extractEnumValues(schema)
			return "string", false, enumValues, nil
		}
		// Integer enum - hoist to top-level
		enumName := internal.ToPascalCase(propertyName)
		_, err := buildEnum(enumName, propProxy, ctx)
		if err != nil {
			return "", false, nil, err
		}
		return enumName, false, nil, nil
	}

	if len(schema.Type) == 0 {
		return "", false, nil, fmt.Errorf("property must have type or $ref")
	}

	var typ string
	if len(schema.Type) > 1 {
		nonNullTypes := []string{}
		for _, t := range schema.Type {
			if !strings.EqualFold(t, "null") {
				nonNullTypes = append(nonNullTypes, t)
			}
		}

		if len(nonNullTypes) != 1 {
			return "", false, nil, fmt.Errorf("multi-type properties not supported (only nullable variants allowed)")
		}

		typ = nonNullTypes[0]
	} else {
		typ = schema.Type[0]
	}
	format := schema.Format

	scalarType, err := MapScalarType(ctx, typ, format)
	return scalarType, false, nil, err
}

// MapScalarType maps OpenAPI type+format to proto3 scalar type.
func MapScalarType(ctx *Context, typ, format string) (string, error) {
	switch typ {
	case "integer":
		if format == "int64" {
			return "int64", nil
		}
		return "int32", nil

	case "number":
		if format == "float" {
			return "float", nil
		}
		return "double", nil

	case "string":
		if format == "date" || format == "date-time" {
			ctx.UsesTimestamp = true
			return "google.protobuf.Timestamp", nil
		}
		if format == "byte" || format == "binary" {
			return "bytes", nil
		}
		return "string", nil

	case "boolean":
		return "bool", nil

	default:
		return "", fmt.Errorf("unsupported type: %s", typ)
	}
}

// ResolveArrayItemType determines the proto3 type for array items.
// Returns type name, enum values (for string enums), and error.
// For inline objects/enums: validates property name is not plural.
func ResolveArrayItemType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *Context, parentMsg *ProtoMessage) (string, []string, error) {
	// Check if Items is defined
	if schema.Items == nil || schema.Items.A == nil {
		return "", nil, fmt.Errorf("array must have items defined")
	}

	itemsProxy := schema.Items.A
	itemsSchema := itemsProxy.Schema()
	if itemsSchema == nil {
		if err := itemsProxy.GetBuildError(); err != nil {
			return "", nil, fmt.Errorf("failed to resolve array items: %w", err)
		}
		return "", nil, fmt.Errorf("array items schema is nil")
	}

	// Check for nested arrays
	if len(itemsSchema.Type) > 0 && internal.Contains(itemsSchema.Type, "array") {
		return "", nil, fmt.Errorf("nested arrays not supported")
	}

	// Check if it's a reference
	if itemsProxy.IsReference() {
		ref := itemsProxy.GetReference()
		resolvedSchema := itemsProxy.Schema()
		if resolvedSchema != nil && isStringEnum(resolvedSchema) {
			enumValues := extractEnumValues(resolvedSchema)
			return "string", enumValues, nil
		}
		if ref != "" {
			// Extract the last segment of the reference path
			parts := strings.Split(ref, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1], nil, nil
			}
		}
		return "", nil, fmt.Errorf("invalid reference format")
	}

	// Check if it's an inline enum
	if internal.IsEnumSchema(itemsSchema) {
		// Check if it's a string enum
		if isStringEnum(itemsSchema) {
			enumValues := extractEnumValues(itemsSchema)
			return "string", enumValues, nil
		}
		// Integer enum - validate property name is not plural
		if strings.HasSuffix(propertyName, "es") {
			return "", nil, fmt.Errorf("cannot derive enum name from plural array property '%s'; use singular form or $ref", propertyName)
		}
		if strings.HasSuffix(propertyName, "s") {
			return "", nil, fmt.Errorf("cannot derive enum name from plural array property '%s'; use singular form or $ref", propertyName)
		}

		// Hoist inline integer enum to top-level
		enumName := internal.ToPascalCase(propertyName)
		_, err := buildEnum(enumName, itemsProxy, ctx)
		if err != nil {
			return "", nil, err
		}
		return enumName, nil, nil
	}

	// Check if it's an inline object
	if len(itemsSchema.Type) > 0 && internal.Contains(itemsSchema.Type, "object") {
		// Validate property name is not plural
		if strings.HasSuffix(propertyName, "es") {
			return "", nil, fmt.Errorf("cannot derive message name from plural array property '%s'; use singular form or $ref", propertyName)
		}
		if strings.HasSuffix(propertyName, "s") {
			return "", nil, fmt.Errorf("cannot derive message name from plural array property '%s'; use singular form or $ref", propertyName)
		}

		// Build nested message for inline object in array
		nestedMsg, err := buildNestedMessage(propertyName, itemsProxy, ctx, parentMsg)
		if err != nil {
			return "", nil, err
		}
		return nestedMsg.Name, nil, nil
	}

	// It's a scalar type
	if len(itemsSchema.Type) == 0 {
		return "", nil, fmt.Errorf("array items must have a type")
	}

	itemType := itemsSchema.Type[0]
	format := itemsSchema.Format
	scalarType, err := MapScalarType(ctx, itemType, format)
	return scalarType, nil, err
}

// validateSchema checks for unsupported OpenAPI features
func validateSchema(schema *base.Schema, propertyName string) error {
	if schema == nil {
		return nil
	}

	// Check for schema composition features
	if len(schema.AllOf) > 0 {
		return fmt.Errorf("property '%s' uses 'allOf' which is not supported", propertyName)
	}

	if len(schema.AnyOf) > 0 {
		return fmt.Errorf("property '%s' uses 'anyOf' which is not supported", propertyName)
	}

	if len(schema.OneOf) > 0 {
		// Require discriminator
		if schema.Discriminator == nil || schema.Discriminator.PropertyName == "" {
			return fmt.Errorf("oneOf in property '%s' requires discriminator", propertyName)
		}

		// Require all variants to be $ref (no inline schemas)
		for i, variant := range schema.OneOf {
			if !variant.IsReference() {
				return fmt.Errorf("oneOf variant %d in property '%s' must use $ref, inline schemas not supported", i, propertyName)
			}
		}

		// Valid oneOf - will be handled as Go code
		return nil
	}

	if schema.Not != nil {
		return fmt.Errorf("property '%s' uses 'not' which is not supported", propertyName)
	}

	return nil
}
