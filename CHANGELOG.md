# Changelog

## [0.2.0] - 2026-03-23

### Breaking

- Default directory changed from `~/.dotmem` to `~/.mem`. Existing users: `mv ~/.dotmem ~/.mem` and update `autoMemoryDirectory` in each project's `.claude/settings.local.json`.

### Added

- `dotmem unlink` -- remove the memory link for the current project.
- `dotmem uninstall-hook` -- remove the Stop hook from `~/.claude/settings.json`.
- `dotmem log [slug]` -- show memory change history for a project.
- `dotmem push` -- push the memory repo to its remote.
- `dotmem cd [slug]` -- open a subshell in a project or memory directory (exit to return).
- `.path` file stored alongside `.repo` during `link`, mapping slug back to the project's canonical path (main worktree).
- `compact` and `log` auto-detect slug from the current directory when omitted.
- `ls` shows project paths when available.
- `compact` checks `claude --version` and fails early if below 2.1.78.

### Changed

- `dotmem status` renamed to `dotmem ls`.

## [0.1.0] - 2026-03-22

Initial release.

- `dotmem init` -- create the central memory repo.
- `dotmem link [slug]` -- link a project to the memory repo.
- `dotmem commit` -- auto-commit changed memory files (Stop hook).
- `dotmem compact <slug>` -- merge memory files via Claude with streaming progress.
- `dotmem install-hook` -- register the Stop hook.
- `dotmem status` -- list linked projects.
