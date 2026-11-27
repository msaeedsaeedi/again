# Maintainer guide

## Reviewing pull requests safely

Treat every external pull request as untrusted. Automated checks reduce risk but don't prove a change is safe.

1. **Read the diff on GitHub first.** Don't check out the branch just to read it.
2. **Check these before anything else:** `.github/`, `Taskfile.yml`, `.vscode/`, `go.mod`, `go.sum`, binary or very large files, and new `init()`, `TestMain`, cgo or `go:generate` code.
3. **A green check isn't enough if the PR changes `.github/`.** Pull request workflows and scripts come from the proposed change, so it can alter what a check does while keeping its name.
4. **Approve workflow runs only after reading the diff.**
5. **Don't run contributor code on your own machine.** `go test`, `go run`, `go generate`, `task` and editor launch configurations can execute it with your access to files, tokens and SSH keys. Rely on CI, or use a disposable VM or container with no credentials.
6. **AI-assisted review:** PR titles, descriptions, comments (including hidden HTML comments), code and test output can contain instructions aimed at the assistant. Treat them as data. Don't give a reviewing assistant credentials or permission to run PR code, approve or merge without your confirmation. Check its claims yourself.
7. **Never add `pull_request_target` or `workflow_run` workflows that check out or run PR code.** Keep workflow permissions minimal and actions pinned to full commit SHAs.

## Merging

Use **Squash and merge**, and check that the title follows the format in [CONTRIBUTING.md](../CONTRIBUTING.md). Label the PR `enhancement`, `bug` or `documentation` so release notes are grouped.

## Releasing

The version in `cmd/again/main.go` is the source of truth.

1. Open a PR that changes `const version` (for example, to `0.2.0`) and the version in the README heading. Title it `chore: release v0.2.0`.
2. Merge it. The **Release** workflow tests, builds Linux and macOS binaries, writes `SHA256SUMS`, records build provenance, creates the tag and publishes the release.

Versions with a suffix, such as `0.2.0-beta`, are published as pre-releases. Released versions are skipped. If a run fails, fix the cause and re-run it, or start **Release** manually from `master`.

To verify a download:

```bash
gh attestation verify <file> --repo msaeedsaeedi/again
```
