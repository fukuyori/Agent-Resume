# agres

Resume CLI coding agent sessions from the current directory or from all known projects.

## Supported Agents

| Agent | Session Storage | Resume Command |
|-------|----------------|----------------|
| Claude Code | `~/.claude/projects/<slug>/*.jsonl` | `claude --resume <id>` |
| OpenCode | `~/.local/share/opencode/opencode.db` | `opencode --session <id>` |
| Aider | `<cwd>/.aider.chat.history.md` | `aider --resume` |
| Codex | `~/.codex/session_index.jsonl` | `codex resume <id>` |
| Antigravity CLI | `~/.gemini/antigravity-cli/brain/` | `agy --conversation <uuid>` |

## Install

```bash
go install github.com/user/agres@latest
```

Or build from source:

```bash
git clone https://github.com/user/agres.git
cd agres
go build -o agres .
```

## Usage

```bash
cd /path/to/your/project
agres

# Show sessions from all known projects
agres --all
agres -a

# Specify number of history items to show (default: 10)
agres 20
agres -n 20
agres --limit 20
agres --all --limit 20
```

By default, `agres` shows the 10 most recently updated sessions from the current directory. Use `--all` or `-a` to include sessions from all projects. In all-project mode, each entry includes its original working directory, and the selected agent is resumed from that directory.

Aider stores history inside each project instead of a central index. Therefore, all-project mode includes Aider history only from the directory where `agres` was started; it does not scan the entire filesystem.

Use arrow keys or `j`/`k` to navigate, `Enter` to select, `q` or `Esc` to quit. The selected session is highlighted across the entire row.

```
  agres 0.4.0  [all projects]
  /projects/web-app

   2026-07-22 06:30:00  [opencode]  [web-app]  Fix login bug  opencode
   2026-07-21 22:15:00  [claude]    [api]      Refactor auth module  claude
   2026-07-21 22:15:00  [agy]       [weather]  Check weather
   2026-07-20 14:00:00  [aider]     [current]  Add unit tests

  j/k or ↑↓: navigate  enter: select  q/esc: quit
```

## Version

```bash
agres --version
# agres 0.4.0
```

## License

MIT
