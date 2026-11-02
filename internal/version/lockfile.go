package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// lockFileUpdate describes a pending version bump inside a lock file that belongs to the
// active version file.
type lockFileUpdate struct {
	Path    string // absolute path of the lock file
	Entries int    // number of version values that will be replaced
	content []byte // new content to write
}

func (u lockFileUpdate) apply() error {
	return os.WriteFile(u.Path, u.content, 0644)
}

// lockFileUpdater computes the spans to replace in a lock file, given the version file's
// content and the lock file's content.
type lockFileUpdater func(manifest, lock []byte, oldVersion string) ([]span, error)

// lockFiles maps a version file's base name to the lock files that record its version.
var lockFiles = map[string]map[string]lockFileUpdater{
	"package.json": {
		"package-lock.json":   npmLockSpans,
		"npm-shrinkwrap.json": npmLockSpans,
	},
	"Cargo.toml": {
		"Cargo.lock": cargoLockSpans,
	},
	"pyproject.toml": {
		"uv.lock": uvLockSpans,
	},
}

// planLockFileUpdates returns the lock-file updates needed when bumping versionFile from
// oldVersion to newVersion. Lock files that do not exist are ignored. Lock files that exist
// but contain no matching entry produce a warning instead of an update.
func planLockFileUpdates(versionFile string, manifest []byte, oldVersion, newVersion string) ([]lockFileUpdate, []string) {
	if versionFile == "" {
		return nil, nil
	}
	updaters, ok := lockFiles[filepath.Base(versionFile)]
	if !ok {
		return nil, nil
	}

	var updates []lockFileUpdate
	var warnings []string
	dir := filepath.Dir(versionFile)
	for _, name := range slices.Sorted(maps.Keys(updaters)) {
		lockPath := filepath.Join(dir, name)
		lock, err := os.ReadFile(lockPath)
		if err != nil {
			if !os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("could not read %s: %v", name, err))
			}
			continue
		}
		spans, err := updaters[name](manifest, lock, oldVersion)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("could not parse %s: %v", name, err))
			continue
		}
		if len(spans) == 0 {
			warnings = append(warnings, fmt.Sprintf("%s has no entry for version %s; leaving it unchanged", name, oldVersion))
			continue
		}
		updates = append(updates, lockFileUpdate{
			Path:    lockPath,
			Entries: len(spans),
			content: replaceSpans(lock, spans, newVersion),
		})
	}
	return updates, warnings
}

// npmLockSpans finds the root project version in package-lock.json / npm-shrinkwrap.json:
// the top-level "version" (all lockfile versions) and packages[""].version (lockfile v2+).
func npmLockSpans(_, lock []byte, oldVersion string) ([]span, error) {
	return findJSONStrings(lock, oldVersion, [][]string{
		{"version"},
		{"packages", "", "version"},
	})
}

// cargoLockSpans finds the [[package]] entries in Cargo.lock that belong to this project.
// For a crate with [package], that is the source-less entry with the crate's name. For a
// workspace with [workspace.package], it is every source-less (local) entry on the old version,
// which covers members that inherit the version via `version.workspace = true`.
func cargoLockSpans(manifest, lock []byte, oldVersion string) ([]span, error) {
	versionEntry, ok := findTomlKey(manifest, "version", tomlVersionTables["Cargo.toml"])
	if !ok {
		return nil, fmt.Errorf("no version found in Cargo.toml")
	}
	name := ""
	if versionEntry.table == "package" {
		nameEntry, ok := findTomlKey(manifest, "name", []string{"package"})
		if !ok {
			return nil, fmt.Errorf("no [package] name found in Cargo.toml")
		}
		name = nameEntry.value
	}

	var spans []span
	for _, pkg := range lockPackages(lock) {
		if pkg.source || pkg.version == nil || pkg.version.value != oldVersion {
			continue
		}
		if name != "" && pkg.name != name {
			continue
		}
		spans = append(spans, pkg.version.pos)
	}
	return spans, nil
}

var pep503Separators = regexp.MustCompile(`[-_.]+`)

func normalizePythonName(name string) string {
	return strings.ToLower(pep503Separators.ReplaceAllString(name, "-"))
}

// uvLockSpans finds the project's own [[package]] entry in uv.lock, matched by its
// normalized [project] name.
func uvLockSpans(manifest, lock []byte, oldVersion string) ([]span, error) {
	nameEntry, ok := findTomlKey(manifest, "name", []string{"project", "tool.poetry"})
	if !ok {
		return nil, fmt.Errorf("no project name found in pyproject.toml")
	}
	name := normalizePythonName(nameEntry.value)

	var spans []span
	for _, pkg := range lockPackages(lock) {
		if pkg.version == nil || pkg.version.value != oldVersion || normalizePythonName(pkg.name) != name {
			continue
		}
		spans = append(spans, pkg.version.pos)
	}
	return spans, nil
}

type lockPackage struct {
	name    string
	version *tomlEntry
	source  bool // a string `source = "..."` is present (remote package in Cargo.lock)
}

// lockPackages groups the [[package]] entries of a TOML lock file.
func lockPackages(lock []byte) []lockPackage {
	var pkgs []lockPackage
	for _, e := range scanToml(lock) {
		if e.table != "package" || e.index < 0 {
			continue
		}
		for len(pkgs) <= e.index {
			pkgs = append(pkgs, lockPackage{})
		}
		pkg := &pkgs[e.index]
		switch e.key {
		case "name":
			pkg.name = e.value
		case "version":
			if pkg.version == nil {
				entry := e
				pkg.version = &entry
			}
		case "source":
			pkg.source = true
		}
	}
	return pkgs
}

// findJSONStrings returns the byte spans (excluding quotes) of string values equal to want at
// any of the given key paths. It walks the token stream so the file can be edited in place
// without reformatting.
func findJSONStrings(content []byte, want string, paths [][]string) ([]span, error) {
	dec := json.NewDecoder(bytes.NewReader(content))
	var spans []span

	matches := func(path []string) bool {
		for _, p := range paths {
			if len(p) != len(path) {
				continue
			}
			equal := true
			for i := range p {
				if p[i] != path[i] {
					equal = false
					break
				}
			}
			if equal {
				return true
			}
		}
		return false
	}

	var walk func(path []string) error
	walk = func(path []string) error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				for dec.More() {
					k, err := dec.Token()
					if err != nil {
						return err
					}
					key, _ := k.(string)
					if err := walk(append(path[:len(path):len(path)], key)); err != nil {
						return err
					}
				}
				_, err = dec.Token()
				return err
			case '[':
				for dec.More() {
					if err := walk(append(path[:len(path):len(path)], "[]")); err != nil {
						return err
					}
				}
				_, err = dec.Token()
				return err
			}
		case string:
			if t == want && matches(path) {
				end := int(dec.InputOffset()) - 1 // index of the closing quote
				start := bytes.LastIndexByte(content[:end], '"') + 1
				if string(content[start:end]) == want {
					spans = append(spans, span{start, end})
				}
			}
		}
		return nil
	}

	if err := walk(nil); err != nil {
		return nil, err
	}
	return spans, nil
}
