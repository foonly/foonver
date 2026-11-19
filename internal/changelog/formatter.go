package changelog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/foonly/foonver/internal/config"
)

// scopeRegex matches "scope: message" style commits (e.g. "gui: move button")
// whose prefix is not a recognised conventional commit type.
var scopeRegex = regexp.MustCompile(`^([A-Za-z0-9][\w./-]*)(!)?:\s+(.+)$`)

// isKnownType reports whether t is a conventional commit type that gets its
// own changelog section: either a built-in type or one listed in the
// changelog-types config option.
func isKnownType(t string) bool {
	if _, ok := typeTitles[t]; ok {
		return true
	}
	for _, ct := range config.Conf.ChangelogTypes {
		if strings.EqualFold(strings.TrimSpace(ct), t) {
			return true
		}
	}
	return false
}

// sortMisc orders uncategorised commits so that scoped entries come first,
// clustered alphabetically by scope, followed by unscoped entries. Commit
// order is preserved within each scope.
func sortMisc(items []CommitInfo) {
	sort.SliceStable(items, func(i, j int) bool {
		si, sj := strings.ToLower(items[i].Scope), strings.ToLower(items[j].Scope)
		if (si == "") != (sj == "") {
			return si != ""
		}
		return si < sj
	})
}

// CommitInfo represents structured information parsed from a conventional commit message.
type CommitInfo struct {
	Hash     string
	Type     string
	Scope    string
	Message  string
	Breaking bool
	Raw      string
}

// Formatter defines the interface for generating changelogs in different formats.
type Formatter interface {
	DocumentHeader() string
	FormatGroup(name string, date string, commits []string) string
	// SectionName reports whether line is a version section heading in this
	// format and, if so, returns the version name it names.
	SectionName(line string) (string, bool)
}

var mdSectionRegex = regexp.MustCompile(`^#{2,3}\s+(\S+)(?:\s+\(.*\))?\s*$`)
var wpSectionRegex = regexp.MustCompile(`^=\s*([^=\s]\S*)(?:\s+\(.*\))?\s*=\s*$`)

func matchSection(re *regexp.Regexp, line string) (string, bool) {
	m := re.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// GetFormatter returns the Formatter implementation for the given format name.
func GetFormatter(format string) (Formatter, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return &MarkdownFormatter{}, nil
	case "wordpress", "wp", "readme", "txt":
		return &WordPressFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown changelog format %q, supported formats are: markdown, wordpress", format)
	}
}

func parseCommit(c string) CommitInfo {
	parts := strings.SplitN(c, " ", 2)
	hash := parts[0]
	if len(parts) < 2 {
		return CommitInfo{
			Hash:    hash,
			Type:    "misc",
			Message: hash,
			Raw:     c,
		}
	}

	matches := typeRegex.FindStringSubmatch(parts[1])
	if len(matches) > 1 && isKnownType(strings.ToLower(matches[1])) {
		return CommitInfo{
			Hash:     hash,
			Type:     strings.ToLower(matches[1]),
			Scope:    matches[2],
			Breaking: matches[3] == "!",
			Message:  matches[4],
			Raw:      parts[1],
		}
	}

	// Not a known type: treat "something: message" as a scoped misc commit.
	if m := scopeRegex.FindStringSubmatch(parts[1]); m != nil {
		return CommitInfo{
			Hash:     hash,
			Type:     "misc",
			Scope:    m[1],
			Breaking: m[2] == "!",
			Message:  m[3],
			Raw:      parts[1],
		}
	}

	return CommitInfo{
		Hash:    hash,
		Type:    "misc",
		Message: parts[1],
		Raw:     parts[1],
	}
}

// MarkdownFormatter formats changelogs in standard Markdown.
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) DocumentHeader() string {
	return "# Changelog\n\n"
}

func (f *MarkdownFormatter) SectionName(line string) (string, bool) {
	return matchSection(mdSectionRegex, line)
}

func (f *MarkdownFormatter) FormatGroup(name string, date string, commits []string) string {
	var b strings.Builder
	title := name
	lvl := "### "
	if strings.HasSuffix(name, ".0") {
		lvl = "## "
	}

	if date != "" {
		title = fmt.Sprintf("%s (%s)", name, date)
	}
	b.WriteString(lvl)
	b.WriteString(title)
	b.WriteString("\n\n")
	if len(commits) > 0 {
		renderGroupedCommitsMarkdown(&b, commits)
	}
	return b.String()
}

