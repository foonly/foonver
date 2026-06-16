package version

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/foonly/foonver/internal/config"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		want     string
		wantErr  bool
	}{
		{
			name:     "package.json valid",
			filename: "package.json",
			content:  `{"name": "test", "version": "1.2.3"}`,
			want:     "1.2.3",
			wantErr:  false,
		},
		{
			name:     "version.json valid",
			filename: "version.json",
			content:  `{"version": "v2.0.0-rc.1"}`,
			want:     "v2.0.0-rc.1",
			wantErr:  false,
		},
		{
			name:     "version.toml valid double quotes",
			filename: "version.toml",
			content:  `version = "1.0.0"`,
			want:     "1.0.0",
			wantErr:  false,
		},
		{
			name:     "version.toml valid single quotes",
			filename: "version.toml",
			content:  `version = '1.0.0'`,
			want:     "1.0.0",
			wantErr:  false,
		},
		{
			name:     "version.txt valid",
			filename: "version.txt",
			content:  "  1.2.3-beta.1  \n",
			want:     "1.2.3-beta.1",
			wantErr:  false,
		},
		{
			name:     "version.md valid header",
			filename: "version.md",
			content:  "# Project Name\n\nVersion 1.2.3\n",
			want:     "1.2.3",
			wantErr:  false,
		},
		{
			name:     "version.md valid badge",
			filename: "version.md",
			content:  "![version](https://img.shields.io/badge/version-v2.0.0-blue)",
			want:     "2.0.0-blue",
			wantErr:  false,
		},
		{
			name:     "version.yaml valid",
			filename: "version.yaml",
			content:  "version: 1.2.3\n",
			want:     "1.2.3",
			wantErr:  false,
		},
		{
			name:     "version.yml valid",
			filename: "version.yml",
			content:  "version: '2.0.0'\n",
			want:     "2.0.0",
			wantErr:  false,
		},
		{
			name:     "composer.json valid",
			filename: "composer.json",
			content:  `{"name": "test/project", "version": "1.0.0-alpha"}`,
			want:     "1.0.0-alpha",
			wantErr:  false,
		},
		{
			name:     "unsupported file",
			filename: "version.xml",
			content:  "<version>1.0.0</version>",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractVersion(tt.filename, []byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("extractVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineNextVersionTarget(t *testing.T) {
	current, _ := semver.NewVersion("1.2.3")

	tests := []struct {
		name    string
		cmd     string
		arg     string
		want    string
		wantErr bool
	}{
		{
			name:    "major",
			cmd:     "major",
			arg:     "",
			want:    "2.0.0",
			wantErr: false,
		},
		{
			name:    "minor",
			cmd:     "minor",
			arg:     "",
			want:    "1.3.0",
			wantErr: false,
		},
		{
			name:    "patch",
			cmd:     "patch",
			arg:     "",
			want:    "1.2.4",
			wantErr: false,
		},
		{
			name:    "specific version",
			cmd:     "ver",
			arg:     "2.0.0-rc.1",
			want:    "2.0.0-rc.1",
			wantErr: false,
		},
		{
			name:    "invalid specific version",
			cmd:     "ver",
			arg:     "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, _, err := determineNextVersion(current, tt.cmd, tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("determineNextVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("determineNextVersion() got = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestParseCommit(t *testing.T) {
	originalParser := config.Conf.Parser
	t.Cleanup(func() {
		config.Conf.Parser = originalParser
	})

	tests := []struct {
		name   string
		parser string
		msg    string
		major  bool
		minor  bool
		patch  bool
	}{
		{
			name:   "angular parser detects breaking change with bang",
			parser: "angular",
			msg:    "feat(api)!: change response format",
			major:  true,
			minor:  false,
			patch:  false,
		},
		{
			name:   "angular parser detects feat as minor",
			parser: "angular",
			msg:    "feat(ui): add dark mode",
			major:  false,
			minor:  true,
			patch:  false,
		},
		{
			name:   "generic parser treats unknown message as patch",
			parser: "generic",
			msg:    "chore: update dependencies",
			major:  false,
			minor:  false,
			patch:  true,
		},
		{
			name:   "angular parser treats unknown message as none",
			parser: "angular",
			msg:    "chore: update dependencies",
			major:  false,
			minor:  false,
			patch:  false,
		},
		{
			name:   "all parsers gives major precedence over generic patch",
			parser: "all",
			msg:    "feat(core)!: rewrite parser",
			major:  true,
			minor:  true,
			patch:  false,
		},
		{
			name:   "all parsers yields minor when angular minor and generic minor",
			parser: "all",
			msg:    "feat: add CLI command",
			major:  false,
			minor:  true,
			patch:  false,
		},
		{
			name:   "all parsers can produce minor and patch together",
			parser: "all",
			msg:    "fix: correct panic on nil input",
			major:  false,
			minor:  false,
			patch:  true,
		},
		{
			name:   "angular detect breaking in footer",
			parser: "angular",
			msg:    "feat: add config parser\nBREAKING CHANGE: config file format changed",
			major:  true,
			minor:  false,
			patch:  false,
		},
		{
			name:   "generic user breaking",
			parser: "generic",
			msg:    "breaking: change in project config",
			major:  true,
			minor:  false,
			patch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Conf.Parser = tt.parser

			major, minor, patch := parseCommit(tt.msg)

			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("parseCommit(%q) = (major=%v, minor=%v, patch=%v), want (major=%v, minor=%v, patch=%v)",
					tt.msg, major, minor, patch, tt.major, tt.minor, tt.patch)
			}
		})
	}
}

func TestSyncVersion(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		content    string
		oldVersion string
		newVersion string
		want       string
		wantErr    bool
	}{
		{
			name:       "readme sync with 'version:' prefix",
			filename:   "README.md",
			content:    "# My Project\n\nThis is version: 1.0.0\n",
			oldVersion: "1.0.0",
			newVersion: "1.1.0",
			want:       "# My Project\n\nThis is version: 1.1.0\n",
			wantErr:    false,
		},
		{
			name:       "docs sync with 'v' prefix",
			filename:   "docs.md",
			content:    "Current version is v1.2.3 and should be updated.",
			oldVersion: "1.2.3",
			newVersion: "2.0.0",
			want:       "Current version is v2.0.0 and should be updated.",
			wantErr:    false,
		},
		{
			name:       "case insensitive 'VERSION' prefix",
			filename:   "INSTALL.txt",
			content:    "VERSION 0.8.0-rc1",
			oldVersion: "0.8.0-rc1",
			newVersion: "0.9.0",
			want:       "VERSION 0.9.0",
			wantErr:    false,
		},
		{
			name:       "punctuation handling",
			filename:   "README.md",
			content:    "### Version: 1.0.0-beta.1",
			oldVersion: "1.0.0-beta.1",
			newVersion: "1.0.0",
			want:       "### Version: 1.0.0",
			wantErr:    false,
		},
		{
			name:       "missing version fails",
			filename:   "missing.md",
			content:    "No version string here.",
			oldVersion: "1.0.0",
			newVersion: "1.1.0",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file
			f, err := os.CreateTemp("", "sync_test_*.md")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())

			if _, err := f.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			f.Close()

			// Set root dir to temp dir so syncVersion can find it
			oldRoot := config.Conf.Info.RootDir
			config.Conf.Info.RootDir = os.TempDir()
			defer func() { config.Conf.Info.RootDir = oldRoot }()

			// syncVersion expects a relative path to RootDir
			relPath := f.Name()[len(os.TempDir()):]
			if relPath[0] == os.PathSeparator {
				relPath = relPath[1:]
			}

			err = syncVersion(relPath, tt.oldVersion, tt.newVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("syncVersion() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got, err := os.ReadFile(f.Name())
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != tt.want {
					t.Errorf("syncVersion() got = %q, want %q", string(got), tt.want)
				}
			}
		})
	}
}

func TestUpdateVersionFile(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		content    string
		oldVersion string
		newVersion string
		want       string
		wantErr    bool
	}{
		{
			name:       "json update",
			filename:   "package.json",
			content:    `{"version": "1.2.3"}`,
			oldVersion: "1.2.3",
			newVersion: "1.2.4",
			want:       `{"version": "1.2.4"}`,
			wantErr:    false,
		},
		{
			name:       "toml update",
			filename:   "version.toml",
			content:    `version = "1.2.3"`,
			oldVersion: "1.2.3",
			newVersion: "2.0.0",
			want:       `version = "2.0.0"`,
			wantErr:    false,
		},
		{
			name:       "txt update",
			filename:   "version.txt",
			content:    "v1.2.3\n",
			oldVersion: "v1.2.3",
			newVersion: "v1.3.0",
			want:       "v1.3.0\n",
			wantErr:    false,
		},
		{
			name:       "yaml update double quotes",
			filename:   "version.yaml",
			content:    "version: \"1.2.3\"\n",
			oldVersion: "1.2.3",
			newVersion: "1.2.4",
			want:       "version: \"1.2.4\"\n",
			wantErr:    false,
		},
		{
			name:       "yaml update single quotes",
			filename:   "version.yaml",
			content:    "version: '1.2.3'\n",
			oldVersion: "1.2.3",
			newVersion: "1.2.4",
			want:       "version: '1.2.4'\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file to safely test file writing
			f, err := os.CreateTemp("", "*_"+tt.filename)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())

			err = updateVersionFile(f.Name(), tt.oldVersion, tt.newVersion, []byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("updateVersionFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got, err := os.ReadFile(f.Name())
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != tt.want {
					t.Errorf("updateVersionFile() got = %v, want %v", string(got), tt.want)
				}
			}
		})
	}
}

