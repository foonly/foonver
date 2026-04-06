// Package git provides helper functions for Git operations like repository validation, status checks, and versioning.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/foonly/foonver/internal/config"
)

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}

	return stdout.String(), nil
}

// EnsureRepo checks if the current directory is a Git repository and populates config.Conf.Info.
func EnsureRepo() error {
	config.Conf.Info = config.GitInfo{
		Ok:        true,
		Clean:     true,
		HasRemote: false,
		RootDir:   ".",
	}

	// Check if in Git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		config.Conf.Info.Ok = false
		return fmt.Errorf("not inside a Git repository")
	}
	config.Conf.Info.RootDir = strings.TrimSpace(string(output))

	// Check for remotes
	cmd = exec.Command("git", "remote")
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git remotes: %w", err)
	}
	if len(bytes.TrimSpace(output)) > 0 {
		config.Conf.Info.HasRemote = true
	}

	return nil
}

// IsClean checks for uncommitted changes in the working directory and updates config.Conf.Info.
func IsClean() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	config.Conf.Info.Clean = len(bytes.TrimSpace(output)) == 0
	return config.Conf.Info.Clean
}

// RunPreflightChecks ensures the current directory is a Git repository and has no uncommitted changes.
func RunPreflightChecks() error {
	if err := EnsureRepo(); err != nil {
		return err
	}
	if !IsClean() {
		return fmt.Errorf("Git working directory not clean. Commit or stash changes first")
	}
	return nil
}

// CommitAndTag stages the version files, commits them with the version as the message, and creates a Git tag.
func CommitAndTag(filenames []string, version string) error {
	var cmd *exec.Cmd

	// Add files
	for _, filename := range filenames {
		cmd = exec.Command("git", "add", filename)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git add failed for %s: %w", filename, err)
		}
	}

	// Append the prefix to version.
	versionString := fmt.Sprintf("%s%s", config.Conf.Prefix, version)

	// Commit
	cmd = exec.Command("git", "commit", "-m", versionString)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// Tag
	cmd = exec.Command("git", "tag", versionString)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git tag failed: %w", err)
	}

	return nil
}

// PushTags pushes local commits and tags to the remote repository.
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

type Tag struct {
	Name string
	Date string
}

// GetTags returns all tags in the repository sorted by creation date.
func GetTags() ([]Tag, error) {
	out, err := runGit("for-each-ref", "--format=%(refname:short) %(creatordate:short)", "refs/tags", "--sort=creatordate")
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	lines := splitNonEmptyLines(out)
	tags := make([]Tag, 0, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			tags = append(tags, Tag{
				Name: parts[0],
				Date: parts[1],
			})
		}
	}
	return tags, nil
}

// GetLatestTag returns the most recent tag name.
func GetLatestTag() (string, error) {
	out, err := runGit("describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetCommits returns a list of commit hashes and messages for the given range.
func GetCommits(revRange string) ([]string, error) {
	args := []string{"log", "--pretty=format:%h %s"}
	if strings.TrimSpace(revRange) != "" {
		args = append(args, revRange)
	}

	out, err := runGit(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for range %q: %w", revRange, err)
	}

	return splitNonEmptyLines(out), nil
}

func splitNonEmptyLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
