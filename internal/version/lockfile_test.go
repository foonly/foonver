package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanLockFileUpdates(t *testing.T) {
	tests := []struct {
		name         string
		manifestName string
		manifest     string
		lockName     string
		lock         string
		want         string
		wantEntries  int
		wantWarning  bool
	}{
		{
			name:         "Cargo.lock single crate",
			manifestName: "Cargo.toml",
			manifest:     "[package]\nname = \"demo\"\nversion = \"0.3.0\"\n",
			lockName:     "Cargo.lock",
			lock: "version = 4\n\n" +
				"[[package]]\nname = \"demo\"\nversion = \"0.3.0\"\ndependencies = [\n \"other\",\n]\n\n" +
				"[[package]]\nname = \"other\"\nversion = \"0.3.0\"\nsource = \"registry+https://github.com/rust-lang/crates.io-index\"\nchecksum = \"abc\"\n\n" +
				"[[package]]\nname = \"local-path-dep\"\nversion = \"0.3.0\"\n",
			want: "version = 4\n\n" +
				"[[package]]\nname = \"demo\"\nversion = \"0.4.0\"\ndependencies = [\n \"other\",\n]\n\n" +
				"[[package]]\nname = \"other\"\nversion = \"0.3.0\"\nsource = \"registry+https://github.com/rust-lang/crates.io-index\"\nchecksum = \"abc\"\n\n" +
				"[[package]]\nname = \"local-path-dep\"\nversion = \"0.3.0\"\n",
			wantEntries: 1,
		},
		{
			name:         "Cargo.lock workspace members",
			manifestName: "Cargo.toml",
			manifest:     "[workspace]\nmembers = [\"a\", \"b\"]\n\n[workspace.package]\nversion = \"1.0.0\"\n",
			lockName:     "Cargo.lock",
			lock: "[[package]]\nname = \"a\"\nversion = \"1.0.0\"\n\n" +
				"[[package]]\nname = \"b\"\nversion = \"1.0.0\"\n\n" +
				"[[package]]\nname = \"serde\"\nversion = \"1.0.0\"\nsource = \"registry+https://github.com/rust-lang/crates.io-index\"\n",
			want: "[[package]]\nname = \"a\"\nversion = \"1.1.0\"\n\n" +
				"[[package]]\nname = \"b\"\nversion = \"1.1.0\"\n\n" +
				"[[package]]\nname = \"serde\"\nversion = \"1.0.0\"\nsource = \"registry+https://github.com/rust-lang/crates.io-index\"\n",
			wantEntries: 2,
		},
		{
			name:         "package-lock.json v3",
			manifestName: "package.json",
			manifest:     `{"name": "demo", "version": "2.0.0"}`,
			lockName:     "package-lock.json",
			lock: "{\n  \"name\": \"demo\",\n  \"version\": \"2.0.0\",\n  \"lockfileVersion\": 3,\n  \"packages\": {\n" +
				"    \"\": {\n      \"name\": \"demo\",\n      \"version\": \"2.0.0\"\n    },\n" +
				"    \"node_modules/dep\": {\n      \"version\": \"2.0.0\"\n    }\n  }\n}\n",
			want: "{\n  \"name\": \"demo\",\n  \"version\": \"2.1.0\",\n  \"lockfileVersion\": 3,\n  \"packages\": {\n" +
				"    \"\": {\n      \"name\": \"demo\",\n      \"version\": \"2.1.0\"\n    },\n" +
				"    \"node_modules/dep\": {\n      \"version\": \"2.0.0\"\n    }\n  }\n}\n",
			wantEntries: 2,
		},
		{
			name:         "uv.lock with normalized name",
			manifestName: "pyproject.toml",
			manifest:     "[project]\nname = \"My_Project\"\nversion = \"0.1.0\"\n",
			lockName:     "uv.lock",
			lock: "version = 1\n\n" +
				"[[package]]\nname = \"my-project\"\nversion = \"0.1.0\"\nsource = { editable = \".\" }\n\n" +
				"[[package]]\nname = \"requests\"\nversion = \"0.1.0\"\nsource = { registry = \"https://pypi.org/simple\" }\n",
			want: "version = 1\n\n" +
				"[[package]]\nname = \"my-project\"\nversion = \"0.2.0\"\nsource = { editable = \".\" }\n\n" +
				"[[package]]\nname = \"requests\"\nversion = \"0.1.0\"\nsource = { registry = \"https://pypi.org/simple\" }\n",
			wantEntries: 1,
		},
		{
			name:         "stale lock file is left alone with a warning",
			manifestName: "Cargo.toml",
			manifest:     "[package]\nname = \"demo\"\nversion = \"0.3.0\"\n",
			lockName:     "Cargo.lock",
			lock:         "[[package]]\nname = \"demo\"\nversion = \"0.2.0\"\n",
			wantWarning:  true,
		},
	}

	bumps := map[string]string{"0.3.0": "0.4.0", "1.0.0": "1.1.0", "2.0.0": "2.1.0", "0.1.0": "0.2.0"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			manifestPath := filepath.Join(dir, tt.manifestName)
			lockPath := filepath.Join(dir, tt.lockName)
			if err := os.WriteFile(lockPath, []byte(tt.lock), 0644); err != nil {
				t.Fatal(err)
			}

			oldVersion, err := extractVersion(manifestPath, []byte(tt.manifest))
			if err != nil {
				t.Fatal(err)
			}
			updates, warnings := planLockFileUpdates(manifestPath, []byte(tt.manifest), oldVersion, bumps[oldVersion])

			if tt.wantWarning {
				if len(updates) != 0 || len(warnings) != 1 {
					t.Fatalf("expected no updates and one warning, got %d updates, warnings %v", len(updates), warnings)
				}
				return
			}
			if len(warnings) != 0 {
				t.Fatalf("unexpected warnings: %v", warnings)
			}
			if len(updates) != 1 {
				t.Fatalf("expected 1 update, got %d", len(updates))
			}
			if updates[0].Path != lockPath || updates[0].Entries != tt.wantEntries {
				t.Errorf("got path %s with %d entries, want %s with %d", updates[0].Path, updates[0].Entries, lockPath, tt.wantEntries)
			}
			if err := updates[0].apply(); err != nil {
				t.Fatal(err)
			}
			got, _ := os.ReadFile(lockPath)
			if string(got) != tt.want {
				t.Errorf("lock file mismatch:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}

	t.Run("missing lock file is ignored", func(t *testing.T) {
		manifestPath := filepath.Join(t.TempDir(), "Cargo.toml")
		updates, warnings := planLockFileUpdates(manifestPath, []byte("[package]\nname = \"x\"\nversion = \"1.0.0\"\n"), "1.0.0", "1.0.1")
		if len(updates) != 0 || len(warnings) != 0 {
			t.Errorf("expected nothing, got %d updates, warnings %v", len(updates), warnings)
		}
	})
}
