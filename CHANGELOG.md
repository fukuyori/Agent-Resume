# Changelog

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.5.2] - 2026-09-12

### Added
- MIT license file.
- Version update checklist.
- macOS PKG creation script with Developer ID signing and Apple notarization.

### Changed
- Removed the release archive from the repository; `*.zip` is now ignored.
- Corrected the installation instructions to use the actual GitHub repository.
- Displayed the recorded model in an aligned column immediately after the agent name; OpenCode now prefers the concrete model ID over the provider name.

## [0.5.1] - 2026-08-27

### Added
- `agres clean` subcommand to delete old or oversized sessions across Claude Code, Codex, OpenCode and Antigravity.
  - `--older-than` (default `30d`), `--larger-than`, `--agent`, `--keep` (default 3 per project), `-a/--all`, `-y/--yes`, `--dry-run`.
  - Conditions combine with AND; sessions updated within the last hour are never deleted; the candidate list is shown and confirmed before deletion.
  - OpenCode sessions are removed from the database with `VACUUM` so space is actually released.
  - Aider is not supported because its history is a single file per project.

## [0.5.0] - 2026-08-27

### Added
- History size column for every session (file size for Claude, Codex and Antigravity; `message` + `part` data for OpenCode; section size for Aider).
- Size color coding: 3MB or more in yellow, 10MB or more in red, as a hint that the session has been compacted many times and may be slow to resume.

### Fixed
- Codex sessions forked from another session were merged into their parent and disappeared from the list. Forked rollouts embed the parent's `session_meta`; only the first one is now used to identify the session.

## [0.4.0] - 2026-08-20

### Added
- `-a` / `--all` to list sessions from all known projects. Each entry shows its original working directory, and the selected agent is resumed from that directory.

### Changed
- Redesigned the list layout: timestamp first, then agent, project and title; the selected row is highlighted across its full width.
- Aider in all-project mode includes only the directory where `agres` was started (no filesystem scan).

## [0.3.0] - 2026-07-24

### Fixed
- Sessions are matched to the current project by comparing real filesystem locations (`samePath`), handling symlinks, case differences and moved directories.
- Codex index entries without a `cwd` no longer count as belonging to the current project.
- OpenCode sessions are filtered by their session directory.
- Antigravity workspace URIs are converted to native paths.

## [0.2.1] - 2026-07-24

### Added
- `-n` / `-l` / `--limit` and a bare count argument to choose how many sessions to show (default 10).
- `-v` / `--version` and `-h` / `--help`.

### Fixed
- Codex: session IDs are taken from the rollout filename; environment context is stripped from titles.
- Antigravity: timestamp parsing.

## [0.2.0] - 2026-07-22

### Changed
- Module renamed to `agres`.
- Claude Code project directories are located by trying several slug forms, with path normalization on Windows (case, slashes, volume names).

## [0.1.0] - 2026-07-22

### Added
- Initial release: list and resume sessions of Claude Code, OpenCode, Aider, Codex and Antigravity CLI from the current directory with a keyboard-driven TUI.

[Unreleased]: https://github.com/fukuyori/Agent-Resume/compare/0.5.2...HEAD
[0.5.2]: https://github.com/fukuyori/Agent-Resume/compare/2ee2747...0.5.2
[0.5.1]: https://github.com/fukuyori/Agent-Resume/compare/645d1a9...2ee2747
[0.5.0]: https://github.com/fukuyori/Agent-Resume/compare/35a94ca...645d1a9
[0.4.0]: https://github.com/fukuyori/Agent-Resume/compare/69d5aac...35a94ca
[0.3.0]: https://github.com/fukuyori/Agent-Resume/compare/556db40...69d5aac
[0.2.1]: https://github.com/fukuyori/Agent-Resume/compare/3d2e560...556db40
[0.2.0]: https://github.com/fukuyori/Agent-Resume/compare/3e609cf...3d2e560
[0.1.0]: https://github.com/fukuyori/Agent-Resume/commit/3e609cf
