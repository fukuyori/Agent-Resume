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

- [ ] Create a commit only when explicitly requested.
- [ ] Create a tag only when explicitly requested.
- [ ] Push commits or tags only when explicitly requested.
