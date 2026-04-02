package main

import (
	"bytes"
	"os"
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

func TestCmdBrowsePlainCurrentProject(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	// Create a real git repo so resolveSlug can use git rev-parse.
	repoDir := makeTempRepo(t, t.TempDir())
	canonicalPath := mainWorktree(repoDir)

	// Project alpha linked to repoDir.
	projA := dotmem + "/alpha"
	mustMkdirAll(t, projA)
	mustWriteFile(t, projA+"/.repo", []byte("git@github.com:user/alpha.git"))
	mustWriteFile(t, projA+"/.path", []byte(canonicalPath+"\n"))
	mustWriteFile(t, projA+"/feedback.md", []byte("---\nname: Alpha feedback\ntype: feedback\n---\n\nAlpha content.\n"))

	// Project beta with no .path (unlinked).
	projB := dotmem + "/beta"
	mustMkdirAll(t, projB)
	mustWriteFile(t, projB+"/.repo", []byte("git@github.com:user/beta.git"))
	mustWriteFile(t, projB+"/notes.md", []byte("---\nname: Beta notes\ntype: user\n---\n\nBeta content.\n"))

	// Chdir into repoDir so resolveSlug matches alpha.
	chdirTo(t, repoDir)

	var buf bytes.Buffer
	if err := cmdBrowsePlain(&buf, "", "", false); err != nil {
		t.Fatalf("cmdBrowsePlain: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Alpha feedback") {
		t.Errorf("expected alpha memory in output:\n%s", out)
	}
	if strings.Contains(out, "Beta notes") {
		t.Errorf("expected beta memory to be filtered out:\n%s", out)
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
		if err := cmdBrowsePlain(&buf, "", "", false); err != nil {
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
		if err := cmdBrowsePlain(&buf, "feedback", "", false); err != nil {
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
		if err := cmdBrowsePlain(&buf, "", "beta", false); err != nil {
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
		if err := cmdBrowsePlain(&buf, "user", "", false); err != nil {
			t.Fatalf("cmdBrowse: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "no memories found") {
			t.Errorf("expected 'no memories found' in:\n%s", out)
		}
	})
}

func TestDeleteMemory(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	proj := dotmem + "/alpha"
	mustMkdirAll(t, proj)
	mustWriteFile(t, proj+"/.repo", []byte("git@github.com:user/alpha.git"))
	mustWriteFile(t, proj+"/feedback.md", []byte("---\nname: Test\ntype: feedback\n---\n\nContent.\n"))
	mustWriteFile(t, proj+"/MEMORY.md", []byte("# Memory\n\n- [Test](feedback.md) -- a test entry\n"))

	mem := memoryFile{Project: "alpha", File: "feedback.md"}
	if err := deleteMemory(dotmem, mem); err != nil {
		t.Fatalf("deleteMemory: %v", err)
	}

	// File should be gone.
	if _, err := os.Stat(proj + "/feedback.md"); !os.IsNotExist(err) {
		t.Error("expected feedback.md to be deleted")
	}

	// MEMORY.md should no longer reference the file.
	data, err := os.ReadFile(proj + "/MEMORY.md")
	if err != nil {
		t.Fatalf("read MEMORY.md: %v", err)
	}
	if strings.Contains(string(data), "feedback.md") {
		t.Errorf("expected MEMORY.md to not reference feedback.md:\n%s", data)
	}
}

func TestDeleteMemoryNoIndex(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	proj := dotmem + "/alpha"
	mustMkdirAll(t, proj)
	mustWriteFile(t, proj+"/.repo", []byte("git@github.com:user/alpha.git"))
	mustWriteFile(t, proj+"/feedback.md", []byte("---\nname: Test\ntype: feedback\n---\n\nContent.\n"))
	// No MEMORY.md.

	mem := memoryFile{Project: "alpha", File: "feedback.md"}
	if err := deleteMemory(dotmem, mem); err != nil {
		t.Fatalf("deleteMemory without MEMORY.md: %v", err)
	}

	if _, err := os.Stat(proj + "/feedback.md"); !os.IsNotExist(err) {
		t.Error("expected feedback.md to be deleted")
	}
}

func TestCommitMemoryChange(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	proj := dotmem + "/alpha"
	mustMkdirAll(t, proj)
	mustWriteFile(t, proj+"/.repo", []byte("git@github.com:user/alpha.git"))

	// Commit initial state so there's a base.
	if _, err := gitExec(dotmem, "add", "alpha"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := gitExec(dotmem, "commit", "-m", "link: add alpha"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	mustWriteFile(t, proj+"/feedback.md", []byte("---\nname: Test\ntype: feedback\n---\n\nContent.\n"))
	mustWriteFile(t, proj+"/MEMORY.md", []byte("# Memory\n\n- [Test](feedback.md) -- entry\n"))

	if err := commitMemoryChange(dotmem, "alpha", "feedback.md", "browse: edit"); err != nil {
		t.Fatalf("commitMemoryChange: %v", err)
	}

	// Verify a commit was made with the right message.
	out, err := gitExec(dotmem, "log", "--oneline", "-1")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(out, "browse: edit: alpha/feedback.md") {
		t.Errorf("unexpected commit message: %s", out)
	}
}

func TestCommitMemoryChangeNoIndex(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	proj := dotmem + "/alpha"
	mustMkdirAll(t, proj)
	mustWriteFile(t, proj+"/.repo", []byte("git@github.com:user/alpha.git"))

	if _, err := gitExec(dotmem, "add", "alpha"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := gitExec(dotmem, "commit", "-m", "link: add alpha"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// No MEMORY.md - commit should still succeed.
	mustWriteFile(t, proj+"/feedback.md", []byte("---\nname: Test\ntype: feedback\n---\n\nContent.\n"))

	if err := commitMemoryChange(dotmem, "alpha", "feedback.md", "browse: edit"); err != nil {
		t.Fatalf("commitMemoryChange without MEMORY.md: %v", err)
	}

	out, err := gitExec(dotmem, "log", "--oneline", "-1")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(out, "browse: edit: alpha/feedback.md") {
		t.Errorf("unexpected commit message: %s", out)
	}
}

func TestCommitMemoryChangeAfterDelete(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	proj := dotmem + "/alpha"
	mustMkdirAll(t, proj)
	mustWriteFile(t, proj+"/.repo", []byte("git@github.com:user/alpha.git"))
	mustWriteFile(t, proj+"/feedback.md", []byte("---\nname: Test\ntype: feedback\n---\n\nContent.\n"))
	mustWriteFile(t, proj+"/MEMORY.md", []byte("# Memory\n\n- [Test](feedback.md) -- entry\n"))

	// Commit initial state.
	if _, err := gitExec(dotmem, "add", "alpha"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := gitExec(dotmem, "commit", "-m", "link: add alpha"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Delete the file then commit (simulates the browse delete flow).
	if err := deleteMemory(dotmem, memoryFile{Project: "alpha", File: "feedback.md"}); err != nil {
		t.Fatalf("deleteMemory: %v", err)
	}
	if err := commitMemoryChange(dotmem, "alpha", "feedback.md", "browse: delete"); err != nil {
		t.Fatalf("commitMemoryChange after delete: %v", err)
	}

	out, err := gitExec(dotmem, "log", "--oneline", "-1")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(out, "browse: delete: alpha/feedback.md") {
		t.Errorf("unexpected commit message: %s", out)
	}
}

func TestCommitMemoryChangeNoop(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	proj := dotmem + "/alpha"
	mustMkdirAll(t, proj)
	mustWriteFile(t, proj+"/.repo", []byte("git@github.com:user/alpha.git"))
	mustWriteFile(t, proj+"/feedback.md", []byte("---\nname: Test\ntype: feedback\n---\n\nContent.\n"))

	if _, err := gitExec(dotmem, "add", "alpha"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := gitExec(dotmem, "commit", "-m", "link: add alpha"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Commit without modifying anything - should be a no-op, not an error.
	if err := commitMemoryChange(dotmem, "alpha", "feedback.md", "browse: edit"); err != nil {
		t.Fatalf("commitMemoryChange no-op should succeed: %v", err)
	}

	// HEAD should still be the original commit.
	out, err := gitExec(dotmem, "log", "--oneline", "-1")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(out, "link: add alpha") {
		t.Errorf("no-op commit should not have created a new commit: %s", out)
	}
}

func TestResolveProjectFilterInvalidSlug(t *testing.T) {
	dotmem := t.TempDir()
	_, err := resolveProjectFilter(dotmem, "../escape", false)
	if err == nil {
		t.Fatal("expected error for invalid slug")
	}
	if !strings.Contains(err.Error(), "invalid project slug") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveProjectFilterNotFound(t *testing.T) {
	setupGitEnv(t)
	dotmem := initDotmem(t)

	_, err := resolveProjectFilter(dotmem, "nonexistent", false)
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCascadeMemoryIndexBulletOnly(t *testing.T) {
	dir := t.TempDir()
	proj := dir + "/alpha"
	mustMkdirAll(t, proj)
	content := "# Memory\n\n- [Test](feedback.md) -- entry\nSome prose mentioning (feedback.md) in text.\n"
	mustWriteFile(t, proj+"/MEMORY.md", []byte(content))

	if err := cascadeMemoryIndex(dir, "alpha", "feedback.md"); err != nil {
		t.Fatalf("cascadeMemoryIndex: %v", err)
	}

	data, err := os.ReadFile(proj + "/MEMORY.md")
	if err != nil {
		t.Fatalf("read MEMORY.md: %v", err)
	}
	out := string(data)

	// Bullet line should be removed.
	if strings.Contains(out, "- [Test]") {
		t.Errorf("expected bullet line to be removed:\n%s", out)
	}
	// Prose line should be preserved.
	if !strings.Contains(out, "Some prose mentioning") {
		t.Errorf("expected prose line to be preserved:\n%s", out)
	}
}

func TestValidTypeFilter(t *testing.T) {
	// Valid types should pass.
	for _, typ := range []string{"", "user", "feedback", "project", "reference", "untyped"} {
		if err := validTypeFilter(typ); err != nil {
			t.Errorf("validTypeFilter(%q) = %v, want nil", typ, err)
		}
	}

	// Invalid type should fail.
	err := validTypeFilter("banana")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "unknown memory type") {
		t.Errorf("unexpected error: %v", err)
	}
}
