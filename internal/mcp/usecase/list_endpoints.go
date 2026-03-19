package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// ListEndpointsUseCase lists detected API endpoints with optional filters.
type ListEndpointsUseCase struct {
	endpointRepo domain.EndpointRepository
}

// NewListEndpointsUseCase constructs a ListEndpointsUseCase.
func NewListEndpointsUseCase(er domain.EndpointRepository) *ListEndpointsUseCase {
	return &ListEndpointsUseCase{endpointRepo: er}
}

// Execute retrieves endpoints filtered by method and service, returning a formatted string.
func (uc *ListEndpointsUseCase) Execute(ctx context.Context, method, service string) (string, error) {
	endpoints, err := uc.endpointRepo.List(ctx, method, service)
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		msg := "No API endpoints found"
		if service != "" {
			msg += fmt.Sprintf(" for service %q", service)
		}
		return msg, nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d API endpoint(s):\n\n", len(endpoints)))
	for _, e := range endpoints {
		sb.WriteString(fmt.Sprintf("  %s %s", e.Method, e.Path))
		if e.HandlerName != "" {
			sb.WriteString(fmt.Sprintf(" -> %s", e.HandlerName))
		}
		if e.Framework != "" {
			sb.WriteString(fmt.Sprintf(" [%s]", e.Framework))
		}
		if e.FilePath != "" {
			sb.WriteString(fmt.Sprintf(" @ %s:%d", e.FilePath, e.Line))
		} else {
			sb.WriteString(fmt.Sprintf(" (line %d)", e.Line))
		}
		if e.RepoName != "" {
			sb.WriteString(fmt.Sprintf(" {%s}", e.RepoName))
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
