package indexer

import (
	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/goatlas/goatlas/internal/indexer/repository/postgres"
	"github.com/goatlas/goatlas/internal/indexer/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service is the top-level indexer facade wiring repositories and use-cases.
type Service struct {
	IndexRepo     *usecase.IndexRepoUseCase
	SearchSymbols *usecase.SearchSymbolsUseCase
	FileRepo      domain.FileRepository
	SymbolRepo    domain.SymbolRepository
	EndpointRepo  domain.EndpointRepository
	ImportRepo    domain.ImportRepository
}

// NewService constructs a Service using the provided database pool.
func NewService(pool *pgxpool.Pool) *Service {
	fileRepo := postgres.NewFileRepo(pool)
	symbolRepo := postgres.NewSymbolRepo(pool)
	endpointRepo := postgres.NewEndpointRepo(pool)
	importRepo := postgres.NewImportRepo(pool)

	return &Service{
		FileRepo:      fileRepo,
		SymbolRepo:    symbolRepo,
		EndpointRepo:  endpointRepo,
		ImportRepo:    importRepo,
		IndexRepo:     usecase.NewIndexRepoUseCase(fileRepo, symbolRepo, endpointRepo, importRepo),
		SearchSymbols: usecase.NewSearchSymbolsUseCase(symbolRepo),
	}
}
