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
	modulePath := parser.ModuleFromGoMod(absPath)

	// Load detection patterns from goatlas.yaml (or embedded defaults)
	patternCfg, _ := parser.LoadPatterns(absPath)

	// Upsert the repository record; scans back existing last_commit from DB.
	repo := &domain.Repository{
		Name: repoName,
		Path: absPath,
	}
	if err := uc.repoRepo.Upsert(ctx, repo); err != nil {
		return nil, fmt.Errorf("upsert repository: %w", err)
	}

	slog.Info("indexing repository", "name", repoName, "id", repo.ID, "module", modulePath)

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
			// if diffErr != nil: fall through to full walk (targetFiles stays nil)
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
			var indexErr error
			switch ext {
			case ".go":
				indexErr = uc.indexGoFile(ctx, repo.ID, absPath, fa.path, modulePath, force, stats, &allConns, patternCfg)
			case ".tsx", ".ts", ".jsx", ".js":
				indexErr = uc.indexJSFile(ctx, repo.ID, absPath, fa.path, force, stats, &allConns, patternCfg)
			case ".py":
				indexErr = uc.indexPythonFile(ctx, repo.ID, absPath, fa.path, force, stats)
			}
			if indexErr != nil {
				slog.Warn("incremental: index file failed", "path", fa.path, "error", indexErr)
			}
		}
	} else {
		err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip unreadable
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
			relPath, _ := filepath.Rel(absPath, path)

			switch ext {
			case ".go":
				return uc.indexGoFile(ctx, repo.ID, absPath, relPath, modulePath, force, stats, &allConns, patternCfg)
			case ".tsx", ".ts", ".jsx", ".js":
				return uc.indexJSFile(ctx, repo.ID, absPath, relPath, force, stats, &allConns, patternCfg)
			case ".py":
				return uc.indexPythonFile(ctx, repo.ID, absPath, relPath, force, stats)
			default:
				return nil
			}
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

	// Update repo last indexed time via second upsert (safe: doesn't overwrite last_commit).
	now := time.Now()
	repo.LastIndexedAt = &now
	_ = uc.repoRepo.Upsert(ctx, repo)

	// Persist the current HEAD commit so future incremental runs can diff from here.
	if headCommit != "" {
		_ = uc.repoRepo.UpdateLastCommit(ctx, repo.ID, headCommit)
	}

	result := map[string]any{
		"status":     "ok",
		"repository": repoName,
		"repo_id":    repo.ID,
		"module":     modulePath,
	}
	for k, v := range stats {
		result[k] = v
	}
	return result, nil
}

