# Changelog

All notable changes to this project are documented here.

This project follows the principles of Keep a Changelog and Semantic Versioning.

## [1.0.0] - 2025-12-17

### Added
- Command repetition: run any shell command a specified number of times (sequential execution).
- Cross‑platform execution: uses the system shell (Unix `/bin/sh -c`, Windows `cmd.exe /c`).
- Result reporting: captures exit code, stdout, stderr, and duration for each run.
- TUI feedback: intro/outro messages, spinner, and colored, structured output.
- Silent mode: suppress per‑run output when you only care about exit status.
- Built‑in help and version display.
- Deno tasks for development, building a binary, and local installation.

### Known Limitations
- Concurrent execution is not implemented in this release.
- Verbose mode, interactive prompts, and execution summaries are not yet available.