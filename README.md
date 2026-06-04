# lemon-chat

A local, open-source AI chat UI. Think self-hosted claude.ai or ChatGPT — minimal, fast, no telemetry.

Talks to locally-running models via HTTP API (tested with [Ollama](https://ollama.com)). Persists conversations in a local SQLite database.

## Features

**Chat** — multi-conversation chat with locally-running models. Responses stream via SSE. Messages render markdown, code blocks, and math (KaTeX). Conversations get auto-generated titles in the background, which can also be regenerated on demand. Conversations can be forked at any point.

**Completions** — a separate raw-text completion mode for open-ended generation. Write a prompt, run it against a model, and iterate with undo/redo. Completions are saved and titled the same way as conversations.

**Characters** — define AI personas with a system prompt, a first message, hidden context messages, a custom avatar, and per-character model and title-generation settings. Characters can be private or shared with all users. SillyTavern character card PNG files can be imported directly.

**Multi-user** — admin users can create accounts, reset passwords, and grant or revoke admin access. Each user's conversations and completions are private to them.

## Requirements

- Go 1.26+ **or** Docker
- A running Ollama instance (or any OpenAI-compatible API)

## Quick start

### With Docker

```sh
mkdir -p data
cp lemon.toml.example data/lemon.toml
# edit data/lemon.toml — set your model API endpoint and credentials
cp docker-compose.override.yml.example docker-compose.override.yml
# edit docker-compose.override.yml as needed
docker compose up
```

### Without Docker

```sh
cp lemon.toml.example lemon.toml
# edit lemon.toml — set your model API endpoint and credentials
go run ./cmd/lemon-chat
```

Then open `http://localhost:8080` and log in with the admin credentials from your config.

## Docker details

The container expects a `lemon.toml` config file at `/data/lemon.toml` and writes the SQLite database to `/data/lemon.db`. Mount a directory or named volume at `/data` to persist both.

`docker-compose.override.yml.example` documents common customisations — bind-mounting the data directory for easy config editing, restricting the port to localhost, and pointing at an Ollama instance running on the host machine.

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

Apache 2.0
