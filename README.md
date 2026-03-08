# marc-mcp

MCP server for accessing [marc.info](https://marc.info/) mailing list archives.

## Features

- Streamable HTTP MCP transport (stateless, SSE-capable)
- Browse mailing lists by category and/or regex filter
- List messages by month with page + per-page limit controls
- Fetch full message content (headers + body)
- Search within a list by subject, author, or body
- Built-in SQLite cache with TTL for scraped results
- Automatic retry with backoff for transient upstream errors
- Optional JWT bearer-token authentication

## Architecture

```text
main.go
internal/
  auth/
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

## Development

```bash
make test
make build
```

Available targets:

- `make test` - run unit tests
- `make build` - build local binary `./marc-mcp`
- `make build-all` - cross-build `marc-mcp-{os}-{arch}` binaries
- `make print-version` - print linker version variables
- `make clean` - remove generated binaries

## Run

```bash
go run .
```

Server defaults to `:8080`. `make build` produces a `marc-mcp` binary in the project root.

## Configuration

| Variable | Description | Default |
|---|---|---|
| `LISTEN_ADDR` | Server listen address | `:8080` |
| `AUTH_MODE` | `disabled` or `jwt` | `disabled` |
| `JWT_PUBLIC_KEY` | RSA public key path for JWT validation | (empty) |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `MARC_TIMEOUT` | HTTP timeout for marc.info requests | `2m` |
| `MARC_CACHE_DB` | Custom SQLite cache path | OS user cache dir + `/marc-mcp/cache.db` |
| `MARC_CACHE_TTL` | Cache TTL (Go duration) | `24h` |
| `READ_TIMEOUT` | HTTP read timeout | `15s` |
| `WRITE_TIMEOUT` | HTTP write timeout | `60s` |
| `IDLE_TIMEOUT` | HTTP idle timeout | `60s` |
| `SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | `10s` |
| `MAX_HEADER_BYTES` | Max HTTP header bytes | `1048576` |

`MARC_TIMEOUT` valid range is 10s to 15m.

## Authentication (Optional)

Bearer token protection is supported with JWT validation.

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
- MCP Streamable HTTP transport at `/mcp`

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
make test
```

## License

MIT
