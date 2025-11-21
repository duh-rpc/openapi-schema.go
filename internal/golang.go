package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/duh-rpc/openapi-schema.go/internal/parser"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// GoStruct represents a Go struct definition with union metadata
type GoStruct struct {
	Name             string
	Description      string
	Fields           []*GoField
	IsUnion          bool
	UnionVariants    []string
	Discriminator    string
	DiscriminatorMap map[string]string // discriminator value -> type name (lowercase keys)
}

// GoField represents a struct field with Go type, JSON tag, pointer flag
type GoField struct {
	Name        string
	Type        string
	JSONName    string
	Description string
	IsPointer   bool
}

// GoContext holds state during Go code generation including package name
type GoContext struct {
	Tracker     *NameTracker
	Structs     []*GoStruct
	PackageName string
	NeedsTime   bool // Flag for time.Time import
}

// NewGoContext initializes empty context with package name
func NewGoContext(packageName string) *GoContext {
	return &GoContext{
		Tracker:     NewNameTracker(),
		Structs:     []*GoStruct{},
		PackageName: packageName,
		NeedsTime:   false,
	}
}

// BuildGoStructs processes schemas marked as Go-only, build GoStruct for each
func BuildGoStructs(entries []*parser.SchemaEntry, goTypes map[string]bool, graph *DependencyGraph, ctx *GoContext) error {
	// Build Go structs for all types marked as Go-only
	for _, entry := range entries {
		// Skip if not a Go type
		if !goTypes[entry.Name] {
			continue
		}

		goStruct, err := buildGoStruct(entry.Name, entry.Proxy, graph, ctx)
		if err != nil {
			return err
		}

		ctx.Structs = append(ctx.Structs, goStruct)
	}

	return nil
}

// buildGoStruct builds Go struct - if oneOf present, create union wrapper; otherwise regular struct
func buildGoStruct(name string, proxy *base.SchemaProxy, graph *DependencyGraph, ctx *GoContext) (*GoStruct, error) {
	schema := proxy.Schema()
	if schema == nil {
		return nil, fmt.Errorf("schema for '%s' is nil", name)
	}

	goStruct := &GoStruct{
		Name:        name,
		Description: schema.Description,
		Fields:      make([]*GoField, 0),
	}

	// Check if this is a union type (schema-level oneOf)
	if len(schema.OneOf) > 0 {
		// This is a union wrapper - create pointer fields for each variant
		goStruct.IsUnion = true
		goStruct.Discriminator = schema.Discriminator.PropertyName

		variants := extractVariantNames(schema.OneOf)
		goStruct.UnionVariants = variants

		// Build discriminator map with validation
		discriminatorMap, err := buildDiscriminatorMap(schema, variants, graph.schemas)
		if err != nil {
			return nil, err
		}
		goStruct.DiscriminatorMap = discriminatorMap

		// Create pointer field for each variant
		for _, variantName := range variants {
			goStruct.Fields = append(goStruct.Fields, &GoField{
				Name:      variantName,
				Type:      "*" + variantName, // Always pointer
				JSONName:  "-",               // Union types don't marshal fields directly
				IsPointer: false,             // Pointer already in Type string
			})
		}

		return goStruct, nil
	}

	// Regular struct - process properties
	if schema.Properties == nil {
		// Empty struct
		return goStruct, nil
	}

	for propName, propProxy := range schema.Properties.FromOldest() {
		// Get Go type for this property
		propSchema := propProxy.Schema()
		if propSchema == nil {
			return nil, fmt.Errorf("property '%s' in schema '%s' has nil schema", propName, name)
		}

		typeName, isPointer, err := goType(propSchema, propName, propProxy, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to map type for property '%s' in schema '%s': %w", propName, name, err)
		}

		// Convert property name to Go field name (PascalCase)
		fieldName := ToPascalCase(propName)

		goStruct.Fields = append(goStruct.Fields, &GoField{
			Name:        fieldName,
			Type:        typeName,
			JSONName:    propName, // Original OpenAPI property name
			Description: propSchema.Description,
			IsPointer:   isPointer, // Not used if Type already has *
		})
	}

	return goStruct, nil
}

