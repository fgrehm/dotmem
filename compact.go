package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newCompactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact [slug]",
		Short: "Compact memory files into a single MEMORY.md",
		Long: `Merge all memory files for a project into a single MEMORY.md using
Claude to extract, deduplicate, and organize. Files that serve as
standalone reference documents (specs, plans, checklists) are kept.
Requires the claude CLI to be on PATH.

If slug is omitted, auto-detects from the current project directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) > 0 {
				slug = args[0]
			}
			yes, _ := cmd.Flags().GetBool("yes")
			model, _ := cmd.Flags().GetString("model")
			effort, _ := cmd.Flags().GetString("effort")
			return cmdCompact(cmd.Context(), cmd.OutOrStdout(), cmd.InOrStdin(), slug, yes, model, effort)
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt")
	cmd.Flags().StringP("model", "m", "", "Claude model to use (default: sonnet)")
	cmd.Flags().StringP("effort", "e", "", "effort level: low, medium, high, max (default: low)")
	return cmd
}

type compactResult struct {
	Memory string       `json:"memory"`
	Keep   []fileAction `json:"keep"`
	Delete []fileAction `json:"delete"`
}

type fileAction struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

func cmdCompact(ctx context.Context, w io.Writer, r io.Reader, slug string, force bool, model string, effort string) error {
	validEfforts := map[string]bool{"": true, "low": true, "medium": true, "high": true, "max": true}
	if !validEfforts[effort] {
		return fmt.Errorf("invalid effort %q (valid: low, medium, high, max)", effort)
	}

	dir, err := dotmemDir()
	if err != nil {
		return err
	}
	if err := requireInit(dir); err != nil {
		return err
	}

	if slug == "" {
		resolved, err := resolveSlug(dir)
		if err != nil {
			return err
		}
		slug = resolved
	}

	projectDir := filepath.Join(dir, slug)
	files, err := readMemoryFiles(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project %q not found in %s", slug, dir)
		}
		return err
	}

	if len(files) == 0 {
		fmt.Fprintf(w, "dotmem: no memory files to compact in %q\n", slug)
		return nil
	}

	if len(files) == 1 {
		fmt.Fprintf(w, "dotmem: only one memory file in %q, nothing to compact\n", slug)
		return nil
	}

	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("\"claude\" not found on PATH. Install Claude Code first.")
	}
	if err := checkClaudeVersion(); err != nil {
		return err
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	totalLines := 0
	var fileList strings.Builder
	for _, name := range names {
		lines := strings.Count(files[name], "\n")
		totalLines += lines
		fmt.Fprintf(&fileList, "- %s (%d lines)\n", name, lines)
	}

	fmt.Fprintf(w, "dotmem: compacting %q (%d files, %d lines total)\n", slug, len(files), totalLines)

	prompt := buildCompactPrompt(slug, projectDir, fileList.String())

	result, err := runClaude(ctx, w, projectDir, prompt, model, effort)
	if err != nil {
		return fmt.Errorf("claude failed: %w", err)
	}

	newLines := strings.Count(result.Memory, "\n")

	fmt.Fprintf(w, "\n")
	if len(result.Delete) > 0 {
		fmt.Fprintf(w, "delete:\n")
		for _, f := range result.Delete {
			fmt.Fprintf(w, "  %s (%s)\n", f.File, f.Reason)
		}
	}
	if len(result.Keep) > 0 {
		fmt.Fprintf(w, "keep:\n")
		for _, f := range result.Keep {
			fmt.Fprintf(w, "  %s (%s)\n", f.File, f.Reason)
		}
	}
	fmt.Fprintf(w, "result: MEMORY.md %d -> %d lines\n", totalLines, newLines)

	if !force {
		if err := confirmPrompt(w, r, "\napply? [y/N] "); err != nil {
			return err
		}
	}

	memoryPath := filepath.Join(projectDir, "MEMORY.md")
	if err := os.WriteFile(memoryPath, []byte(result.Memory), 0644); err != nil {
		return err
	}

	for _, f := range result.Delete {
		target := filepath.Clean(filepath.Join(projectDir, f.File))
		if rel, err := filepath.Rel(projectDir, target); err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if target == memoryPath {
			continue // already overwritten
		}
		os.Remove(target)
	}

	if _, err := gitExec(dir, "add", "-A"); err != nil {
		return err
	}
	if _, err := gitExec(dir, "diff", "--cached", "--quiet"); err != nil {
		if _, err := gitExec(dir, "commit", "-m", fmt.Sprintf("compact: %s", slug)); err != nil {
			return err
		}
	}

	fmt.Fprintf(w, "dotmem: compacted %q\n", slug)
	return nil
}

func readMemoryFiles(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() || isMetaFile(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = string(data)
	}
	return files, nil
}

func buildCompactPrompt(slug, projectDir, fileList string) string {
	return fmt.Sprintf(`You are compacting memory files for the project %q.

