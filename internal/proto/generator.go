package proto

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

const protoTemplate = `syntax = "proto3";

package {{.PackageName}};
{{if .UsesTimestamp}}
import "google/protobuf/timestamp.proto";
{{end}}
option go_package = "{{.GoPackage}}";
{{range .Definitions}}{{renderDefinition .}}{{end}}
`

type templateData struct {
	PackageName   string
	Messages      []*ProtoMessage
	Enums         []*ProtoEnum
	Definitions   []interface{}
	UsesTimestamp bool
	GoPackage     string
}

// Generate creates proto3 output from messages and enums in order
func Generate(packageName string, packagePath string, ctx *Context) ([]byte, error) {
	funcMap := template.FuncMap{
		"formatComment":    formatCommentForTemplate,
		"renderDefinition": renderDefinition,
	}

	tmpl, err := template.New("proto").Funcs(funcMap).Parse(protoTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	data := templateData{
		PackageName:   packageName,
		Messages:      ctx.Messages,
		Enums:         ctx.Enums,
		Definitions:   ctx.Definitions,
		UsesTimestamp: ctx.UsesTimestamp,
		GoPackage:     packagePath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// renderDefinition renders either an enum or message definition
func renderDefinition(def interface{}) string {
	switch d := def.(type) {
	case *ProtoEnum:
		return renderEnum(d)
	case *ProtoMessage:
		return renderMessage(d)
	default:
		return ""
	}
}

// renderEnum renders an enum definition
func renderEnum(enum *ProtoEnum) string {
	var result strings.Builder
	result.WriteString("\n")

	if enum.Description != "" {
		result.WriteString(formatCommentForTemplate(enum.Description))
	}

	result.WriteString(fmt.Sprintf("enum %s {\n", enum.Name))
	for _, value := range enum.Values {
		result.WriteString(fmt.Sprintf("  %s = %d;\n", value.Name, value.Number))
	}
	result.WriteString("}\n")

	return result.String()
}

// renderMessage renders a message definition
func renderMessage(msg *ProtoMessage) string {
	return renderMessageWithIndent(msg, "")
}

// renderMessageWithIndent renders a message definition with custom indentation
func renderMessageWithIndent(msg *ProtoMessage, indent string) string {
	var result strings.Builder
	result.WriteString("\n")

	if msg.Description != "" {
		result.WriteString(formatComment(msg.Description, indent))
	}

	result.WriteString(indent)
	result.WriteString(fmt.Sprintf("message %s {\n", msg.Name))

	// Render nested messages first (with proper indentation)
	for _, nested := range msg.Nested {
		nestedContent := renderMessageWithIndent(nested, indent+"  ")
		// Remove the leading newline from nested message since we're inside parent
		result.WriteString(strings.TrimPrefix(nestedContent, "\n"))
		result.WriteString("\n")
	}

	// Render fields
	for _, field := range msg.Fields {
		if field.Description != "" {
			result.WriteString(formatComment(field.Description, indent+"  "))
		}

		if len(field.EnumValues) > 0 {
			result.WriteString(formatEnumComment(field.EnumValues, indent+"  "))
		}

		result.WriteString(indent)
		result.WriteString("  ")
		if field.Repeated {
			result.WriteString("repeated ")
		}
		result.WriteString(fmt.Sprintf("%s %s = %d", field.Type, field.Name, field.Number))
		if field.JSONName != "" {
			result.WriteString(fmt.Sprintf(" [json_name = \"%s\"]", field.JSONName))
		}
		result.WriteString(";\n")
	}

	result.WriteString(indent)
	result.WriteString("}\n")

	return result.String()
}

// formatCommentForTemplate formats a description as a proto3 comment for use in templates
func formatCommentForTemplate(description string) string {
	return formatComment(description, "")
}

// formatComment formats a description as a proto3 comment with indentation
func formatComment(description, indent string) string {
	if strings.TrimSpace(description) == "" {
		return ""
	}

	lines := strings.Split(description, "\n")
	var result strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		result.WriteString(indent)
		if trimmed == "" {
			result.WriteString("//\n")
		} else {
			result.WriteString("// ")
			result.WriteString(trimmed)
			result.WriteString("\n")
		}
	}

	return result.String()
}

// formatEnumComment formats enum values as a proto3 comment
func formatEnumComment(values []string, indent string) string {
	if len(values) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString(indent)
	result.WriteString("// enum: [")
	for i, value := range values {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(value)
	}
	result.WriteString("]\n")
	return result.String()
}
