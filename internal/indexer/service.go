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
	RepoRepo      domain.RepositoryRepository
	FileRepo      domain.FileRepository
	SymbolRepo    domain.SymbolRepository
	EndpointRepo  domain.EndpointRepository
	ImportRepo    domain.ImportRepository
	ConnRepo      domain.ServiceConnectionRepository
}

// NewService constructs a Service using the provided database pool.
func NewService(pool *pgxpool.Pool) *Service {
	repoRepo := postgres.NewRepoRepo(pool)
	fileRepo := postgres.NewFileRepo(pool)
	symbolRepo := postgres.NewSymbolRepo(pool)
	endpointRepo := postgres.NewEndpointRepo(pool)
	importRepo := postgres.NewImportRepo(pool)
	connRepo := postgres.NewServiceConnectionRepo(pool)

	return &Service{
		RepoRepo:      repoRepo,
		FileRepo:      fileRepo,
		SymbolRepo:    symbolRepo,
		EndpointRepo:  endpointRepo,
		ImportRepo:    importRepo,
		ConnRepo:      connRepo,
		IndexRepo:     usecase.NewIndexRepoUseCase(repoRepo, fileRepo, symbolRepo, endpointRepo, importRepo, connRepo),
		SearchSymbols: usecase.NewSearchSymbolsUseCase(symbolRepo),
	}
}
