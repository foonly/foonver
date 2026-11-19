package changelog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/foonly/foonver/internal/config"
)

// setupRepo creates a temporary git repository, chdirs into it and points the
// config at it. The returned function runs git commands inside it.
func setupRepo(t *testing.T) (string, func(args ...string)) {
	t.Helper()
	dir := t.TempDir()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, string(out))
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")
	runGit("config", "commit.gpgsign", "false")
	runGit("config", "tag.gpgsign", "false")

	oldConf := config.Conf
	oldCwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.Conf = oldConf
		os.Chdir(oldCwd)
	})

	config.Conf.Info.RootDir = dir
	config.Conf.ChangelogFile = "CHANGELOG.md"
	config.Conf.ChangelogFormat = "markdown"
	config.Conf.ChangelogStart = ""
	config.Conf.ChangelogEnd = ""
	config.Conf.ChangelogForce = false
	config.Conf.IncludePrereleases = false
	return dir, runGit
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertSingleTrailingNewline(t *testing.T, content string) {
	t.Helper()
	if !strings.HasSuffix(content, "\n") || strings.HasSuffix(content, "\n\n") {
		t.Errorf("expected content to end with exactly one newline, got tail %q", content[max(0, len(content)-20):])
	}
}

func TestParseExistingSections(t *testing.T) {
	content := "# Changelog\n\nSome intro.\n\n## Notes heading that is not a version\n\n" +
		"## v1.1.0 (2026-01-02)\n\n#### Features\n\n- a (abc)\n\n" +
		"### 1.0.1\n\n- b\n" +
		"## v1.0.0\n\n- c\n"
	log := parseExisting(&MarkdownFormatter{}, content, map[string]bool{})

	if !strings.Contains(log.preamble, "Some intro.") || !strings.Contains(log.preamble, "Notes heading") {
		t.Errorf("unexpected preamble: %q", log.preamble)
	}
	if len(log.sections) != 3 {
		t.Fatalf("expected 3 sections, got %d: %+v", len(log.sections), log.sections)
	}
	if got, _ := log.take(versionKey("v1.0.1")); got != "### 1.0.1\n\n- b\n\n" {
		t.Errorf("unexpected 1.0.1 section: %q", got)
	}
	if got, _ := log.take(versionKey("1.1.0")); !strings.HasPrefix(got, "## v1.1.0 (2026-01-02)\n") || !strings.Contains(got, "#### Features") {
		t.Errorf("unexpected 1.1.0 section: %q", got)
	}

	wp := parseExisting(&WordPressFormatter{}, "== Changelog ==\n\n= 1.0.0 (2026-01-01) =\n\n* x\n\n= v0.9.0 =\n* y\n", map[string]bool{})
	if len(wp.sections) != 2 || wp.sections[1].key != versionKey("0.9.0") {
		t.Errorf("unexpected wordpress sections: %+v", wp.sections)
	}
}

func TestWriteChangelogPreservesEdits(t *testing.T) {
	dir, runGit := setupRepo(t)
	path := filepath.Join(dir, "CHANGELOG.md")

	runGit("commit", "--allow-empty", "-m", "feat: first feature")
	runGit("tag", "v1.0.0")
	runGit("commit", "--allow-empty", "-m", "fix: second fix")
	runGit("tag", "v1.1.0")
	runGit("commit", "--allow-empty", "-m", "feat: third feature")
	runGit("tag", "v1.2.0")

	if _, err := WriteChangelog(""); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}
	initial := readFile(t, path)
	assertSingleTrailingNewline(t, initial)
	for _, want := range []string{"first feature", "second fix", "third feature"} {
		if !strings.Contains(initial, want) {
			t.Fatalf("expected %q in initial changelog:\n%s", want, initial)
		}
	}

	// Hand edit: custom intro, reworded v1.2.0 (using an unprefixed heading),
	// drop the v1.1.0 section entirely, and add pre-history at the bottom.
	edited := "# Changelog\n\nAll notable changes.\n\n" +
		"## 1.2.0 (2026-01-01)\n\n- Hand-written release notes\n\n" +
		"## v1.0.0\n\n- Edited first release\n\n" +
		"## 0.9.0\n\n- Before foonver\n"
	if err := os.WriteFile(path, []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}

	runGit("commit", "--allow-empty", "-m", "fix: unreleased fix")
	if _, err := WriteChangelog("v1.3.0"); err != nil {
		t.Fatalf("preserving write failed: %v", err)
	}
	got := readFile(t, path)
	assertSingleTrailingNewline(t, got)

	checks := []string{
		"All notable changes.",
		"## 1.2.0 (2026-01-01)\n\n- Hand-written release notes",
		"- Edited first release",
		"second fix",     // v1.1.0 regenerated because it was deleted
		"unreleased fix", // new release section generated
		"## 0.9.0\n\n- Before foonver",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in changelog:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"third feature", "first feature"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("edited section was regenerated (found %q):\n%s", unwanted, got)
		}
	}

	order := []string{"v1.3.0", "1.2.0", "v1.1.0", "v1.0.0", "0.9.0"}
	last := -1
	for _, h := range order {
		idx := strings.Index(got, "## "+h)
		if idx == -1 {
			idx = strings.Index(got, "### "+h)
		}
		if idx <= last {
			t.Errorf("section %q out of order (idx %d, previous %d):\n%s", h, idx, last, got)
		}
		last = idx
	}

	// Running again is stable.
	if _, err := WriteChangelog("v1.3.0"); err != nil {
		t.Fatal(err)
	}
	if again := readFile(t, path); again != got {
		t.Errorf("second write changed output:\n--- first\n%s\n--- second\n%s", got, again)
	}

	// Force regenerates everything and drops hand-written content.
	config.Conf.ChangelogForce = true
	if _, err := WriteChangelog("v1.3.0"); err != nil {
		t.Fatal(err)
	}
	forced := readFile(t, path)
	assertSingleTrailingNewline(t, forced)
	for _, unwanted := range []string{"All notable changes.", "Hand-written", "Edited first release", "0.9.0"} {
		if strings.Contains(forced, unwanted) {
			t.Errorf("forced write kept %q:\n%s", unwanted, forced)
		}
	}
	for _, want := range []string{"first feature", "third feature"} {
		if !strings.Contains(forced, want) {
			t.Errorf("forced write missing %q:\n%s", want, forced)
		}
	}
}

