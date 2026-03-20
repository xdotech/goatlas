package graph

// PackageNode represents a Go package/module in the knowledge graph.
type PackageNode struct {
	Name   string
	Module string
}

// FileNode represents a Go source file in the knowledge graph.
type FileNode struct {
	Path    string
	Package string
}

// FunctionNode represents a function or method in the knowledge graph.
type FunctionNode struct {
	Name          string
	QualifiedName string
	Signature     string
	File          string
}

// TypeNode represents a named type (struct, interface, alias) in the knowledge graph.
type TypeNode struct {
	Name          string
	QualifiedName string
	Kind          string // struct|interface|type
	File          string
}

// ServiceDependency captures a package dependency edge.
type ServiceDependency struct {
	Service    string
	Dependency string
}

// APIHandlerResult holds resolved handler info for an endpoint pattern.
type APIHandlerResult struct {
	Method      string
	Path        string
	HandlerName string
	File        string
	Line        int
}

// CallerResult holds a function that calls (or references) a target function.
type CallerResult struct {
	Name          string
	QualifiedName string
	File          string
	Line          int
	Depth         int
	Confidence    float64
	IsNameMatch   bool // true when result is a name-match fallback (no CALLS edge in graph)
}

// AffectedEndpoint represents an API endpoint affected by a code change.
type AffectedEndpoint struct {
	Method string
	Path   string
}

// ImpactReport is the result of a change impact analysis.
type ImpactReport struct {
	Callers              []CallerResult
	AffectedEndpoints    []AffectedEndpoint
	AffectedComponents   []string
}
