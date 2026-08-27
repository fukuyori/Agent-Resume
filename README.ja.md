# agres

カレントディレクトリ、または既知の全プロジェクトからCLIコーディングエージェントのセッションを再開するツール。

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

# 既知の全プロジェクトの履歴を表示
agres --all
agres -a

# 履歴の表示件数を指定 (デフォルト: 10)
agres 20
agres -n 20
agres --limit 20
agres --all --limit 20
```

デフォルトでは、カレントディレクトリで最近更新されたセッションを10件表示します。`--all`または`-a`を指定すると、全プロジェクトの履歴を対象にします。全体表示では各履歴の元の作業フォルダも表示し、選択したエージェントをそのフォルダから再開します。

Aiderは中央の索引ではなく各プロジェクト内に履歴を保存します。そのため、全体表示でもAiderについては`agres`を起動したフォルダの履歴だけを含め、ファイルシステム全体は走査しません。

矢印キーまたは `j`/`k` で選択、`Enter` で確定、`q` または `Esc` で終了。選択中のセッションは行全体の背景色で表示します。各行には保存されている履歴のサイズも表示します(不明な場合は `-`)。履歴 1MB がおおよそ文脈 1 杯分に相当するため、3MB 以上は黄色(何度か圧縮済み。新しいセッションへの引き継ぎを検討)、10MB 以上は赤(再開が遅く、初期の文脈もほぼ残っていない)で表示します。

```
  agres 0.5.1  [all projects]
  /projects/web-app

   2026-07-22 06:30:00  [opencode]   45.2K  [web-app]  ログインバグ修正  opencode
   2026-07-21 22:15:00  [claude]      1.3M  [api]      認証モジュールリファクタ  claude
   2026-07-21 22:15:00  [agy]       210.0K  [weather]  天気を確認
   2026-07-20 14:00:00  [aider]       8.1K  [current]  ユニットテスト追加

  j/k or ↑↓: 移動  enter: 選択  q/esc: 終了
```

## 古いセッションの整理

古いセッションを自動で削除するのは Claude Code だけ(`cleanupPeriodDays`、既定 30 日)で、Codex・OpenCode・Antigravity は無期限に溜まり続けます。`agres clean` はこれらを横断して、古い、または肥大化したセッションを削除します。

```bash
# カレントプロジェクト: 30 日更新のないセッションを、直近 3 件を残して削除
agres clean

# 全プロジェクト
agres clean -a

# 日付に関係なく 10MB 以上のセッションを削除
agres clean -a --larger-than 10M --older-than 0 --keep 0

# Codex だけを確認なしで削除(定期実行向け)
agres clean -a --agent codex --yes

# 削除対象を表示するだけ
agres clean -a --dry-run
```

| オプション | 既定 | 意味 |
|---|---|---|
| `-a`, `--all` | オフ | 全プロジェクトを対象にする |
| `--older-than <期間>` | `30d` | 最終更新がこの期間より前のものだけ(`12h`、`2w`、`0` で無効) |
| `--larger-than <サイズ>` | オフ | 履歴サイズがこれ以上のものだけ(`512K`、`10M`、`1G`) |
| `--agent <名前>` | 全部 | `claude`、`codex`、`opencode`、`agy` のいずれか |
| `--keep <件数>` | `3` | プロジェクトごとに直近 N 件は必ず残す |
| `-y`, `--yes` | オフ | 確認プロンプトを省略 |
| `--dry-run` | オフ | 対象を表示するだけ |

複数の条件は AND で組み合わせます。直近 1 時間以内に更新されたセッションは、実行中のエージェントのものである可能性があるため削除しません。削除は即時(ゴミ箱なし)ですが、削除前に一覧を表示して確認します。Aider はプロジェクトごとの 1 ファイルに全履歴が入っているため対象外です。

## バージョン

```bash
agres --version
# agres 0.5.1
```

## ライセンス

MIT
