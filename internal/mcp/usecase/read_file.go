package usecase

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileUseCase reads a file from the repository root with optional line range.
type ReadFileUseCase struct {
	RepoRoot string
}

// NewReadFileUseCase constructs a ReadFileUseCase.
func NewReadFileUseCase(repoRoot string) *ReadFileUseCase {
	return &ReadFileUseCase{RepoRoot: repoRoot}
}

// Execute reads the file at path (relative to RepoRoot) and returns the content with line numbers.
func (uc *ReadFileUseCase) Execute(_ context.Context, path string, startLine, endLine int) (string, error) {
	clean := filepath.Clean(filepath.Join(uc.RepoRoot, path))
	if !strings.HasPrefix(clean, uc.RepoRoot) {
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
