# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-17

### Added
- Initial MCP server implementation for ticket lifecycle management
- SQLite-based persistent storage with automatic schema migration
- Five core tools: `ticket_create`, `ticket_get`, `ticket_claim`, `ticket_update_status`, `ticket_list_ready`
- Support for epic, story, and bug ticket types
- Status transitions: todo → in_progress → blocked → done
- Ticket ownership claim rules with optional force override
- Repo-scoped database isolation via `COTICKET_REPO_PATH`
- PowerShell setup script for per-repo launcher generation
- Comprehensive lifecycle validation and business rules
- Event-sourced ticket history in `ticket_events` table
- MCP protocol version 2024-11-05 compliance
- MIT License
- npm package distribution with cross-platform binary downloads
- GitHub Actions release automation for Windows, macOS (Intel/ARM), and Linux (Intel/ARM)
- SKILL documentation for CoTicket ticket lifecycle workflow

[0.1.0]: https://github.com/paletas/coticket/releases/tag/v0.1.0
