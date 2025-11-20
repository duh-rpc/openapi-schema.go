package internal

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/duh-rpc/openapi-proto.go/internal/parser"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

// ExampleContext holds state during example generation
type ExampleContext struct {
	schemas        map[string]*parser.SchemaEntry // All available schemas (name + proxy)
	path           []string                       // Current path for circular detection (e.g., ["User", "Address"])
	depth          int                            // Current nesting depth
	maxDepth       int                            // Maximum allowed depth
	rand           *rand.Rand                     // Random number generator (seeded for determinism)
	fieldOverrides map[string]interface{}         // Field name to value overrides
}

// GenerateExamples generates JSON examples for specified schemas
func GenerateExamples(entries []*parser.SchemaEntry, schemaNames []string, maxDepth int, seed int64, fieldOverrides map[string]interface{}) (map[string]json.RawMessage, error) {
	schemaMap := make(map[string]*parser.SchemaEntry)
	for _, entry := range entries {
		schemaMap[entry.Name] = entry
	}

	ctx := &ExampleContext{
		schemas:        schemaMap,
		path:           make([]string, 0),
		depth:          0,
		maxDepth:       maxDepth,
		rand:           rand.New(rand.NewSource(seed)),
		fieldOverrides: fieldOverrides,
	}

	targetSchemas := entries
	if len(schemaNames) > 0 {
		targetSchemas = make([]*parser.SchemaEntry, 0, len(schemaNames))
		for _, name := range schemaNames {
			if entry, ok := schemaMap[name]; ok {
				targetSchemas = append(targetSchemas, entry)
			}
		}
	}

	result := make(map[string]json.RawMessage)
	for _, entry := range targetSchemas {
		ctx.path = make([]string, 0)
		ctx.depth = 0

		value, err := generateExample(entry.Name, entry.Proxy, ctx)
		if err != nil {
			return nil, err
		}

		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal example for %s: %w", entry.Name, err)
		}

		result[entry.Name] = json.RawMessage(jsonBytes)
	}

	return result, nil
}

// generateExample generates a JSON example for a single schema
func generateExample(name string, proxy *base.SchemaProxy, ctx *ExampleContext) (interface{}, error) {
	for _, p := range ctx.path {
		if p == name {
			return nil, nil
		}
	}

	if ctx.depth >= ctx.maxDepth {
		return nil, nil
	}

	ctx.path = append(ctx.path, name)
	defer func() {
		ctx.path = ctx.path[:len(ctx.path)-1]
	}()

	schema := proxy.Schema()
	if schema == nil {
		return nil, fmt.Errorf("schema %s is nil", name)
	}

	if proxy.IsReference() {
		ref := proxy.GetReference()
		refName, err := extractReferenceName(ref)
		if err != nil {
			return nil, err
		}

		entry, ok := ctx.schemas[refName]
		if !ok {
			return nil, fmt.Errorf("schema '%s' not found", refName)
		}

		return generateExample(refName, entry.Proxy, ctx)
	}

	if len(schema.Type) > 0 && contains(schema.Type, "array") {
		return generateArrayExample(schema, name, ctx)
	}

	if len(schema.Type) > 0 && contains(schema.Type, "object") {
		return generateObjectExample(schema, name, ctx)
	}

	if isEnumSchema(schema) {
		if len(schema.Enum) > 0 {
			return extractYAMLNodeValue(schema.Enum[0]), nil
		}
	}

	if len(schema.Type) == 0 {
		return nil, fmt.Errorf("schema must have type or $ref")
	}

	typ := schema.Type[0]
	format := schema.Format

	return generateScalarValue(name, schema, typ, format, ctx)
}

