package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

func makeTempRepo(t *testing.T, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"commit", "--allow-empty", "-m", "initial"},
		{"remote", "add", "origin", remoteURL},
	} {
		if _, err := gitExec(dir, args...); err != nil {
			t.Fatalf("makeTempRepo: git %v: %v", args, err)
		}
	}
	return dir
}

func initDotmem(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".mem")
	t.Setenv("DOTMEM_DIR", dir)
	var buf bytes.Buffer
	if err := cmdInit(&buf); err != nil {
		t.Fatalf("initDotmem: %v", err)
	}
	return dir
}

func chdirTo(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func putDotmemOnPath(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	fake := filepath.Join(binDir, "dotmem")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func fakeHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestReadJSONSettings_NotFound(t *testing.T) {
	settings, err := readJSONSettings("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("expected empty map, got %v", settings)
	}
}

func TestReadJSONSettings_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(""), 0644)

	settings, err := readJSONSettings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("expected empty map, got %v", settings)
	}
}

func TestReadJSONSettings_CorruptJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("{bad json}"), 0644)

	_, err := readJSONSettings(path)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected parse error, got %q", err.Error())
	}
}

func TestReadJSONSettings_ValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"key": "value"}`), 0644)

	settings, err := readJSONSettings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings["key"] != "value" {
		t.Errorf("expected key=value, got %v", settings)
	}
}

func TestWriteJSONSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	settings := map[string]any{"foo": "bar", "num": float64(42)}
	if err := writeJSONSettings(path, settings); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read back and verify.
	got, err := readJSONSettings(path)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got["foo"] != "bar" || got["num"] != float64(42) {
		t.Errorf("round-trip failed, got %v", got)
	}

	// Verify trailing newline.
	data, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("expected trailing newline")
	}
}
