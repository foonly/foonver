package version

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	tomlTableHeader      = regexp.MustCompile(`^\s*\[\s*([^\[\]]+?)\s*\]\s*(?:#.*)?$`)
	tomlArrayTableHeader = regexp.MustCompile(`^\s*\[\[\s*([^\[\]]+?)\s*\]\]\s*(?:#.*)?$`)
	tomlStringAssignment = regexp.MustCompile(`^\s*([A-Za-z0-9_-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
)

// span is a half-open byte range [start, end) within a file.
type span struct{ start, end int }

// tomlEntry is a `key = "string"` assignment found by scanToml.
type tomlEntry struct {
	table string // normalized table name ("" for the root table)
	index int    // occurrence number for array tables ([[table]]), -1 for regular tables
	key   string
	value string
	pos   span // location of the value, excluding quotes
}

// normalizeTomlTable turns a header like `tool . "poetry"` into `tool.poetry`.
func normalizeTomlTable(name string) string {
	parts := strings.Split(name, ".")
	for i, p := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(p), `"'`)
	}
	return strings.Join(parts, ".")
}

// scanToml is a minimal, line-based TOML scanner that reports single-line string assignments
// together with the table they belong to and their byte offsets. It does not handle multi-line
// strings or dotted keys, which is sufficient for locating versions in manifests and lock files
// while leaving the rest of the file byte-for-byte untouched.
func scanToml(content []byte) []tomlEntry {
	var entries []tomlEntry
	current, index := "", -1
	arrayCounts := map[string]int{}

	offset := 0
	for _, line := range bytes.SplitAfter(content, []byte("\n")) {
		lineStart := offset
		offset += len(line)
		trimmed := bytes.TrimRight(line, "\r\n")

		if m := tomlArrayTableHeader.FindSubmatch(trimmed); m != nil {
			current = normalizeTomlTable(string(m[1]))
			index = arrayCounts[current]
			arrayCounts[current]++
			continue
		}
		if m := tomlTableHeader.FindSubmatch(trimmed); m != nil {
			current, index = normalizeTomlTable(string(m[1])), -1
			continue
		}
		loc := tomlStringAssignment.FindSubmatchIndex(trimmed)
		if loc == nil {
			continue
		}
		start, end := loc[4], loc[5]
		if start < 0 {
			start, end = loc[6], loc[7]
		}
		entries = append(entries, tomlEntry{
			table: current,
			index: index,
			key:   string(trimmed[loc[2]:loc[3]]),
			value: string(trimmed[start:end]),
			pos:   span{lineStart + start, lineStart + end},
		})
	}
	return entries
}

// findTomlKey returns the first string assignment of key in the first of the given tables that
// has one, along with the table it was found in. A nil tables slice matches any regular table.
func findTomlKey(content []byte, key string, tables []string) (tomlEntry, bool) {
	entries := scanToml(content)
	if tables == nil {
		for _, e := range entries {
			if e.key == key && e.index == -1 {
				return e, true
			}
		}
		return tomlEntry{}, false
	}
	for _, table := range tables {
		for _, e := range entries {
			if e.key == key && e.index == -1 && e.table == table {
				return e, true
			}
		}
	}
	return tomlEntry{}, false
}

// findTomlVersion locates the version value in TOML content and returns it together with its
// byte offsets (excluding the quotes). For known manifests only the configured tables are
// considered, so versions in e.g. [dependencies.foo] or [workspace.dependencies] are ignored.
func findTomlVersion(filename string, content []byte) (string, int, int, error) {
	tables, known := tomlVersionTables[filepath.Base(filename)]
	if !known {
		tables = nil
	}
	e, ok := findTomlKey(content, "version", tables)
	if !ok {
		if known {
			return "", 0, 0, fmt.Errorf("no version string found in [%s]", strings.Join(tables, "] or ["))
		}
		return "", 0, 0, fmt.Errorf("no version string found")
	}
	return e.value, e.pos.start, e.pos.end, nil
}

// replaceSpans replaces each span (which must be sorted and non-overlapping) with repl.
func replaceSpans(content []byte, spans []span, repl string) []byte {
	var out bytes.Buffer
	last := 0
	for _, s := range spans {
		out.Write(content[last:s.start])
		out.WriteString(repl)
		last = s.end
	}
	out.Write(content[last:])
	return out.Bytes()
}
