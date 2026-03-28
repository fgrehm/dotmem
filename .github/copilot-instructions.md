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
- Git staging discipline: never use `git add -A` or `git add .` in commands that create/update specific files. Stage only the intended paths (e.g., `git add <slug>`, `git commit -m "..." <file>`). Broad staging silently commits unrelated pending changes.
- Hook matching: when searching for a specific hook entry to remove, check all discriminating fields (`type` and `command`), not just one.
- Error wrapping: wrap underlying errors with `%w` so callers can inspect root cause. Don't discard `gitExec` errors by replacing them with a bare string.

### Slugs

- All user-supplied slugs must be normalized via `normalizeSlug()` before validation and lookup. Applies to every command that accepts a slug argument (`link`, `unlink`, `log`, `compact`, `cd`). Auto-detected slugs from `resolveSlug()` are already normalized.
- Paths read from `.path` files must be validated non-empty before use (`os.Stat("")` gives an unhelpful error).
- When a `.path` target is expected to be a directory, verify `info.IsDir()` after stat.

### Testing

- All `os.ReadFile` and `json.Unmarshal` calls in tests must check and `t.Fatal` on errors. Silent ignoring can mask regressions or cause misleading panics on type assertions.

### Architecture

- `.repo` stores remote origin URL (project identity marker).
- `.path` stores canonical project path (main worktree).
- `readMemoryFiles()` must skip `.repo` and `.path`.
- `resolveSlug()` matches cwd's main worktree against `.path` files.
- JSON round-tripped via `map[string]any` to preserve unknown fields.
- `ensureGitignoreRule(path)` appends the `**/.path` rule if not present. If it modifies the file, commit the change immediately with `git commit -m "..." <file>` to prevent auto-commit sweeping it up.

## Tooling

- Go version: see `go.mod`.
- Linter: golangci-lint v2, managed as a Go tool dependency. Run `make lint` or
  `go tool golangci-lint run ./...`. Config in `.golangci.yml`.
- Formatting: `make fmt` runs gofumpt + goimports via `go tool golangci-lint fmt`.
- Dead code: `make deadcode` runs `go tool deadcode ./...` (hard gate in CI).
- Complexity: `make audit` runs gocyclo (informational at 15, hard gate at 30 in CI).
- Tests: `make test` runs with `-race -shuffle=on`.
- Pre-commit hook: `.githooks/pre-commit` auto-formats and lints staged files.
  Run `make setup-hooks` to activate.
- Release: tag-triggered via GoReleaser. Release notes extracted from `CHANGELOG.md`.
  See the Releasing section in CLAUDE.md.

## CHANGELOG

When reviewing PRs, verify that `CHANGELOG.md` has an `[Unreleased]` entry for any
user-facing change (features, fixes, breaking changes). Use
[Keep a Changelog](https://keepachangelog.com/) format.
