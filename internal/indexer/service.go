package indexer

import (
	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/xdotech/goatlas/internal/indexer/repository/postgres"
	"github.com/xdotech/goatlas/internal/indexer/usecase"
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
	CACRepo       domain.ComponentAPICallRepository
	FCRepo        domain.FunctionCallRepository
	TURepo        domain.TypeUsageRepository
	IIRepo        domain.InterfaceImplRepository
	ProcessRepo   domain.ProcessRepository
	CommunityRepo domain.CommunityRepository
}

// NewService constructs a Service using the provided database pool.
func NewService(pool *pgxpool.Pool) *Service {
	repoRepo := postgres.NewRepoRepo(pool)
	fileRepo := postgres.NewFileRepo(pool)
	symbolRepo := postgres.NewSymbolRepo(pool)
	endpointRepo := postgres.NewEndpointRepo(pool)
	importRepo := postgres.NewImportRepo(pool)
	connRepo := postgres.NewServiceConnectionRepo(pool)
	cacRepo := postgres.NewComponentAPICallRepo(pool)
	fcRepo := postgres.NewFunctionCallRepo(pool)
	tuRepo := postgres.NewTypeUsageRepo(pool)
	iiRepo := postgres.NewInterfaceImplRepo(pool)
	processRepo := postgres.NewProcessRepo(pool)
	communityRepo := postgres.NewCommunityRepo(pool)

	return &Service{
		RepoRepo:      repoRepo,
		FileRepo:      fileRepo,
		SymbolRepo:    symbolRepo,
		EndpointRepo:  endpointRepo,
		ImportRepo:    importRepo,
		ConnRepo:      connRepo,
		CACRepo:       cacRepo,
		FCRepo:        fcRepo,
		TURepo:        tuRepo,
		IIRepo:        iiRepo,
		ProcessRepo:   processRepo,
		CommunityRepo: communityRepo,
		IndexRepo:     usecase.NewIndexRepoUseCase(repoRepo, fileRepo, symbolRepo, endpointRepo, importRepo, connRepo, cacRepo, fcRepo, tuRepo, iiRepo),
		SearchSymbols: usecase.NewSearchSymbolsUseCase(symbolRepo),
	}
}

