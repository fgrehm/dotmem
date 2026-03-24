package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdInstallHook_NotOnPath(t *testing.T) {
	fakeHome(t)
	t.Setenv("PATH", "")
	var buf bytes.Buffer
	err := cmdInstallHook(&buf)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found on PATH") {
		t.Errorf("expected PATH error, got %q", err.Error())
	}
}

func TestCmdInstallHook_HappyPath(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)
	var buf bytes.Buffer
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "hook installed") {
		t.Errorf("expected 'hook installed', got %q", buf.String())
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	hooks := settings["hooks"].(map[string]any)
	stop := hooks["Stop"].([]any)
	found := false
	for _, entry := range stop {
		e := entry.(map[string]any)
		for _, h := range e["hooks"].([]any) {
			hm := h.(map[string]any)
			if hm["command"] == "dotmem commit" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("dotmem commit hook not found in settings: %s", string(raw))
	}
}

func TestCmdInstallHook_AlreadyInstalled(t *testing.T) {
	fakeHome(t)
	putDotmemOnPath(t)
	var buf bytes.Buffer
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("first install: %v", err)
	}
	buf.Reset()
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if !strings.Contains(buf.String(), "already installed") {
		t.Errorf("expected 'already installed', got %q", buf.String())
	}
}

func TestCmdInstallHook_MergesExistingSettings(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{"theme": "dark"}`), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("install: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("failed to unmarshal settings.json: %v (raw: %s)", err, string(raw))
	}
	if settings["theme"] != "dark" {
		t.Error("existing 'theme' key was lost")
	}
	if _, ok := settings["hooks"]; !ok {
		t.Error("hooks key not added")
	}
}

func TestCmdInstallHook_EmptySettings(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("expected success with empty settings: %v", err)
	}
}

func TestCmdInstallHook_CorruptSettings(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := cmdInstallHook(&buf)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected parse error, got %q", err.Error())
	}
}
