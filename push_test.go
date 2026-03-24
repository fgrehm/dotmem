package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdPush_NotInitialized(t *testing.T) {
	t.Setenv("DOTMEM_DIR", filepath.Join(t.TempDir(), "nonexistent"))
	var buf bytes.Buffer
	err := cmdPush(&buf)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected 'not initialized', got %q", err.Error())
	}
}

func TestCmdPush_NoRemote(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)

	var buf bytes.Buffer
	err := cmdPush(&buf)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no remote configured") {
		t.Errorf("expected 'no remote configured', got %q", err.Error())
	}
}

func TestCmdPush_HappyPath(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)

	// Create a bare repo to push to.
	bareDir := t.TempDir()
	if _, err := gitExec(bareDir, "init", "--bare"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitExec(dotmemDir, "remote", "add", "origin", bareDir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdPush(&buf); err != nil {
		t.Fatalf("push: %v", err)
	}
	if !strings.Contains(buf.String(), "pushed to remote") {
		t.Errorf("expected 'pushed to remote', got %q", buf.String())
	}

	// Verify the bare repo received the commits.
	log, err := gitExec(bareDir, "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "init: create dotmem repo") {
		t.Errorf("expected init commit in bare repo, got %q", log)
	}
}
