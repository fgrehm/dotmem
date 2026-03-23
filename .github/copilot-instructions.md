# Copilot Instructions

## Project context

dotmem is a Go CLI tool that centralizes Claude Code memory files into `~/.mem/` (a git repo). Uses cobra for CLI framework. See CLAUDE.md for full details.

## Review guidelines

When reviewing PRs, check for:

### Code style

- One file per command (`<command>.go`), thin cobra wrapper delegating to `cmdX()` function.
- Tests call `cmdX()` directly, not through cobra.
- No external dependencies beyond cobra. Use `encoding/json`, `os/exec`, `path/filepath` from stdlib.
- Use `io.Writer` parameter for output (not `fmt.Println`), so tests can capture output.
- Error messages start lowercase, no trailing period.
- Conventional commits, present tense, under 72 chars.

### Testing

- Every new command or behavior needs integration tests.
- Tests use `t.TempDir()` and `t.Setenv()`, sequential (not `t.Parallel()`).
- Use local paths as git remotes in tests, never `https://` URLs.
- `cmdLink` tests pass `strings.NewReader("")` or `&bytes.Buffer{}` for stdin.

### Safety

- `commit` must always exit 0 (it runs as a Stop hook).
- Path traversal: any file operations derived from external input (e.g., Claude output in compact) must validate paths stay within the project directory.
- Atomic writes (temp file + rename) for settings files.
- No `.gitignore` manipulation in user projects.

### Architecture

- `.repo` stores remote origin URL (project identity marker).
- `.path` stores canonical project path (main worktree).
- `readMemoryFiles()` must skip `.repo` and `.path`.
- `resolveSlug()` matches cwd's main worktree against `.path` files.
- JSON round-tripped via `map[string]any` to preserve unknown fields.
