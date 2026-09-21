package changelog

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/foonly/foonver/internal/config"
	"github.com/foonly/foonver/internal/git"
)

var findVer = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
var typeRegex = regexp.MustCompile(`(?i)^([a-z]+)(?:\((.*)\))?(!)?:\s*(.*)$`)

var typeTitles = map[string]string{
	"feat":     "Features",
	"fix":      "Bug Fixes",
	"revert":   "Reverts",
	"chore":    "Maintenance",
	"docs":     "Documentation",
	"style":    "Styles",
	"refactor": "Refactor",
	"perf":     "Performance Improvements",
	"test":     "Tests",
	"build":    "Build System",
	"ci":       "Continuous Integration",
}

// Generate builds a changelog in the specified format from git tags and commits.
func Generate(format string, nextVersion string, latestOnly bool) (string, error) {
	formatter, err := GetFormatter(format)
	if err != nil {
		return "", err
	}
	return GenerateWithFormatter(formatter, nextVersion, latestOnly, !latestOnly)
}

// GenerateMarkdown builds a changelog in Markdown from git tags and commits.
//
// Output format:
//
//	# Changelog
//
//	## v1.2.0
//
//	### Features
//	- abc1234 feat: some feature
//
//	### Bug Fixes
//	- def5678 fix: some bug
//
//	## v1.1.0
//	- ...
//
// Tags are discovered using creation date order and rendered newest-first.
// Each tag section includes commits between previousTag..tag.
// The first tag includes all commits reachable from that tag.
// If there are commits since the last tag, they are grouped under nextVersion (or "Unreleased" if empty).
func GenerateMarkdown(nextVersion string, latestOnly bool) (string, error) {
	return Generate("markdown", nextVersion, latestOnly)
}

// GenerateWithFormatter builds a changelog using the given Formatter.
func GenerateWithFormatter(formatter Formatter, nextVersion string, latestOnly bool, includeDocHeader bool) (string, error) {
	tags, err := git.GetTags()
	if err != nil {
		return "", err
	}

	sort.SliceStable(tags, func(i, j int) bool {
		v1, err1 := semver.NewVersion(tags[i].Name)
		v2, err2 := semver.NewVersion(tags[j].Name)
		if err1 == nil && err2 == nil {
			return v1.LessThan(v2)
		}
		if err1 == nil {
			return false
		}
		if err2 == nil {
			return true
		}
		// Neither parses as semver: leave their relative order untouched
		// (SliceStable already preserves the original order for ties).
		return false
	})

	var b strings.Builder
	if includeDocHeader {
		b.WriteString(formatter.DocumentHeader())
	}

	includePrereleases := config.Conf.IncludePrereleases

	inPrereleaseMode := false
	if nextVersion != "" && nextVersion != "Unreleased" {
		inPrereleaseMode = isPrerelease(nextVersion)
	} else if len(tags) > 0 {
		inPrereleaseMode = isPrerelease(tags[len(tags)-1].Name)
	}

	lastStableIdx := -1
	for i := len(tags) - 1; i >= 0; i-- {
		if !isPrerelease(tags[i].Name) {
			lastStableIdx = i
			break
		}
	}

	var renderedTags []git.Tag
	if includePrereleases {
		renderedTags = tags
	} else {
		for i, tag := range tags {
			if !isPrerelease(tag.Name) {
				renderedTags = append(renderedTags, tag)
			} else if inPrereleaseMode && i > lastStableIdx {
				renderedTags = append(renderedTags, tag)
			}
		}
	}

	// No tags to render: fall back to full history.
	if len(renderedTags) == 0 {
		title := nextVersion
		if title == "" {
			title = "Unreleased"
		}
		group, err := generateGroup(formatter, "", title, "")
		if err != nil {
			return "", err
		}
		b.WriteString(group)
		return b.String(), nil
	}

	// Include any unreleased changes since the last rendered tag.
	lastRenderedTag := renderedTags[len(renderedTags)-1]
	currentTime := time.Now()

	title := "Unreleased"
	dateNow := ""
	if nextVersion != "" {
		title = nextVersion
		dateNow = currentTime.Format("2006-01-02")
	}
	unreleasedCommits, err := filteredCommits(fmt.Sprintf("%s..HEAD", lastRenderedTag.Name), title)
	if err == nil && len(unreleasedCommits) > 0 {
		group, err := generateGroup(formatter, fmt.Sprintf("%s..HEAD", lastRenderedTag.Name), title, dateNow)
		if err == nil {
			b.WriteString(group)
			if latestOnly {
				return b.String(), nil
			}
		}
	}

	// tags are oldest -> newest; render newest -> oldest
	for i := len(renderedTags) - 1; i >= 0; i-- {
		tag := renderedTags[i]

		var revRange string
		if i == 0 {
			// First tag: include all commits up to this tag.
			revRange = tag.Name
		} else {
			prev := renderedTags[i-1]
			revRange = fmt.Sprintf("%s..%s", prev.Name, tag.Name)
		}
		group, err := generateGroup(formatter, revRange, tag.Name, tag.Date)
		if err != nil {
			return "", err
		}
		b.WriteString(group)

		if latestOnly {
			return b.String(), nil
		}
	}

	return b.String(), nil
}