// buildDiscriminatorMap builds map from discriminator values to type names
func buildDiscriminatorMap(schema *base.Schema, variants []string, schemas map[string]*base.SchemaProxy) (map[string]string, error) {
	mapping := make(map[string]string)
	discriminatorProp := schema.Discriminator.PropertyName

	// If explicit mapping exists, use it
	if schema.Discriminator != nil && !schema.Discriminator.Mapping.IsZero() {
		for value, ref := range schema.Discriminator.Mapping.FromOldest() {
			// Extract "Dog" from "#/components/schemas/Dog"
			typeName, err := extractReferenceName(ref)
			if err != nil {
				return nil, fmt.Errorf("failed to extract type name from discriminator mapping value '%s': %w", value, err)
			}

			// Check for conflicts (case-insensitive)
			lowerValue := strings.ToLower(value)
			if existing, exists := mapping[lowerValue]; exists && existing != typeName {
				return nil, fmt.Errorf("discriminator conflict: values '%s' and '%s' both map to lowercase '%s'",
					existing, value, lowerValue)
			}

			mapping[lowerValue] = typeName // Store lowercase for case-insensitive lookup
		}

		// Validate that all variants are covered by mapping
		for _, variant := range variants {
			found := false
			for _, mappedType := range mapping {
				if mappedType == variant {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("variant '%s' not covered by discriminator mapping", variant)
			}
		}

		return mapping, nil
	}

	// Otherwise, build case-insensitive mapping from variant names
	for _, variant := range variants {
		lowerVariant := strings.ToLower(variant)

		// Check for conflicts (e.g., "Dog" and "dog" both exist)
		if existing, exists := mapping[lowerVariant]; exists && existing != variant {
			return nil, fmt.Errorf("discriminator conflict: variants '%s' and '%s' both map to lowercase '%s'",
				existing, variant, lowerVariant)
		}

		mapping[lowerVariant] = variant // "dog" -> "Dog"
	}

	// Validate that discriminator property exists in all variant schemas
	for _, variant := range variants {
		variantProxy, exists := schemas[variant]
		if !exists {
			return nil, fmt.Errorf("variant '%s' not found in schemas", variant)
		}

		variantSchema := variantProxy.Schema()
		if variantSchema == nil {
			return nil, fmt.Errorf("variant '%s' has nil schema", variant)
		}

		// Check if discriminator property exists
		if variantSchema.Properties == nil {
			return nil, fmt.Errorf("discriminator property '%s' missing in variant '%s' (no properties)",
				discriminatorProp, variant)
		}

		hasDiscriminator := false
		for propName := range variantSchema.Properties.FromOldest() {
			if propName == discriminatorProp {
				hasDiscriminator = true
				break
			}
		}

		if !hasDiscriminator {
			return nil, fmt.Errorf("discriminator property '%s' missing in variant '%s'",
				discriminatorProp, variant)
		}
	}

	return mapping, nil
}

// goType maps OpenAPI type to Go type using type mapping table
func goType(schema *base.Schema, propertyName string, propProxy *base.SchemaProxy, ctx *GoContext) (string, bool, error) {
	// Check if it's a reference first
	if propProxy.IsReference() {
		ref := propProxy.GetReference()
		typeName, err := extractReferenceName(ref)
		if err != nil {
			return "", false, fmt.Errorf("property '%s': %w", propertyName, err)
		}
		// Objects/refs are always pointers in Go
		return "*" + typeName, false, nil
	}

	// Check if it's an array
	if len(schema.Type) > 0 && contains(schema.Type, "array") {
		arrayType, err := mapGoArrayType(schema, propProxy, ctx)
		if err != nil {
			return "", false, err
		}
		return arrayType, false, nil
	}

	// Check if it's an inline object
	if len(schema.Type) > 0 && contains(schema.Type, "object") {
		// For inline objects, derive type name from property name
		typeName := ToPascalCase(propertyName)
		return "*" + typeName, false, nil
	}

	// It's a scalar type
	if len(schema.Type) == 0 {
		return "", false, fmt.Errorf("property '%s' must have type or $ref", propertyName)
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
			return "", false, fmt.Errorf("property '%s' has multi-type which is not supported (only nullable variants allowed)", propertyName)
		}

		typ = nonNullTypes[0]
	} else {
		typ = schema.Type[0]
	}
	format := schema.Format

	scalarType, err := mapGoScalarType(typ, format, ctx)
	if err != nil {
		return "", false, err
	}

	return scalarType, false, nil
}

// mapGoScalarType maps OpenAPI scalars using type table
func mapGoScalarType(typ, format string, ctx *GoContext) (string, error) {
	switch typ {
	case "integer":
		switch format {
		case "int8":
			return "int8", nil
		case "int16":
			return "int16", nil
		case "int32":
			return "int32", nil
		case "int64":
			return "int64", nil
		case "uint8":
			return "uint8", nil
		case "uint16":
			return "uint16", nil
		case "uint32":
			return "uint32", nil
		case "uint64":
			return "uint64", nil
		case "int", "":
			return "int32", nil // Default to int32 for proto3 consistency
		default:
			return "", fmt.Errorf("unsupported integer format: %s", format)
		}

	case "number":
		switch format {
		case "float":
			return "float32", nil
		case "double", "":
			return "float64", nil // Default to float64 (double precision)
		default:
			return "", fmt.Errorf("unsupported number format: %s", format)
		}

	case "string":
		switch format {
		case "date", "date-time":
			ctx.NeedsTime = true
			return "time.Time", nil
		case "byte", "binary":
			return "[]byte", nil
		case "email", "uuid", "password", "":
			// Phase 1: All string formats map to string
			return "string", nil
		default:
			// Unknown format defaults to string
			return "string", nil
		}

	case "boolean":
		return "bool", nil

	default:
		return "", fmt.Errorf("unsupported type: %s", typ)
	}
}

// mapGoArrayType maps arrays to Go slices
func mapGoArrayType(schema *base.Schema, propProxy *base.SchemaProxy, ctx *GoContext) (string, error) {
	// Check if Items is defined
	if schema.Items == nil || schema.Items.A == nil {
		return "", fmt.Errorf("array must have items defined")
	}

	itemsProxy := schema.Items.A
	itemsSchema := itemsProxy.Schema()
	if itemsSchema == nil {
		if err := itemsProxy.GetBuildError(); err != nil {
			return "", fmt.Errorf("failed to resolve array items: %w", err)
		}
		return "", fmt.Errorf("array items schema is nil")
	}

	// Get element type
	elementType, _, err := goType(itemsSchema, "item", itemsProxy, ctx)
	if err != nil {
		return "", err
	}

	// Build slice type
	return "[]" + elementType, nil
}

// ExtractPackageName extracts package name from full Go package path
func ExtractPackageName(packagePath string) string {
	if packagePath == "" {
		return "main"
	}

	// Split by / to get path components
	parts := strings.Split(packagePath, "/")
	if len(parts) == 0 {
		return "main"
	}

	// Get last component
	last := parts[len(parts)-1]

	// Check if last component is version (v1, v2, etc.)
	if strings.HasPrefix(last, "v") && len(last) > 1 {
		// Try to parse as number
		if _, err := strconv.Atoi(last[1:]); err == nil {
			// It's a version, use second-to-last component if available
			if len(parts) > 1 {
				return parts[len(parts)-2]
			}
		}
	}

	return last
}
