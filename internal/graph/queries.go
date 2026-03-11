package graph

import (
	"context"
	"fmt"
	"strings"
)

// Querier provides higher-level graph query operations over the Neo4j client.
type Querier struct {
	client *Client
}

// NewQuerier constructs a Querier.
func NewQuerier(client *Client) *Querier {
	return &Querier{client: client}
}

// GetServiceDependencies returns the packages imported by files belonging to service.
func (q *Querier) GetServiceDependencies(ctx context.Context, service string) ([]string, error) {
	records, err := q.client.QueryNodes(ctx, `
		MATCH (pkg:Package {name: $service})-[:DEFINES]->(f:File)-[:IMPORTS]->(dep:Package)
		WHERE dep.name <> $service
		RETURN DISTINCT dep.name AS dependency
		ORDER BY dep.name
	`, map[string]any{"service": service})
	if err != nil {
		return nil, err
	}

	var deps []string
	for _, r := range records {
		if dep, ok := r["dependency"].(string); ok {
			deps = append(deps, dep)
		}
	}
	return deps, nil
}

// GetAPIHandlers finds functions whose name or qualified name contains the pattern.
func (q *Querier) GetAPIHandlers(ctx context.Context, endpointPattern string) ([]APIHandlerResult, error) {
	records, err := q.client.QueryNodes(ctx, `
		MATCH (fn:Function)
		WHERE fn.name CONTAINS $pattern OR fn.qualified CONTAINS $pattern
		RETURN fn.name AS name, fn.qualified AS qualified, fn.line AS line
		LIMIT 20
	`, map[string]any{"pattern": endpointPattern})
	if err != nil {
		return nil, err
	}

	var results []APIHandlerResult
	for _, r := range records {
		res := APIHandlerResult{}
		if v, ok := r["name"].(string); ok {
			res.HandlerName = v
		}
		if v, ok := r["qualified"].(string); ok {
			res.Path = v
		}
		if v, ok := r["line"].(int64); ok {
			res.Line = int(v)
		}
		results = append(results, res)
	}
	return results, nil
}

// FindCallers returns functions whose name or qualified name matches functionName.
// depth is accepted for API compatibility but the current Cypher is non-recursive.
func (q *Querier) FindCallers(ctx context.Context, functionName string, depth int) ([]CallerResult, error) {
	records, err := q.client.QueryNodes(ctx, `
		MATCH (fn:Function)
		WHERE fn.name = $name OR fn.qualified CONTAINS $name
		RETURN fn.name AS name, fn.qualified AS qualified, fn.line AS line
		LIMIT 30
	`, map[string]any{"name": functionName})
	if err != nil {
		return nil, err
	}

	var results []CallerResult
	for _, r := range records {
		cr := CallerResult{Depth: 1}
		if v, ok := r["name"].(string); ok {
			cr.Name = v
		}
		if v, ok := r["qualified"].(string); ok {
			cr.QualifiedName = v
		}
		if v, ok := r["line"].(int64); ok {
			cr.Line = int(v)
		}
		results = append(results, cr)
	}
	return results, nil
}

// FormatServiceDependencies renders dependency list as human-readable text.
func (q *Querier) FormatServiceDependencies(service string, deps []string) string {
	if len(deps) == 0 {
		return fmt.Sprintf("No dependencies found for service: %s", service)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dependencies for %s:\n\n", service))
	for _, dep := range deps {
		sb.WriteString(fmt.Sprintf("  - %s\n", dep))
	}
	return sb.String()
}

// FormatCallers renders caller results as human-readable text.
func (q *Querier) FormatCallers(functionName string, callers []CallerResult) string {
	if len(callers) == 0 {
		return fmt.Sprintf("No callers found for: %s", functionName)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Graph callers/references for %q:\n\n", functionName))
	for _, c := range callers {
		sb.WriteString(fmt.Sprintf("  [depth %d] %s", c.Depth, c.QualifiedName))
		if c.File != "" {
			sb.WriteString(fmt.Sprintf(" @ %s:%d", c.File, c.Line))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
