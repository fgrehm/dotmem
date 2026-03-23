package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsTerminal_NonFile(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("expected false for non-file reader")
	}
}

func TestIsTerminal_RegularFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("expected false for regular file (not a chardev)")
	}
}

func TestIsTerminal_ClosedFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if isTerminal(f) {
		t.Error("expected false for closed file")
	}
}

func TestCmdLink_HappyPath(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "my-app", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `linked "my-app"`) {
		t.Errorf("expected linked message, got %q", out)
	}
	if !strings.Contains(out, "gitignored") {
		t.Errorf("expected gitignore hint, got %q", out)
	}

	actualRemote, err := gitExec(repoDir, "remote", "get-url", "origin")
	if err != nil {
		t.Fatal(err)
	}
	repoFile := filepath.Join(dotmemDir, "my-app", ".repo")
	data, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("missing .repo file: %v", err)
	}
	if strings.TrimSpace(string(data)) != actualRemote {
		t.Errorf(".repo content: got %q, want %q", strings.TrimSpace(string(data)), actualRemote)
	}

	pathFile := filepath.Join(dotmemDir, "my-app", ".path")
	pathData, err := os.ReadFile(pathFile)
	if err != nil {
		t.Fatalf("missing .path file: %v", err)
	}
	if strings.TrimSpace(string(pathData)) != repoDir {
		t.Errorf(".path content: got %q, want %q", strings.TrimSpace(string(pathData)), repoDir)
	}

	log, err := gitExec(dotmemDir, "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "link: add my-app") {
		t.Errorf("expected link commit, got: %s", log)
	}

	settingsPath := filepath.Join(repoDir, ".claude", "settings.local.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("missing settings.local.json: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	want := filepath.Join(dotmemDir, "my-app")
	if settings["autoMemoryDirectory"] != want {
		t.Errorf("autoMemoryDirectory: got %q, want %q", settings["autoMemoryDirectory"], want)
	}
}

func TestCmdLink_RelativeDotmemDir(t *testing.T) {
	setupGitEnv(t)
	t.Setenv("DOTMEM_DIR", "relative/path")
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)
	var buf bytes.Buffer
	err := cmdLink(&buf, strings.NewReader(""), "", false)
	if err == nil {
		t.Fatal("expected error for relative DOTMEM_DIR")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("expected 'absolute' in error, got %q", err.Error())
	}
}

func TestCmdLink_NotInitialized(t *testing.T) {
	setupGitEnv(t)
	t.Setenv("DOTMEM_DIR", filepath.Join(t.TempDir(), "nonexistent"))
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)
	var buf bytes.Buffer
	err := cmdLink(&buf, strings.NewReader(""), "", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected 'not initialized', got %q", err.Error())
	}
}

func TestCmdLink_NotGitRepo(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	chdirTo(t, t.TempDir())
	var buf bytes.Buffer
	err := cmdLink(&buf, strings.NewReader(""), "", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository', got %q", err.Error())
	}
}

func TestCmdLink_NoRemote(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	dir := t.TempDir()
	if _, err := gitExec(dir, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitExec(dir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	chdirTo(t, dir)
	var buf bytes.Buffer
	err := cmdLink(&buf, strings.NewReader(""), "", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no remote origin") {
		t.Errorf("expected 'no remote origin', got %q", err.Error())
	}
}

func TestCmdLink_SlugCollision(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)

	remote1 := t.TempDir()
	remote2 := t.TempDir()

	repo1 := makeTempRepo(t, remote1)
	chdirTo(t, repo1)
	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "api", false); err != nil {
		t.Fatalf("first link failed: %v", err)
	}

	repo2 := makeTempRepo(t, remote2)
	chdirTo(t, repo2)
	buf.Reset()
	err := cmdLink(&buf, strings.NewReader(""), "api", false)
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "already linked to") {
		t.Errorf("expected collision message, got %q", err.Error())
	}
}

