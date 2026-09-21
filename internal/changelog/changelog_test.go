package changelog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/foonly/foonver/internal/config"
)

func TestFindVerRegex(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"0.0.0", true},
		{"1.2.3", true},
		{"v1.2.3", true},
		{"v0.11.0", true},
		{"10.20.30", true},
		{"v1.2.3-beta.1", true},
		{"1.2.3-rc.20", true},
		{"feat: add something", false},
		{"fix: 1.2.3", false},
		{"version 1.2.3", false},
		{"v1.2", false},
		{"1.2", false},
		{"not a version", false},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := findVer.MatchString(tt.msg)
			if got != tt.want {
				t.Errorf("findVer.MatchString(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestMessageFiltering(t *testing.T) {
	tests := []struct {
		msg  string
		want bool // true if it should be filtered out
	}{
		{"feat: add something", false},
		{"fix: bug", false},
		{"v1.2.3", true},
		{"1.2.3", true},
		{"chore: bump version [skip ci]", true},
		{"docs: update readme [skip action]", true},
		{"Merge branch 'main'", true},
		{"v0.11.1", true},
		{"just a message", false},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			// This matches the logic in filteredCommits
			isFiltered := false
			if strings.HasPrefix(tt.msg, "Merge ") {
				isFiltered = true
			} else if strings.Contains(tt.msg, "[skip ci]") || strings.Contains(tt.msg, "[skip action]") {
				isFiltered = true
			} else if findVer.MatchString(tt.msg) {
				isFiltered = true
			}

			if isFiltered != tt.want {
				t.Errorf("Filtering for %q = %v, want %v", tt.msg, isFiltered, tt.want)
			}
		})
	}
}

