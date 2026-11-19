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
	return generate(formatter, nextVersion, latestOnly, includeDocHeader, nil)
}

// generate builds the changelog. If existing is non-nil it is the current
// changelog content: sections for already released tags are taken from it
// verbatim instead of being regenerated, its preamble is kept, and sections
// that don't belong to any tag are carried over.
func generate(formatter Formatter, nextVersion string, latestOnly bool, includeDocHeader bool, existing *string) (string, error) {
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

	unreleasedTitle := nextVersion
	if unreleasedTitle == "" {
		unreleasedTitle = "Unreleased"
	}

	// Sections "owned" by the generator: every tag (rendered or not) and the
	// unreleased section. Existing sections with these keys are never carried
	// over as orphans.
	owned := map[string]bool{
		versionKey(unreleasedTitle): true,
		versionKey("Unreleased"):    true,
	}
	for _, tag := range tags {
		owned[versionKey(tag.Name)] = true
	}

	var prev *existingLog
	if existing != nil {
		known := make(map[string]bool, len(owned))
		for k := range owned {
			known[k] = true
		}
		prev = parseExisting(formatter, *existing, known)
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

	// Sections newest-first.
	var entries []section

	if len(renderedTags) == 0 {
		// No tags to render: fall back to full history.
		group, err := generateGroup(formatter, "", unreleasedTitle, "")
		if err != nil {
			return "", err
		}
		entries = append(entries, section{key: versionKey(unreleasedTitle), text: group})
	} else {
		// Include any unreleased changes since the last rendered tag.
		lastRenderedTag := renderedTags[len(renderedTags)-1]
		dateNow := ""
		if nextVersion != "" {
			dateNow = time.Now().Format("2006-01-02")
		}
		unreleasedRange := fmt.Sprintf("%s..HEAD", lastRenderedTag.Name)
		unreleasedCommits, err := filteredCommits(unreleasedRange, unreleasedTitle)
		if err == nil && len(unreleasedCommits) > 0 {
			group, err := generateGroup(formatter, unreleasedRange, unreleasedTitle, dateNow)
			if err == nil {
				entries = append(entries, section{key: versionKey(unreleasedTitle), text: group})
			}
		}

		// tags are oldest -> newest; render newest -> oldest
		for i := len(renderedTags) - 1; i >= 0; i-- {
			if latestOnly && len(entries) > 0 {
				break
			}
			tag := renderedTags[i]
			key := versionKey(tag.Name)

			if prev != nil {
				if text, ok := prev.take(key); ok {
					entries = append(entries, section{key: key, text: text})
					continue
				}
			}

			var revRange string
			if i == 0 {
				// First tag: include all commits up to this tag.
				revRange = tag.Name
			} else {
				revRange = fmt.Sprintf("%s..%s", renderedTags[i-1].Name, tag.Name)
			}
			group, err := generateGroup(formatter, revRange, tag.Name, tag.Date)
			if err != nil {
				return "", err
			}
			entries = append(entries, section{key: key, text: group})
		}
	}

	if prev != nil && !latestOnly {
		entries = mergeOrphans(entries, prev, owned)
	}

	var b strings.Builder
	if prev != nil && strings.TrimSpace(prev.preamble) != "" {
		b.WriteString(normaliseSection(prev.preamble))
	} else if includeDocHeader {
		b.WriteString(formatter.DocumentHeader())
	}
	for _, e := range entries {
		b.WriteString(e.text)
		if latestOnly {
			break
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

// regionBounds locates the changelog region in existing content: it starts on
// the line after startPattern and ends right before endPattern (or at the end
// of the content when endPattern is empty).
func regionBounds(existing string, startPattern string, endPattern string) (int, int, error) {
	startIdx := strings.Index(existing, startPattern)
	if startIdx == -1 {
		return 0, 0, fmt.Errorf("start pattern %q not found in content", startPattern)
	}

	afterStartIdx := startIdx + len(startPattern)
	lineEndIdx := strings.IndexByte(existing[afterStartIdx:], '\n')
	if lineEndIdx != -1 {
		afterStartIdx += lineEndIdx + 1
	} else {
		afterStartIdx = len(existing)
	}

	if endPattern == "" {
		return afterStartIdx, len(existing), nil
	}

	endIdx := strings.Index(existing[afterStartIdx:], endPattern)
	if endIdx == -1 {
		return 0, 0, fmt.Errorf("end pattern %q not found after start pattern in content", endPattern)
	}
	return afterStartIdx, afterStartIdx + endIdx, nil
}

// InjectChangelog replaces the changelog section in existing content bounded by startPattern and optional endPattern.
func InjectChangelog(existing string, changelog string, startPattern string, endPattern string) (string, error) {
	regionStart, regionEnd, err := regionBounds(existing, startPattern, endPattern)
	if err != nil {
		return "", err
	}

	before := strings.TrimRight(existing[:regionStart], "\r\n")
	trimmedChangelog := strings.TrimSpace(changelog)

	if endPattern == "" {
		if trimmedChangelog == "" {
			return before + "\n", nil
		}
		return before + "\n\n" + trimmedChangelog + "\n", nil
	}

	after := strings.TrimLeft(existing[regionEnd:], "\r\n")

	if trimmedChangelog == "" {
		return before + "\n\n" + after, nil
	}
	return before + "\n\n" + trimmedChangelog + "\n\n" + after, nil
}

// WriteChangelog generates the changelog and writes or injects it into the configured file.
func WriteChangelog(nextVersion string) (string, error) {
	filePath, content, err := RenderChangelog(nextVersion)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write changelog to %s: %w", filePath, err)
	}
	return filePath, nil
}

// RenderChangelog returns the configured changelog file path and the full
// content it should have after updating, without writing anything.
//
// Unless config.Conf.ChangelogForce is set, sections of already released
// versions that exist in the file are kept as they are, so manual edits
// survive. Deleting a section makes it be regenerated.
func RenderChangelog(nextVersion string) (string, string, error) {
	format := config.Conf.ChangelogFormat
	if format == "" {
		format = "markdown"
	}
	formatter, err := GetFormatter(format)
	if err != nil {
		return "", "", err
	}

	filePath := path.Join(config.Conf.Info.RootDir, config.Conf.ChangelogFile)
	startPattern := config.Conf.ChangelogStart
	endPattern := config.Conf.ChangelogEnd
	force := config.Conf.ChangelogForce

	if startPattern == "" {
		var existing *string
		if !force {
			data, err := os.ReadFile(filePath)
			if err != nil && !os.IsNotExist(err) {
				return "", "", fmt.Errorf("failed to read changelog file %s: %w", filePath, err)
			}
			if err == nil {
				str := string(data)
				existing = &str
			}
		}
		content, err := generate(formatter, nextVersion, false, true, existing)
		if err != nil {
			return "", "", err
		}
		return filePath, content, nil
	}

	existingBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read changelog file %s for pattern replacement: %w", filePath, err)
	}
	existingContent := string(existingBytes)

	var region *string
	if !force {
		regionStart, regionEnd, err := regionBounds(existingContent, startPattern, endPattern)
		if err != nil {
			return "", "", fmt.Errorf("failed to update changelog in %s: %w", filePath, err)
		}
		r := existingContent[regionStart:regionEnd]
		region = &r
	}

	changelogContent, err := generate(formatter, nextVersion, false, false, region)
	if err != nil {
		return "", "", err
	}

	updatedContent, err := InjectChangelog(existingContent, changelogContent, startPattern, endPattern)
	if err != nil {
		return "", "", fmt.Errorf("failed to update changelog in %s: %w", filePath, err)
	}

	return filePath, updatedContent, nil
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
