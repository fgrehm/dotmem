# Ideas

Future directions for dotmem. Nothing here is committed to, just things worth exploring.

## Memory intelligence

- ~~**Streaming compact**~~ -- done (stream-json with file-read progress)
- **Incremental compaction** -- detect which files changed since last compact and only re-process those, instead of rewriting the full MEMORY.md each time
- **Deep compaction** -- `dotmem compact --deep` to also compress surviving reference documents (specs, plans) individually, not just merge into MEMORY.md
- **Per-directory CLAUDE.md from memories** -- generate project-specific CLAUDE.md instructions based on accumulated memory patterns
- **Compact dry-run** -- `dotmem compact --dry-run` to preview the new MEMORY.md content before applying
- **Configurable line limit** -- `dotmem compact --max-lines 300` to override the default 200-line limit for projects that need more context

## More commands

- `dotmem push` -- auto-push to remote on a schedule or post-commit
- `dotmem diff` -- show what changed in memory across sessions
- `dotmem unlink` / `dotmem uninstall-hook` -- teardown commands
- `dotmem repair` -- fix `.repo` URLs after GitHub repo renames/transfers (currently manual edit)
- **Import detection** -- detect when Claude Code has written memories to the default `~/.claude/` path (outside dotmem) and offer to import them

## Organization

- ~~**Rename default directory**~~ -- done (changed to `~/.mem`)
- **Branch-aware memory** -- organize as `<slug>/branches/<branch>/` or tag memories with branch metadata
- **Pruning** -- detect and clean up stale project directories

## Wiring

- **Symlink-based wiring** -- `dotmem link --symlink` for tool-agnostic memory (Cursor, Copilot, etc.) via `.ai/` symlinks
- **Multi-machine sync** -- document or automate remote repo setup for cross-machine use

## Session context

- **Richer commit messages** -- include project name or session summary instead of just a timestamp

## Hardening

- **Claude CLI version check** -- `dotmem compact` could check `claude --version` and fail early if below minimum (tested against 2.1.78+)
- **Compact recovery docs** -- document that `git checkout -- .` in `~/.mem` recovers from a bad compact
