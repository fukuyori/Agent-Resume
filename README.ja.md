# agres

カレントディレクトリのCLIコーディングエージェントのセッションを再開するツール。

## 対応エージェント

| エージェント | セッション保存先 | 再開コマンド |
|------------|----------------|-------------|
| Claude Code | `~/.claude/projects/<slug>/*.jsonl` | `claude --resume <id>` |
| OpenCode | `~/.local/share/opencode/opencode.db` | `opencode --session <id>` |
| Aider | `<cwd>/.aider.chat.history.md` | `aider --resume` |
| Codex | `~/.codex/session_index.jsonl` | `codex resume <id>` |
| Antigravity CLI | `~/.gemini/antigravity-cli/brain/` | `agy --conversation <uuid>` |

## インストール

```bash
go install github.com/user/agres@latest
```

ソースからビルド:

```bash
git clone https://github.com/user/agres.git
cd agres
go build -o agres .
```

## 使い方

```bash
cd /path/to/your/project
agres
```

矢印キーまたは `j`/`k` で選択、`Enter` で確定、`q` または `Esc` で終了。

```
  agres 0.2.0

 > [opencode]  ログインバグ修正              2026-07-22 06:30:00  opencode
   [claude]    認証モジュールリファクタ      2026-07-21 22:15:00  claude
   [agy]       天気を確認                    2026-07-21 22:15:00
   [aider]     ユニットテスト追加            2026-07-20 14:00:00

  j/k or ↑↓: 移動  enter: 選択  q/esc: 終了
```

## バージョン

```bash
agres --version
# agres 0.2.0
```

## ライセンス

MIT
