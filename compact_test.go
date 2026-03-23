package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClaude creates a fake "claude" executable that returns canned JSON output.
func fakeClaude(t *testing.T, result compactResult) {
	t.Helper()
	binDir := t.TempDir()

	// Emit stream-json format: a result event with the compact result as a string.
	inner, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	resultEvent, err := json.Marshal(map[string]string{"type": "result", "result": string(inner)})
	if err != nil {
		t.Fatal(err)
	}

	// Write the JSON to a file and cat it, avoiding shell string interpretation.
	dataFile := filepath.Join(binDir, "response.json")
	if err := os.WriteFile(dataFile, append(resultEvent, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	fake := filepath.Join(binDir, "claude")
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"99.0.0 (Claude Code)\"; exit 0; fi\ncat %s\n", dataFile)
	if err := os.WriteFile(fake, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestCmdCompact_ProjectNotFound(t *testing.T) {
	setupGitEnv(t)
	initDotmem(t)
	fakeClaude(t, compactResult{})

	var buf bytes.Buffer
	err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "nonexistent", false, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got %q", err.Error())
	}
}

func TestCmdCompact_NoFiles(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "myapp")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Only .repo exists.
	if err := os.WriteFile(filepath.Join(projectDir, ".repo"), []byte("url\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "myapp", false, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no memory files") {
		t.Errorf("expected 'no memory files', got %q", buf.String())
	}
}

func TestCmdCompact_SingleFile(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "myapp")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("# Memory\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "myapp", false, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "nothing to compact") {
		t.Errorf("expected 'nothing to compact', got %q", buf.String())
	}
}

func TestCmdCompact_ClaudeNotOnPath(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "myapp")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("# Memory\n"), 0644)
	os.WriteFile(filepath.Join(projectDir, "notes.md"), []byte("# Notes\n"), 0644)
	t.Setenv("PATH", "")

	var buf bytes.Buffer
	err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "myapp", false, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found on PATH") {
		t.Errorf("expected PATH error, got %q", err.Error())
	}
}

func TestCmdCompact_NonTTYAborts(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "myapp")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("# Memory\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "notes.md"), []byte("# Notes\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fakeClaude(t, compactResult{
		Memory: "# Memory\ncompacted\n",
		Delete: []fileAction{
			{File: "MEMORY.md", Reason: "replaced"},
			{File: "notes.md", Reason: "merged"},
		},
	})

	var buf bytes.Buffer
	err := cmdCompact(context.Background(), &buf, &bytes.Buffer{}, "myapp", false, "", "")
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Errorf("expected aborted for non-TTY, got %v", err)
	}
}

func TestCmdCompact_WithForce(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "myapp")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("# Old Memory\nold stuff\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "gotchas.md"), []byte("# Gotchas\nsome gotcha\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "SPEC.md"), []byte("# Spec\nkeep this\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fakeClaude(t, compactResult{
		Memory: "# Memory\n\n## Gotchas\n\n- some gotcha\n",
		Keep:   []fileAction{{File: "SPEC.md", Reason: "standalone reference"}},
		Delete: []fileAction{
			{File: "MEMORY.md", Reason: "replaced"},
			{File: "gotchas.md", Reason: "merged into Gotchas section"},
		},
	})

	var buf bytes.Buffer
	if err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "myapp", true, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MEMORY.md should have new content.
	data, err := os.ReadFile(filepath.Join(projectDir, "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "some gotcha") {
		t.Errorf("expected compacted content, got %q", string(data))
	}

	// gotchas.md should be deleted.
	if _, err := os.Stat(filepath.Join(projectDir, "gotchas.md")); err == nil {
		t.Error("gotchas.md should have been deleted")
	}

	// SPEC.md should survive.
	if _, err := os.Stat(filepath.Join(projectDir, "SPEC.md")); err != nil {
		t.Error("SPEC.md should have survived")
	}

	// Should have committed.
	log, err := gitExec(dotmemDir, "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "compact: myapp") {
		t.Errorf("expected compact commit, got: %s", log)
	}

	out := buf.String()
	if !strings.Contains(out, "compacted") {
		t.Errorf("expected 'compacted' in output, got %q", out)
	}
}

func TestCmdCompact_InvalidEffort(t *testing.T) {
	var buf bytes.Buffer
	err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "myapp", false, "", "turbo")
	if err == nil {
		t.Fatal("expected error for invalid effort")
	}
	if !strings.Contains(err.Error(), "invalid effort") {
		t.Errorf("expected 'invalid effort', got %q", err.Error())
	}
}

func TestCmdCompact_PathTraversal(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "myapp")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("# Memory\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "notes.md"), []byte("# Notes\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file outside projectDir that a traversal attack would target.
	outsideFile := filepath.Join(dotmemDir, "important.txt")
	if err := os.WriteFile(outsideFile, []byte("do not delete\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fakeClaude(t, compactResult{
		Memory: "# Memory\ncompacted\n",
		Delete: []fileAction{
			{File: "MEMORY.md", Reason: "replaced"},
			{File: "notes.md", Reason: "merged"},
			{File: "../important.txt", Reason: "traversal attempt"},
		},
	})

	var buf bytes.Buffer
	if err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "myapp", true, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The file outside projectDir must survive.
	if _, err := os.Stat(outsideFile); err != nil {
		t.Error("path traversal should have been blocked, but file was deleted")
	}

	// notes.md (legitimate delete) should be gone.
	if _, err := os.Stat(filepath.Join(projectDir, "notes.md")); err == nil {
		t.Error("notes.md should have been deleted")
	}
}

func TestCmdCompact_OldClaudeVersion(t *testing.T) {
	setupGitEnv(t)
	dotmemDir := initDotmem(t)
	projectDir := filepath.Join(dotmemDir, "myapp")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("# Memory\n"), 0644)
	os.WriteFile(filepath.Join(projectDir, "notes.md"), []byte("# Notes\n"), 0644)

	// Create a fake claude that reports an old version.
	binDir := t.TempDir()
	fake := filepath.Join(binDir, "claude")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"1.0.0 (Claude Code)\"; exit 0; fi\n"
	os.WriteFile(fake, []byte(script), 0755)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var buf bytes.Buffer
	err := cmdCompact(context.Background(), &buf, strings.NewReader(""), "myapp", true, "", "")
	if err == nil {
		t.Fatal("expected error for old version")
	}
	if !strings.Contains(err.Error(), "too old") {
		t.Errorf("expected 'too old' error, got %q", err.Error())
	}
}

func TestReadMemoryFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("memory\n"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("notes\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".repo"), []byte("url\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".path"), []byte("/some/path\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	files, err := readMemoryFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}
	if _, ok := files[".repo"]; ok {
		t.Error(".repo should be excluded")
	}
	if _, ok := files[".path"]; ok {
		t.Error(".path should be excluded")
	}
	if _, ok := files["MEMORY.md"]; !ok {
		t.Error("MEMORY.md should be included")
	}
}
