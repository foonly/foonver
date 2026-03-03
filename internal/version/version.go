// Package version provides utilities for discovering, extracting, and updating version information
// in various file formats, as well as determining the next version based on Git history.
package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"foonly.dev/foonver/internal/git"
	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
)

var versionFiles = []string{
	"package.json",
	"version.json",
	"version.toml",
	"version.txt",
	"version.md",
}

func RunVersion(cmd *cobra.Command, args []string) error {

	projectRoot, err := git.RunPreflightChecks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fileName, currentVersion, fileContent, err := discoverVersion(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found version %s in %s\n", currentVersion.Original(), fileName)

	fmt.Printf("Command %s\n", cmd.Name())

	// WIP
	os.Exit(0)

	target := ""
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	nextVersion, err := determineNextVersion(currentVersion, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining next version: %v\n", err)
		os.Exit(1)
	}

	nextVersionStr := nextVersion.String()
	if strings.HasPrefix(currentVersion.Original(), "v") {
		nextVersionStr = "v" + nextVersionStr
	}

	if currentVersion.String() == nextVersion.String() {
		fmt.Println("Version is already up to date.")
		os.Exit(0)
	}

	fmt.Printf("Bumping version from %s to %s\n", currentVersion.Original(), nextVersionStr)

	if err := updateVersionFile(fileName, currentVersion.Original(), nextVersionStr, fileContent); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating version file: %v\n", err)
		os.Exit(1)
	}

	if err := git.CommitAndTag(fileName, nextVersionStr); err != nil {
		fmt.Fprintf(os.Stderr, "Git error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully bumped version to %s\n", nextVersionStr)

	return nil
}

// discoverVersion searches for a version file in the current directory and returns its path,
// the parsed version, and its raw content.
func discoverVersion(projectRoot string) (string, *semver.Version, []byte, error) {
	for _, file := range versionFiles {
		filePath := path.Join(projectRoot, file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", nil, nil, fmt.Errorf("error reading %s: %w", file, err)
		}

		vStr, err := extractVersion(file, content)
		if err != nil {
			continue // Might be the wrong format, try next file
		}

		v, err := semver.NewVersion(vStr)
		if err != nil {
			return "", nil, nil, fmt.Errorf("invalid semver string '%s' in %s: %w", vStr, file, err)
		}

		return filePath, v, content, nil
	}

	return "", nil, nil, fmt.Errorf("no valid version file found")
}

// ExtractVersion extracts a version string from the given file content based on the filename's extension.
// It supports .json, .toml, .txt, and .md files.
func extractVersion(filename string, content []byte) (string, error) {
	switch {
	case strings.HasSuffix(filename, ".json"):
		var data map[string]any
		if err := json.Unmarshal(content, &data); err != nil {
			return "", err
		}
		if v, ok := data["version"].(string); ok {
			return v, nil
		}
		return "", fmt.Errorf("no 'version' string field found")

	case strings.HasSuffix(filename, ".toml"):
		re := regexp.MustCompile(`(?m)^version\s*=\s*['"]([^'"]+)['"]`)
		matches := re.FindSubmatch(content)
		if len(matches) > 1 {
			return string(matches[1]), nil
		}
		return "", fmt.Errorf("no version string found")

	case strings.HasSuffix(filename, ".txt"):
		return strings.TrimSpace(string(content)), nil

	case strings.HasSuffix(filename, ".md"):
		// Very simple regex for version in MD files (e.g. # Version 1.2.3 or v1.2.3)
		re := regexp.MustCompile(`(?i)(?:version\s+|v)?(\d+\.\d+\.\d+(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?)`)
		matches := re.FindSubmatch(content)
		if len(matches) > 1 {
			return string(matches[1]), nil
		}
		return "", fmt.Errorf("no version string found")
	}
	return "", fmt.Errorf("unsupported file type")
}

// determineNextVersion calculates the next version based on a target ("major", "minor", "patch",
// or a specific version string) or automatically by analyzing Git commit messages.
func determineNextVersion(current *semver.Version, target string) (*semver.Version, error) {
	if target != "" {
		switch strings.ToLower(target) {
		case "major":
			v := current.IncMajor()
			return &v, nil
		case "minor":
			v := current.IncMinor()
			return &v, nil
		case "patch":
			v := current.IncPatch()
			return &v, nil
		default:
			return semver.NewVersion(target)
		}
	}

	// Automatic mode based on Git history
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	lastTag, err := cmd.Output()
	var logCmd *exec.Cmd

	if err != nil {
		// No tags found, get all commits
		logCmd = exec.Command("git", "log", "--oneline")
	} else {
		tag := strings.TrimSpace(string(lastTag))
		logCmd = exec.Command("git", "log", fmt.Sprintf("%s..HEAD", tag), "--oneline")
	}

	logOutput, err := logCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git log: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(logOutput)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return current, nil // No new commits
	}

	bumpMajor := false
	bumpMinor := false
	bumpPatch := false

	for _, line := range lines {
		// Strip the commit hash
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		msg := strings.ToLower(strings.TrimSpace(parts[1]))

		if strings.HasPrefix(msg, "breaking change") || strings.Contains(msg, "breaking change") {
			bumpMajor = true
		} else if strings.HasPrefix(msg, "feat:") || strings.HasPrefix(msg, "feature") || strings.HasPrefix(msg, "new feature") {
			bumpMinor = true
		} else {
			bumpPatch = true
		}
	}

	var next semver.Version
	if bumpMajor {
		next = current.IncMajor()
	} else if bumpMinor {
		next = current.IncMinor()
	} else if bumpPatch {
		next = current.IncPatch()
	} else {
		next = *current
	}

	return &next, nil
}

// updateVersionFile replaces the old version string with the new version string in the specified file
// and writes the changes back to disk, attempting to preserve file formatting.
func updateVersionFile(filename, oldVersion, newVersion string, content []byte) error {
	var newContent []byte

	switch {
	case strings.HasSuffix(filename, ".json"), strings.HasSuffix(filename, ".toml"):
		// Simple string replacement for JSON and TOML to preserve formatting.
		// A more robust approach would use an AST, but this works for basic cases.
		oldStr := fmt.Sprintf(`"%s"`, oldVersion)
		newStr := fmt.Sprintf(`"%s"`, newVersion)
		newContent = bytes.Replace(content, []byte(oldStr), []byte(newStr), 1)

		// TOML might use single quotes
		if bytes.Equal(content, newContent) && strings.HasSuffix(filename, ".toml") {
			oldStr := fmt.Sprintf(`'%s'`, oldVersion)
			newStr := fmt.Sprintf(`'%s'`, newVersion)
			newContent = bytes.Replace(content, []byte(oldStr), []byte(newStr), 1)
		}

	case strings.HasSuffix(filename, ".txt"), strings.HasSuffix(filename, ".md"):
		newContent = bytes.Replace(content, []byte(oldVersion), []byte(newVersion), 1)
	}

	if bytes.Equal(content, newContent) {
		return fmt.Errorf("failed to replace version string in %s", filename)
	}

	return os.WriteFile(filename, newContent, 0644)
}
