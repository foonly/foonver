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

type StepType string

const (
	StepUpdateVersionFile StepType = "UpdateVersionFile"
	StepUpdateChangelog   StepType = "UpdateChangelog"
	StepGitCommit         StepType = "GitCommit"
	StepGitTag            StepType = "GitTag"
	StepGitPush           StepType = "GitPush"
	StepWriteReleaseNotes StepType = "WriteReleaseNotes"
	StepSyncVersion       StepType = "SyncVersion"
)

type PlanStep struct {
	Type        StepType     `json:"type"`
	Description string       `json:"description"`
	Action      func() error `json:"-"`
}

type ExecutionPlan struct {
	CurrentVersion    *semver.Version `json:"-"`
	CurrentVersionStr string          `json:"current_version"`
	NextVersion       *semver.Version `json:"-"`
	NextVersionStr    string          `json:"next_version"`
	VersionFile       string          `json:"version_file"`
	Commits           []string        `json:"commits"`
	LastTag           string          `json:"last_tag"`
	IsDirty           bool            `json:"is_dirty"`
	Steps             []PlanStep      `json:"steps"`
}

var versionFiles = []string{
	"package.json",
	"version.json",
	"version.toml",
	"version.txt",
	"version.md",
}

func RunVersion(cmd *cobra.Command, args []string) error {
	plan, err := BuildPlan(cmd, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	nextVersionStr := plan.NextVersionStr
	quiet := config.Conf.PrintVersion || config.Conf.JSON

	if !quiet {
		fmt.Printf("Found version %s in %s\n", plan.CurrentVersion.Original(), plan.VersionFile)
		if plan.LastTag != "" {
			fmt.Printf("Last version tag: %s\n", plan.LastTag)
		}

		if len(plan.Commits) > 0 {
			fmt.Println("Commits since last tag:")
			for _, c := range plan.Commits {
				fmt.Printf("  %s\n", c)
			}
		}
	}

	if plan.CurrentVersion.String() == plan.NextVersion.String() {
		if config.Conf.JSON {
			data, _ := json.MarshalIndent(plan, "", "  ")
			fmt.Println(string(data))
		} else if config.Conf.PrintVersion {
			fmt.Println(nextVersionStr)
		} else {
			fmt.Println("Version is already up to date.")
		}
		return nil
	}

	if !quiet {
		fmt.Printf("Next version: %s\n", nextVersionStr)
	}

	if config.Conf.DryRun {
		if config.Conf.JSON {
			data, _ := json.MarshalIndent(plan, "", "  ")
			fmt.Println(string(data))
		} else if config.Conf.PrintVersion {
			fmt.Println(nextVersionStr)
		} else {
			fmt.Println("\nMode: dry-run (no changes will be made)")
			fmt.Printf("Repo clean: %t\n", !plan.IsDirty)
			fmt.Println("Planned actions:")
			for _, step := range plan.Steps {
				fmt.Printf("  - %s\n", step.Description)
			}
		}
		return nil
	}

	if plan.IsDirty {
		fmt.Fprintf(os.Stderr, "Error: Git working directory not clean. Commit or stash changes first.\n")
		os.Exit(1)
	}

	for _, step := range plan.Steps {
		if config.Conf.Verbosity >= config.Normal && !quiet {
			fmt.Printf("Executing: %s...\n", step.Description)
		}
		if err := step.Action(); err != nil {
			return fmt.Errorf("step %s failed: %w", step.Type, err)
		}
	}

	if config.Conf.JSON {
		data, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Println(string(data))
	} else if config.Conf.PrintVersion {
		fmt.Println(nextVersionStr)
	} else {
		fmt.Printf("Successfully bumped version to %s\n", nextVersionStr)
	}
	return nil
}

func BuildPlan(cmd *cobra.Command, args []string) (*ExecutionPlan, error) {
	if err := git.EnsureRepo(); err != nil {
		return nil, err
	}

	fileName, currentVersion, fileContent, err := discoverVersion()
	if err != nil {
		return nil, err
	}

	newVersion := ""
	if len(args) > 0 {
		newVersion = args[0]
	}

	action := cmd.Name()
	if cmd.Parent() == nil {
		action = "auto"
	}

	nextVersion, commits, lastTag, err := determineNextVersion(currentVersion, action, newVersion)
	if err != nil {
		return nil, err
	}

	nextVersionStr := nextVersion.String()
	if strings.HasPrefix(currentVersion.Original(), "v") {
		nextVersionStr = "v" + nextVersionStr
	}

	plan := &ExecutionPlan{
		CurrentVersion:    currentVersion,
		CurrentVersionStr: currentVersion.Original(),
		NextVersion:       nextVersion,
		NextVersionStr:    nextVersionStr,
		VersionFile:       fileName,
		Commits:           commits,
		LastTag:           lastTag,
		IsDirty:           !git.IsClean(),
		Steps:             []PlanStep{},
	}

	if currentVersion.String() == nextVersion.String() {
		return plan, nil
	}

	// 1. Update version file
	plan.Steps = append(plan.Steps, PlanStep{
		Type:        StepUpdateVersionFile,
		Description: fmt.Sprintf("Update %s: %s -> %s", fileName, currentVersion.Original(), nextVersionStr),
		Action: func() error {
			return updateVersionFile(fileName, currentVersion.Original(), nextVersionStr, fileContent)
		},
	})

	// 2. Version Sync
	for _, syncFile := range config.Conf.VersionSync {
		sFile := syncFile // capture for closure
		plan.Steps = append(plan.Steps, PlanStep{
			Type:        StepSyncVersion,
			Description: fmt.Sprintf("Sync version in %s: %s -> %s", sFile, currentVersion.Original(), nextVersionStr),
			Action: func() error {
				return syncVersion(sFile, currentVersion.Original(), nextVersionStr)
			},
		})
	}

	// 3. Changelog
	if config.Conf.Changelog {
		plan.Steps = append(plan.Steps, PlanStep{
			Type:        StepUpdateChangelog,
			Description: fmt.Sprintf("Update changelog: %s", config.Conf.File),
			Action: func() error {
				_, err := changelog.WriteChangelog(nextVersionStr)
				return err
			},
		})
	}

	if config.Conf.ReleaseNotes != "" {
		plan.Steps = append(plan.Steps, PlanStep{
			Type:        StepWriteReleaseNotes,
			Description: fmt.Sprintf("Write release notes: %s", config.Conf.ReleaseNotes),
			Action: func() error {
				md, err := changelog.GenerateMarkdown(nextVersionStr, true)
				if err != nil {
					return err
				}
				return os.WriteFile(config.Conf.ReleaseNotes, []byte(md), 0644)
			},
		})
	}

	// 4. Commit and Tag
	plan.Steps = append(plan.Steps, PlanStep{
		Type:        StepGitCommit,
		Description: fmt.Sprintf("Git commit and tag: %s", nextVersionStr),
		Action: func() error {
			files := []string{fileName}
			for _, syncFile := range config.Conf.VersionSync {
				files = append(files, path.Join(config.Conf.Info.RootDir, syncFile))
			}
			if config.Conf.Changelog {
				changelogPath := path.Join(config.Conf.Info.RootDir, config.Conf.File)
				files = append(files, changelogPath)
			}
			return git.CommitAndTag(files, nextVersionStr)
		},
	})

	// 5. Push
	if config.Conf.Push {
		if config.Conf.Info.HasRemote {
			plan.Steps = append(plan.Steps, PlanStep{
				Type:        StepGitPush,
				Description: "Git push commits and tags",
				Action: func() error {
					return git.PushTags()
				},
			})
		}
	}

	return plan, nil
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
func determineNextVersion(current *semver.Version, target string, setVersion string) (*semver.Version, []string, string, error) {
	action := strings.TrimSpace(strings.ToLower(target))
	var commits []string
	var lastTag string

	if action == "ver" {
		v, err := semver.NewVersion(setVersion)
		return v, nil, "", err
	}

	if action == "auto" {
		var auto string
		var err error
		auto, commits, lastTag, err = autoVersion()
		if err != nil {
			return nil, nil, "", err
		}
		if auto != "" && !config.Conf.PrintVersion && !config.Conf.JSON {
			fmt.Printf("Auto-detected version bump: %s\n", auto)
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
	return &next, commits, lastTag, nil
}

func autoVersion() (string, []string, string, error) {
	// Automatic mode based on Git history
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	lastTagBytes, err := cmd.Output()
	var logCmd *exec.Cmd
	lastTag := ""

	if err != nil {
		// No tags found, get all commits
		logCmd = exec.Command("git", "log", "--oneline")
	} else {
		lastTag = strings.TrimSpace(string(lastTagBytes))
		logCmd = exec.Command("git", "log", fmt.Sprintf("%s..HEAD", lastTag), "--oneline")
	}

	logOutput, err := logCmd.Output()
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to get git log: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(logOutput)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return "", nil, lastTag, nil // No new commits
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

	action := ""
	if bumpMajor {
		action = "major"
	} else if bumpMinor {
		action = "minor"
	} else if bumpPatch {
		action = "patch"
	}

	return action, lines, lastTag, nil
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

// syncVersion finds the first mention of the old version in the file preceded by "version" or "v"
// (case insensitive, optional punctuation/whitespace) and replaces it with the new version.
func syncVersion(filename, oldVersion, newVersion string) error {
	filePath := path.Join(config.Conf.Info.RootDir, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for sync: %w", err)
	}

	// Regex explained:
	// (?i)         : Case-insensitive
	// (version|v)  : Match "version" or "v"
	// [[:punct:]\s]* : Optional punctuation or whitespace
	// (%s)         : The old version string (quoted/escaped)
	quotedOld := regexp.QuoteMeta(oldVersion)
	reStr := fmt.Sprintf(`(?i)(version|v)[[:punct:]\s]*(%s)`, quotedOld)
	re, err := regexp.Compile(reStr)
	if err != nil {
		return fmt.Errorf("failed to compile sync regex: %w", err)
	}

	loc := re.FindSubmatchIndex(content)
	if loc == nil {
		return fmt.Errorf("could not find version '%s' with prefix 'version' or 'v' in %s", oldVersion, filename)
	}

	// Submatch indices:
	// loc[0], loc[1] : full match
	// loc[2], loc[3] : "version" or "v"
	// loc[4], loc[5] : the version string itself
	start := loc[4]
	end := loc[5]

	var newContent bytes.Buffer
	newContent.Write(content[:start])
	newContent.WriteString(newVersion)
	newContent.Write(content[end:])

	return os.WriteFile(filePath, newContent.Bytes(), 0644)
}
