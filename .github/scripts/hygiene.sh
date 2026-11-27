#!/usr/bin/env bash
# Rejects content that can mislead a reviewer or run on a reviewer's machine.
# A pull request can edit this script, so also review any change under .github/.
set -euo pipefail
export LC_ALL=C.UTF-8
status=0

# Invisible and bidirectional-control characters ("Trojan Source"), and raw
# control characters such as terminal escape sequences.
pattern='[\x{00}-\x{08}\x{0B}\x{0C}\x{0E}-\x{1F}\x{7F}\x{200B}-\x{200F}\x{202A}-\x{202E}\x{2060}-\x{2064}\x{2066}-\x{2069}\x{FEFF}]'
rc=0
git grep -nIP "$pattern" || rc=$?
case "$rc" in
  0) echo "::error::Hidden or control characters found in the lines above."; status=1 ;;
  1) ;;
  *) echo "::error::git grep failed (exit $rc)."; exit "$rc" ;;
esac

# Files that editors, shells and AI assistants may load or execute automatically.
blocked="$(git ls-files | grep -Ei '(^|/)(\.claude|\.cursor|\.devcontainer|\.githooks)/|(^|/)(CLAUDE(\.local)?\.md|AGENTS\.md|GEMINI\.md|\.cursorrules|\.envrc|\.mcp\.json|copilot-instructions\.md)$|(^|/)\.vscode/(tasks|settings|mcp)\.json$' || true)"
if [ -n "$blocked" ]; then
  printf '::error::These files are not accepted without a maintainer decision:\n'
  printf '%s\n' "$blocked"
  status=1
fi

exit "$status"
