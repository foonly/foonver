package changelog

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"foonly.dev/foonver/internal/config"
	"foonly.dev/foonver/internal/git"
)

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
func GenerateMarkdown(nextVersion string) (string, error) {
	tags, err := git.GetTags()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# Changelog\n\n")

	// No tags: fall back to full history.
	if len(tags) == 0 {
		title := nextVersion
		if title == "" {
			title = "Unreleased"
		}
		group, err := generateGroup("", title, "")
		if err != nil {
			return "", err
		}
		b.WriteString(group)
		return b.String(), nil
	}

	// Include any unreleased changes since the last tag.
	lastTag := tags[len(tags)-1]
	currentTime := time.Now()

	title := "Unreleased"
	dateNow := ""
	if nextVersion != "" {
		title = nextVersion
		dateNow = currentTime.Format("2006-01-02")
	}
	unreleasedCommits, err := filteredCommits(fmt.Sprintf("%s..HEAD", lastTag.Name), title)
	if err == nil && len(unreleasedCommits) > 0 {
		group, err := generateGroup(fmt.Sprintf("%s..HEAD", lastTag.Name), title, dateNow)
		if err == nil {
			b.WriteString(group)
		}
	}

	// tags are oldest -> newest; render newest -> oldest
	for i := len(tags) - 1; i >= 0; i-- {
		tag := tags[i]

		var revRange string
		if i == 0 {
			// First tag: include all commits up to this tag.
			revRange = tag.Name
		} else {
			prev := tags[i-1]
			revRange = fmt.Sprintf("%s..%s", prev.Name, tag.Name)
		}
		group, err := generateGroup(revRange, tag.Name, tag.Date)
		if err != nil {
			return "", err
		}
		b.WriteString(group)
	}

	return b.String(), nil
}

// WriteChangelog generates the markdown and writes it to the configured file.
func WriteChangelog(nextVersion string) (string, error) {
	md, err := GenerateMarkdown(nextVersion)
	if err != nil {
		return "", err
	}

	filePath := path.Join(config.Conf.Info.RootDir, config.Conf.File)
	if err := os.WriteFile(filePath, []byte(md), 0644); err != nil {
		return "", fmt.Errorf("failed to write changelog to %s: %w", filePath, err)
	}

	return filePath, nil
}

func generateGroup(revRange string, name string, date string) (string, error) {
	commits, err := filteredCommits(revRange, name)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	title := name
	lvl := "### "
	if strings.HasSuffix(name, ".0") {
		lvl = "## "
	}

	if date != "" {
		title = fmt.Sprintf("%s (%s)", name, date)
	}
	b.WriteString(lvl + title + "\n\n")
	if len(commits) > 0 {
		renderGroupedCommits(&b, commits)
	}
	return b.String(), nil
}

var findVer = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
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

func renderGroupedCommits(b *strings.Builder, commits []string) {
	groups := make(map[string][]string)
	for _, c := range commits {
		parts := strings.SplitN(c, " ", 2)
		hash := parts[0]
		matches := typeRegex.FindStringSubmatch(parts[1])
		if len(matches) > 1 {
			scp := ""
			tpe := strings.ToLower(matches[1])
			msg := matches[4]
			if matches[2] != "" {
				scp = fmt.Sprintf("%s: ", matches[2])
			}
			if matches[3] == "!" {
				msg += " (BREAKING CHANGE)"
			}

			groups[tpe] = append(groups[tpe], fmt.Sprintf("%s%s (%s)", scp, msg, hash))
		} else {
			groups["misc"] = append(groups["misc"], fmt.Sprintf("%s (%s)", parts[1], hash))
		}
	}

	// Preferred order for common types
	order := []string{"feat", "fix", "revert", "perf", "refactor", "docs", "style", "test", "build", "ci", "chore"}
	seen := make(map[string]bool)

	for _, t := range order {
		if items, ok := groups[t]; ok {
			title := typeTitles[t]
			b.WriteString("#### " + title + "\n")
			for _, item := range items {
				b.WriteString("- " + item + "\n")
			}
			b.WriteString("\n")
			seen[t] = true
		}
	}

	// Any other types discovered
	var remaining []string
	for t := range groups {
		if !seen[t] && t != "misc" {
			remaining = append(remaining, t)
		}
	}
	sort.Strings(remaining)
	for _, t := range remaining {
		title := strings.ToUpper(t[:1]) + t[1:]
		b.WriteString("### " + title + "\n")
		for _, item := range groups[t] {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}

	// Misc last
	if items, ok := groups["misc"]; ok {
		b.WriteString("### Misc\n")
		for _, item := range items {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}
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

		if findVer.MatchString(msg) {
			continue
		}

		filtered = append(filtered, c)
	}
	return filtered, nil
}
