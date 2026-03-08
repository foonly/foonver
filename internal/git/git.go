// Package git provides helper functions for Git operations like preflight checks and committing version changes.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type GitInfo struct {
	Ok        bool
	Clean     bool
	HasRemote bool
	RootDir   string
}

var Info GitInfo

// RunPreflightChecks ensures the current directory is a Git repository and has no uncommitted changes.
func RunPreflightChecks() error {
	Info = GitInfo{
		Ok:        true,
		Clean:     true,
		HasRemote: false,
		RootDir:   ".",
	}

	// Check if in Git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		Info.Ok = false
		return fmt.Errorf("not inside a Git repository")
	}
	Info.RootDir = strings.TrimSpace(string(output))

	// Check for remotes
	cmd = exec.Command("git", "remote")
	output, err = cmd.Output()
	if err != nil {
		Info.Ok = false
		return fmt.Errorf("failed to check git remotes: %w", err)
	}
	if len(bytes.TrimSpace(output)) > 0 {
		Info.HasRemote = true
	}

	// Check for clean working directory
	cmd = exec.Command("git", "status", "--porcelain")
	output, err = cmd.Output()
	if err != nil {
		Info.Ok = false
		return fmt.Errorf("failed to check git status: %w", err)
	}
	if len(bytes.TrimSpace(output)) > 0 {
		Info.Clean = false
		return fmt.Errorf("git working directory not clean. Commit or stash changes first")
	}
	return nil
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

func PushTags() error {
	cmd := exec.Command("git", "push")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to push commits: %v\n", err)
	}
	cmd = exec.Command("git", "push", "--tags")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to push tags: %v\n", err)
	}
	return nil
}