func TestDiscoverVersion_Fallback(t *testing.T) {
	// Mock git repository
	dir, err := os.MkdirTemp("", "foonver-discover-test-*")
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
	runGit("commit", "--allow-empty", "-m", "initial")
	runGit("tag", "v1.2.3")

	// Set root dir to temp dir
	oldRoot := config.Conf.Info.RootDir
	config.Conf.Info.RootDir = dir
	oldCwd, _ := os.Getwd()
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		os.Chdir(oldCwd)
	}()

	t.Run("fallback to tag", func(t *testing.T) {
		path, v, content, err := discoverVersion()
		if err != nil {
			t.Fatalf("discoverVersion() failed: %v", err)
		}
		if path != "" {
			t.Errorf("expected empty path, got %s", path)
		}
		if v.String() != "1.2.3" {
			t.Errorf("expected version 1.2.3, got %s", v.String())
		}
		if v.Original() != "v1.2.3" {
			t.Errorf("expected original version v1.2.3, got %s", v.Original())
		}
		if content != nil {
			t.Errorf("expected nil content")
		}
	})

	t.Run("prefers file over tag", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(dir, "version.json"), []byte(`{"version": "2.0.0"}`), 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(filepath.Join(dir, "version.json"))

		path, v, _, err := discoverVersion()
		if err != nil {
			t.Fatalf("discoverVersion() failed: %v", err)
		}
		if !strings.HasSuffix(path, "version.json") {
			t.Errorf("expected version.json, got %s", path)
		}
		if v.String() != "2.0.0" {
			t.Errorf("expected version 2.0.0, got %s", v.String())
		}
	})

	t.Run("uses specified version file", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(dir, "custom.json"), []byte(`{"version": "3.0.0"}`), 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(filepath.Join(dir, "custom.json"))

		// Precedence check: also create a standard version file
		err = os.WriteFile(filepath.Join(dir, "version.json"), []byte(`{"version": "2.0.0"}`), 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(filepath.Join(dir, "version.json"))

		config.Conf.VersionFile = "custom.json"
		defer func() { config.Conf.VersionFile = "" }()

		path, v, _, err := discoverVersion()
		if err != nil {
			t.Fatalf("discoverVersion() failed: %v", err)
		}
		if !strings.HasSuffix(path, "custom.json") {
			t.Errorf("expected custom.json, got %s", path)
		}
		if v.String() != "3.0.0" {
			t.Errorf("expected version 3.0.0, got %s", v.String())
		}
	})

	t.Run("defaults to 0.0.0 when nothing found", func(t *testing.T) {
		// Create a fresh empty repo
		emptyDir, err := os.MkdirTemp("", "foonver-empty-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(emptyDir)

		runGit := func(args ...string) {
			cmd := exec.Command("git", args...)
			cmd.Dir = emptyDir
			cmd.Run()
		}
		runGit("init")

		oldRoot := config.Conf.Info.RootDir
		config.Conf.Info.RootDir = emptyDir
		oldCwd, _ := os.Getwd()
		os.Chdir(emptyDir)
		defer func() {
			config.Conf.Info.RootDir = oldRoot
			os.Chdir(oldCwd)
		}()

		path, v, content, err := discoverVersion()
		if err != nil {
			t.Fatalf("discoverVersion() failed: %v", err)
		}
		if path != "" {
			t.Errorf("expected empty path")
		}
		if v.String() != "0.0.0" {
			t.Errorf("expected 0.0.0, got %s", v.String())
		}
		if content != nil {
			t.Errorf("expected nil content")
		}
	})
}
