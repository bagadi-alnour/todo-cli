# Changelog

All notable changes to this project are documented in this file.

## [0.6.0] - 2026-05-18

### Added

- **Per-creator storage** — todos are stored in `.todos/users/<firstname-lastname>.json` (from `git config user.name`), so teammates rarely conflict in Git.
- **`createdBy` field** on each todo (owner slug, not email).
- **Assignees** — `--assign` on `add`/`edit`, `--assignee` filter on `list`, assignee in Web UI and `todo export`.
- **`todo contributors`** — list git contributors from `git shortlog` (cached in `.todos/contributors.json`).
- **Assignee suggestions** — `todo add` with paths prints a blame-based assignee hint when `--assign` is omitted.
- **`todo stats --by-assignee`** — workload breakdown by assignee.
- Shell completion for `--assign` / `--assignee` flags.
- `TODO_USER_NAME` env override for CI/tests (maps to the same slug rules as `git user.name`).

### Changed

- **`todo init`** creates `.todos/users/` instead of a monolithic todo list file.
- **Web UI** — assignee dropdown and filter; add form layout improvements; list sorted newest-first; done todos hidden from **all** view (use **done** filter).
- **Legacy migration** — existing `.todos/todos.json` is migrated into `users/*.json` on first load (`legacy.json` for items without `createdBy`).

### Notes

- Assignee values still use **git author email** internally (for matching contributors). Only **filenames** avoid email for privacy.
- Requires `git config user.name` to be set for `todo add` (or set `TODO_USER_NAME`).

## [0.5.0] and earlier

See [GitHub Releases](https://github.com/bagadi-alnour/todo-cli/releases).
