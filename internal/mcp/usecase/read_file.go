package usecase

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReadFileUseCase reads a file from any indexed repository with optional line range.
// It resolves the file path by looking up the repository root from the database,
// supporting multi-repo setups where files may belong to different repositories.
type ReadFileUseCase struct {
	pool *pgxpool.Pool
}

// NewReadFileUseCase constructs a ReadFileUseCase.
func NewReadFileUseCase(pool *pgxpool.Pool) *ReadFileUseCase {
	return &ReadFileUseCase{pool: pool}
}

// Execute reads the file at the given relative path and returns the content with line numbers.
// It looks up the repository root from the database by matching the path against indexed files.
func (uc *ReadFileUseCase) Execute(ctx context.Context, path string, startLine, endLine int) (string, error) {
	// Look up the file path and its repository's absolute root from the DB.
	var repoPath, filePath string
	err := uc.pool.QueryRow(ctx, `
		SELECT r.path, f.path
		FROM files f
		JOIN repositories r ON r.id = f.repo_id
		WHERE f.path = $1
		LIMIT 1
	`, path).Scan(&repoPath, &filePath)
	if err != nil {
		return "", fmt.Errorf("file %q not found in index: %w", path, err)
	}

	absPath := filepath.Join(repoPath, filePath)
	clean := filepath.Clean(absPath)

	// Safety check: ensure resolved path is within the repo root.
	if !strings.HasPrefix(clean, filepath.Clean(repoPath)) {
		return "", errors.New("path outside repo root")
	}

	f, err := os.Open(clean)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 1
	for scanner.Scan() {
		if startLine > 0 && lineNum < startLine {
			lineNum++
			continue
		}
		if endLine > 0 && lineNum > endLine {
			break
		}
		lines = append(lines, fmt.Sprintf("%4d: %s", lineNum, scanner.Text()))
		lineNum++
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}
