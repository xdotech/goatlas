package usecase

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// GetHeadCommit returns the current HEAD commit hash for the given repo path.
// Returns empty string if the path is not a git repository.
func GetHeadCommit(repoPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", nil // not a git repo or no commits — silently degrade
	}
	return strings.TrimSpace(string(out)), nil
}

// commitExists checks whether a given commit hash exists in the repo.
func commitExists(repoPath, commit string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := exec.CommandContext(ctx, "git", "-C", repoPath, "cat-file", "-e", commit).Run()
	return err == nil
}

// GetChangedFiles returns the added/modified/deleted file paths between two commits.
// Falls back to (nil, nil, nil, nil) if git commands fail.
func GetChangedFiles(repoPath, fromCommit, toCommit string) (added, modified, deleted []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--name-status", fromCommit, toCommit).Output()
	if err != nil {
		return nil, nil, nil, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		switch {
		case status == "A":
			added = append(added, parts[1])
		case status == "M":
			modified = append(modified, parts[1])
		case status == "D":
			deleted = append(deleted, parts[1])
		case strings.HasPrefix(status, "R") && len(parts) == 3:
			// Rename: old path is deleted, new path is added
			deleted = append(deleted, parts[1])
			added = append(added, parts[2])
		}
	}
	return added, modified, deleted, nil
}
