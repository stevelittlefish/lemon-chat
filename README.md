# lemon-chat

A local, open-source AI chat UI. Think self-hosted claude.ai or ChatGPT — minimal, fast, no telemetry.

Talks to locally-running models via HTTP API (tested with [Ollama](https://ollama.com)). Persists conversations in a local SQLite database.

## Requirements

- Go 1.22+
- A running Ollama instance (or any OpenAI-compatible API)

## Quick start

```sh
cp lemon.toml.example lemon.toml
# edit lemon.toml — set your model API endpoint and credentials
go run ./cmd/lemon-chat
```

Then open `http://localhost:8080` and log in with the admin credentials from your config.

## Configuration

All configuration lives in `lemon.toml`. See `lemon.toml.example` for the full reference. The following environment variables override config values at runtime:

| Variable | Overrides |
|---|---|
| `LEMON_PORT` | `server.port` |
| `LEMON_DB_PATH` | `server.db_path` |
| `LEMON_ADMIN_USERNAME` | `bootstrap.admin_username` |
| `LEMON_ADMIN_PASSWORD` | `bootstrap.admin_password` |

## Stack

- **Backend:** Go (stdlib HTTP, no CGo)
- **Database:** SQLite via `modernc.org/sqlite`
- **Frontend:** vanilla JS ES modules, no build step
- **Streaming:** SSE for message streaming, WebSocket for sidebar updates (e.g. auto-titles)

## License

MIT
