# dotmem

Centralize Claude Code memory files into a single git-tracked repo with automatic versioning.

## Why

Claude Code's [auto memory](https://code.claude.com/docs/en/memory#auto-memory) saves context
to `~/.claude/projects/<project>/memory/`. dotmem takes this further by centralizing
all project memories into a single git repo:

- **Version history** - every session auto-commits, so you can diff and revert
- **Cross-project search** - one repo means `grep -r` across all your AI context
- **Portable** - `git push` to a private remote for backup or multi-machine sync
- **Worktree-aware** - multiple worktrees of the same project share one memory directory

## How it works

dotmem creates a central git repo at `~/.dotmem/` (override with `DOTMEM_DIR`, must
be an absolute path). Each linked project gets its own subdirectory:

```
~/.dotmem/                      <- central git repo
├── .gitignore
├── README.md
├── my-app/                     <- one dir per project
│   ├── .repo                   <- remote origin URL (identity marker)
│   ├── MEMORY.md
│   ├── debugging.md
│   └── api-conventions.md
└── another-repo/
    ├── .repo
    └── MEMORY.md
```

Two pieces of configuration make this work:

**Per-project** (`.claude/settings.local.json`, gitignored): `dotmem link` writes
this to redirect auto memory into the central repo.

```json
{
  "autoMemoryDirectory": "/home/you/.dotmem/my-app"
}
```

**Global** (`~/.claude/settings.json`): `dotmem install-hook` adds a
[Stop hook](https://code.claude.com/docs/en/hooks#stop) that auto-commits after every
Claude Code session.

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "dotmem commit"
          }
        ]
      }
    ]
  }
}
```

You don't need to write either of these manually. `dotmem link` and
`dotmem install-hook` handle it.

## Install

Download a binary from [GitHub releases](https://github.com/fgrehm/dotmem/releases)
and add it to your PATH.

Or if you have Go installed:

```
go install github.com/fgrehm/dotmem@latest
```

## Quickstart

```sh
dotmem init              # create ~/.dotmem as a git repo
dotmem install-hook      # register the Claude Code Stop hook
cd ~/projects/my-app     # must be a git repo with a remote origin
dotmem link              # link this project's memory to ~/.dotmem/my-app
```

That's it. Claude Code now writes memory files to `~/.dotmem/my-app/` and the
Stop hook auto-commits them after every session.

Repeat `dotmem link` inside each project you want to track. Make sure
`.claude/settings.local.json` is in your global or project `.gitignore` so you
don't commit machine-specific paths.

> [!IMPORTANT]
> If you push your dotmem repo to a remote, **keep it private**. Memory files may
> contain project-specific context and internal details.

## Commands

| Command | Description |
|---|---|
| `dotmem init` | Create the central memory repo at `~/.dotmem` |
| `dotmem link [slug]` | Link the current project (derives slug from dir name if omitted, `-y` to skip prompts) |
| `dotmem commit` | Auto-commit changed memory files (always exits 0, silent by default) |
| `dotmem compact <slug>` | Merge memory files into a single MEMORY.md via Claude (requires `claude` CLI; `-m` model, `-e` effort, `-y` skip prompt) |
| `dotmem install-hook` | Register the Stop hook in `~/.claude/settings.json` |
| `dotmem status` | List linked projects with last-modified dates |

Set `DOTMEM_DIR` to an absolute path to override the default `~/.dotmem` location.
If you use a custom `DOTMEM_DIR`, make sure it's set in your shell profile so the
Stop hook can see it. Set `DOTMEM_DEBUG=1` for verbose output from `dotmem commit`.

Shell completions are available via `dotmem completion` (bash, zsh, fish, powershell).

## Troubleshooting

### Auto memory is not enabled

`autoMemoryEnabled` must be on (the default). If you previously disabled it, remove
`"autoMemoryEnabled": false` from your settings or unset the
`CLAUDE_CODE_DISABLE_AUTO_MEMORY` environment variable. See the
[auto memory docs](https://code.claude.com/docs/en/memory#enable-or-disable-auto-memory).

### Claude writes to the wrong memory directory

If Claude ignores `autoMemoryDirectory` and writes to the default location, add an
explicit instruction to your project's `CLAUDE.md`:

```markdown
Write memory files to ~/.dotmem/<your-slug>/ (not the default auto memory location).
```

Related issues: [#33535](https://github.com/anthropics/claude-code/issues/33535),
[#36636](https://github.com/anthropics/claude-code/issues/36636).

### Existing memory is not migrated

`dotmem link` does not move existing memory from `~/.claude/projects/`. To migrate
manually:

```sh
cp ~/.claude/projects/<project>/memory/* ~/.dotmem/<slug>/
```

### Reverting a bad compact

`dotmem compact` auto-commits after applying changes. If the result looks wrong:

```sh
cd ~/.dotmem
git log --oneline     # find the commit before compact
git revert HEAD       # undo the compact commit
```

### Remote origin URL changed

If your project's remote origin URL changes (repo rename or transfer), `dotmem link`
will report a slug collision. Update the `.repo` file in `~/.dotmem/<slug>/.repo`
with the new URL and re-run `dotmem link`.

### Uninstalling

To stop using dotmem:

1. Remove `autoMemoryDirectory` from each project's `.claude/settings.local.json`
2. Remove the `dotmem commit` Stop hook from `~/.claude/settings.json`
3. Optionally delete `~/.dotmem/` (or keep it as a read-only archive)

## Alternatives

- [**Claude Code auto memory**](https://code.claude.com/docs/en/memory) (built-in) - per-project, no git history, no cross-project search
- [**claude-mem**](https://github.com/thedotmack/claude-mem) - session capture + AI compression + searchable SQLite archive via MCP
- [**supermemory**](https://github.com/supermemoryai/claude-supermemory) - cross-project + team memory via external API

dotmem fills a different niche: local plain-text files, git, zero infrastructure.

## Ideas

See [IDEAS.md](IDEAS.md) for future directions.

## License

[MIT](LICENSE)
