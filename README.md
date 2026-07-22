# agres

Resume CLI coding agent sessions from the current directory.

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
```

Use arrow keys or `j`/`k` to navigate, `Enter` to select, `q` or `Esc` to quit.

```
  agres 0.2.0

 > [opencode]  Fix login bug                2026-07-22 06:30:00  opencode
   [claude]    Refactor auth module         2026-07-21 22:15:00  claude
   [agy]       Check weather                2026-07-21 22:15:00
   [aider]     Add unit tests               2026-07-20 14:00:00

  j/k or ↑↓: navigate  enter: select  q/esc: quit
```

## Version

```bash
agres --version
# agres 0.2.0
```

## License

MIT
