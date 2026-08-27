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

Use arrow keys or `j`/`k` to navigate, `Enter` to select, `q` or `Esc` to quit. The selected session is highlighted across the entire row. Each row also shows the size of the stored history (`-` when unknown). Roughly 1MB of history corresponds to one full context window, so sizes of 3MB or more are shown in yellow (compacted several times; consider handing off to a new session) and 10MB or more in red (slow to resume and unlikely to retain early context).

```
  agres 0.5.1  [all projects]
  /projects/web-app

   2026-07-22 06:30:00  [opencode]   45.2K  [web-app]  Fix login bug  opencode
   2026-07-21 22:15:00  [claude]      1.3M  [api]      Refactor auth module  claude
   2026-07-21 22:15:00  [agy]       210.0K  [weather]  Check weather
   2026-07-20 14:00:00  [aider]       8.1K  [current]  Add unit tests

  j/k or ↑↓: navigate  enter: select  q/esc: quit
```

## Cleaning up old sessions

Only Claude Code deletes old sessions on its own (`cleanupPeriodDays`, 30 days by default); Codex, OpenCode and Antigravity keep every session forever. `agres clean` removes old or oversized sessions across all of them.

```bash
# Current project: sessions not updated for 30 days, keeping the newest 3
agres clean

# All projects
agres clean -a

# Sessions of 10MB or more regardless of age
agres clean -a --larger-than 10M --older-than 0 --keep 0

# Only Codex, delete without asking (for scheduled runs)
agres clean -a --agent codex --yes

# Show what would be deleted and exit
agres clean -a --dry-run
```

| Option | Default | Meaning |
|---|---|---|
| `-a`, `--all` | off | Clean sessions from all projects |
| `--older-than <dur>` | `30d` | Only sessions last updated before this long ago (`12h`, `2w`, `0` disables) |
| `--larger-than <size>` | off | Only sessions whose history is at least this big (`512K`, `10M`, `1G`) |
| `--agent <name>` | all | `claude`, `codex`, `opencode` or `agy` |
| `--keep <count>` | `3` | Always keep the newest N sessions of each project |
| `-y`, `--yes` | off | Skip the confirmation prompt |
| `--dry-run` | off | List candidates only |

Multiple conditions are combined with AND. Sessions updated within the last hour are never deleted, since they may belong to a running agent. Deletion is immediate (no trash); the list is shown and confirmed before anything is removed. Aider is not supported because its whole history lives in a single file per project.

## Version

```bash
agres --version
# agres 0.5.1
```

## License

MIT
