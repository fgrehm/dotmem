# CLAUDE.md

## What is this

Go CLI tool that centralizes Claude Code memory files into `~/.mem/` (a git repo) with automatic versioning via Stop hooks. Uses cobra for the CLI framework.

## Commands

```
dotmem init, link, unlink, commit, compact, install-hook, uninstall-hook, ls, log, push, cd
```

## Build and test

```sh
go build ./...          # build
go test ./...           # run all tests
go vet ./...            # static analysis
golangci-lint run       # linting
make install            # symlink to ~/.local/bin
```

## Code structure

One file per command. Thin cobra wrappers (`newXCmd()`) delegate to `cmdX()` functions. Tests call `cmdX()` directly.

```
main.go          entry point
root.go          cobra root command, registers subcommands
helpers.go       dotmemDir(), gitExec(), resolveSlug(), JSON helpers
<command>.go     one per command
<command>_test.go
```

## Testing conventions

- Integration tests with `t.TempDir()` and `t.Setenv()`. Sequential (not parallel) because `t.Setenv` is process-global.
- `setupGitEnv(t)` sets git author/committer env vars for CI.
- `initDotmem(t)` creates a temp dotmem repo via `DOTMEM_DIR`.
- `makeTempRepo(t, remoteURL)` creates a temp git repo with a fake remote.
- `fakeHome(t)` overrides `HOME` to a temp dir.
- Use local paths as remotes (e.g., `t.TempDir()`), NOT `https://` URLs. Git may rewrite https to SSH via global config.
- `cmdLink` accepts `io.Reader` for stdin. Pass `strings.NewReader("")` or `&bytes.Buffer{}` in tests (both non-TTY).
- `fakeClaude(t, result)` creates a fake `claude` CLI for compact tests.

## Key conventions

- `commit` always exits 0 (it runs as a hook). All other commands fail hard on errors.
- `.repo` file stores remote origin URL (project identity). `.path` file stores canonical project path (main worktree).
- `readMemoryFiles()` skips `.repo` and `.path`.
- JSON: `encoding/json` only, `map[string]any` for round-tripping. Atomic writes (temp file + rename) for settings.
- No external dependencies beyond cobra.
- Slug normalization: call `normalizeSlug(slug)` on any user-supplied slug before `validateSlug` and directory lookup. Auto-detected slugs from `resolveSlug()` are already normalized.
- Git staging: stage only the intended paths. Never `git add -A` in link/commit-style operations on the dotmem repo; use `git add <path>` or `git commit -m "..." <file>` to limit scope.
- Error wrapping: wrap `gitExec` errors with `%w`, don't replace them with bare strings.
