package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdUnlink_HappyPath(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdLink(&buf, strings.NewReader(""), "myapp", false); err != nil {
		t.Fatalf("link: %v", err)
	}

	buf.Reset()
	if err := cmdUnlink(&buf); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if !strings.Contains(buf.String(), "unlinked") {
		t.Errorf("expected 'unlinked', got %q", buf.String())
	}

	settingsPath := filepath.Join(repoDir, ".claude", "settings.local.json")
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Error("expected settings.local.json to be removed when empty")
	}
}

func TestCmdUnlink_PreservesOtherSettings(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	claudeDir := filepath.Join(repoDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(`{"autoMemoryDirectory": "/some/path", "theme": "dark"}`), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdUnlink(&buf); err != nil {
		t.Fatalf("unlink: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(claudeDir, "settings.local.json"))
	if err != nil {
		t.Fatalf("settings file missing: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := settings["autoMemoryDirectory"]; ok {
		t.Error("autoMemoryDirectory should have been removed")
	}
	if settings["theme"] != "dark" {
		t.Error("other settings should be preserved")
	}
}

func TestCmdUnlink_NotLinked(t *testing.T) {
	setupGitEnv(t)
	repoDir := makeTempRepo(t, t.TempDir())
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdUnlink(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "not linked") {
		t.Errorf("expected 'not linked', got %q", buf.String())
	}
}

func TestCmdUnlink_NotGitRepo(t *testing.T) {
	chdirTo(t, t.TempDir())
	var buf bytes.Buffer
	err := cmdUnlink(&buf)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository', got %q", err.Error())
	}
}
