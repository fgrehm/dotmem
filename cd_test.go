package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdCd_NotInitialized(t *testing.T) {
	t.Setenv("DOTMEM_DIR", filepath.Join(t.TempDir(), "nonexistent"))
	err := cmdCd("")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected 'not initialized', got %q", err.Error())
	}
}

func TestCmdCd_ProjectNotFound(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)

	err := cmdCd("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got %q", err.Error())
	}
}

func TestCmdCd_ProjectPathGone(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)

	projectDir := filepath.Join(dotmemDir, "myapp")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, ".path"), []byte("/nonexistent/path\n"), 0644)

	err := cmdCd("myapp")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no longer exists") {
		t.Errorf("expected 'no longer exists', got %q", err.Error())
	}
}

func TestCmdCd_MemoryRepo(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)

	// Use a shell that exits immediately to test the subshell spawns correctly.
	t.Setenv("SHELL", "/bin/true")
	if err := cmdCd(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdCd_SubshellExitCode(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)

	// A shell that exits non-zero should not cause dotmem to report an error.
	t.Setenv("SHELL", "/bin/false")
	if err := cmdCd(""); err != nil {
		t.Fatalf("subshell exit code should be ignored, got %v", err)
	}
}

func TestCmdCd_ProjectDir(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)

	targetDir := t.TempDir()
	projectDir := filepath.Join(dotmemDir, "myapp")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, ".path"), []byte(targetDir+"\n"), 0644)

	t.Setenv("SHELL", "/bin/true")
	if err := cmdCd("myapp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
