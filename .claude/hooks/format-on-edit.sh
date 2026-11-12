#!/usr/bin/env bash
# PostToolUse hook: format the edited file the same way the pre-commit hook does.
# Never blocks: missing tools or formatter errors are ignored.
f=$(jq -r '.tool_input.file_path // empty')
[ -f "$f" ] || exit 0
root="${CLAUDE_PROJECT_DIR:-$(pwd)}"

case "$f" in
  *.go)
    command -v gofmt >/dev/null && gofmt -w "$f"
    ;;
  "$root"/frontend/wailsjs/*)
    ;; # generated bindings: leave alone
  "$root"/frontend/*.ts|"$root"/frontend/*.js|"$root"/frontend/*.svelte|"$root"/frontend/*.css|"$root"/frontend/*.html)
    if [ -x "$root/node_modules/.bin/prettier" ]; then
      (cd "$root" && node_modules/.bin/prettier --write "$f" >/dev/null 2>&1)
    fi
    ;;
esac
exit 0