// indexGoFile processes a single Go file.
func (uc *IndexRepoUseCase) indexGoFile(
	ctx context.Context,
	repoID int64,
	absPath, relPath, modulePath string,
	force bool,
	stats map[string]int,
	allConns *[]domain.ServiceConnection,
	patternCfg *parser.PatternConfig,
) error {
	fullPath := filepath.Join(absPath, relPath)

	hash, err := hashFile(fullPath)
	if err != nil {
		return nil
	}

	// Check existing file: skip if hash unchanged
	existing, _ := uc.fileRepo.GetByPath(ctx, repoID, relPath)
	if existing != nil && existing.Hash == hash && !force {
		stats["files_skipped"]++
		return nil
	}

	// Parse AST
	parsed, err := parser.ParseFile(fullPath)
	if err != nil {
		slog.Warn("parse failed", "file", relPath, "error", err)
		return nil
	}

	// Determine module for this file
	pkgModule := modulePath
	if pkgModule == "" {
		pkgModule = parsed.Module
	}

	// Upsert file record
	fileRec := &domain.File{
		RepoID: repoID,
		Path:   relPath,
		Module: pkgModule,
		Hash:   hash,
	}
	if err := uc.fileRepo.Upsert(ctx, fileRec); err != nil {
		return fmt.Errorf("upsert file %s: %w", relPath, err)
	}

	// Clear old data for re-index
	_ = uc.symbolRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.epRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.importRepo.DeleteByFileID(ctx, fileRec.ID)

	// Insert symbols
	if len(parsed.Symbols) > 0 {
		for i := range parsed.Symbols {
			parsed.Symbols[i].FileID = fileRec.ID
		}
		if err := uc.symbolRepo.BulkInsert(ctx, parsed.Symbols); err != nil {
			return fmt.Errorf("insert symbols: %w", err)
		}
		stats["symbols"] += len(parsed.Symbols)
	}

	// Insert imports
	if len(parsed.Imports) > 0 {
		for i := range parsed.Imports {
			parsed.Imports[i].FileID = fileRec.ID
		}
		if err := uc.importRepo.BulkInsert(ctx, parsed.Imports); err != nil {
			return fmt.Errorf("insert imports: %w", err)
		}
		stats["imports"] += len(parsed.Imports)
	}

	// Extract API endpoints
	endpoints, _ := parser.ExtractRoutes(fullPath, parsed.Imports)
	if len(endpoints) > 0 {
		for i := range endpoints {
			endpoints[i].FileID = fileRec.ID
		}
		if err := uc.epRepo.BulkInsert(ctx, endpoints); err != nil {
			return fmt.Errorf("insert endpoints: %w", err)
		}
		stats["endpoints"] += len(endpoints)
	}

	// Detect cross-service connections (gRPC, Kafka)
	connResult, err := parser.DetectConnections(fullPath, patternCfg)
	if err == nil && len(connResult.Connections) > 0 {
		for i := range connResult.Connections {
			connResult.Connections[i].RepoID = repoID
			connResult.Connections[i].FileID = fileRec.ID
		}
		*allConns = append(*allConns, connResult.Connections...)
	}

	// Extract function call graph
	_ = uc.fcRepo.DeleteByFileID(ctx, fileRec.ID)
	calls, callErr := parser.ExtractFunctionCalls(fullPath)
	if callErr == nil && len(calls) > 0 {
		for i := range calls {
			calls[i].FileID = fileRec.ID
		}
		if err := uc.fcRepo.BulkInsert(ctx, calls); err != nil {
			slog.Warn("insert function calls failed", "file", relPath, "error", err)
		}
		stats["function_calls"] += len(calls)
	}

	// Extract type usages from function signatures
	_ = uc.tuRepo.DeleteByFileID(ctx, fileRec.ID)
	typeUsages, tuErr := parser.ExtractTypeUsages(fullPath)
	if tuErr == nil && len(typeUsages) > 0 {
		for i := range typeUsages {
			typeUsages[i].FileID = fileRec.ID
		}
		if err := uc.tuRepo.BulkInsert(ctx, typeUsages); err != nil {
			slog.Warn("insert type usages failed", "file", relPath, "error", err)
		}
		stats["type_usages"] += len(typeUsages)
	}

	// Extract interface implementations
	_ = uc.iiRepo.DeleteByFileID(ctx, fileRec.ID)
	impls, iiErr := parser.ExtractInterfaceImpls(fullPath)
	if iiErr == nil && len(impls) > 0 {
		for i := range impls {
			impls[i].FileID = fileRec.ID
		}
		if err := uc.iiRepo.BulkInsert(ctx, impls); err != nil {
			slog.Warn("insert interface impls failed", "file", relPath, "error", err)
		}
		stats["interface_impls"] += len(impls)
	}

	stats["files_indexed"]++
	return nil
}