func TestWriteChangelogDropsPromotedPrereleases(t *testing.T) {
	dir, runGit := setupRepo(t)
	path := filepath.Join(dir, "CHANGELOG.md")

	runGit("commit", "--allow-empty", "-m", "feat: stable")
	runGit("tag", "v1.0.0")
	runGit("commit", "--allow-empty", "-m", "feat: beta work")
	runGit("tag", "v1.1.0-beta.1")

	if _, err := WriteChangelog(""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, path), "v1.1.0-beta.1") {
		t.Fatalf("expected beta section before promotion")
	}

	if _, err := WriteChangelog("v1.1.0"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, path)
	if strings.Contains(got, "v1.1.0-beta.1") {
		t.Errorf("beta section should be dropped after promotion:\n%s", got)
	}
	if !strings.Contains(got, "beta work") {
		t.Errorf("beta commits should be consolidated under v1.1.0:\n%s", got)
	}
}

func TestWriteChangelogPreservesWordPressRegion(t *testing.T) {
	dir, runGit := setupRepo(t)
	path := filepath.Join(dir, "readme.txt")
	config.Conf.ChangelogFile = "readme.txt"
	config.Conf.ChangelogFormat = "wordpress"
	config.Conf.ChangelogStart = "== Changelog =="
	config.Conf.ChangelogEnd = "== Upgrade Notice =="

	runGit("commit", "--allow-empty", "-m", "feat: plugin feature")
	runGit("tag", "v1.0.0")
	runGit("commit", "--allow-empty", "-m", "fix: plugin fix")

	readme := "=== Plugin ===\n\n== Changelog ==\n\n= 1.0.0 =\n\n* Edited by hand\n\n= 0.1.0 =\n* Initial\n\n== Upgrade Notice ==\nUpgrade!\n"
	if err := os.WriteFile(path, []byte(readme), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := WriteChangelog("v1.0.1"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, path)
	for _, want := range []string{"=== Plugin ===", "* Fix: plugin fix", "= 1.0.0 =\n\n* Edited by hand", "= 0.1.0 =\n* Initial", "== Upgrade Notice ==\nUpgrade!"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in readme:\n%s", want, got)
		}
	}
	if strings.Contains(got, "plugin feature") {
		t.Errorf("1.0.0 section should not be regenerated:\n%s", got)
	}
}

func TestWriteChangelogDuplicateVersionTags(t *testing.T) {
	dir, runGit := setupRepo(t)
	path := filepath.Join(dir, "CHANGELOG.md")

	runGit("commit", "--allow-empty", "-m", "feat: first")
	runGit("tag", "v1.0.0")
	runGit("commit", "--allow-empty", "-m", "fix: second")
	runGit("tag", "1.0.1")
	runGit("tag", "v1.0.1") // same release tagged twice

	if _, err := WriteChangelog(""); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, path)
	if _, err := WriteChangelog(""); err != nil {
		t.Fatal(err)
	}
	if second := readFile(t, path); second != first {
		t.Errorf("rewrite changed output:\n--- first\n%s\n--- second\n%s", first, second)
	}
	if !strings.Contains(first, "- second") || strings.Count(first, "1.0.1") != 2 {
		t.Errorf("expected both 1.0.1 sections and the fix:\n%s", first)
	}
}
