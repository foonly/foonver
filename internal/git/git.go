// Package git provides helper functions for Git operations like preflight checks and committing version changes.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
)

// RunPreflightChecks ensures the current directory is a Git repository and has no uncommitted changes.
func RunPreflightChecks() (string, error) {
	// Check if in Git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not inside a Git repository")
	}

	// Check for clean working directory
	cmd = exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to check git status: %w", err)
	}
	if len(bytes.TrimSpace(output)) > 0 {
		return "", fmt.Errorf("git working directory not clean. Commit or stash changes first")
	}

	// Return git root folder.
	cmd = exec.Command("git", "rev-parse", "--show-toplevel")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("unable to determine git root")
	}
	return string(output), nil
}

// CommitAndTag stages the version file, commits it with the version as the message, and creates a Git tag.
func CommitAndTag(filename, version string) error {
	// Add file
	cmd := exec.Command("git", "add", filename)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Commit
	// Standardizing to just the version string as per PLAN.md
	// Alternatively could prefix with "v" depending on convention.
	cmd = exec.Command("git", "commit", "-m", version)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// Tag
	cmd = exec.Command("git", "tag", version)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git tag failed: %w", err)
	}

	return nil
}