// indexJSFile processes a single JS/TS/JSX/TSX file.
func (uc *IndexRepoUseCase) indexJSFile(
	ctx context.Context,
	repoID int64,
	absPath, relPath string,
	force bool,
	stats map[string]int,
	allConns *[]domain.ServiceConnection,
	patternCfg *parser.PatternConfig,
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

	// Parse JSX/TS
	parsed, err := parser.ParseJSXFile(fullPath)
	if err != nil {
		slog.Warn("parse JS failed", "file", relPath, "error", err)
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

	_ = uc.symbolRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.importRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.epRepo.DeleteByFileID(ctx, fileRec.ID)

	if len(parsed.Symbols) > 0 {
		for i := range parsed.Symbols {
			parsed.Symbols[i].FileID = fileRec.ID
		}
		if err := uc.symbolRepo.BulkInsert(ctx, parsed.Symbols); err != nil {
			return fmt.Errorf("insert symbols: %w", err)
		}
		stats["symbols"] += len(parsed.Symbols)
	}

	if len(parsed.Imports) > 0 {
		for i := range parsed.Imports {
			parsed.Imports[i].FileID = fileRec.ID
		}
		if err := uc.importRepo.BulkInsert(ctx, parsed.Imports); err != nil {
			return fmt.Errorf("insert imports: %w", err)
		}
		stats["imports"] += len(parsed.Imports)
	}

	// Extract React routes
	endpoints, _ := parser.ExtractReactRoutes(fullPath, parsed.Imports)
	if len(endpoints) > 0 {
		for i := range endpoints {
			endpoints[i].FileID = fileRec.ID
		}
		if err := uc.epRepo.BulkInsert(ctx, endpoints); err != nil {
			return fmt.Errorf("insert endpoints: %w", err)
		}
		stats["endpoints"] += len(endpoints)
	}

	// Detect API service references in TS files (e.g. SVC_PREFIX constants)
	if patternCfg != nil && len(patternCfg.TypeScript.APIPrefix) > 0 {
		tsConns, _ := parser.DetectTSAPIConnections(fullPath, patternCfg.TypeScript.APIPrefix)
		if len(tsConns) > 0 {
			for i := range tsConns {
				tsConns[i].RepoID = repoID
				tsConns[i].FileID = fileRec.ID
			}
			*allConns = append(*allConns, tsConns...)
		}
	}

	// Detect component-level API calls (React component → backend API)
	var apiPatterns []parser.TSAPIPattern
	if patternCfg != nil {
		apiPatterns = patternCfg.TypeScript.APIPrefix
	}
	_ = uc.cacRepo.DeleteByFileID(ctx, fileRec.ID)
	componentCalls, _ := parser.DetectComponentAPICalls(fullPath, apiPatterns)
	if len(componentCalls) > 0 {
		for i := range componentCalls {
			componentCalls[i].FileID = fileRec.ID
		}
		if err := uc.cacRepo.BulkInsert(ctx, componentCalls); err != nil {
			return fmt.Errorf("insert component API calls: %w", err)
		}
	}

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

// indexPythonFile processes a single Python file.
func (uc *IndexRepoUseCase) indexPythonFile(
	ctx context.Context,
	repoID int64,
	absPath, relPath string,
	force bool,
	stats map[string]int,
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

	parsed, err := parser.ParsePythonFile(fullPath)
	if err != nil {
		slog.Warn("parse Python failed", "file", relPath, "error", err)
		return nil
	}

	fileRec := &domain.File{
		RepoID: repoID,
		Path:   relPath,
		Module: parsed.Module,
		Hash:   hash,
	}
	if err := uc.fileRepo.Upsert(ctx, fileRec); err != nil {
		return fmt.Errorf("upsert file %s: %w", relPath, err)
	}

	_ = uc.symbolRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.importRepo.DeleteByFileID(ctx, fileRec.ID)
	_ = uc.epRepo.DeleteByFileID(ctx, fileRec.ID)

	if len(parsed.Symbols) > 0 {
		for i := range parsed.Symbols {
			parsed.Symbols[i].FileID = fileRec.ID
		}
		if err := uc.symbolRepo.BulkInsert(ctx, parsed.Symbols); err != nil {
			return fmt.Errorf("insert symbols: %w", err)
		}
		stats["symbols"] += len(parsed.Symbols)
	}

	if len(parsed.Imports) > 0 {
		for i := range parsed.Imports {
			parsed.Imports[i].FileID = fileRec.ID
		}
		if err := uc.importRepo.BulkInsert(ctx, parsed.Imports); err != nil {
			return fmt.Errorf("insert imports: %w", err)
		}
		stats["imports"] += len(parsed.Imports)
	}

	if len(parsed.Endpoints) > 0 {
		for i := range parsed.Endpoints {
			parsed.Endpoints[i].FileID = fileRec.ID
		}
		if err := uc.epRepo.BulkInsert(ctx, parsed.Endpoints); err != nil {
			return fmt.Errorf("insert endpoints: %w", err)
		}
		stats["endpoints"] += len(parsed.Endpoints)
	}

	stats["files_indexed"]++
	return nil
}
