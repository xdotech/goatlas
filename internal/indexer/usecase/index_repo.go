package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/xdotech/goatlas/internal/indexer/parser"
)

// IndexRepoUseCase indexes (or re-indexes) a repository.
type IndexRepoUseCase struct {
	repoRepo   domain.RepositoryRepository
	fileRepo   domain.FileRepository
	symbolRepo domain.SymbolRepository
	epRepo     domain.EndpointRepository
	importRepo domain.ImportRepository
	connRepo   domain.ServiceConnectionRepository
	cacRepo    domain.ComponentAPICallRepository
	fcRepo     domain.FunctionCallRepository
	tuRepo     domain.TypeUsageRepository
	iiRepo     domain.InterfaceImplRepository
}

// NewIndexRepoUseCase constructs a new IndexRepoUseCase.
func NewIndexRepoUseCase(
	repoRepo domain.RepositoryRepository,
	fileRepo domain.FileRepository,
	symbolRepo domain.SymbolRepository,
	epRepo domain.EndpointRepository,
	importRepo domain.ImportRepository,
	connRepo domain.ServiceConnectionRepository,
	cacRepo domain.ComponentAPICallRepository,
	fcRepo domain.FunctionCallRepository,
	tuRepo domain.TypeUsageRepository,
	iiRepo domain.InterfaceImplRepository,
) *IndexRepoUseCase {
	return &IndexRepoUseCase{
		repoRepo:   repoRepo,
		fileRepo:   fileRepo,
		symbolRepo: symbolRepo,
		epRepo:     epRepo,
		importRepo: importRepo,
		connRepo:   connRepo,
		cacRepo:    cacRepo,
		fcRepo:     fcRepo,
		tuRepo:     tuRepo,
		iiRepo:     iiRepo,
	}
}

// Execute walks the repository at absPath, parses files, and stores results.
// When incremental is true and force is false, only files changed since the last
// indexed commit are processed.
func (uc *IndexRepoUseCase) Execute(ctx context.Context, absPath string, force, incremental bool) (map[string]any, error) {
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	repoName := filepath.Base(absPath)

	// Load detection patterns (embedded defaults + catalog + overrides)
	patternCfg, _ := parser.LoadPatterns(absPath)

	registry := parser.NewRegistry()

	// Upsert the repository record
	repo := &domain.Repository{
		Name: repoName,
		Path: absPath,
	}
	if err := uc.repoRepo.Upsert(ctx, repo); err != nil {
		return nil, fmt.Errorf("upsert repository: %w", err)
	}

	slog.Info("indexing repository", "name", repoName, "id", repo.ID)

	// Clear cross-service connections for this repo (re-detected each index)
	if err := uc.connRepo.DeleteByRepoID(ctx, repo.ID); err != nil {
		return nil, fmt.Errorf("clear connections: %w", err)
	}

	stats := map[string]int{
		"files_indexed": 0,
		"files_skipped": 0,
		"symbols":       0,
		"endpoints":     0,
		"imports":       0,
		"connections":   0,
	}

	var allConns []domain.ServiceConnection
	var allCAC []domain.ComponentAPICall

	// Get current HEAD commit for incremental logic and final update.
	headCommit, _ := GetHeadCommit(absPath)

	// Determine which files to process (nil = full walk).
	type fileAction struct {
		path    string
		deleted bool
	}
	var targetFiles []fileAction

	if incremental && !force && repo.LastCommit != "" && headCommit != "" && headCommit != repo.LastCommit {
		if commitExists(absPath, repo.LastCommit) {
			added, modified, deleted, diffErr := GetChangedFiles(absPath, repo.LastCommit, headCommit)
			if diffErr == nil {
				for _, p := range added {
					targetFiles = append(targetFiles, fileAction{path: p})
				}
				for _, p := range modified {
					targetFiles = append(targetFiles, fileAction{path: p})
				}
				for _, p := range deleted {
					targetFiles = append(targetFiles, fileAction{path: p, deleted: true})
				}
				slog.Info("incremental index", "changed", len(added)+len(modified), "deleted", len(deleted))
			}
		}
	}

	if targetFiles != nil {
		for _, fa := range targetFiles {
			if fa.deleted {
				if err := uc.fileRepo.DeleteByPath(ctx, repo.ID, fa.path); err != nil {
					slog.Warn("incremental: delete file failed", "path", fa.path, "error", err)
				}
				continue
			}
			ext := strings.ToLower(filepath.Ext(fa.path))
			handler := registry.HandlerFor(ext)
			if handler == nil {
				continue
			}
			if indexErr := uc.indexFile(ctx, repo.ID, absPath, fa.path, handler, patternCfg, force, stats, &allConns, &allCAC); indexErr != nil {
				slog.Warn("incremental: index file failed", "path", fa.path, "error", indexErr)
			}
		}
	} else {
		err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := info.Name()
				if base == "vendor" || base == "node_modules" || base == ".git" || base == "testdata" ||
					base == "__pycache__" || base == ".venv" || base == "venv" || base == ".tox" || base == ".eggs" {
					return filepath.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			handler := registry.HandlerFor(ext)
			if handler == nil {
				return nil
			}

			relPath, _ := filepath.Rel(absPath, path)
			return uc.indexFile(ctx, repo.ID, absPath, relPath, handler, patternCfg, force, stats, &allConns, &allCAC)
		})
		if err != nil {
			return nil, fmt.Errorf("walk: %w", err)
		}
	}

	// Bulk insert all detected connections
	if len(allConns) > 0 {
		if err := uc.connRepo.BulkInsert(ctx, allConns); err != nil {
			return nil, fmt.Errorf("insert connections: %w", err)
		}
		stats["connections"] = len(allConns)
	}

	// Bulk insert component API calls
	if len(allCAC) > 0 {
		if err := uc.cacRepo.BulkInsert(ctx, allCAC); err != nil {
			return nil, fmt.Errorf("insert component api calls: %w", err)
		}
	}

	// Update repo last indexed time
	now := time.Now()
	repo.LastIndexedAt = &now
	_ = uc.repoRepo.Upsert(ctx, repo)

	// Persist current HEAD commit for incremental runs
	if headCommit != "" {
		_ = uc.repoRepo.UpdateLastCommit(ctx, repo.ID, headCommit)
	}

	result := map[string]any{
		"status":     "ok",
		"repository": repoName,
		"repo_id":    repo.ID,
	}
	for k, v := range stats {
		result[k] = v
	}
	return result, nil
}

