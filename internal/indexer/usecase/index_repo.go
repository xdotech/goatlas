package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/goatlas/goatlas/internal/indexer/parser"
)

// IndexResult holds statistics from a completed indexing run.
type IndexResult struct {
	FilesIndexed   int
	FilesSkipped   int
	SymbolsFound   int
	EndpointsFound int
	Duration       time.Duration
}

// IndexRepoUseCase walks a repository and indexes all Go source files.
type IndexRepoUseCase struct {
	fileRepo     domain.FileRepository
	symbolRepo   domain.SymbolRepository
	endpointRepo domain.EndpointRepository
	importRepo   domain.ImportRepository
}

// NewIndexRepoUseCase creates a new IndexRepoUseCase with the given repositories.
func NewIndexRepoUseCase(fr domain.FileRepository, sr domain.SymbolRepository, er domain.EndpointRepository, ir domain.ImportRepository) *IndexRepoUseCase {
	return &IndexRepoUseCase{fileRepo: fr, symbolRepo: sr, endpointRepo: er, importRepo: ir}
}

var skipDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, "testdata": true, ".idea": true,
}

// Execute walks repoPath and indexes every .go file. When force is false,
// files whose hash hasn't changed are skipped.
func (uc *IndexRepoUseCase) Execute(ctx context.Context, repoPath string, force bool) (*IndexResult, error) {
	start := time.Now()
	result := &IndexResult{}

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("repo path not found: %w", err)
	}

	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !isIndexableFile(path) {
			return nil
		}
		return uc.indexFile(ctx, absPath, path, force, result)
	})

	result.Duration = time.Since(start)
	return result, err
}

func (uc *IndexRepoUseCase) indexFile(ctx context.Context, repoPath, filePath string, force bool, result *IndexResult) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil // skip unreadable files
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	relPath, _ := filepath.Rel(repoPath, filePath)

	existing, err := uc.fileRepo.GetByPath(ctx, relPath)
	if err != nil {
		return err
	}

	if existing != nil && existing.Hash == hash && !force {
		result.FilesSkipped++
		return nil
	}

	parsed, err := parseByExtension(filePath)
	if err != nil {
		return nil // skip unparseable files
	}

	endpoints, _ := extractEndpointsByExtension(filePath, parsed.Imports)

	file := &domain.File{
		Path:   relPath,
		Module: parsed.Module,
		Hash:   hash,
	}

	if err := uc.fileRepo.Upsert(ctx, file); err != nil {
		return fmt.Errorf("upsert file %s: %w", relPath, err)
	}

	// Delete stale data when re-indexing an existing file
	if existing != nil {
		if err := uc.symbolRepo.DeleteByFileID(ctx, file.ID); err != nil {
			return err
		}
		if err := uc.endpointRepo.DeleteByFileID(ctx, file.ID); err != nil {
			return err
		}
		if err := uc.importRepo.DeleteByFileID(ctx, file.ID); err != nil {
			return err
		}
	}

	symbols := make([]domain.Symbol, len(parsed.Symbols))
	for i, s := range parsed.Symbols {
		symbols[i] = s
		symbols[i].FileID = file.ID
	}

	imports := make([]domain.Import, len(parsed.Imports))
	for i, imp := range parsed.Imports {
		imports[i] = imp
		imports[i].FileID = file.ID
	}

	eps := make([]domain.APIEndpoint, len(endpoints))
	for i, ep := range endpoints {
		eps[i] = ep
		eps[i].FileID = file.ID
	}

	if err := uc.symbolRepo.BulkInsert(ctx, symbols); err != nil {
		return fmt.Errorf("bulk insert symbols for %s: %w", relPath, err)
	}
	if err := uc.importRepo.BulkInsert(ctx, imports); err != nil {
		return fmt.Errorf("bulk insert imports for %s: %w", relPath, err)
	}
	if err := uc.endpointRepo.BulkInsert(ctx, eps); err != nil {
		return fmt.Errorf("bulk insert endpoints for %s: %w", relPath, err)
	}

	result.FilesIndexed++
	result.SymbolsFound += len(symbols)
	result.EndpointsFound += len(endpoints)
	return nil
}

var indexableExts = map[string]bool{
	".go":  true,
	".tsx": true,
	".ts":  true,
	".jsx": true,
	".js":  true,
}

func isIndexableFile(path string) bool {
	return indexableExts[filepath.Ext(path)]
}

func parseByExtension(filePath string) (*parser.ParseResult, error) {
	switch filepath.Ext(filePath) {
	case ".tsx", ".ts", ".jsx", ".js":
		return parser.ParseJSXFile(filePath)
	default:
		return parser.ParseFile(filePath)
	}
}

func extractEndpointsByExtension(filePath string, imports []domain.Import) ([]domain.APIEndpoint, error) {
	switch filepath.Ext(filePath) {
	case ".tsx", ".ts", ".jsx", ".js":
		return parser.ExtractReactRoutes(filePath, imports)
	default:
		return parser.ExtractRoutes(filePath, imports)
	}
}
