# CoTicket Ticket Lifecycle Skill

## Purpose
Use CoTicket as a durable work queue so AI agents can create, claim, execute, and complete tickets across sessions.

## Enforcement
- Treat this SKILL as mandatory for ticket lifecycle work in this repository and downstream repositories using CoTicket.
- Do not skip required server-start, claim, and status updates.

## When to use
- You need to turn a plan/backlog item into actionable work.
- You need the next unowned ticket instead of scanning docs manually.
- You need to persist ownership and status updates so future sessions can resume work.

## Installation
Install CoTicket MCP globally via npm:
```bash
npm install -g coticket-mcp
```

Or for Go users:
```bash
go install github.com/paletas/coticket/cmd/coticket-mcp@latest
```

### MCP Client Configuration
Add CoTicket to your MCP-compatible client (e.g., Claude Desktop, Cline) configuration:

```json
{
  "mcpServers": {
    "coticket": {
      "command": "coticket-mcp",
      "env": {
        "COTICKET_REPO_PATH": "/path/to/your/repository"
      }
    }
  }
}
```

For repository-specific isolation, use the generated launcher (see Required setup below).

## Required setup (portable across repositories)
Pick a database path inside the target repository so each repo has isolated state (example: `<repo>\.coticket\coticket.db`).

Recommended bootstrap:
```powershell
powershell -File D:\path\to\coticket\scripts\setup-repo.ps1 -RepoPath D:\path\to\target-repo
```

If running from a CoTicket checkout:
```powershell
$env:COTICKET_DB_PATH = "D:\path\to\target-repo\.coticket\coticket.db"
go run .\cmd\coticket-mcp
```

If running from a built binary:
```powershell
$env:COTICKET_DB_PATH = "D:\path\to\target-repo\.coticket\coticket.db"
coticket-mcp.exe
```

## Server-start gate (required)
Before any `ticket_*` tool call, ensure CoTicket MCP is running for the target repo.

Preferred start command (repo-local launcher):
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File D:\path\to\target-repo\.coticket\start-coticket-mcp.ps1
```

If launcher is missing, bootstrap first:
```powershell
powershell -File D:\path\to\coticket\scripts\setup-repo.ps1 -RepoPath D:\path\to\target-repo
```

## Standard workflow
1. **Start/verify server** using the required server-start gate above.
2. **Create work** with `ticket_create`
   - Required fields: `type` (`epic|story|bug`), `title`, `created_by`
3. **Pull next work** with `ticket_list_ready`
   - Use `limit` and optional `type` filter
4. **Take ownership** with `ticket_claim`
   - Provide `id` and `owner` (use `force=true` only for explicit reassignment)
5. **Execute and report progress** with `ticket_update_status`
   - Valid statuses: `todo`, `in_progress`, `blocked`, `done`
6. **Verify ticket state** with `ticket_get`

## Operational rules
- `ticket_list_ready` returns only unassigned `todo` tickets.
- Keep business logic in service-layer code, not in ad-hoc MCP caller logic.
- Use a stable owner identity (agent/session name) so ownership history is meaningful.
- Update status as work changes; do not leave completed work as `in_progress`.
- Always scope `COTICKET_DB_PATH` per repository to avoid cross-repo ticket contamination.
- Confirm startup logs show the expected repository scope and DB path before executing ticket operations.
- If CoTicket server is not active, start it first; never run ticket workflow steps without an active repo-scoped server.

## Repo-scoped setup checklist
- Run `scripts/setup-repo.ps1` once per target repository.
- Use the generated `<repo>\.coticket\start-coticket-mcp.ps1` launcher in MCP server config.
- Confirm startup logs print the expected `repo scope` and `database path`.
- Keep each repository on its own `.coticket\coticket.db` path.

## Quick examples
Create:
```json
{"name":"ticket_create","arguments":{"type":"story","title":"Add retry policy","description":"Handle transient failures","created_by":"agent-copilot"}}
```

Claim:
```json
{"name":"ticket_claim","arguments":{"id":"<ticket-id>","owner":"agent-copilot"}}
```

Move to done:
```json
{"name":"ticket_update_status","arguments":{"id":"<ticket-id>","status":"done","updated_by":"agent-copilot"}}
```

## Tool Reference

### ticket_create
Create a new ticket (epic, story, or bug).

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "enum": ["epic", "story", "bug"],
      "description": "Ticket type"
    },
    "title": {
      "type": "string",
      "description": "Ticket title (required)"
    },
    "description": {
      "type": "string",
      "description": "Ticket description (optional)"
    },
    "created_by": {
      "type": "string",
      "description": "Creator identity (agent/user name)"
    },
    "parent_id": {
      "type": "string",
      "description": "Parent epic ID for stories (optional)"
    }
  },
  "required": ["type", "title", "created_by"]
}
```

**Output:** Returns ticket ID, type, title, status (defaults to `todo`), and timestamps.

### ticket_get
Retrieve a ticket by ID.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Ticket UUID"
    }
  },
  "required": ["id"]
}
```

**Output:** Full ticket details including ID, type, title, description, status, assignee, parent_id, timestamps, and events history.

### ticket_claim
Claim ownership of a ticket or reassign to a different owner.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Ticket UUID"
    },
    "owner": {
      "type": "string",
      "description": "New owner identity"
    },
    "force": {
      "type": "boolean",
      "description": "Force reassignment if already owned (default: false)"
    }
  },
  "required": ["id", "owner"]
}
```

**Output:** Updated ticket with new assignee. Enforces claim rules unless `force=true`.

**Claim Rules:**
- Cannot claim already-owned ticket unless `force=true`
- Cannot claim tickets in `done` status

### ticket_update_status
Update ticket status through the lifecycle.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Ticket UUID"
    },
    "status": {
      "type": "string",
      "enum": ["todo", "in_progress", "blocked", "done"],
      "description": "New status"
    },
    "updated_by": {
      "type": "string",
      "description": "Who is making the update"
    }
  },
  "required": ["id", "status", "updated_by"]
}
```

**Output:** Updated ticket with new status and updated_at timestamp.

**Status Transition Rules:**
- `todo` can transition to: `in_progress`
- `in_progress` can transition to: `blocked`, `done`
- `blocked` can transition to: `in_progress`, `done`
- `done` is terminal (no transitions allowed from done)

### ticket_list_ready
List unassigned tickets in `todo` status, ordered by oldest `updated_at` first.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 200,
      "description": "Max tickets to return (default: 25)"
    },
    "type": {
      "type": "string",
      "enum": ["epic", "story", "bug"],
      "description": "Filter by ticket type (optional)"
    }
  },
  "required": []
}
```

**Output:** Array of tickets with ID, type, title, description, status (always `todo`), created_at, and updated_at. Returns empty array if no tickets match.
