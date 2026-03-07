# marc-mcp

MCP server for accessing [marc.info](https://marc.info/) mailing list archives.

## Features

- Streamable HTTP MCP transport (SSE-capable)
- Browse mailing lists by category or regex filter
- List messages with month pagination
- Fetch full message content
- Search by subject, author, or body
- Built-in SQLite cache for scraped results

## Architecture

```text
main.go
internal/
  cache/
  config/
  httpserver/
  marc/
  middleware/
  tools/
  transport/
```

- `internal/marc`: HTML scraper client for marc.info
- `internal/cache`: cache + FTS schema
- `internal/tools`: MCP tool implementations
- `internal/transport`: Streamable HTTP MCP handler
- `internal/httpserver`: route wiring (`/health`, `/mcp`)

## Requirements

- Go 1.25+

## Run

```bash
go run .
```

Server defaults to `:8080`.

## Configuration

| Variable | Description | Default |
|---|---|---|
| `LISTEN_ADDR` | Server listen address | `:8080` |
| `AUTH_MODE` | `disabled` or `jwt` | `disabled` |
| `JWT_PUBLIC_KEY` | RSA public key path for JWT validation | (empty) |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `MARC_TIMEOUT` | HTTP timeout for marc.info requests | `60s` |
| `MARC_CACHE_DB` | Custom SQLite cache path | OS user cache dir |
| `MARC_CACHE_TTL` | Cache TTL (Go duration) | `24h` |
| `READ_TIMEOUT` | HTTP read timeout | `15s` |
| `WRITE_TIMEOUT` | HTTP write timeout | `60s` |
| `IDLE_TIMEOUT` | HTTP idle timeout | `60s` |
| `SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | `10s` |
| `MAX_HEADER_BYTES` | Max HTTP header bytes | `1048576` |

`MARC_TIMEOUT` valid range is 10s to 15m.

## Authentication (Optional)

OAuth-style bearer token protection is supported with JWT validation.

Enable with:

```bash
AUTH_MODE=jwt
JWT_PUBLIC_KEY=public.pem
```

Requests must include:

```text
Authorization: Bearer <token>
```

## Endpoints

- `GET /health`
- MCP transport at `POST /mcp`

## Tools

### `list_mailing_lists`

List all available mailing lists, optionally filtered by category and/or name regex.

Parameters:
- `category` (optional)
- `filter` (optional, regex)

### `list_messages`

List messages from a mailing list with pagination.

Parameters:
- `list` (required)
- `month` (optional, `YYYYMM`, default current month)
- `page` (optional, 1-based, default `1`)
- `limit` (optional)

### `get_message`

Fetch full content of a specific message.

Parameters:
- `list` (required)
- `message_id` (required)

### `search_messages`

Search messages in a mailing list.

Parameters:
- `list` (required)
- `query` (required)
- `search_type` (optional: `s` subject, `a` author, `b` body; default `s`)

## Tests

```bash
go test ./...
```

## License

MIT
