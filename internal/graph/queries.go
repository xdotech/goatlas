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

// FindCallers returns functions that call the target function, using recursive
// CALLS edges in the Neo4j graph (variable-length paths up to depth).
// minConfidence filters edges below the threshold (0.0 = no filter).
// repo optionally restricts results to a single repository (empty = all repos).
func (q *Querier) FindCallers(ctx context.Context, functionName string, depth int, minConfidence float64, repo string) ([]CallerResult, error) {
	if depth <= 0 {
		depth = 5
	}

	// Build confidence filter clause
	confFilter := ""
	if minConfidence > 0 {
		confFilter = fmt.Sprintf("AND ALL(r IN relationships(path) WHERE r.confidence >= %f)", minConfidence)
	}

	repoFilter := ""
	if repo != "" {
		repoFilter = "AND ($repo = '' OR target.repo = $repo)"
	}

	// First try recursive CALLS-based query (includes IMPLEMENTS traversal)
	records, err := q.client.QueryNodes(ctx, fmt.Sprintf(`
		MATCH path = (caller:Function)-[:CALLS*1..%d]->(target:Function)
		WHERE (target.name = $name OR target.qualified CONTAINS $name)
		%s
		%s
		WITH caller, path,
		     REDUCE(minConf = 1.0, r IN relationships(path) | CASE WHEN r.confidence < minConf THEN r.confidence ELSE minConf END) AS pathConf
		RETURN DISTINCT caller.name AS name, caller.qualified AS qualified,
		       caller.file AS file, caller.line AS line,
		       length(path) AS depth, pathConf AS confidence
		UNION
		MATCH (target:Function)-[:IMPLEMENTS]->(iface:Type)<-[:IMPLEMENTS]-(impl:Function)
		WHERE (target.name = $name OR target.qualified CONTAINS $name)
		  AND ($repo = '' OR target.repo = $repo)
		WITH impl
		MATCH path = (caller:Function)-[:CALLS*1..%d]->(impl)
		WITH caller, path,
		     REDUCE(minConf = 1.0, r IN relationships(path) | CASE WHEN r.confidence < minConf THEN r.confidence ELSE minConf END) AS pathConf
		RETURN DISTINCT caller.name AS name, caller.qualified AS qualified,
		       caller.file AS file, caller.line AS line,
		       length(path) + 1 AS depth, pathConf AS confidence
		ORDER BY confidence DESC, depth, name
		LIMIT 50
	`, depth, confFilter, repoFilter, depth), map[string]any{"name": functionName, "repo": repo})
	if err != nil {
		return nil, err
	}

	var results []CallerResult
	for _, r := range records {
		cr := CallerResult{Depth: 1, Confidence: 0.5}
		if v, ok := r["name"].(string); ok {
			cr.Name = v
		}
		if v, ok := r["qualified"].(string); ok {
			cr.QualifiedName = v
		}
		if v, ok := r["file"].(string); ok {
			cr.File = v
		}
		if v, ok := r["line"].(int64); ok {
			cr.Line = int(v)
		}
		if v, ok := r["depth"].(int64); ok {
			cr.Depth = int(v)
		}
		if v, ok := r["confidence"].(float64); ok {
			cr.Confidence = v
		}
		results = append(results, cr)
	}

	// Fall back to simple name-match if no CALLS edges exist yet.
	// These are labeled as name-matches (not proven callers).
	if len(results) == 0 {
		fallbackRecords, err := q.client.QueryNodes(ctx, `
			MATCH (fn:Function)
			WHERE (fn.name = $name OR fn.qualified CONTAINS $name)
			  AND ($repo = '' OR fn.repo = $repo)
			RETURN fn.name AS name, fn.qualified AS qualified, fn.line AS line
			LIMIT 30
		`, map[string]any{"name": functionName, "repo": repo})
		if err != nil {
			return nil, err
		}
		for _, r := range fallbackRecords {
			cr := CallerResult{Depth: 1, Confidence: 0.3, IsNameMatch: true}
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
// Graph callers (CALLS edges) and name-match fallbacks are shown in separate sections.
func (q *Querier) FormatCallers(functionName string, callers []CallerResult) string {
	if len(callers) == 0 {
		return fmt.Sprintf("No callers found for: %s\n(Tip: run build_graph to enable graph-based caller search)", functionName)
	}

	var graphCallers, nameMatches []CallerResult
	for _, c := range callers {
		if c.IsNameMatch {
			nameMatches = append(nameMatches, c)
		} else {
			graphCallers = append(graphCallers, c)
		}
	}

	var sb strings.Builder
	if len(graphCallers) > 0 {
		sb.WriteString(fmt.Sprintf("Graph callers for %q (%d via CALLS edges):\n\n", functionName, len(graphCallers)))
		for _, c := range graphCallers {
			sb.WriteString(fmt.Sprintf("  [depth %d, conf %.2f] %s", c.Depth, c.Confidence, c.QualifiedName))
			if c.File != "" {
				sb.WriteString(fmt.Sprintf(" @ %s:%d", c.File, c.Line))
			}
			sb.WriteString("\n")
		}
	}
	if len(nameMatches) > 0 {
		if len(graphCallers) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Name matches for %q (%d — no graph; run build_graph for true callers):\n\n", functionName, len(nameMatches)))
		for _, c := range nameMatches {
			sb.WriteString(fmt.Sprintf("  %s", c.QualifiedName))
			if c.Line > 0 {
				sb.WriteString(fmt.Sprintf(" (line %d)", c.Line))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// AnalyzeImpact performs a transitive impact analysis — finds all callers,
// affected endpoints, and affected UI components for a given symbol.
// repo optionally restricts analysis to a single repository (empty = all repos).
func (q *Querier) AnalyzeImpact(ctx context.Context, symbol string, maxDepth int, repo string) (*ImpactReport, error) {
	report := &ImpactReport{}

	// 1. Find all transitive callers via CALLS edges (no confidence filter for impact — show all)
	callers, err := q.FindCallers(ctx, symbol, maxDepth, 0.0, repo)
	if err != nil {
		return nil, fmt.Errorf("find callers: %w", err)
	}
	report.Callers = callers

	// 2. Find affected API endpoints: callers that are handler functions
	//    connected to Endpoint nodes via HANDLES edge
	//    Also check if the target symbol itself is a handler
	allSymbols := []string{symbol}
	for _, c := range callers {
		if c.QualifiedName != "" {
			allSymbols = append(allSymbols, c.QualifiedName)
		}
		if c.Name != "" {
			allSymbols = append(allSymbols, c.Name)
		}
	}

	epRecords, err := q.client.QueryNodes(ctx, `
		UNWIND $names AS funcName
		MATCH (fn:Function)-[:HANDLES]->(ep:Endpoint)
		WHERE fn.name = funcName OR fn.qualified = funcName
		RETURN DISTINCT ep.method AS method, ep.path AS path
	`, map[string]any{"names": allSymbols})
	if err == nil {
		for _, r := range epRecords {
			ep := AffectedEndpoint{}
			if v, ok := r["method"].(string); ok {
				ep.Method = v
			}
			if v, ok := r["path"].(string); ok {
				ep.Path = v
			}
			report.AffectedEndpoints = append(report.AffectedEndpoints, ep)
		}
	}

	// 3. Find affected UI components that call the affected endpoints
	if len(report.AffectedEndpoints) > 0 {
		var paths []string
		for _, ep := range report.AffectedEndpoints {
			paths = append(paths, ep.Path)
		}

		compRecords, err := q.client.QueryNodes(ctx, `
			UNWIND $paths AS apiPath
			MATCH (comp:Component)-[:CALLS_API]->(ep:Endpoint)
			WHERE ep.path = apiPath
			RETURN DISTINCT comp.name AS name
		`, map[string]any{"paths": paths})
		if err == nil {
			for _, r := range compRecords {
				if v, ok := r["name"].(string); ok {
					report.AffectedComponents = append(report.AffectedComponents, v)
				}
			}
		}
	}

	return report, nil
}