func isPrerelease(tagName string) bool {
	v, err := semver.NewVersion(tagName)
	if err != nil {
		return false
	}
	return v.Prerelease() != ""
}

// InjectChangelog replaces the changelog section in existing content bounded by startPattern and optional endPattern.
func InjectChangelog(existing string, changelog string, startPattern string, endPattern string) (string, error) {
	startIdx := strings.Index(existing, startPattern)
	if startIdx == -1 {
		return "", fmt.Errorf("start pattern %q not found in content", startPattern)
	}

	afterStartIdx := startIdx + len(startPattern)
	lineEndIdx := strings.IndexByte(existing[afterStartIdx:], '\n')
	if lineEndIdx != -1 {
		afterStartIdx += lineEndIdx + 1
	} else {
		afterStartIdx = len(existing)
	}

	before := strings.TrimRight(existing[:afterStartIdx], "\r\n")
	trimmedChangelog := strings.TrimSpace(changelog)

	if endPattern == "" {
		if trimmedChangelog == "" {
			return before + "\n", nil
		}
		return before + "\n\n" + trimmedChangelog + "\n", nil
	}

	rest := existing[afterStartIdx:]
	endIdx := strings.Index(rest, endPattern)
	if endIdx == -1 {
		return "", fmt.Errorf("end pattern %q not found after start pattern in content", endPattern)
	}

	after := strings.TrimLeft(rest[endIdx:], "\r\n")

	if trimmedChangelog == "" {
		return before + "\n\n" + after, nil
	}
	return before + "\n\n" + trimmedChangelog + "\n\n" + after, nil
}

// WriteChangelog generates the changelog and writes or injects it into the configured file.
func WriteChangelog(nextVersion string) (string, error) {
	format := config.Conf.ChangelogFormat
	if format == "" {
		format = "markdown"
	}
	formatter, err := GetFormatter(format)
	if err != nil {
		return "", err
	}

	filePath := path.Join(config.Conf.Info.RootDir, config.Conf.File)
	startPattern := config.Conf.ChangelogStart
	endPattern := config.Conf.ChangelogEnd

	if startPattern == "" {
		content, err := GenerateWithFormatter(formatter, nextVersion, false, true)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write changelog to %s: %w", filePath, err)
		}
		return filePath, nil
	}

	existingBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read changelog file %s for pattern replacement: %w", filePath, err)
	}

	changelogContent, err := GenerateWithFormatter(formatter, nextVersion, false, false)
	if err != nil {
		return "", err
	}

	updatedContent, err := InjectChangelog(string(existingBytes), changelogContent, startPattern, endPattern)
	if err != nil {
		return "", fmt.Errorf("failed to update changelog in %s: %w", filePath, err)
	}

	if err := os.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write changelog to %s: %w", filePath, err)
	}

	return filePath, nil
}

func generateGroup(formatter Formatter, revRange string, name string, date string) (string, error) {
	commits, err := filteredCommits(revRange, name)
	if err != nil {
		return "", err
	}

	return formatter.FormatGroup(name, date, commits), nil
}

func filteredCommits(revRange string, tag string) ([]string, error) {
	commits, err := git.GetCommits(revRange)
	if err != nil {
		return nil, err
	}

	var filtered []string
	for _, c := range commits {
		parts := strings.SplitN(c, " ", 2)
		if len(parts) < 2 {
			continue
		}
		msg := strings.TrimSpace(parts[1])

		if msg == tag {
			continue
		}

		if strings.HasPrefix(msg, "Merge ") {
			continue
		}

		if strings.Contains(msg, "[skip ci]") || strings.Contains(msg, "[skip action]") {
			continue
		}

		if findVer.MatchString(msg) {
			continue
		}

		filtered = append(filtered, c)
	}
	return filtered, nil
}