// generateScalarValue generates a value for a scalar type with constraints
func generateScalarValue(fieldName string, schema *base.Schema, typ, format string, ctx *ExampleContext) (interface{}, error) {
	if schema.Example != nil {
		return extractYAMLNodeValue(schema.Example), nil
	}

	if schema.Default != nil {
		return extractYAMLNodeValue(schema.Default), nil
	}

	// Check field overrides (after Example and Default, before type generation)
	if ctx.fieldOverrides != nil {
		if overrideValue, ok := ctx.fieldOverrides[fieldName]; ok {
			// Validate type matches schema type
			switch typ {
			case "integer":
				switch v := overrideValue.(type) {
				case int:
					return v, nil
				case float64:
					// JSON unmarshaling produces float64 for all numbers
					if math.Mod(v, 1.0) == 0 {
						return int(v), nil
					}
					return nil, fmt.Errorf("field override for '%s' has wrong type: expected integer, got float with decimal", fieldName)
				default:
					return nil, fmt.Errorf("field override for '%s' has wrong type: expected integer, got %T", fieldName, overrideValue)
				}
			case "number":
				switch v := overrideValue.(type) {
				case int:
					return float64(v), nil
				case float64:
					return v, nil
				default:
					return nil, fmt.Errorf("field override for '%s' has wrong type: expected number, got %T", fieldName, overrideValue)
				}
			case "string":
				if v, ok := overrideValue.(string); ok {
					return v, nil
				}
				return nil, fmt.Errorf("field override for '%s' has wrong type: expected string, got %T", fieldName, overrideValue)
			case "boolean":
				if v, ok := overrideValue.(bool); ok {
					return v, nil
				}
				return nil, fmt.Errorf("field override for '%s' has wrong type: expected boolean, got %T", fieldName, overrideValue)
			}
		}
	}

	switch typ {
	case "integer":
		min := 0
		max := 100
		if schema.Minimum != nil {
			min = int(*schema.Minimum)
		}
		if schema.Maximum != nil {
			max = int(*schema.Maximum)
		}

		if min > max {
			return nil, fmt.Errorf("invalid schema: minimum > maximum")
		}

		if schema.Minimum != nil || schema.Maximum != nil {
			return ctx.rand.Intn(max-min+1) + min, nil
		}
		return ctx.rand.Intn(100) + 1, nil

	case "number":
		min := 0.0
		max := 100.0
		if schema.Minimum != nil {
			min = *schema.Minimum
		}
		if schema.Maximum != nil {
			max = *schema.Maximum
		}

		if min > max {
			return nil, fmt.Errorf("invalid schema: minimum > maximum")
		}

		if schema.Minimum != nil || schema.Maximum != nil {
			return ctx.rand.Float64()*(max-min) + min, nil
		}
		return ctx.rand.Float64()*99.0 + 1.0, nil

	case "string":
		return generateStringValue(fieldName, schema, format, ctx)

	case "boolean":
		return ctx.rand.Intn(2) == 1, nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}
}

// generateStringValue generates string value honoring format and length constraints
func generateStringValue(fieldName string, schema *base.Schema, format string, ctx *ExampleContext) (string, error) {
	var minLength int
	var maxLength int

	if schema.MinLength != nil {
		minLength = int(*schema.MinLength)
	}

	if schema.MaxLength != nil {
		maxLength = int(*schema.MaxLength)
	}

	if minLength > 0 && maxLength > 0 && minLength > maxLength {
		return "", fmt.Errorf("invalid schema: minLength > maxLength")
	}

	lowerFieldName := strings.ToLower(fieldName)
	if lowerFieldName == "cursor" || lowerFieldName == "first" || lowerFieldName == "after" {
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/"
		length := ctx.rand.Intn(17) + 16
		result := make([]byte, length)
		for i := range result {
			result[i] = charset[ctx.rand.Intn(len(charset))]
		}
		return string(result), nil
	}

	if lowerFieldName == "error" {
		return "An error occurred", nil
	}

	if lowerFieldName == "message" {
		return "This is a message", nil
	}

	var template string

	switch format {
	case "email":
		template = "user@example.com"
	case "uuid":
		template = "123e4567-e89b-12d3-a456-426614174000"
	case "uri", "url":
		template = "https://example.com"
	case "date":
		template = "2024-01-15"
	case "date-time":
		template = "2024-01-15T10:30:00Z"
	case "hostname":
		template = "example.com"
	default:
		length := 10
		if minLength > 0 {
			if maxLength > 0 {
				length = ctx.rand.Intn(maxLength-minLength+1) + minLength
			} else {
				length = minLength
			}
		} else if maxLength > 0 {
			length = ctx.rand.Intn(maxLength + 1)
		}

		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		result := make([]byte, length)
		for i := range result {
			result[i] = charset[ctx.rand.Intn(len(charset))]
		}
		return string(result), nil
	}

	if minLength > 0 && len(template) < minLength {
		padding := minLength - len(template)
		for i := 0; i < padding; i++ {
			template += "x"
		}
	}

	if maxLength > 0 && len(template) > maxLength {
		template = template[:maxLength]
	}

	return template, nil
}

