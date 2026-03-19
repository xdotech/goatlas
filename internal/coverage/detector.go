package coverage

import (
	"context"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Detector checks whether feature components are present in the indexed codebase.
type Detector struct {
	pool         *pgxpool.Pool
	symbolRepo   domain.SymbolRepository
	endpointRepo domain.EndpointRepository
}

// NewDetector creates a Detector backed by the given pool and repositories.
func NewDetector(pool *pgxpool.Pool, sr domain.SymbolRepository, er domain.EndpointRepository) *Detector {
	return &Detector{pool: pool, symbolRepo: sr, endpointRepo: er}
}

// DetectFeature checks all components of a feature and returns a FeatureResult.
func (d *Detector) DetectFeature(ctx context.Context, feature *Feature) (*FeatureResult, error) {
	result := &FeatureResult{Feature: *feature}

	for i := range feature.Backend {
		if err := d.detectComponent(ctx, &feature.Backend[i]); err != nil {
			return nil, err
		}
	}
	for i := range feature.Frontend {
		if err := d.detectComponent(ctx, &feature.Frontend[i]); err != nil {
			return nil, err
		}
	}

	result.BackendStatus = aggregateStatus(feature.Backend)
	result.FrontendStatus = aggregateStatus(feature.Frontend)
	result.OverallStatus = overallStatus(result.BackendStatus, result.FrontendStatus)
	result.Feature = *feature
	return result, nil
}

func (d *Detector) detectComponent(ctx context.Context, c *Component) error {
	switch c.Type {
	case "api_endpoint":
		return d.detectAPIEndpoint(ctx, c)
	case "service_method":
		return d.detectServiceMethod(ctx, c)
	case "ui_screen", "api_call":
		c.Status = StatusMissing
		c.Confidence = 0.3
		c.Evidence = []string{"Frontend detection requires full file indexing"}
		return nil
	default:
		c.Status = StatusMissing
		return nil
	}
}

func (d *Detector) detectAPIEndpoint(ctx context.Context, c *Component) error {
	parts := strings.SplitN(c.Identifier, " ", 2)
	if len(parts) != 2 {
		c.Status = StatusMissing
		return nil
	}
	method, path := parts[0], parts[1]

	// Replace path params for LIKE matching (e.g., {id} -> %)
	pathPattern := strings.ReplaceAll(path, "{", "%")
	pathPattern = strings.ReplaceAll(pathPattern, "}", "%")

	rows, err := d.pool.Query(ctx, `
		SELECT id, path, handler_name, framework FROM api_endpoints
		WHERE method = $1 AND path LIKE $2
		LIMIT 5
	`, method, pathPattern)
	if err != nil {
		return err
	}
	defer rows.Close()

	var evidence []string
	for rows.Next() {
		var id int64
		var epPath, handlerName, framework string
		if err := rows.Scan(&id, &epPath, &handlerName, &framework); err != nil {
			return err
		}
		evidence = append(evidence, handlerName+" ["+framework+"]")
	}

	if len(evidence) > 0 {
		c.Status = StatusImplemented
		c.Evidence = evidence
		c.Confidence = 0.9
	} else {
		c.Status = StatusMissing
		c.Confidence = 0.0
	}
	return nil
}

func (d *Detector) detectServiceMethod(ctx context.Context, c *Component) error {
	parts := strings.SplitN(c.Identifier, ".", 2)
	methodName := c.Identifier
	if len(parts) == 2 {
		methodName = parts[1]
	}

	symbols, err := d.symbolRepo.Search(ctx, methodName, 5, "method")
	if err != nil {
		return err
	}

	if len(symbols) > 0 {
		var evidence []string
		for _, s := range symbols {
			evidence = append(evidence, s.QualifiedName)
		}
		c.Status = StatusImplemented
		c.Evidence = evidence
		c.Confidence = 0.8
	} else {
		c.Status = StatusMissing
		c.Confidence = 0.0
	}
	return nil
}

func aggregateStatus(components []Component) Status {
	if len(components) == 0 {
		return StatusImplemented
	}
	implemented := 0
	for _, c := range components {
		if c.Status == StatusImplemented {
			implemented++
		}
	}
	if implemented == len(components) {
		return StatusImplemented
	}
	if implemented == 0 {
		return StatusMissing
	}
	return StatusPartial
}

func overallStatus(backend, frontend Status) Status {
	if backend == StatusImplemented && (frontend == StatusImplemented || frontend == StatusMissing) {
		return StatusImplemented
	}
	if backend == StatusMissing && frontend == StatusMissing {
		return StatusMissing
	}
	return StatusPartial
}
