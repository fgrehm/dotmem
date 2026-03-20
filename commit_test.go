package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdCommit_NoInit(t *testing.T) {
	setupGitEnv(t)
	t.Setenv("DOTMEM_DIR", filepath.Join(t.TempDir(), "nonexistent"))
	var buf bytes.Buffer
	if err := cmdCommit(&buf); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestCmdCommit_NoChanges(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	var buf bytes.Buffer
	if err := cmdCommit(&buf); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestCmdCommit_WithChanges(t *testing.T) {
	setupGitEnv(t)
	dir := initDotmem(t)
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := cmdCommit(&buf); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	log, err := gitExec(dir, "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "auto:") {
		t.Errorf("expected auto: commit, got log:\n%s", log)
	}
}

func TestCmdCommit_Debug(t *testing.T) {
	setupGitEnv(t)
	t.Setenv("DOTMEM_DEBUG", "1")
	t.Setenv("DOTMEM_DIR", filepath.Join(t.TempDir(), "nonexistent"))
	var buf bytes.Buffer
	if err := cmdCommit(&buf); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !strings.Contains(buf.String(), "debug:") {
		t.Errorf("expected debug output, got %q", buf.String())
	}
}

func TestCmdCommit_NoChanges_Debug(t *testing.T) {
	setupGitEnv(t)
	t.Setenv("DOTMEM_DEBUG", "1")
	initDotmem(t)
	var buf bytes.Buffer
	if err := cmdCommit(&buf); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !strings.Contains(buf.String(), "no changes") {
		t.Errorf("expected 'no changes' in debug output, got %q", buf.String())
	}
}

func TestCmdCommit_WithChanges_Debug(t *testing.T) {
	setupGitEnv(t)
	t.Setenv("DOTMEM_DEBUG", "1")
	dir := initDotmem(t)
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := cmdCommit(&buf); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "git add") {
		t.Errorf("expected 'git add' in debug output, got %q", out)
	}
	if !strings.Contains(out, "committed") {
		t.Errorf("expected 'committed' in debug output, got %q", out)
	}
}

func TestCmdCommit_RelativeDotmemDir(t *testing.T) {
	setupGitEnv(t)
	t.Setenv("DOTMEM_DEBUG", "1")
	t.Setenv("DOTMEM_DIR", "relative/path")
	var buf bytes.Buffer
	if err := cmdCommit(&buf); err != nil {
		t.Fatalf("expected nil (commit swallows errors), got %v", err)
	}
}
