package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdLs_NotInitialized(t *testing.T) {
	t.Setenv("DOTMEM_DIR", filepath.Join(t.TempDir(), "nonexistent"))
	var buf bytes.Buffer
	err := cmdLs(&buf)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected 'not initialized', got %q", err.Error())
	}
}

func TestCmdLs_NoProjects(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	var buf bytes.Buffer
	if err := cmdLs(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no linked projects") {
		t.Errorf("expected 'no linked projects', got %q", buf.String())
	}
}

func TestCmdLs_WithProjects(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "my-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"MEMORY.md", "debugging.md", ".repo"} {
		if err := os.WriteFile(filepath.Join(projectDir, f), []byte("test\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".path"), []byte("/home/user/my-project\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdLs(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "my-project") {
		t.Errorf("expected 'my-project' in output, got %q", out)
	}
	if !strings.Contains(out, dotmemDir) {
		t.Errorf("expected dotmem dir path in output, got %q", out)
	}
	if !strings.Contains(out, "2 files") {
		t.Errorf("expected '2 files' in output, got %q", out)
	}
	if !strings.Contains(out, "/home/user/my-project") {
		t.Errorf("expected project path in output, got %q", out)
	}
}

func TestCmdLs_SingleFile(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "solo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdLs(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "1 file") {
		t.Errorf("expected '1 file' (singular), got %q", buf.String())
	}
}
