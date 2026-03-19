package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// GetComponentAPIsUseCase returns API endpoints called by a React component.
type GetComponentAPIsUseCase struct {
	cacRepo domain.ComponentAPICallRepository
}

// NewGetComponentAPIsUseCase constructs a GetComponentAPIsUseCase.
func NewGetComponentAPIsUseCase(cacRepo domain.ComponentAPICallRepository) *GetComponentAPIsUseCase {
	return &GetComponentAPIsUseCase{cacRepo: cacRepo}
}

// Execute finds all API calls made by the given component.
func (uc *GetComponentAPIsUseCase) Execute(ctx context.Context, component string) (string, error) {
	calls, err := uc.cacRepo.FindByComponent(ctx, component)
	if err != nil {
		return "", err
	}
	if len(calls) == 0 {
		return fmt.Sprintf("No API calls found for component %q", component), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Component %q makes %d API call(s):\n\n", component, len(calls))
	for _, c := range calls {
		method := c.HttpMethod
		if method == "" {
			method = "?"
		}
		fmt.Fprintf(&sb, "  %s %s\n", method, c.APIPath)
		if c.TargetService != "" {
			fmt.Fprintf(&sb, "    → service: %s\n", c.TargetService)
		}
		fmt.Fprintf(&sb, "    line: %d\n\n", c.Line)
	}
	return sb.String(), nil
}
