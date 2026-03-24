package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdUninstallHook_HappyPath(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)

	var buf bytes.Buffer
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("install: %v", err)
	}

	buf.Reset()
	if err := cmdUninstallHook(&buf); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !strings.Contains(buf.String(), "hook removed") {
		t.Errorf("expected 'hook removed', got %q", buf.String())
	}

	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("failed to unmarshal settings.json: %v (raw: %s)", err, string(raw))
	}
	if _, ok := settings["hooks"]; ok {
		t.Errorf("hooks key should have been removed when empty, got %s", string(raw))
	}
}

func TestCmdUninstallHook_NotInstalled(t *testing.T) {
	fakeHome(t)

	var buf bytes.Buffer
	if err := cmdUninstallHook(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "not installed") {
		t.Errorf("expected 'not installed', got %q", buf.String())
	}
}

func TestCmdUninstallHook_PreservesOtherSettings(t *testing.T) {
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

	buf.Reset()
	if err := cmdUninstallHook(&buf); err != nil {
		t.Fatalf("uninstall: %v", err)
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
		t.Error("other settings should be preserved")
	}
	if _, ok := settings["hooks"]; ok {
		t.Error("hooks key should have been removed")
	}
}

func TestCmdUninstallHook_PreservesOtherHooks(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)

	var buf bytes.Buffer
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Add another Stop hook entry alongside dotmem's.
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readJSONSettings(settingsPath)
	if err != nil {
		t.Fatalf("readJSONSettings: %v", err)
	}
	hooks := settings["hooks"].(map[string]any)
	stopHooks := hooks["Stop"].([]any)
	stopHooks = append(stopHooks, map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{"type": "command", "command": "echo other"},
		},
	})
	hooks["Stop"] = stopHooks
	if err := writeJSONSettings(settingsPath, settings); err != nil {
		t.Fatalf("writeJSONSettings: %v", err)
	}

	buf.Reset()
	if err := cmdUninstallHook(&buf); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	var after map[string]any
	if err := json.Unmarshal(raw, &after); err != nil {
		t.Fatalf("failed to unmarshal settings.json: %v (raw: %s)", err, string(raw))
	}
	afterHooks := after["hooks"].(map[string]any)
	afterStop := afterHooks["Stop"].([]any)
	if len(afterStop) != 1 {
		t.Errorf("expected 1 remaining stop hook, got %d", len(afterStop))
	}
}

func TestCmdUninstallHook_PreservesUnrecognizedHooksShape(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)

	// Write a Stop hook entry with a non-[]any "hooks" field.
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	raw := `{"hooks":{"Stop":[{"matcher":"","hooks":"not-an-array"},{"matcher":"","hooks":[{"type":"command","command":"dotmem commit"}]}]}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdUninstallHook(&buf); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v (raw: %s)", err, string(data))
	}
	hooks := settings["hooks"].(map[string]any)
	stopHooks := hooks["Stop"].([]any)
	if len(stopHooks) != 1 {
		t.Errorf("expected 1 remaining stop entry (the unrecognized one), got %d", len(stopHooks))
	}
	entry := stopHooks[0].(map[string]any)
	if entry["hooks"] != "not-an-array" {
		t.Error("unrecognized hooks shape should be preserved unchanged")
	}
}

func TestCmdUninstallHook_Idempotent(t *testing.T) {
	home := fakeHome(t)
	putDotmemOnPath(t)

	var buf bytes.Buffer
	if err := cmdInstallHook(&buf); err != nil {
		t.Fatalf("install: %v", err)
	}

	buf.Reset()
	if err := cmdUninstallHook(&buf); err != nil {
		t.Fatalf("first uninstall: %v", err)
	}

	buf.Reset()
	if err := cmdUninstallHook(&buf); err != nil {
		t.Fatalf("second uninstall: %v", err)
	}
	if !strings.Contains(buf.String(), "not installed") {
		t.Errorf("expected 'not installed', got %q", buf.String())
	}

	// Verify settings.json still valid.
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("invalid JSON after double uninstall: %v", err)
	}
}
