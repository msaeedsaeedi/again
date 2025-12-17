# xn

Run any command or script n times, sequentially. Useful for quick retries, basic load/benchmark loops, or repeated task execution with simple feedback and colored output.

## Features (v1.0.0)

- Execute any command n times (sequential)
- Cross‑platform execution (Unix `/bin/sh -c`, Windows `cmd.exe /c`)
- Captures exit code, stdout, stderr, and duration per run
- TUI feedback: intro/outro, spinner, and colored output
- Silent mode to suppress per‑run output
- Built‑in help and version display
- Deno tasks for development, building, and local install

## Installation

Build and install from source with Deno:

```bash
# Install locally (script entry)
deno task install

# Or build a standalone binary
deno task build
# Binary will be created at ./bin/xn
```

## Usage

```bash
# Run a command 5 times
xn -n 5 echo "Hello"

# Run a script multiple times
xn -n 3 node script.js

# Suppress per‑run output (silent mode)
xn -n 10 -s node script.js
```

## Options

```
-n, --count <number>   Number of times to execute (default: 1)
-s, --silent           Silent mode – suppress per‑run output
-h, --help             Show help information
-v, --version          Show version information
```

## Limitations / Roadmap

- Concurrent execution is not available in v1.0.0.
- Verbose mode, interactive prompts, and execution summaries are not yet implemented.

## Acknowledgments

- Built with [Deno](https://deno.land/)
- Terminal UI via [@clack/prompts](https://github.com/natemoo-re/clack)
- Colors by [picocolors](https://github.com/alexeyraspopov/picocolors)
