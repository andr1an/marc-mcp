This project is an MCP to access https://marc.info/ mailing list archive.

Language: Go latest (1.25)

Uses Nix flake, all commands for launching tools should be prefixed by `nix develop --command`.

## Spec

**Transport:** Streaming HTTP (SSE)
**Data source:** HTML scraping (marc.info has no RSS/API)
**Scope:** All mailing lists

### Tools

1. `list_mailing_lists` - Browse available mailing lists by category
2. `list_messages` - List messages in a mailing list (with pagination by month)
3. `get_message` - Fetch full message content
4. `search_messages` - Search messages by subject, author, or body

### Dependencies

- `github.com/mark3labs/mcp-go` - MCP SDK for Go
- `golang.org/x/net/html` - HTML parsing
