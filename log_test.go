package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdLog_HappyPath(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("link: %v", err)
	}

	// Add a memory file and commit.
	mustWriteFile(t, filepath.Join(dotmemDir, "myapp", "MEMORY.md"), []byte("# Memory\n"))
	if _, err := gitExec(dotmemDir, "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := gitExec(dotmemDir, "commit", "-m", "commit: auto-save"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	buf.Reset()
	if err := cmdLog(&buf, "myapp"); err != nil {
		t.Fatalf("log: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "link: add myapp") {
		t.Errorf("expected link commit in log, got %q", out)
	}
	if !strings.Contains(out, "auto-save") {
		t.Errorf("expected auto-save commit in log, got %q", out)
	}
}

func TestCmdLog_ProjectNotFound(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)

	var buf bytes.Buffer
	err := cmdLog(&buf, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got %q", err.Error())
	}
}

func TestCmdLog_NotInitialized(t *testing.T) {
	t.Setenv("DOTMEM_DIR", filepath.Join(t.TempDir(), "nonexistent"))
	var buf bytes.Buffer
	err := cmdLog(&buf, "myapp")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected 'not initialized', got %q", err.Error())
	}
}

func TestCmdLog_AutoDetectSlug(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("link: %v", err)
	}

	mustWriteFile(t, filepath.Join(dotmemDir, "myapp", "MEMORY.md"), []byte("# Memory\n"))
	if _, err := gitExec(dotmemDir, "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := gitExec(dotmemDir, "commit", "-m", "commit: auto-save"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	buf.Reset()
	// Empty slug triggers auto-detection from cwd.
	if err := cmdLog(&buf, ""); err != nil {
		t.Fatalf("log auto-detect: %v", err)
	}
	if !strings.Contains(buf.String(), "auto-save") {
		t.Errorf("expected auto-save commit, got %q", buf.String())
	}
}

func TestCmdLog_AutoDetectNotLinked(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	err := cmdLog(&buf, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no linked project") {
		t.Errorf("expected 'no linked project', got %q", err.Error())
	}
}

func TestCmdLog_NoHistory(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)

	// Create project dir manually (no commits touching it).
	projectDir := filepath.Join(dotmemDir, "empty")
	mustMkdirAll(t, projectDir)

	var buf bytes.Buffer
	if err := cmdLog(&buf, "empty"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no history") {
		t.Errorf("expected 'no history', got %q", buf.String())
	}
}