// generateArrayExample generates example for array schema
func generateArrayExample(schema *base.Schema, propertyName string, ctx *ExampleContext) ([]interface{}, error) {
	if schema.Items == nil || schema.Items.A == nil {
		return nil, fmt.Errorf("array must have items defined")
	}

	minItems := 1
	if schema.MinItems != nil && *schema.MinItems > 0 {
		minItems = int(*schema.MinItems)
	}

	maxItems := minItems
	if schema.MaxItems != nil {
		maxItems = int(*schema.MaxItems)
		if maxItems < minItems {
			return nil, fmt.Errorf("invalid schema: minItems > maxItems")
		}
	}

	if ctx.depth >= ctx.maxDepth {
		return []interface{}{}, nil
	}

	ctx.depth++
	defer func() {
		ctx.depth--
	}()

	// Generate random number of items between minItems and maxItems
	numItems := minItems
	if maxItems > minItems {
		numItems = ctx.rand.Intn(maxItems-minItems+1) + minItems
	}

	itemProxy := schema.Items.A
	result := make([]interface{}, 0, numItems)

	for i := 0; i < numItems; i++ {
		itemValue, err := generatePropertyValue(propertyName, itemProxy, ctx)
		if err != nil {
			return nil, err
		}
		if itemValue != nil {
			result = append(result, itemValue)
		}
	}

	return result, nil
}

// generateObjectExample generates example for object schema
func generateObjectExample(schema *base.Schema, name string, ctx *ExampleContext) (map[string]interface{}, error) {
	if ctx.depth >= ctx.maxDepth {
		return nil, nil
	}

	result := make(map[string]interface{})

	if schema.Properties == nil {
		return result, nil
	}

	ctx.depth++
	defer func() {
		ctx.depth--
	}()

	for propName, propProxy := range schema.Properties.FromOldest() {
		propValue, err := generatePropertyValue(propName, propProxy, ctx)
		if err != nil {
			return nil, err
		}

		if propValue != nil {
			result[propName] = propValue
		}
	}

	return result, nil
}

// generatePropertyValue generates example value for object property
func generatePropertyValue(propertyName string, propProxy *base.SchemaProxy, ctx *ExampleContext) (interface{}, error) {
	schema := propProxy.Schema()
	if schema == nil {
		return nil, fmt.Errorf("property %s has nil schema", propertyName)
	}

	if propProxy.IsReference() {
		ref := propProxy.GetReference()
		refName, err := extractReferenceName(ref)
		if err != nil {
			return nil, err
		}

		for _, p := range ctx.path {
			if p == refName {
				return nil, nil
			}
		}

		entry, ok := ctx.schemas[refName]
		if !ok {
			return nil, fmt.Errorf("schema '%s' not found", refName)
		}

		return generateExample(refName, entry.Proxy, ctx)
	}

	if len(schema.Type) > 0 && contains(schema.Type, "array") {
		return generateArrayExample(schema, propertyName, ctx)
	}

	if len(schema.Type) > 0 && contains(schema.Type, "object") {
		obj, err := generateObjectExample(schema, propertyName, ctx)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			return nil, nil
		}
		return obj, nil
	}

	if isEnumSchema(schema) {
		if len(schema.Enum) > 0 {
			return extractYAMLNodeValue(schema.Enum[0]), nil
		}
	}

	if len(schema.Type) == 0 {
		return nil, fmt.Errorf("property must have type or $ref")
	}

	typ := schema.Type[0]
	format := schema.Format

	return generateScalarValue(propertyName, schema, typ, format, ctx)
}

// extractYAMLNodeValue extracts the actual value from a yaml.Node
func extractYAMLNodeValue(node *yaml.Node) interface{} {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!int":
			if val, err := strconv.ParseInt(node.Value, 10, 64); err == nil {
				return int(val)
			}
		case "!!float":
			if val, err := strconv.ParseFloat(node.Value, 64); err == nil {
				return val
			}
		case "!!bool":
			if val, err := strconv.ParseBool(node.Value); err == nil {
				return val
			}
		case "!!str", "":
			return node.Value
		}
		return node.Value
	default:
		return node.Value
	}
}