func TestIsPrerelease(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"v1.0.0", false},
		{"1.0.0", false},
		{"v1.2.0-beta.1", true},
		{"1.2.0-beta.1", true},
		{"v1.2.0-rc.2", true},
		{"v2.0.0-alpha", true},
		{"not-a-semver", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := isPrerelease(tt.tag)
			if got != tt.want {
				t.Errorf("isPrerelease(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestInjectChangelog(t *testing.T) {
	t.Run("start pattern only", func(t *testing.T) {
		existing := "=== Plugin Name ===\n== Description ==\nDescription here\n== Changelog ==\n= 0.9.0 =\n* Initial release\n"
		changelog := "= 1.0.0 =\n* Feature: new feature (abc1234)\n\n= 0.9.0 =\n* Initial release"
		start := "== Changelog =="

		got, err := InjectChangelog(existing, changelog, start, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "=== Plugin Name ===\n== Description ==\nDescription here\n== Changelog ==\n\n= 1.0.0 =\n* Feature: new feature (abc1234)\n\n= 0.9.0 =\n* Initial release\n"
		if got != expected {
			t.Errorf("InjectChangelog mismatch.\nGot:\n%q\nExpected:\n%q", got, expected)
		}
	})

	t.Run("start and end pattern", func(t *testing.T) {
		existing := "=== Plugin Name ===\n== Changelog ==\n= 0.9.0 =\n* Initial release\n== Upgrade Notice ==\n= 1.0.0 =\nUpgrade now!\n"
		changelog := "= 1.0.0 =\n* Feature: new feature (abc1234)\n\n= 0.9.0 =\n* Initial release"
		start := "== Changelog =="
		end := "== Upgrade Notice =="

		got, err := InjectChangelog(existing, changelog, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "=== Plugin Name ===\n== Changelog ==\n\n= 1.0.0 =\n* Feature: new feature (abc1234)\n\n= 0.9.0 =\n* Initial release\n\n== Upgrade Notice ==\n= 1.0.0 =\nUpgrade now!\n"
		if got != expected {
			t.Errorf("InjectChangelog mismatch.\nGot:\n%q\nExpected:\n%q", got, expected)
		}
	})

	t.Run("start pattern not found", func(t *testing.T) {
		existing := "=== Plugin Name ===\nNo changelog header\n"
		_, err := InjectChangelog(existing, "changelog", "== Changelog ==", "")
		if err == nil {
			t.Error("expected error when start pattern is missing, got nil")
		}
	})

	t.Run("end pattern not found", func(t *testing.T) {
		existing := "=== Plugin Name ===\n== Changelog ==\nOld stuff\n"
		_, err := InjectChangelog(existing, "changelog", "== Changelog ==", "== Upgrade Notice ==")
		if err == nil {
			t.Error("expected error when end pattern is missing after start pattern, got nil")
		}
	})
}

func TestWordPressFormatter(t *testing.T) {
	wp := &WordPressFormatter{}

	if header := wp.DocumentHeader(); header != "== Changelog ==\n\n" {
		t.Errorf("unexpected document header: %q", header)
	}

	commits := []string{
		"abc1234 feat(core)!: major feature",
		"def5678 fix: resolve bug",
		"ghi9012 custom: something special",
		"jkl3456 misc non-conventional message",
	}

	group := wp.FormatGroup("v1.0.0", "2026-09-21", commits)
	expectedLines := []string{
		"= v1.0.0 (2026-09-21) =",
		"* Feature: core: major feature (BREAKING CHANGE) (abc1234)",
		"* Fix: resolve bug (def5678)",
		"* Custom: something special (ghi9012)",
		"* misc non-conventional message (jkl3456)",
	}

	for _, line := range expectedLines {
		if !strings.Contains(group, line) {
			t.Errorf("expected line %q in group output:\n%s", line, group)
		}
	}
}

func TestGenerateMarkdown_Prerelease(t *testing.T) {
	dir, err := os.MkdirTemp("", "foonver-changelog-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	runGit := func(args ...string) {
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

	// Commit 1 -> v1.0.0
	runGit("commit", "--allow-empty", "-m", "feat: initial release")
	runGit("tag", "v1.0.0")

	// Commit 2 -> v1.1.0-beta.1
	runGit("commit", "--allow-empty", "-m", "feat: beta feature in 1.1.0")
	runGit("tag", "v1.1.0-beta.1")

	// Commit 3 -> v1.1.0
	runGit("commit", "--allow-empty", "-m", "fix: finalize 1.1.0")
	runGit("tag", "v1.1.0")

	// Commit 4 -> v1.2.0-beta.1
	runGit("commit", "--allow-empty", "-m", "feat: beta 1 in 1.2.0")
	runGit("tag", "v1.2.0-beta.1")

	// Commit 5 -> v1.2.0-beta.2
	runGit("commit", "--allow-empty", "-m", "feat: beta 2 in 1.2.0")
	runGit("tag", "v1.2.0-beta.2")

	// Unreleased commit
	runGit("commit", "--allow-empty", "-m", "fix: unreleased fix")

	oldRoot := config.Conf.Info.RootDir
	config.Conf.Info.RootDir = dir
	oldCwd, _ := os.Getwd()
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		os.Chdir(oldCwd)
		config.Conf.IncludePrereleases = false
	}()

	t.Run("in prerelease mode", func(t *testing.T) {
		config.Conf.IncludePrereleases = false
		md, err := GenerateMarkdown("v1.2.0-beta.3", false)
		if err != nil {
			t.Fatalf("GenerateMarkdown failed: %v", err)
		}

		// Active prereleases of 1.2.0 should be present
		if !strings.Contains(md, "v1.2.0-beta.3") {
			t.Errorf("expected v1.2.0-beta.3 in changelog, got:\n%s", md)
		}
		if !strings.Contains(md, "v1.2.0-beta.2") {
			t.Errorf("expected v1.2.0-beta.2 in changelog, got:\n%s", md)
		}
		if !strings.Contains(md, "v1.2.0-beta.1") {
			t.Errorf("expected v1.2.0-beta.1 in changelog, got:\n%s", md)
		}
		if !strings.Contains(md, "v1.1.0") {
			t.Errorf("expected v1.1.0 in changelog, got:\n%s", md)
		}
		// Past finalized prerelease v1.1.0-beta.1 should be omitted
		if strings.Contains(md, "v1.1.0-beta.1") {
			t.Errorf("v1.1.0-beta.1 should NOT be shown separately in changelog, got:\n%s", md)
		}
		// Commit from v1.1.0-beta.1 should still be combined under v1.1.0
		if !strings.Contains(md, "beta feature in 1.1.0") {
			t.Errorf("expected 'beta feature in 1.1.0' combined under v1.1.0, got:\n%s", md)
		}
	})

	t.Run("promoted to stable", func(t *testing.T) {
		config.Conf.IncludePrereleases = false
		md, err := GenerateMarkdown("v1.2.0", false)
		if err != nil {
			t.Fatalf("GenerateMarkdown failed: %v", err)
		}

		if !strings.Contains(md, "v1.2.0") {
			t.Errorf("expected v1.2.0 in changelog, got:\n%s", md)
		}
		// Prereleases of 1.2.0 should now be omitted
		if strings.Contains(md, "v1.2.0-beta.2") {
			t.Errorf("v1.2.0-beta.2 should NOT be shown in changelog, got:\n%s", md)
		}
		if strings.Contains(md, "v1.2.0-beta.1") {
			t.Errorf("v1.2.0-beta.1 should NOT be shown in changelog, got:\n%s", md)
		}
		// Commits from 1.2.0 beta cycle should be combined under v1.2.0
		if !strings.Contains(md, "beta 1 in 1.2.0") {
			t.Errorf("expected 'beta 1 in 1.2.0' combined under v1.2.0, got:\n%s", md)
		}
		if !strings.Contains(md, "beta 2 in 1.2.0") {
			t.Errorf("expected 'beta 2 in 1.2.0' combined under v1.2.0, got:\n%s", md)
		}
	})

	t.Run("with include-prereleases", func(t *testing.T) {
		config.Conf.IncludePrereleases = true
		md, err := GenerateMarkdown("v1.2.0", false)
		if err != nil {
			t.Fatalf("GenerateMarkdown failed: %v", err)
		}

		if !strings.Contains(md, "v1.2.0-beta.2") {
			t.Errorf("expected v1.2.0-beta.2 with include-prereleases, got:\n%s", md)
		}
		if !strings.Contains(md, "v1.2.0-beta.1") {
			t.Errorf("expected v1.2.0-beta.1 with include-prereleases, got:\n%s", md)
		}
		if !strings.Contains(md, "v1.1.0-beta.1") {
			t.Errorf("expected v1.1.0-beta.1 with include-prereleases, got:\n%s", md)
		}
	})

	t.Run("wordpress format generation and writing", func(t *testing.T) {
		config.Conf.ChangelogFormat = "wordpress"
		config.Conf.ChangelogFile = "readme.txt"
		config.Conf.ChangelogStart = "== Changelog =="
		config.Conf.ChangelogEnd = "== Upgrade Notice =="

		readmePath := filepath.Join(dir, "readme.txt")
		initialReadme := "=== My Plugin ===\n== Changelog ==\n= 0.1.0 =\n* Initial\n== Upgrade Notice ==\nUpgrade!\n"
		if err := os.WriteFile(readmePath, []byte(initialReadme), 0644); err != nil {
			t.Fatal(err)
		}

		writtenPath, err := WriteChangelog("v1.2.0")
		if err != nil {
			t.Fatalf("WriteChangelog failed: %v", err)
		}
		if writtenPath != readmePath {
			t.Errorf("expected written path %q, got %q", readmePath, writtenPath)
		}

		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatal(err)
		}
		strContent := string(content)

		if !strings.Contains(strContent, "=== My Plugin ===") {
			t.Errorf("expected header preserved in readme, got:\n%s", strContent)
		}
		if !strings.Contains(strContent, "== Changelog ==") {
			t.Errorf("expected changelog start marker preserved, got:\n%s", strContent)
		}
		if !strings.Contains(strContent, "= v1.2.0") {
			t.Errorf("expected v1.2.0 in wordpress changelog, got:\n%s", strContent)
		}
		if !strings.Contains(strContent, "* Fix: unreleased fix") {
			t.Errorf("expected '* Fix: unreleased fix' in changelog, got:\n%s", strContent)
		}
		if !strings.Contains(strContent, "== Upgrade Notice ==\nUpgrade!\n") {
			t.Errorf("expected upgrade notice preserved, got:\n%s", strContent)
		}
	})
}
