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
	return runGitWithCode(args...)
}

func runGitWithCode(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		// Return the original err as the cause so ExitCode can inspect it
		return "", &GitError{Args: args, Message: msg, Err: err}
	}

	return stdout.String(), nil
}

type GitError struct {
	Args    []string
	Message string
	Err     error
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), e.Message)
}

func (e *GitError) Unwrap() error {
	return e.Err
}

// ExitCode returns the exit code of a git command error, if available.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	// Many of our errors are wrapped with fmt.Errorf, so we need to check the cause
	type causer interface {
		Unwrap() error
	}
	if c, ok := err.(causer); ok {
		return ExitCode(c.Unwrap())
	}
	return -1
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
	output, err := runGit("rev-parse", "--show-toplevel")
	if err != nil {
		config.Conf.Info.Ok = false
		return fmt.Errorf("not inside a Git repository")
	}
	config.Conf.Info.RootDir = strings.TrimSpace(output)

	// Check for remotes
	output, err = runGit("remote")
	if err != nil {
		return fmt.Errorf("failed to check git remotes: %w", err)
	}
	if len(strings.TrimSpace(output)) > 0 {
		config.Conf.Info.HasRemote = true
	}

	return nil
}

// IsClean checks for uncommitted changes in the working directory and updates config.Conf.Info.
func IsClean() bool {
	output, err := runGit("status", "--porcelain")
	if err != nil {
		return false
	}
	config.Conf.Info.Clean = len(strings.TrimSpace(output)) == 0
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
	// Add files
	for _, filename := range filenames {
		if _, err := runGit("add", filename); err != nil {
			return err
		}
	}

	// Append the prefix to version if it's not already there.
	versionString := version
	if config.Conf.Prefix != "" && !strings.HasPrefix(version, config.Conf.Prefix) {
		versionString = fmt.Sprintf("%s%s", config.Conf.Prefix, version)
	}

	// Determine commit message
	commitMsg := config.Conf.CommitMessage
	if commitMsg == "" {
		commitMsg = versionString
	}
	if config.Conf.CommitSuffix != "" {
		commitMsg = fmt.Sprintf("%s %s", commitMsg, config.Conf.CommitSuffix)
	}

	// Commit only if there are staged changes
	// git diff --cached --quiet returns 1 if there are staged changes, 0 if none.
	_, err := runGit("diff", "--cached", "--quiet")
	if err != nil {
		code := ExitCode(err)
		if code == 1 {
			// There are staged changes, proceed with commit
			if _, err := runGit("commit", "-m", commitMsg); err != nil {
				return err
			}
		} else {
			// A real error occurred (code != 1 and err != nil)
			return err
		}
	} else {
		// err == nil means code == 0, so no staged changes.
		if config.Conf.Verbosity >= config.Normal {
			fmt.Printf("No changes to commit for %s\n", versionString)
		}
	}

	// Tag
	if _, err := runGit("tag", versionString); err != nil {
		return err
	}

	return nil
}

// PushTags pushes local commits and tags to the remote repository.
func PushTags() error {
	remote := config.Conf.Remote
	if remote == "" {
		remote = "origin"
	}

	if _, err := runGit("push", remote, "HEAD"); err != nil {
		return err
	}
	if _, err := runGit("push", remote, "--tags"); err != nil {
		return err
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