func renderGroupedCommitsMarkdown(b *strings.Builder, commits []string) {
	format := func(info CommitInfo) string {
		scp := ""
		msg := info.Message
		if info.Scope != "" {
			scp = fmt.Sprintf("%s: ", info.Scope)
		}
		if info.Breaking {
			msg += " (BREAKING CHANGE)"
		}
		return fmt.Sprintf("%s%s (%s)", scp, msg, info.Hash)
	}

	groups := make(map[string][]string)
	var misc []CommitInfo
	for _, c := range commits {
		info := parseCommit(c)
		if info.Type == "misc" {
			misc = append(misc, info)
			continue
		}
		groups[info.Type] = append(groups[info.Type], format(info))
	}
	sortMisc(misc)
	for _, info := range misc {
		groups["misc"] = append(groups["misc"], format(info))
	}

	order := []string{"feat", "fix", "revert", "perf", "refactor", "docs", "style", "test", "build", "ci", "chore"}
	seen := make(map[string]bool)

	for _, t := range order {
		if items, ok := groups[t]; ok {
			title := typeTitles[t]
			b.WriteString("#### ")
			b.WriteString(title)
			b.WriteString("\n\n")
			for _, item := range items {
				b.WriteString("- ")
				b.WriteString(item)
				b.WriteString("\n")
			}
			b.WriteString("\n")
			seen[t] = true
		}
	}

	var remaining []string
	for t := range groups {
		if !seen[t] && t != "misc" {
			remaining = append(remaining, t)
		}
	}
	sort.Strings(remaining)
	for _, t := range remaining {
		title := strings.ToUpper(t[:1]) + t[1:]
		b.WriteString("#### ")
		b.WriteString(title)
		b.WriteString("\n\n")
		for _, item := range groups[t] {
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if items, ok := groups["misc"]; ok {
		b.WriteString("#### Misc\n\n")
		for _, item := range items {
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
}

var wpTypeTitles = map[string]string{
	"feat":     "Feature",
	"fix":      "Fix",
	"revert":   "Revert",
	"chore":    "Maintenance",
	"docs":     "Documentation",
	"style":    "Style",
	"refactor": "Refactor",
	"perf":     "Performance",
	"test":     "Test",
	"build":    "Build",
	"ci":       "CI",
}

// WordPressFormatter formats changelogs in WordPress readme.txt style.
type WordPressFormatter struct{}

func (f *WordPressFormatter) DocumentHeader() string {
	return "== Changelog ==\n\n"
}

func (f *WordPressFormatter) SectionName(line string) (string, bool) {
	return matchSection(wpSectionRegex, line)
}

func (f *WordPressFormatter) FormatGroup(name string, date string, commits []string) string {
	var b strings.Builder
	title := name
	if date != "" {
		title = fmt.Sprintf("%s (%s)", name, date)
	}
	b.WriteString(fmt.Sprintf("= %s =\n\n", title))
	if len(commits) > 0 {
		renderGroupedCommitsWordPress(&b, commits)
	}
	return b.String()
}

func renderGroupedCommitsWordPress(b *strings.Builder, commits []string) {
	groups := make(map[string][]string)
	var misc []CommitInfo
	for _, c := range commits {
		info := parseCommit(c)
		if info.Type == "misc" {
			misc = append(misc, info)
			continue
		}
		prefix, ok := wpTypeTitles[info.Type]
		if !ok {
			prefix = strings.ToUpper(info.Type[:1]) + info.Type[1:]
		}
		scp := ""
		msg := info.Message
		if info.Scope != "" {
			scp = fmt.Sprintf("%s - ", info.Scope)
		}
		if info.Breaking {
			msg += " (BREAKING CHANGE)"
		}
		groups[info.Type] = append(groups[info.Type], fmt.Sprintf("* %s: %s%s", prefix, scp, msg))
	}
	sortMisc(misc)
	for _, info := range misc {
		msg := info.Message
		if info.Scope != "" {
			msg = fmt.Sprintf("%s: %s", info.Scope, msg)
		}
		if info.Breaking {
			msg += " (BREAKING CHANGE)"
		}
		groups["misc"] = append(groups["misc"], fmt.Sprintf("* %s", msg))
	}

	order := []string{"feat", "fix", "revert", "perf", "refactor", "docs", "style", "test", "build", "ci", "chore"}
	seen := make(map[string]bool)

	for _, t := range order {
		if items, ok := groups[t]; ok {
			for _, item := range items {
				b.WriteString(item)
				b.WriteByte('\n')
			}
			seen[t] = true
		}
	}

	var remaining []string
	for t := range groups {
		if !seen[t] && t != "misc" {
			remaining = append(remaining, t)
		}
	}
	sort.Strings(remaining)
	for _, t := range remaining {
		for _, item := range groups[t] {
			b.WriteString(item)
			b.WriteByte('\n')
		}
	}

	if items, ok := groups["misc"]; ok {
		for _, item := range items {
			b.WriteString(item)
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n")
}
