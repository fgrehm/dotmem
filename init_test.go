package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdInit_HappyPath(t *testing.T) {
	setupGitEnv(t)
	dir := filepath.Join(t.TempDir(), ".dotmem")
	t.Setenv("DOTMEM_DIR", dir)
	var buf bytes.Buffer
	if err := cmdInit(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "initialized at") {
		t.Errorf("expected 'initialized at' in output, got %q", out)
	}
	if !strings.Contains(out, "install-hook") {
		t.Errorf("expected 'install-hook' hint in output, got %q", out)
	}
	for _, f := range []string{".gitignore", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing file: %s", f)
		}
	}
	log, err := gitExec(dir, "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "init: create dotmem repo") {
		t.Errorf("expected init commit, got: %s", log)
	}
}

func TestCmdInit_AlreadyExists(t *testing.T) {
	setupGitEnv(t)
	dir := filepath.Join(t.TempDir(), ".dotmem")
	t.Setenv("DOTMEM_DIR", dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err := cmdInit(&buf)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("expected 'already initialized', got %q", err.Error())
	}
}

func TestCmdInit_RelativeDotmemDir(t *testing.T) {
	t.Setenv("DOTMEM_DIR", "not/absolute")
	var buf bytes.Buffer
	err := cmdInit(&buf)
	if err == nil {
		t.Fatal("expected error for relative DOTMEM_DIR")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("expected 'absolute' in error, got %q", err.Error())
	}
}
