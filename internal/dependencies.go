package internal

import (
	"fmt"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// DependencyGraph tracks schema dependencies and union types for transitive closure computation
type DependencyGraph struct {
	schemas       map[string]*base.SchemaProxy
	edges         map[string][]string // from -> []to dependencies
	hasUnion      map[string]bool
	unionReasons  map[string]string
	unionVariants map[string][]string // union name -> variant names
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		schemas:       make(map[string]*base.SchemaProxy),
		edges:         make(map[string][]string),
		hasUnion:      make(map[string]bool),
		unionReasons:  make(map[string]string),
		unionVariants: make(map[string][]string),
	}
}

// AddSchema registers a schema in the graph
func (g *DependencyGraph) AddSchema(name string, proxy *base.SchemaProxy) error {
	g.schemas[name] = proxy
	return nil
}

// AddDependency records that 'from' schema references 'to' schema
func (g *DependencyGraph) AddDependency(from, to string) {
	if g.edges[from] == nil {
		g.edges[from] = make([]string, 0)
	}
	g.edges[from] = append(g.edges[from], to)
}

// MarkUnion marks a schema as containing a union with the given reason and variant names
func (g *DependencyGraph) MarkUnion(schemaName, reason string, variants []string) {
	g.hasUnion[schemaName] = true
	g.unionReasons[schemaName] = reason
	g.unionVariants[schemaName] = variants
}

// ComputeTransitiveClosure performs BFS to find all schemas that should be Go-only
// Returns goTypes (Go-only schemas), protoTypes (proto schemas), and reasons
func (g *DependencyGraph) ComputeTransitiveClosure() (goTypes, protoTypes map[string]bool, reasons map[string]string) {
	goTypes = make(map[string]bool)
	reasons = make(map[string]string)
	rootCause := make(map[string]string) // tracks root union type for each Go-only type
	visited := make(map[string]bool)

	// Mark direct union types
	for name, reason := range g.unionReasons {
		goTypes[name] = true
		reasons[name] = reason
		rootCause[name] = name // union types are their own root cause
		visited[name] = true
	}

	// Mark union variants
	for unionName, variants := range g.unionVariants {
		for _, variant := range variants {
			if !goTypes[variant] {
				goTypes[variant] = true
				reasons[variant] = fmt.Sprintf("variant of union type %s", unionName)
				rootCause[variant] = unionName // root cause is the union containing this variant
				visited[variant] = true
			}
		}
	}

	// BFS to find all types referencing Go-only types
	queue := make([]string, 0)
	for name := range goTypes {
		queue = append(queue, name)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find all types that depend on (reference) current
		for from, deps := range g.edges {
			if visited[from] {
				continue
			}

			// Check if 'from' references 'current'
			for _, to := range deps {
				if to == current {
					// Mark 'from' as Go-only because it references a Go-only type
					goTypes[from] = true
					// Use the root cause union type, not the immediate dependency
					unionType := rootCause[current]
					reasons[from] = fmt.Sprintf("references union type %s", unionType)
					rootCause[from] = unionType // propagate root cause
					visited[from] = true
					queue = append(queue, from)
					break
				}
			}
		}
	}

	// Proto types are everything else
	protoTypes = make(map[string]bool)
	for name := range g.schemas {
		if !goTypes[name] {
			protoTypes[name] = true
		}
	}

	return goTypes, protoTypes, reasons
}

// Schemas returns the schemas map for external package access
func (g *DependencyGraph) Schemas() map[string]*base.SchemaProxy {
	return g.schemas
}

// ExtractVariantNames extracts schema names from oneOf variant references
func ExtractVariantNames(oneOf []*base.SchemaProxy) []string {
	variants := make([]string, 0, len(oneOf))
	for _, variant := range oneOf {
		if variant.IsReference() {
			ref := variant.GetReference()
			// Use ExtractReferenceName for proper validation
			name, err := ExtractReferenceName(ref)
			if err == nil && name != "" {
				variants = append(variants, name)
			}
		}
	}
	return variants
}
