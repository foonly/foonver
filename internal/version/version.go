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

	"github.com/Masterminds/semver/v3"
	"github.com/foonly/foonver/internal/changelog"
	"github.com/foonly/foonver/internal/config"
	"github.com/foonly/foonver/internal/git"
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
	if err := git.RunPreflightChecks(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fileName, currentVersion, fileContent, err := discoverVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found version %s in %s\n", currentVersion.Original(), fileName)

	newVersion := ""
	if len(args) > 0 {
		newVersion = args[0]
	}

	nextVersion, err := determineNextVersion(currentVersion, cmd.Name(), newVersion)
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

	filesToCommit := []string{fileName}

	if config.Conf.Changelog {
		changelogFile, err := changelog.WriteChangelog(nextVersionStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating changelog: %v\n", err)
			os.Exit(1)
		}
		filesToCommit = append(filesToCommit, changelogFile)
		fmt.Printf("Updated changelog: %s\n", changelogFile)
	}

	if err := git.CommitAndTag(filesToCommit, nextVersionStr); err != nil {
		fmt.Fprintf(os.Stderr, "Git error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully bumped version to %s\n", nextVersionStr)

	if config.Conf.Push {
		if !config.Conf.Info.HasRemote {
			fmt.Fprintf(os.Stderr, "Cannot push: no git remote configured\n")
			os.Exit(1)
		}
		if err := git.PushTags(); err != nil {
			fmt.Fprintf(os.Stderr, "Git push error: %v\n", err)
			os.Exit(1)
		}
	}

	return nil
}

// discoverVersion searches for a version file in the current directory and returns its path,
// the parsed version, and its raw content.
func discoverVersion() (string, *semver.Version, []byte, error) {
	for _, file := range versionFiles {
		filePath := path.Join(config.Conf.Info.RootDir, file)
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
func determineNextVersion(current *semver.Version, target string, setVersion string) (*semver.Version, error) {
	action := strings.ToLower(target)

	if action == "ver" {
		return semver.NewVersion(setVersion)
	}

	if action == "auto" {
		auto, err := autoVersion()
		if err != nil {
			return nil, err
		}
		action = auto
	}

	var next semver.Version
	switch action {
	case "major":
		next = current.IncMajor()
	case "minor":
		next = current.IncMinor()
	case "patch":
		next = current.IncPatch()
	default:
		next = *current
	}
	return &next, nil

}

func autoVersion() (string, error) {
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
		return "", fmt.Errorf("failed to get git log: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(logOutput)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return "", nil // No new commits
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
		msg := strings.TrimSpace(parts[1])

		major, minor, patch := parseCommit(msg)
		if major {
			bumpMajor = true
		} else if minor {
			bumpMinor = true
		} else if patch {
			bumpPatch = true
		}
	}

	if bumpMajor {
		return "major", nil
	} else if bumpMinor {
		return "minor", nil
	} else if bumpPatch {
		return "patch", nil
	}

	return "", nil
}

func parseCommit(msg string) (bool, bool, bool) {
	bumpMajor := false
	bumpMinor := false
	bumpPatch := false

	parsers := make([]string, 2)
	parsers[0] = "angular"
	parsers[1] = "generic"

	for _, parser := range parsers {
		if config.Conf.Parser != parser && config.Conf.Parser != "all" {
			continue
		}

		var major, minor, patch bool
		switch parser {
		case "angular":
			major, minor, patch = parseAngular(msg)
		case "generic":
			major, minor, patch = parseGeneric(msg)
		}
		if major {
			bumpMajor = true
		} else if minor {
			bumpMinor = true
		} else if patch {
			bumpPatch = true
		}
	}
	return bumpMajor, bumpMinor, bumpPatch
}

func parseGeneric(msg string) (bool, bool, bool) {
	msg = strings.ToLower(msg)

	if strings.HasPrefix(msg, "breaking") || strings.Contains(msg, "breaking change:") {
		return true, false, false
	} else if strings.HasPrefix(msg, "feat") || strings.Contains(msg, "new feature:") {
		return false, true, false
	}

	return false, false, true
}

var findTypeScope = regexp.MustCompile(`(?i)^([a-z]+)(?:\([^)]+\))?(!)?:`)

func parseAngular(msg string) (bool, bool, bool) {
	if strings.Contains(msg, "BREAKING CHANGE:") {
		return true, false, false
	}

	matches := findTypeScope.FindStringSubmatch(msg)
	if len(matches) > 0 {
		if len(matches) > 2 && matches[2] == "!" {
			// Has bang, which indicates a breaking change
			return true, false, false
		}
		switch strings.ToLower(matches[1]) {
		case "feat":
			return false, true, false
		case "fix":
			return false, false, true
		}
	}

	return false, false, false
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
