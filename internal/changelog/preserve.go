package changelog

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// section is one version section of a changelog: its heading line and body.
type section struct {
	key  string // normalised version key, see versionKey
	text string // heading + body, normalised to end with exactly one blank line
}

// existingLog is a previously written changelog split into version sections.
type existingLog struct {
	preamble string              // anything before the first version heading
	sections []section           // in file order
	byKey    map[string][]string // section texts per key, in file order
}

// take returns the next unused section text for key. Multiple tags can share
// a key (e.g. both "0.9.1" and "v0.9.1" exist), so each gets its own section.
func (l *existingLog) take(key string) (string, bool) {
	texts := l.byKey[key]
	if len(texts) == 0 {
		return "", false
	}
	l.byKey[key] = texts[1:]
	return texts[0], true
}

// versionKey normalises a version name so that "v1.2.0" and "1.2.0" refer to
// the same release. Names that are not semver are compared case-insensitively.
func versionKey(name string) string {
	if v, err := semver.NewVersion(name); err == nil {
		return "v:" + v.String()
	}
	return strings.ToLower(strings.TrimSpace(name))
}

// normaliseSection trims trailing whitespace and terminates the section with
// one blank line, matching the output of the formatters.
func normaliseSection(text string) string {
	return strings.TrimRight(text, " \t\r\n") + "\n\n"
}

// parseExisting splits content into version sections using the formatter's
// heading syntax. A heading only starts a section if its name looks like a
// version or is one of the known keys (tag names, the unreleased title);
// other headings are treated as part of the surrounding text.
func parseExisting(f Formatter, content string, known map[string]bool) *existingLog {
	log := &existingLog{byKey: make(map[string][]string)}

	var cur *section
	var pre strings.Builder
	var body strings.Builder

	flush := func() {
		if cur == nil {
			return
		}
		cur.text = normaliseSection(body.String())
		log.sections = append(log.sections, *cur)
		log.byKey[cur.key] = append(log.byKey[cur.key], cur.text)
		body.Reset()
	}

	for _, line := range strings.SplitAfter(content, "\n") {
		if name, ok := f.SectionName(strings.TrimRight(line, "\r\n")); ok {
			key := versionKey(name)
			if findVer.MatchString(name) || known[key] {
				flush()
				cur = &section{key: key}
				body.WriteString(line)
				continue
			}
		}
		if cur == nil {
			pre.WriteString(line)
		} else {
			body.WriteString(line)
		}
	}
	flush()

	log.preamble = pre.String()
	return log
}

// mergeOrphans inserts sections from the existing file that don't belong to
// any git tag (e.g. hand-written history from before foonver was used) into
// the newest-first list of generated sections. Semver orphans are placed by
// version; anything else goes at the end, in file order.
func mergeOrphans(entries []section, existing *existingLog, owned map[string]bool) []section {
	for _, s := range existing.sections {
		if owned[s.key] {
			continue
		}
		owned[s.key] = true // only keep the first copy of a duplicated section

		v, err := semver.NewVersion(strings.TrimPrefix(s.key, "v:"))
		if err != nil || !strings.HasPrefix(s.key, "v:") {
			entries = append(entries, s)
			continue
		}

		pos := len(entries)
		for i, e := range entries {
			if !strings.HasPrefix(e.key, "v:") {
				continue
			}
			if ev, err := semver.NewVersion(strings.TrimPrefix(e.key, "v:")); err == nil && ev.LessThan(v) {
				pos = i
				break
			}
		}
		entries = append(entries[:pos], append([]section{s}, entries[pos:]...)...)
	}
	return entries
}
