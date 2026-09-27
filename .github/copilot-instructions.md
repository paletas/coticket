# Copilot Instructions

## Build, test, and lint
- Format Go code:
  - `Get-ChildItem -Path . -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }`
- Run tests:
  - Full suite: `go test .\...`
  - Single test: `go test .\internal\service -run TestClaimTicketOwnershipRules -v`
- Build MCP server binary:
  - `go build .\cmd\coticket-mcp`

## High-level architecture
- `cmd/coticket-mcp/main.go` is the runtime entrypoint; it loads config, opens SQLite storage, wires the lifecycle service, and starts the MCP stdio server.
- `internal/mcp` owns JSON-RPC framing (Content-Length protocol) plus MCP method/tool routing (`initialize`, `tools/list`, `tools/call`).
- `internal/service` is the business layer and enforces lifecycle rules (ticket type validation, claim ownership rules, status transitions).
- `internal/store` is the SQLite persistence layer, including schema bootstrap for `tickets` and `ticket_events`.
- `internal/domain` defines canonical ticket types/statuses and allowed status transition rules.

## Key conventions
- Supported ticket types are `epic`, `story`, and `bug`; supported statuses are `todo`, `in_progress`, `blocked`, and `done`.
- `ticket_list_ready` is intentionally the unassigned todo queue (`status='todo'` and empty `assignee`), ordered by oldest `updated_at`.
- Keep behavior rules in `internal/service`; avoid placing business transition/ownership logic directly in MCP handlers or SQL queries.
- SQLite path defaults to `<repo>\.coticket\coticket.db`; use `COTICKET_REPO_PATH`, `-repo-path`, and optionally `COTICKET_DB_PATH`/`-db-path` for explicit per-repo isolation.
- Use `scripts/setup-repo.ps1 -RepoPath <repo>` to bootstrap repo-local launcher/config under `.coticket\`.
- Update README and this file in the same change when commands, tool contracts, or lifecycle behavior changes.
- If work becomes repetitive across sessions, codify it as a reusable SKILL and reference that SKILL in agent instructions.
- Use `.github/skills/coticket-ticket-lifecycle/SKILL.md` for the standard ticket-create → list-ready → claim → update-status workflow.
- For any ticket operation, start by following that SKILL checklist and do not bypass it with ad-hoc workflow steps.
- Enforce the SKILL server-start gate: before any `ticket_*` operation, ensure repo-scoped CoTicket MCP is started (bootstrap via `scripts/setup-repo.ps1` if needed).