// indexFile processes a single source file via the appropriate language handler.
func (uc *IndexRepoUseCase) indexFile(
	ctx context.Context,
	repoID int64,
	absPath, relPath string,
	handler parser.LanguageHandler,
	cfg *parser.PatternConfig,
	force bool,
	stats map[string]int,
	allConns *[]domain.ServiceConnection,
	allCAC *[]domain.ComponentAPICall,
) error {
	fullPath := filepath.Join(absPath, relPath)

	hash, err := hashFile(fullPath)
	if err != nil {
		return nil
	}

	existing, _ := uc.fileRepo.GetByPath(ctx, repoID, relPath)
	if existing != nil && existing.Hash == hash && !force {
		stats["files_skipped"]++
		return nil
	}

	fileRec := &domain.File{
		RepoID: repoID,
		Path:   relPath,
		Hash:   hash,
	}
	if err := uc.fileRepo.Upsert(ctx, fileRec); err != nil {
		return fmt.Errorf("upsert file %s: %w", relPath, err)
	}

	// Clear old data for re-index
	_ = uc.symbolRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.epRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.importRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.fcRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.tuRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.iiRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.cacRepo.DeleteByFileID(ctx, fileRec.ID)

	// Parse via language handler
	parsed, err := handler.Parse(ctx, fullPath, cfg)
	if err != nil {
		slog.Warn("parse failed", "file", relPath, "lang", handler.Language(), "error", err)
		return nil
	}

	// Assign FileID to all results
	for i := range parsed.Symbols {
		parsed.Symbols[i].FileID = fileRec.ID
	}
	for i := range parsed.Imports {
		parsed.Imports[i].FileID = fileRec.ID
	}
	for i := range parsed.Endpoints {
		parsed.Endpoints[i].FileID = fileRec.ID
	}
	for i := range parsed.FuncCalls {
		parsed.FuncCalls[i].FileID = fileRec.ID
	}
	for i := range parsed.TypeUsages {
		parsed.TypeUsages[i].FileID = fileRec.ID
	}
	for i := range parsed.IfaceImpls {
		parsed.IfaceImpls[i].FileID = fileRec.ID
	}
	for i := range parsed.CompAPICalls {
		parsed.CompAPICalls[i].FileID = fileRec.ID
	}

	// Bulk insert
	if len(parsed.Symbols) > 0 {
		if err := uc.symbolRepo.BulkInsert(ctx, parsed.Symbols); err != nil {
			return fmt.Errorf("insert symbols: %w", err)
		}
		stats["symbols"] += len(parsed.Symbols)
	}
	if len(parsed.Imports) > 0 {
		if err := uc.importRepo.BulkInsert(ctx, parsed.Imports); err != nil {
			return fmt.Errorf("insert imports: %w", err)
		}
		stats["imports"] += len(parsed.Imports)
	}
	if len(parsed.Endpoints) > 0 {
		if err := uc.epRepo.BulkInsert(ctx, parsed.Endpoints); err != nil {
			return fmt.Errorf("insert endpoints: %w", err)
		}
		stats["endpoints"] += len(parsed.Endpoints)
	}
	if len(parsed.FuncCalls) > 0 {
		if err := uc.fcRepo.BulkInsert(ctx, parsed.FuncCalls); err != nil {
			slog.Warn("insert function calls failed", "file", relPath, "error", err)
		}
		stats["function_calls"] += len(parsed.FuncCalls)
	}
	if len(parsed.TypeUsages) > 0 {
		if err := uc.tuRepo.BulkInsert(ctx, parsed.TypeUsages); err != nil {
			slog.Warn("insert type usages failed", "file", relPath, "error", err)
		}
		stats["type_usages"] += len(parsed.TypeUsages)
	}
	if len(parsed.IfaceImpls) > 0 {
		if err := uc.iiRepo.BulkInsert(ctx, parsed.IfaceImpls); err != nil {
			slog.Warn("insert interface impls failed", "file", relPath, "error", err)
		}
		stats["interface_impls"] += len(parsed.IfaceImpls)
	}

	// Accumulate connections and component API calls (bulk inserted after all files)
	for i := range parsed.Connections {
		parsed.Connections[i].RepoID = repoID
		parsed.Connections[i].FileID = fileRec.ID
	}
	*allConns = append(*allConns, parsed.Connections...)
	*allCAC = append(*allCAC, parsed.CompAPICalls...)

	stats["files_indexed"]++
	return nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
