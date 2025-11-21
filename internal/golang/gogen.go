package golang

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/duh-rpc/openapi-schema.go/internal"
)

// GenerateGo produces Go source code from GoStruct IR with custom JSON marshaling
func GenerateGo(ctx *GoContext) ([]byte, error) {
	funcMap := template.FuncMap{
		"renderStruct": renderStruct,
	}

	tmpl, err := template.New("go").Funcs(funcMap).Parse(goTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go template: %w", err)
	}

	data := goTemplateData{
		PackageName: ctx.PackageName,
		Structs:     ctx.Structs,
		NeedsTime:   ctx.NeedsTime,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute Go template: %w", err)
	}

	return buf.Bytes(), nil
}

const goTemplate = `package {{.PackageName}}

import (
	"encoding/json"
	"fmt"
{{if .NeedsTime}}	"strings"
	"time"
{{else}}	"strings"
{{end}}
)
{{range .Structs}}
{{renderStruct .}}{{end}}
`

type goTemplateData struct {
	PackageName string
	Structs     []*GoStruct
	NeedsTime   bool
}

// renderStruct renders struct definition with fields, add MarshalJSON/UnmarshalJSON for unions
func renderStruct(s *GoStruct) string {
	var result strings.Builder

	// Add struct comment if present
	if s.Description != "" {
		result.WriteString(formatGoComment(s.Description, ""))
	}

	// Struct definition
	result.WriteString(fmt.Sprintf("type %s struct {\n", s.Name))

	// Render fields
	for _, field := range s.Fields {
		result.WriteString(renderField(field, "\t"))
	}

	result.WriteString("}\n")

	// Add custom marshaling for union types
	if s.IsUnion {
		result.WriteString("\n")
		result.WriteString(renderUnionMarshal(s))
		result.WriteString("\n")
		result.WriteString(renderUnionUnmarshal(s))
	}

	return result.String()
}

// renderField renders individual field with JSON tag and pointer notation
func renderField(f *GoField, indent string) string {
	var result strings.Builder

	// Add field comment if present
	if f.Description != "" {
		result.WriteString(formatGoComment(f.Description, indent))
	}

	result.WriteString(indent)
	result.WriteString(fmt.Sprintf("%s %s", f.Name, f.Type))

	// Add JSON tag
	if f.JSONName != "" {
		result.WriteString(fmt.Sprintf(" `json:\"%s\"`", f.JSONName))
	}

	result.WriteString("\n")

	return result.String()
}

// renderUnionMarshal generates MarshalJSON for union - check which variant is non-nil, marshal that variant
func renderUnionMarshal(s *GoStruct) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("func (u *%s) MarshalJSON() ([]byte, error) {\n", s.Name))

	// Count non-nil variants to ensure exactly one is set
	result.WriteString("\tcount := 0\n")
	for _, field := range s.Fields {
		result.WriteString(fmt.Sprintf("\tif u.%s != nil {\n", field.Name))
		result.WriteString("\t\tcount++\n")
		result.WriteString("\t}\n")
	}
	result.WriteString("\tif count > 1 {\n")
	result.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s: multiple variants set\")\n", s.Name))
	result.WriteString("\t}\n\n")

	// Check each variant pointer and marshal the non-nil one
	for _, field := range s.Fields {
		result.WriteString(fmt.Sprintf("\tif u.%s != nil {\n", field.Name))
		result.WriteString(fmt.Sprintf("\t\treturn json.Marshal(u.%s)\n", field.Name))
		result.WriteString("\t}\n")
	}

	// Error if no variant is set
	result.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s: no variant set\")\n", s.Name))
	result.WriteString("}\n")

	return result.String()
}

// renderUnionUnmarshal generates UnmarshalJSON for union - read discriminator, unmarshal into correct variant
func renderUnionUnmarshal(s *GoStruct) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("func (u *%s) UnmarshalJSON(data []byte) error {\n", s.Name))

	// Create anonymous struct to read discriminator
	discriminatorFieldName := internal.ToPascalCase(s.Discriminator)
	result.WriteString("\tvar discriminator struct {\n")
	result.WriteString(fmt.Sprintf("\t\t%s string `json:\"%s\"`\n", discriminatorFieldName, s.Discriminator))
	result.WriteString("\t}\n")

	result.WriteString("\tif err := json.Unmarshal(data, &discriminator); err != nil {\n")
	result.WriteString("\t\treturn err\n")
	result.WriteString("\t}\n\n")

	// Clear all variant pointers to maintain union invariant
	for _, field := range s.Fields {
		result.WriteString(fmt.Sprintf("\tu.%s = nil\n", field.Name))
	}
	result.WriteString("\n")

	// Switch on discriminator value (case-insensitive)
	result.WriteString(fmt.Sprintf("\tswitch strings.ToLower(discriminator.%s) {\n", discriminatorFieldName))

	// Generate case for each discriminator value
	for discValue, typeName := range s.DiscriminatorMap {
		result.WriteString(fmt.Sprintf("\tcase \"%s\":\n", discValue))
		result.WriteString(fmt.Sprintf("\t\tu.%s = &%s{}\n", typeName, typeName))
		result.WriteString(fmt.Sprintf("\t\treturn json.Unmarshal(data, u.%s)\n", typeName))
	}

	// Default case for unknown discriminator values
	result.WriteString("\tdefault:\n")
	result.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"unknown %s: %%s\", discriminator.%s)\n", s.Discriminator, discriminatorFieldName))
	result.WriteString("\t}\n")

	result.WriteString("}\n")

	return result.String()
}

// formatGoComment formats a description as a Go comment with indentation
func formatGoComment(description, indent string) string {
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
