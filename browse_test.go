package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantMeta memoryMeta
		wantBody string
	}{
		{
			name:    "all fields",
			content: "---\nname: Integration test safety\ndescription: Tests must not interfere with running workspaces\ntype: feedback\n---\n\nNever run tests without warning.\n",
			wantMeta: memoryMeta{
				Name:        "Integration test safety",
				Description: "Tests must not interfere with running workspaces",
				Type:        "feedback",
			},
			wantBody: "\nNever run tests without warning.\n",
		},
		{
			name:    "missing type field",
			content: "---\nname: Some note\ndescription: A description\n---\n\nBody text.\n",
			wantMeta: memoryMeta{
				Name:        "Some note",
				Description: "A description",
			},
			wantBody: "\nBody text.\n",
		},
		{
			name:     "no frontmatter",
			content:  "# Session Log\n\nJust a plain markdown file.\n",
			wantMeta: memoryMeta{},
			wantBody: "# Session Log\n\nJust a plain markdown file.\n",
		},
		{
			name:     "empty file",
			content:  "",
			wantMeta: memoryMeta{},
			wantBody: "",
		},
		{
			name:     "opening delimiter only",
			content:  "---\nname: Broken\nNo closing delimiter.\n",
			wantMeta: memoryMeta{},
			wantBody: "---\nname: Broken\nNo closing delimiter.\n",
		},
		{
			name:    "extra unknown fields ignored",
			content: "---\nname: Test\ntype: project\ncustom: value\n---\n\nBody.\n",
			wantMeta: memoryMeta{
				Name: "Test",
				Type: "project",
			},
			wantBody: "\nBody.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMeta, gotBody := parseFrontmatter(tt.content)
			if gotMeta != tt.wantMeta {
				t.Errorf("meta = %+v, want %+v", gotMeta, tt.wantMeta)
			}
			if gotBody != tt.wantBody {
				t.Errorf("body = %q, want %q", gotBody, tt.wantBody)
			}
		})
	}
}

func TestCollectMemories(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	// Project A: two files with frontmatter.
	projA := dotmem + "/alpha"
	mustMkdirAll(t, projA)
	mustWriteFile(t, projA+"/.repo", []byte("git@github.com:user/alpha.git"))
	mustWriteFile(t, projA+"/.path", []byte("/home/user/alpha"))
	mustWriteFile(t, projA+"/feedback_tests.md", []byte("---\nname: Test safety\ndescription: Be careful\ntype: feedback\n---\n\nContent.\n"))
	mustWriteFile(t, projA+"/project_release.md", []byte("---\nname: Release plan\ntype: project\n---\n\nRelease stuff.\n"))

	// Project B: one file without frontmatter.
	projB := dotmem + "/beta"
	mustMkdirAll(t, projB)
	mustWriteFile(t, projB+"/.repo", []byte("git@github.com:user/beta.git"))
	mustWriteFile(t, projB+"/notes.md", []byte("# Notes\n\nPlain markdown.\n"))

	memories, err := collectMemories(dotmem)
	if err != nil {
		t.Fatalf("collectMemories: %v", err)
	}

	if got := len(memories); got != 3 {
		t.Fatalf("got %d memories, want 3", got)
	}

	byFile := make(map[string]memoryFile)
	for _, m := range memories {
		byFile[m.Project+"/"+m.File] = m
	}

	// Check typed file.
	fb, ok := byFile["alpha/feedback_tests.md"]
	if !ok {
		t.Fatal("missing alpha/feedback_tests.md")
	}
	if fb.Meta.Type != "feedback" {
		t.Errorf("type = %q, want feedback", fb.Meta.Type)
	}
	if fb.Meta.Name != "Test safety" {
		t.Errorf("name = %q, want Test safety", fb.Meta.Name)
	}

	// Check untyped file.
	notes, ok := byFile["beta/notes.md"]
	if !ok {
		t.Fatal("missing beta/notes.md")
	}
	if notes.Meta.Type != "" {
		t.Errorf("type = %q, want empty", notes.Meta.Type)
	}
}

func TestCmdBrowse(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	projA := dotmem + "/alpha"
	mustMkdirAll(t, projA)
	mustWriteFile(t, projA+"/.repo", []byte("git@github.com:user/alpha.git"))
	mustWriteFile(t, projA+"/feedback_tests.md", []byte("---\nname: Test safety\ntype: feedback\n---\n\nContent.\n"))
	mustWriteFile(t, projA+"/project_release.md", []byte("---\nname: Release plan\ntype: project\n---\n\nRelease stuff.\n"))

	projB := dotmem + "/beta"
	mustMkdirAll(t, projB)
	mustWriteFile(t, projB+"/.repo", []byte("git@github.com:user/beta.git"))
	mustWriteFile(t, projB+"/notes.md", []byte("# Notes\n\nPlain markdown.\n"))

	t.Run("no filters", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdBrowsePlain(&buf, "", ""); err != nil {
			t.Fatalf("cmdBrowse: %v", err)
		}
		out := buf.String()

		// Check group headers present.
		if !strings.Contains(out, "feedback (1)") {
			t.Errorf("missing feedback group header in:\n%s", out)
		}
		if !strings.Contains(out, "project (1)") {
			t.Errorf("missing project group header in:\n%s", out)
		}
		if !strings.Contains(out, "untyped (1)") {
			t.Errorf("missing untyped group header in:\n%s", out)
		}

		// Check entries.
		if !strings.Contains(out, "Test safety") {
			t.Errorf("missing 'Test safety' entry in:\n%s", out)
		}
		if !strings.Contains(out, "notes.md") {
			t.Errorf("missing 'notes.md' (untyped display name) in:\n%s", out)
		}

		// Check group order: feedback before project before untyped.
		fbIdx := strings.Index(out, "feedback")
		projIdx := strings.Index(out, "project")
		untIdx := strings.Index(out, "untyped")
		if fbIdx > projIdx {
			t.Error("feedback should appear before project")
		}
		if projIdx > untIdx {
			t.Error("project should appear before untyped")
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdBrowsePlain(&buf, "feedback", ""); err != nil {
			t.Fatalf("cmdBrowse: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "feedback (1)") {
			t.Errorf("missing feedback group in:\n%s", out)
		}
		if strings.Contains(out, "project") {
			t.Errorf("project group should be filtered out:\n%s", out)
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdBrowsePlain(&buf, "", "beta"); err != nil {
			t.Fatalf("cmdBrowse: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "notes.md") {
			t.Errorf("missing beta entry in:\n%s", out)
		}
		if strings.Contains(out, "alpha") {
			t.Errorf("alpha entries should be filtered out:\n%s", out)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdBrowsePlain(&buf, "user", ""); err != nil {
			t.Fatalf("cmdBrowse: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "no memories found") {
			t.Errorf("expected 'no memories found' in:\n%s", out)
		}
	})
}
