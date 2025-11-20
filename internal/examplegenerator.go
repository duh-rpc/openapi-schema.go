package internal

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/duh-rpc/openapi-proto.go/internal/parser"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

// ExampleContext holds state during example generation
type ExampleContext struct {
	schemas  map[string]*parser.SchemaEntry // All available schemas (name + proxy)
	path     []string                       // Current path for circular detection (e.g., ["User", "Address"])
	depth    int                            // Current nesting depth
	maxDepth int                            // Maximum allowed depth
	rand     *rand.Rand                     // Random number generator (seeded for determinism)
}

// GenerateExamples generates JSON examples for specified schemas
func GenerateExamples(entries []*parser.SchemaEntry, schemaNames []string, maxDepth int, seed int64) (map[string]json.RawMessage, error) {
	schemaMap := make(map[string]*parser.SchemaEntry)
	for _, entry := range entries {
		schemaMap[entry.Name] = entry
	}

	ctx := &ExampleContext{
		schemas:  schemaMap,
		path:     make([]string, 0),
		depth:    0,
		maxDepth: maxDepth,
		rand:     rand.New(rand.NewSource(seed)),
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

	return generateScalarValue(schema, typ, format, ctx)
}

// generateScalarValue generates a value for a scalar type with constraints
func generateScalarValue(schema *base.Schema, typ, format string, ctx *ExampleContext) (interface{}, error) {
	if schema.Example != nil {
		return extractYAMLNodeValue(schema.Example), nil
	}

	if schema.Default != nil {
		return extractYAMLNodeValue(schema.Default), nil
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
		return 0, nil

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
		return 0.0, nil

	case "string":
		return "example", nil

	case "boolean":
		return ctx.rand.Intn(2) == 1, nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}
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

	itemProxy := schema.Items.A
	result := make([]interface{}, 0, minItems)

	for i := 0; i < minItems; i++ {
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

	return generateScalarValue(schema, typ, format, ctx)
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
