package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// GetAPIConsumersUseCase returns UI components that call a given API endpoint.
type GetAPIConsumersUseCase struct {
	cacRepo domain.ComponentAPICallRepository
}

// NewGetAPIConsumersUseCase constructs a GetAPIConsumersUseCase.
func NewGetAPIConsumersUseCase(cacRepo domain.ComponentAPICallRepository) *GetAPIConsumersUseCase {
	return &GetAPIConsumersUseCase{cacRepo: cacRepo}
}

// Execute finds all components that call the given API path.
func (uc *GetAPIConsumersUseCase) Execute(ctx context.Context, apiPath, method string) (string, error) {
	calls, err := uc.cacRepo.FindByAPIPath(ctx, apiPath)
	if err != nil {
		return "", err
	}

	// Filter by method if specified
	if method != "" {
		method = strings.ToUpper(method)
		var filtered []domain.ComponentAPICall
		for _, c := range calls {
			if c.HttpMethod == method {
				filtered = append(filtered, c)
			}
		}
		calls = filtered
	}

	if len(calls) == 0 {
		return fmt.Sprintf("No UI consumers found for API %q", apiPath), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "API %q is called by %d component(s):\n\n", apiPath, len(calls))
	for _, c := range calls {
		fmt.Fprintf(&sb, "  [%s] %s\n", c.HttpMethod, c.Component)
		if c.TargetService != "" {
			fmt.Fprintf(&sb, "    service: %s\n", c.TargetService)
		}
		fmt.Fprintf(&sb, "    line: %d\n\n", c.Line)
	}
	return sb.String(), nil
}
