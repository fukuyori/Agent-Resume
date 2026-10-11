# Version Update Checklist

Use this checklist whenever the application version changes.

## Files to update

- [ ] `main.go`: update the `version` value.
- [ ] `README.md`: update the version command output and any version shown in examples.
- [ ] `README.ja.md`: update the version command output and any version shown in examples.
- [ ] `CHANGELOG.md`: move the applicable entries from `Unreleased` into a dated version section.
- [ ] `CHANGELOG.md`: add or update the comparison links at the bottom of the file.

## Verification

- [ ] Search the repository for the previous version and review every match.
- [ ] Run `go test ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `go build .`.
- [ ] Run `git diff --check`.
- [ ] Confirm that only intended files and generated artifacts are present in `git status --short`.

## Release operations

- [ ] Confirm that the numeric release tag (for example, `0.6.0`) matches the `version` value in `main.go`.
- [ ] Confirm that the release workflow is expected to publish Windows x86_64 ZIP, macOS Universal ZIP, Linux x86_64 tar.gz, and `SHA256SUMS` assets.
- [ ] Create a commit only when explicitly requested.
- [ ] Create a tag only when explicitly requested.
- [ ] Push commits or tags only when explicitly requested.
