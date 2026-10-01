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
	run("config", "commit.gpgsign", "false")
	run("config", "tag.gpgsign", "false")
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

func TestCommitAndTag_MismatchedPrefix(t *testing.T) {
	dir := setupTestRepo(t)
	defer os.RemoveAll(dir)

	oldRoot := config.Conf.Info.RootDir
	oldPrefix := config.Conf.Prefix
	oldCwd, _ := os.Getwd()
	config.Conf.Info.RootDir = dir
	config.Conf.Prefix = "rel-"
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		config.Conf.Prefix = oldPrefix
		os.Chdir(oldCwd)
	}()

	// The version already carries its own "v" prefix (e.g. preserved from a
	// version file) that doesn't match the configured tag prefix. The tag
	// must not become the malformed "rel-v1.2.0".
	version := "v1.2.0"
	if err := CommitAndTag([]string{}, version); err != nil {
		t.Fatalf("CommitAndTag failed: %v", err)
	}

	out, err := runGit("tag", "-l")
	if err != nil {
		t.Fatal(err)
	}
	tags := strings.Fields(out)
	if len(tags) != 1 || tags[0] != "v1.2.0" {
		t.Errorf("expected exactly tag v1.2.0, got: %v", tags)
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

func TestGetDirtyFiles(t *testing.T) {
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

	// Track initial files
	file1 := "consts.php"
	file2 := "readme.txt"
	if err := os.WriteFile(filepath.Join(dir, file1), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file2), []byte("readme"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", file1, file2)
	runGit("commit", "-m", "add files")

	// Modify files (unstaged changes -> " M consts.php")
	if err := os.WriteFile(filepath.Join(dir, file1), []byte("<?php // edited"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file2), []byte("readme edited"), 0644); err != nil {
		t.Fatal(err)
	}

	dirty := GetDirtyFiles()
	if len(dirty) != 2 {
		t.Fatalf("expected 2 dirty files, got %d: %v", len(dirty), dirty)
	}

	// Ensure filenames are intact and not missing first character
	expected := map[string]bool{"consts.php": true, "readme.txt": true}
	for _, f := range dirty {
		if !expected[f] {
			t.Errorf("unexpected dirty file %q (expected one of consts.php, readme.txt)", f)
		}
	}
}

func TestGetDirtyFiles_SpecialCharacters(t *testing.T) {
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

	// Git C-quotes filenames with spaces or special characters in the
	// default porcelain output; these names must survive intact.
	file1 := "file with space.txt"
	file2 := `weird"name.txt`
	for _, f := range []string{file1, file2} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("v1"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runGit("add", file1, file2)
	runGit("commit", "-m", "add special files")

	for _, f := range []string{file1, file2} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("v2"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	dirty := GetDirtyFiles()
	expected := map[string]bool{file1: true, file2: true}
	if len(dirty) != len(expected) {
		t.Fatalf("expected %d dirty files, got %d: %v", len(expected), len(dirty), dirty)
	}
	for _, f := range dirty {
		if !expected[f] {
			t.Errorf("unexpected dirty file %q", f)
		}
	}
}

func TestGetDirtyFiles_Rename(t *testing.T) {
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

	oldName := "old-name.txt"
	newName := "new-name.txt"
	// Content needs to be long enough for git's rename detection to kick in.
	content := strings.Repeat("hello world\n", 20)
	if err := os.WriteFile(filepath.Join(dir, oldName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", oldName)
	runGit("commit", "-m", "add file")

	if err := os.Rename(filepath.Join(dir, oldName), filepath.Join(dir, newName)); err != nil {
		t.Fatal(err)
	}
	runGit("add", "-A")

	dirty := GetDirtyFiles()
	if len(dirty) != 1 || dirty[0] != newName {
		t.Errorf("expected only %q, got %v", newName, dirty)
	}
}

func TestFormatTag(t *testing.T) {
	oldPrefix := config.Conf.Prefix
	defer func() { config.Conf.Prefix = oldPrefix }()

	config.Conf.Prefix = "v"
	if got := FormatTag("1.2.3"); got != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %q", got)
	}
	if got := FormatTag("v1.2.3"); got != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %q", got)
	}

	config.Conf.Prefix = "rel-"
	if got := FormatTag("1.2.3"); got != "rel-1.2.3" {
		t.Errorf("expected rel-1.2.3, got %q", got)
	}
	if got := FormatTag("v1.2.3"); got != "v1.2.3" {
		t.Errorf("expected v1.2.3 (preserved), got %q", got)
	}

	config.Conf.Prefix = ""
	if got := FormatTag("1.2.3"); got != "1.2.3" {
		t.Errorf("expected 1.2.3, got %q", got)
	}
}

func TestPush(t *testing.T) {
	remoteDir, err := os.MkdirTemp("", "foonver-git-remote-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(remoteDir)

	// Initialize bare remote repo
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init bare repo: %v\n%s", err, out)
	}

	dir := setupTestRepo(t)
	defer os.RemoveAll(dir)

	oldRoot := config.Conf.Info.RootDir
	oldCwd, _ := os.Getwd()
	oldRemote := config.Conf.Remote
	config.Conf.Info.RootDir = dir
	config.Conf.Remote = "origin"
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		config.Conf.Remote = oldRemote
		os.Chdir(oldCwd)
	}()

	// Add remote and push initial commit
	if _, err := runGit("remote", "add", "origin", remoteDir); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}
	if _, err := runGit("push", "-u", "origin", "HEAD"); err != nil {
		t.Fatalf("failed to push initial commit: %v", err)
	}

	// Create a new commit and tag
	version := "v1.0.0"
	testFile := "version.txt"
	if err := os.WriteFile(filepath.Join(dir, testFile), []byte("1.0.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAndTag([]string{testFile}, version); err != nil {
		t.Fatalf("CommitAndTag failed: %v", err)
	}

	// Also create an unrelated local tag that should NOT be pushed
	if _, err := runGit("tag", "unrelated-tag"); err != nil {
		t.Fatalf("failed to create unrelated tag: %v", err)
	}

	// Push just the version tag and commit
	if err := Push(version); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify remote has the tag
	out, err := exec.Command("git", "-C", remoteDir, "tag", "-l").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote tags: %v", err)
	}
	tags := strings.Fields(string(out))
	if len(tags) != 1 || tags[0] != version {
		t.Errorf("expected remote to have only %s, got: %v", version, tags)
	}

	// Verify remote has the new commit
	out, err = exec.Command("git", "-C", remoteDir, "log", "-1", "--pretty=%s").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get remote head commit: %v", err)
	}
	if strings.TrimSpace(string(out)) != version {
		t.Errorf("expected remote head commit %q, got %q", version, strings.TrimSpace(string(out)))
	}
}

func TestPush_AtomicFailure(t *testing.T) {
	remoteDir, err := os.MkdirTemp("", "foonver-git-remote-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(remoteDir)

	// Initialize bare remote repo
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init bare repo: %v\n%s", err, out)
	}

	dir := setupTestRepo(t)
	defer os.RemoveAll(dir)

	oldRoot := config.Conf.Info.RootDir
	oldCwd, _ := os.Getwd()
	oldRemote := config.Conf.Remote
	config.Conf.Info.RootDir = dir
	config.Conf.Remote = "origin"
	os.Chdir(dir)
	defer func() {
		config.Conf.Info.RootDir = oldRoot
		config.Conf.Remote = oldRemote
		os.Chdir(oldCwd)
	}()

	// Add remote and push initial commit
	if _, err := runGit("remote", "add", "origin", remoteDir); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}
	if _, err := runGit("push", "-u", "origin", "HEAD"); err != nil {
		t.Fatalf("failed to push initial commit: %v", err)
	}

	// Push a tag "v1.0.0" on remote pointing to initial commit
	if _, err := runGit("tag", "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := runGit("push", "origin", "refs/tags/v1.0.0"); err != nil {
		t.Fatal(err)
	}

	// Now make a new commit locally and move the tag
	testFile := "version.txt"
	if err := os.WriteFile(filepath.Join(dir, testFile), []byte("1.0.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := runGit("add", testFile); err != nil {
		t.Fatal(err)
	}
	if _, err := runGit("commit", "-m", "new commit"); err != nil {
		t.Fatal(err)
	}
	if _, err := runGit("tag", "-f", "v1.0.0"); err != nil {
		t.Fatal(err)
	}

	// Atomic push should fail because tag already exists on remote and is conflicting
	err = Push("v1.0.0")
	if err == nil {
		t.Fatal("expected Push to fail due to conflicting tag, but it succeeded")
	}

	// Verify remote branch was NOT updated to the new commit (atomic rollback)
	out, err := exec.Command("git", "-C", remoteDir, "log", "-1", "--pretty=%s").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get remote head commit: %v", err)
	}
	if strings.TrimSpace(string(out)) != "initial commit" {
		t.Errorf("expected remote commit to remain 'initial commit', got %q", strings.TrimSpace(string(out)))
	}
}