The memory files are in: %s

Files:
%s
Read each file, then:
1. Extract atomic facts, decisions, patterns, and gotchas from every file.
2. Deduplicate: if the same information appears in multiple files, keep it once.
3. Organize into a single MEMORY.md with semantic sections (e.g., "## Decisions", "## Patterns", "## Architecture", "## Gotchas"). Use whatever sections make sense for the content.
4. Stay under 200 lines. Be concise. Drop stale or redundant information.
5. For files that serve as standalone reference documents (specs, plans, checklists, manual QA guides), keep them separate and do NOT merge their content into MEMORY.md.

Return ONLY valid JSON (no markdown fences, no explanation) in this exact format:

{"memory": "the full MEMORY.md content as a string", "keep": [{"file": "filename.md", "reason": "why it should survive"}], "delete": [{"file": "filename.md", "reason": "why it was merged or removed"}]}

Every file must appear in either "keep" or "delete". MEMORY.md should appear in "delete" (it gets replaced by the new content).`, slug, projectDir, fileList)
}

type streamEvent struct {
	Type    string         `json:"type"`
	Message *streamMessage `json:"message,omitempty"`
	Result  string         `json:"result,omitempty"`
}

type streamMessage struct {
	Content []streamContent `json:"content"`
}

type streamContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

func runClaude(ctx context.Context, w io.Writer, projectDir string, prompt string, model string, effort string) (*compactResult, error) {
	if model == "" {
		model = "sonnet"
	}
	if effort == "" {
		effort = "low"
	}
	args := []string{
		"--print", "-p", prompt,
		"--output-format", "stream-json", "--verbose",
		"--model", model, "--effort", effort,
		"--allowedTools", "Read",
		"--add-dir", projectDir,
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = projectDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	var resultText strings.Builder
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			var ev streamEvent
			if json.Unmarshal([]byte(line), &ev) == nil {
				switch ev.Type {
				case "assistant":
					if ev.Message != nil {
						for _, c := range ev.Message.Content {
							switch c.Type {
							case "text":
								resultText.WriteString(c.Text)
							case "tool_use":
								if c.Name == "Read" {
									var input struct {
										FilePath string `json:"file_path"`
									}
									if json.Unmarshal(c.Input, &input) == nil {
										fmt.Fprintf(w, "  reading %s\n", filepath.Base(input.FilePath))
									}
								}
							}
						}
					}
				case "result":
					if ev.Result != "" {
						resultText.Reset()
						resultText.WriteString(ev.Result)
					}
				}
			}
		}
		if err != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
				return nil, fmt.Errorf("%s", stderr)
			}
		}
		return nil, fmt.Errorf("claude exited with error: %w", err)
	}

	text := strings.TrimSpace(resultText.String())
	// Strip markdown fences if Claude wrapped them despite instructions.
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result compactResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w\n\nraw output:\n%s", err, text)
	}

	if result.Memory == "" {
		return nil, fmt.Errorf("claude returned empty MEMORY.md content")
	}

	return &result, nil
}

var minClaudeVersion = [3]int{2, 1, 78}

func checkClaudeVersion() error {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return fmt.Errorf("failed to check claude version: %w", err)
	}
	// Output format: "2.1.81 (Claude Code)"
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return fmt.Errorf("unexpected claude version output: %q", string(out))
	}
	versionStr := fields[0]
	parts := strings.SplitN(versionStr, ".", 3)
	if len(parts) < 3 {
		return fmt.Errorf("unexpected claude version format: %q", versionStr)
	}
	var ver [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("unexpected claude version format: %q", versionStr)
		}
		ver[i] = n
	}
	if ver[0] < minClaudeVersion[0] ||
		(ver[0] == minClaudeVersion[0] && ver[1] < minClaudeVersion[1]) ||
		(ver[0] == minClaudeVersion[0] && ver[1] == minClaudeVersion[1] && ver[2] < minClaudeVersion[2]) {
		return fmt.Errorf("claude %s is too old. dotmem compact requires %d.%d.%d or later.",
			versionStr, minClaudeVersion[0], minClaudeVersion[1], minClaudeVersion[2])
	}
	return nil
}