func TestCmdLink_SameRemoteWorktree(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	remote := t.TempDir()

	repo1 := makeTempRepo(t, remote)
	chdirTo(t, repo1)
	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("first link: %v", err)
	}
	log1, _ := gitExec(dotmemDir, "log", "--oneline")
	n1 := len(strings.Split(strings.TrimSpace(log1), "\n"))

	repo2 := makeTempRepo(t, remote)
	chdirTo(t, repo2)
	buf.Reset()
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("second link (worktree): %v", err)
	}
	log2, _ := gitExec(dotmemDir, "log", "--oneline")
	n2 := len(strings.Split(strings.TrimSpace(log2), "\n"))
	if n2 != n1 {
		t.Errorf("expected no new commit for worktree link, commit count %d -> %d", n1, n2)
	}
}

func TestCmdLink_AlreadyLinked(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("first link: %v", err)
	}
	buf.Reset()
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("second link: %v", err)
	}
	if !strings.Contains(buf.String(), "already linked") {
		t.Errorf("expected 'already linked', got %q", buf.String())
	}
}

func TestCmdLink_MergesExistingSettings(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	claudeDir := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"someOtherKey": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("link: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(claudeDir, "settings.local.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if settings["someOtherKey"] != "value" {
		t.Error("existing key was lost")
	}
	want := filepath.Join(dotmemDir, "myapp")
	if settings["autoMemoryDirectory"] != want {
		t.Errorf("autoMemoryDirectory: got %v, want %q", settings["autoMemoryDirectory"], want)
	}
}

func TestCmdLink_EmptySettingsFile(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	claudeDir := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("expected success with empty settings file: %v", err)
	}
}

func TestCmdLink_CorruptSettings(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	claudeDir := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := cmdLink(&buf, strings.NewReader(""), "myapp", false)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected parse error, got %q", err.Error())
	}
}

func TestCmdLink_OverwritePrompt_NonTTY(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	claudeDir := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"autoMemoryDirectory": "/some/other/path"}`), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := cmdLink(&buf, &bytes.Buffer{}, "myapp", false)
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Errorf("expected aborted error for non-TTY stdin, got %v", err)
	}
}

func TestCmdLink_OverwriteForce(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	claudeDir := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"autoMemoryDirectory": "/some/other/path"}`), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", true); err != nil {
		t.Fatalf("expected success with -y, got %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(claudeDir, "settings.local.json"))
	if err != nil {
		t.Fatalf("failed to read settings.local.json: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("failed to unmarshal settings.local.json: %v (raw: %s)", err, string(raw))
	}
	want := filepath.Join(dotmemDir, "myapp")
	if settings["autoMemoryDirectory"] != want {
		t.Errorf("autoMemoryDirectory not updated: got %v, want %q", settings["autoMemoryDirectory"], want)
	}
}

func TestCmdLink_DerivedSlug(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)

	parent := t.TempDir()
	repoDir := filepath.Join(parent, "my-project")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init"},
		{"commit", "--allow-empty", "-m", "initial"},
		{"remote", "add", "origin", t.TempDir()},
	} {
		if _, err := gitExec(repoDir, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "", false); err != nil {
		t.Fatalf("link: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dotmemDir, "my-project")); err != nil {
		t.Error("expected my-project dir derived from dirname")
	}
}

func TestCmdLink_EnsuresPathGitignore(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	// Simulate a legacy dotmem repo without the **/.path rule.
	gitignorePath := filepath.Join(dotmemDir, ".gitignore")
	os.WriteFile(gitignorePath, []byte(".DS_Store\n"), 0644)

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("link: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "**/.path") {
		t.Error("expected **/.path to be added to .gitignore for backwards compat")
	}
}

func TestCmdLink_SlugNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"MyApp", "myapp"},
		{"my_app", "my-app"},
		{"my app", "my-app"},
		{"my-app.git", "my-app"},
		{"My_App.git", "my-app"},
	}
	for _, tt := range tests {
		got := normalizeSlug(tt.input)
		if got != tt.want {
			t.Errorf("normalizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
