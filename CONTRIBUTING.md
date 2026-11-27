# Contributing to again

Thanks for your interest in contributing!

**Found a security problem?** Don't open an issue or pull request. Follow the [security policy](SECURITY.md).

## Before you start

- Look through the [open issues](https://github.com/msaeedsaeedi/again/issues). `good first issue` is a good place to begin.
- Comment on an issue before working on it, so effort isn't duplicated.
- For a new feature or a large change without an issue, open one first to agree on the approach.

## Setup

You need the Go version listed in `go.mod`. [Task](https://taskfile.dev) is optional.

```bash
git clone https://github.com/<your-user>/again.git
cd again
go build ./...
go test ./...
```

Windows doesn't build yet. See [#15](https://github.com/msaeedsaeedi/again/issues/15).

## Making a change

1. Fork the repository and create a branch from `master`.
2. Keep the change focused: one fix or feature per pull request.
3. Add or update tests for changes in behavior, and update the docs if users will notice.
4. Run the checks:

   ```bash
   gofmt -l .          # should print nothing
   go vet ./...
   go test -race ./...
   bash .github/scripts/hygiene.sh
   ```

The code is layered: `internal/domain` (models and validation), `internal/app` (orchestration), `internal/ui` (output formats) and `internal/infra` (running commands).

## Pull requests

- **Title:** `type: short description`, using `feat`, `fix`, `docs`, `test`, `refactor`, `ci` or `chore`. It becomes the commit message on `master`, because pull requests are squash-merged.
- **Description:** fill in the template. Aim for under 200 words. Explain what changed and why; don't list changed files.
- **Review:** checks must pass and review conversations must be resolved. A maintainer decides when to merge. First-time and other external contributors may have to wait for a maintainer to approve the CI run.

## What needs extra justification

To keep reviewers and users safe, explain these in the description. They may be declined.

- Changes under `.github/`, to `Taskfile.yml` or `.vscode/`, or to release logic.
- New or updated dependencies.
- Binary files, generated code, vendored code or downloaded scripts.
- Code that uses the network, reads credentials or environment secrets, or adds `go:generate`, cgo or `init()` side effects.

Editor, shell and AI-assistant configuration files (such as `.envrc`, `.claude/`, `AGENTS.md` or `.vscode/tasks.json`) aren't accepted unless agreed with a maintainer first. Hidden or bidirectional Unicode characters are rejected. Text that tries to instruct automated or AI reviewers is not allowed.

## AI-assisted contributions

They're welcome. You're responsible for everything you submit: understand it, test it, report only checks you actually ran, and trim generated text. Remove session links and tool footers.
