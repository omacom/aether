---
name: verify
description: Run the full pre-commit verification suite (Go tests, svelte-check, Vitest, format checks) and report results.
disable-model-invocation: true
---

# Verify

Run every check below from the repo root, even if an earlier one fails, then report a pass/fail summary with the relevant output for failures. Do not fix anything unless asked.

1. **Go tests**: `make test` (`go test ./internal/... ./cli/...`)
2. **Go formatting**: `gofmt -l $(git ls-files '*.go')` — any output is a failure
3. **Frontend types**: `cd frontend && npm run check`
4. **Frontend tests**: `cd frontend && npm test`
5. **Prettier**: `npx prettier --check $(git diff --name-only HEAD -- 'frontend/**/*.ts' 'frontend/**/*.js' 'frontend/**/*.svelte' 'frontend/**/*.css' 'frontend/**/*.html')` — skip if no frontend files changed

Notes:
- If a tool is missing (`go`, `node`, `node_modules`), report it as "skipped: <reason>", never as a pass.
- App entry-point tests need native deps and are not part of this skill: `go test -race -tags webkit2_41 ./...`.
- Final line: `VERIFY: PASS` only if nothing failed and nothing was skipped; otherwise `VERIFY: FAIL` or `VERIFY: INCOMPLETE`.
