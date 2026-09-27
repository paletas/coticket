# CoTicket

CoTicket is an MCP-first ticket lifecycle service for AI agents. It stores durable work items in SQLite so agents can create work, claim ownership, and update status across sessions.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.23-blue.svg)](https://golang.org)
[![MCP Protocol](https://img.shields.io/badge/MCP-2024--11--05-green.svg)](https://modelcontextprotocol.io)

## Features
- **MCP-native**: Built specifically for Model Context Protocol integration
- **Durable storage**: SQLite-backed persistence across AI agent sessions
- **Repo-scoped**: Isolated ticket databases per repository
- **Lifecycle management**: Full ticket workflow from creation to completion
- **Event sourcing**: Complete audit trail of ticket changes
- **Cross-platform**: Supports Windows, macOS (Intel/ARM), and Linux (Intel/ARM)

## Installation

### Via npm (recommended for MCP clients)
```bash
npm install -g coticket-mcp
```

### Via Go
```bash
go install github.com/paletas/coticket/cmd/coticket-mcp@latest
```

### From source
```bash
git clone https://github.com/paletas/coticket.git
cd coticket
go build ./cmd/coticket-mcp
```

## Quick Start

### 1. Install CoTicket
```bash
npm install -g coticket-mcp
```

### 2. Configure your MCP client
Add to your MCP client configuration (e.g., Claude Desktop, Cline):

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

### 3. Use with AI agents
Refer to the [CoTicket Ticket Lifecycle SKILL](.github/skills/coticket-ticket-lifecycle/SKILL.md) for comprehensive workflow guidance.

## Current scope (v0.1.0)
- **Ticket types**: `epic`, `story`, `bug`
- **Lifecycle operations**: create, get, claim, update status, list ready
- **Durable state**: SQLite database at `<repo>\.coticket\coticket.db` by default
- **Status workflow**: `todo` → `in_progress` → `blocked` → `done`

## Configuration

### Environment Variables
- `COTICKET_REPO_PATH` - Target repository path (default: current working directory)
- `COTICKET_DATA_DIR` - Data directory within repo (default: `<repo>\.coticket`)
- `COTICKET_DB_PATH` - Explicit DB path override (must be within repo scope)

### Command-Line Flags
- `-repo-path` - Target repository path used for DB scoping
- `-db-path` - Explicit DB path override (validated to stay inside repo scope)

### Per-Repository Setup Helper
For convenience, generate a repo-specific launcher and MCP config:

```powershell
powershell -File .\scripts\setup-repo.ps1 -RepoPath D:\path\to\target-repo
```

This creates:
- `<repo>\.coticket\start-coticket-mcp.ps1` - Launcher with repo-scoped environment wiring
- `<repo>\.coticket\mcp.coticket.example.json` - Example MCP server configuration

On startup, the server prints the effective repository scope and database path, and logs each MCP action/tool call outcome to the command output.

## MCP Tools

### ticket_create
Create a new ticket (epic, story, or bug).
- **Required**: `type`, `title`, `created_by`
- **Optional**: `description`, `parent_id`

### ticket_get
Retrieve a ticket by ID with full details and event history.
- **Required**: `id`

### ticket_claim
Claim or reassign ticket ownership.
- **Required**: `id`, `owner`
- **Optional**: `force` (boolean, default: false)

### ticket_update_status
Update ticket status through the lifecycle.
- **Required**: `id`, `status`, `updated_by`
- **Valid statuses**: `todo`, `in_progress`, `blocked`, `done`

### ticket_list_ready
List unassigned tickets in `todo` status, ordered by oldest `updated_at`.
- **Optional**: `limit` (1-200, default: 25), `type` (filter by ticket type)

See the [SKILL documentation](.github/skills/coticket-ticket-lifecycle/SKILL.md) for complete tool schemas and usage examples.

## Development

### Run from source
```powershell
go run ./cmd/coticket-mcp
```

### Build binary
```powershell
go build -ldflags "-X main.version=$(Get-Content VERSION)" ./cmd/coticket-mcp
```

### Run tests
Full test suite:
```powershell
go test ./...
```

Single test:
```powershell
go test ./internal/service -run TestClaimTicketOwnershipRules -v
```

### Format code
```powershell
Get-ChildItem -Path . -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
```

## Agent Skill Documentation
For AI agents using CoTicket, refer to the comprehensive workflow guide:
- **[CoTicket Ticket Lifecycle SKILL](.github/skills/coticket-ticket-lifecycle/SKILL.md)**

This SKILL provides:
- Installation and setup instructions
- Server-start gate enforcement
- Standard workflow (create → list → claim → update → verify)
- Complete tool reference with JSON schemas
- Operational rules and best practices

## Architecture
- **Entry point**: [cmd/coticket-mcp/main.go](cmd/coticket-mcp/main.go) - Loads config, initializes storage, starts MCP server
- **MCP layer**: [internal/mcp](internal/mcp) - JSON-RPC protocol handling and tool routing
- **Business logic**: [internal/service](internal/service) - Lifecycle rules, validation, status transitions
- **Persistence**: [internal/store](internal/store) - SQLite storage with schema auto-bootstrap
- **Domain model**: [internal/domain](internal/domain) - Canonical types, statuses, transition rules

## Contributing
Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## License
MIT License - see [LICENSE](LICENSE) for details.

## Links
- **GitHub**: https://github.com/paletas/coticket
- **Issues**: https://github.com/paletas/coticket/issues
- **CHANGELOG**: [CHANGELOG.md](CHANGELOG.md)


