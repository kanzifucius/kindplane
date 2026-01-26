package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// CloneRepo clones a git repository and returns the path
func CloneRepo(ctx context.Context, repoURL, branch string) (string, error) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "kindplane-compositions-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clone options
	opts := &git.CloneOptions{
		URL:      repoURL,
		Progress: nil, // Suppress progress output
		Depth:    1,   // Shallow clone
	}

	if branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
		opts.SingleBranch = true
	}

	// Clone repository
	_, err = git.PlainCloneContext(ctx, tmpDir, false, opts)
	if err != nil {
		// Clean up on error
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return tmpDir, nil
}

// CloneRepoToPath clones a git repository to a specific path
func CloneRepoToPath(ctx context.Context, repoURL, branch, destPath string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Clone options
	opts := &git.CloneOptions{
		URL:      repoURL,
		Progress: nil,
		Depth:    1,
	}

	if branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
		opts.SingleBranch = true
	}

	// Clone repository
	_, err := git.PlainCloneContext(ctx, destPath, false, opts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// CleanupTempDir removes a temporary directory created by CloneRepo
func CleanupTempDir(path string) error {
	return os.RemoveAll(path)
}
