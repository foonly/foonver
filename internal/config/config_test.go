package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDiscovery_DotfileAndFallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "foonver-config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	oldRootDir := Conf.Info.RootDir
	Conf.Info.RootDir = tempDir
	defer func() {
		Conf.Info.RootDir = oldRootDir
	}()

	t.Run("loads .foonver.toml", func(t *testing.T) {
		dotfilePath := filepath.Join(tempDir, ".foonver.toml")
		err := os.WriteFile(dotfilePath, []byte(`prefix = "dot-prefix"`), 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(dotfilePath)

		Init()
		if Conf.Prefix != "dot-prefix" {
			t.Errorf("expected prefix 'dot-prefix', got %q", Conf.Prefix)
		}
	})

	t.Run("falls back to foonver.toml", func(t *testing.T) {
		legacyPath := filepath.Join(tempDir, "foonver.toml")
		err := os.WriteFile(legacyPath, []byte(`prefix = "legacy-prefix"`), 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(legacyPath)

		Init()
		if Conf.Prefix != "legacy-prefix" {
			t.Errorf("expected prefix 'legacy-prefix', got %q", Conf.Prefix)
		}
	})

	t.Run("prefers .foonver.toml over foonver.toml", func(t *testing.T) {
		dotfilePath := filepath.Join(tempDir, ".foonver.toml")
		legacyPath := filepath.Join(tempDir, "foonver.toml")

		if err := os.WriteFile(dotfilePath, []byte(`prefix = "preferred-dot"`), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(dotfilePath)

		if err := os.WriteFile(legacyPath, []byte(`prefix = "ignored-legacy"`), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(legacyPath)

		Init()
		if Conf.Prefix != "preferred-dot" {
			t.Errorf("expected prefix 'preferred-dot', got %q", Conf.Prefix)
		}
	})
}
