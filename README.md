# marc-mcp

MCP server for accessing [marc.info](https://marc.info/) mailing list archives.

## Features

- Browse mailing lists by category or regex filter
- List messages with pagination
- Fetch full message content
- Search by subject, author, or body
- Built-in caching

## Requirements

- Go 1.25+
- Nix (optional, for development environment)

## Installation

```bash
go install github.com/andr1an/marc-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/andr1an/marc-mcp
cd marc-mcp
go build
```

With Nix:

```bash
nix develop --command go build
```

## Usage

Start the server:

```bash
./marc-mcp
```

The server listens on `:8080` by default (streamable HTTP transport).

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_ADDR` | Server listen address | `:8080` |
| `MARC_TIMEOUT` | HTTP timeout for marc.info requests | `60s` |
| `DEBUG` | Enable debug logging | (disabled) |

`MARC_TIMEOUT` accepts Go duration strings (e.g., `30s`, `2m`). Valid range: 10s to 15m.

## Tools

### list_mailing_lists

List all available mailing lists, optionally filtered by category and/or name regex.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `category` | No | Filter by category (e.g., "Development", "Linux", "Security") |
| `filter` | No | Filter list names by regular expression (e.g., `^linux`, `git.*`, `kernel`) |

### list_messages

List messages from a mailing list with pagination.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `list` | Yes | Mailing list name (e.g., "git", "linux-kernel") |
| `month` | No | Month in YYYYMM format (default: current month) |
| `page` | No | Page number, 1-based (default: 1) |
| `limit` | No | Max messages to return (default: all) |

### get_message

Fetch full content of a specific message.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `list` | Yes | Mailing list name |
| `message_id` | Yes | Message ID from list_messages |

### search_messages

Search messages in a mailing list.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `list` | Yes | Mailing list name |
| `query` | Yes | Search query |
| `search_type` | No | `s` = subject (default), `a` = author, `b` = body |

## MCP Client Configuration

### Claude Code

Add to `~/.claude/claude_code_config.json`:

```json
{
  "mcpServers": {
    "marc": {
      "type": "http",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

## License

MIT
