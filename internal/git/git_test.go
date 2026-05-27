package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/foonly/foonver/internal/config"
)

func setupTestRepo(t *testing.T) string {
	dir, err := os.MkdirTemp("", "foonver-git-test-*")
	if err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, string(out))
		}
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test User")
	run("commit", "--allow-empty", "-m", "initial commit")

	return dir
}

func TestCommitAndTag_NoChanges(t *testing.T) {
	dir := setupTestRepo(t)
	defer os.RemoveAll(dir)

	// Save and restore config/cwd
	oldRoot := config.Conf.Info.RootDir
	oldCwd, _ := os.Getwd()
	config.Conf.Info.RootDir = dir
	config.Conf.Verbosity = config.Normal
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		os.Chdir(oldCwd)
	}()

	// No changes staged. CommitAndTag should skip commit but still tag.
	version := "1.0.1"
	err := CommitAndTag([]string{}, version)
	if err != nil {
		t.Fatalf("CommitAndTag failed: %v", err)
	}

	// Verify tag exists
	out, err := runGit("tag", "-l", version)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, version) {
		t.Errorf("Expected tag %s to exist, but not found in: %s", version, out)
	}

	// Verify no new commit was created (HEAD should still be "initial commit")
	out, err = runGit("log", "-1", "--pretty=%s")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "initial commit" {
		t.Errorf("Expected no new commit, but found: %s", strings.TrimSpace(out))
	}
}

func TestCommitAndTag_WithChanges(t *testing.T) {
	dir := setupTestRepo(t)
	defer os.RemoveAll(dir)

	oldRoot := config.Conf.Info.RootDir
	oldCwd, _ := os.Getwd()
	config.Conf.Info.RootDir = dir
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		os.Chdir(oldCwd)
	}()

	// Create a change
	testFile := "version.txt"
	err := os.WriteFile(filepath.Join(dir, testFile), []byte("1.1.0"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	version := "1.1.0"
	err = CommitAndTag([]string{testFile}, version)
	if err != nil {
		t.Fatalf("CommitAndTag failed: %v", err)
	}

	// Verify tag exists
	out, err := runGit("tag", "-l", version)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, version) {
		t.Errorf("Expected tag %s to exist", version)
	}

	// Verify commit was created with version as message
	out, err = runGit("log", "-1", "--pretty=%s")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != version {
		t.Errorf("Expected commit message %s, got: %s", version, strings.TrimSpace(out))
	}
}

func TestCommitAndTag_DoublePrefix(t *testing.T) {
	dir := setupTestRepo(t)
	defer os.RemoveAll(dir)

	oldRoot := config.Conf.Info.RootDir
	oldCwd, _ := os.Getwd()
	config.Conf.Info.RootDir = dir
	config.Conf.Prefix = "v"
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		os.Chdir(oldCwd)
	}()

	// version already has the prefix
	version := "v1.2.0"
	err := CommitAndTag([]string{}, version)
	if err != nil {
		t.Fatalf("CommitAndTag failed: %v", err)
	}

	// Verify tag is NOT "vv1.2.0"
	out, err := runGit("tag", "-l", "v1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "v1.2.0" {
		t.Errorf("Expected tag v1.2.0, got: %s", out)
	}

	out, err = runGit("tag", "-l", "vv1.2.0")
	if err == nil && strings.Contains(out, "vv1.2.0") {
		t.Errorf("Tag vv1.2.0 should not exist")
	}
}

func TestExitCode(t *testing.T) {
	// Note: We can't easily test the actual exit code value without a real process,
	// but we can verify the Unwrap logic works.
	if ExitCode(nil) != 0 {
		t.Errorf("ExitCode(nil) should be 0")
	}

	// Real world test with runGit
	_, err2 := runGit("non-existent-command")
	if err2 == nil {
		t.Fatal("Expected error for non-existent command")
	}
	code := ExitCode(err2)
	if code == 0 {
		t.Errorf("Expected non-zero exit code for failed command, got 0")
	}
}
