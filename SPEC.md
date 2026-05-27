# lemon-chat specification

## What it does

lemon-chat is a browser-based chat interface for locally-running AI models. A Go HTTP server handles persistence (SQLite), model communication, and static file serving. Users interact through a clean, paper-and-ink UI with no cloud dependency. Configuration — models, server settings, and initial setup — lives in a TOML file.

---

## MVP

### In scope

- Go HTTP server serving the static chat UI
- Sidebar: list, create, and delete conversations
- Thread: display messages, stream AI responses via SSE
- Composer: send messages, Shift+Enter for newline
- Persist conversations and messages in SQLite
- Model list read from `lemon.toml`
- Basic markdown rendering: paragraphs, bold, inline code, fenced code blocks

### Out of scope for MVP

- Auth and user profiles (single anonymous user)
- Personas
- File and image attachments
- Model management in the UI (config file only)
- Conversation search
- Message editing and regeneration

---

## Architecture

```
Browser  ←→  Go HTTP server  ←→  Local model API (Ollama etc.)
                   ↕
                SQLite DB
```

- Go 1.22+ server listens on configurable port (default `8080`)
- Serves `static/` at `/`
- REST API under `/api/`
- Streaming AI responses delivered via SSE (`text/event-stream`)
- SQLite database at configurable path (default `lemon.db`)
- Model API calls proxied server-side (browser never talks to Ollama directly)

---

## Data model

```sql
CREATE TABLE users (
  id           INTEGER PRIMARY KEY,
  username     TEXT NOT NULL UNIQUE,
  password_hash TEXT,          -- nullable: no password = login by name only
  is_admin     INTEGER NOT NULL DEFAULT 0,
  created_at   TEXT NOT NULL
);

CREATE TABLE personas (
  id            INTEGER PRIMARY KEY,
  name          TEXT NOT NULL,
  description   TEXT,
  system_prompt TEXT NOT NULL,
  created_by    INTEGER REFERENCES users(id),
  is_global     INTEGER NOT NULL DEFAULT 0,  -- 1 = visible to all users
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL
);

CREATE TABLE conversations (
  id         INTEGER PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id),
  persona_id INTEGER REFERENCES personas(id),
  title      TEXT NOT NULL DEFAULT 'New conversation',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE messages (
  id              INTEGER PRIMARY KEY,
  conversation_id INTEGER NOT NULL REFERENCES conversations(id),
  role            TEXT NOT NULL,   -- 'user' | 'assistant'
  content         TEXT NOT NULL,
  created_at      TEXT NOT NULL
);
```

---

## API surface

### Auth

```
POST  /api/auth/login    { username, password? }  →  sets session cookie
POST  /api/auth/logout
GET   /api/auth/me       →  { id, username, is_admin }
```

### Conversations (scoped to authenticated user)

```
GET    /api/conversations
POST   /api/conversations          { title?, persona_id? }
DELETE /api/conversations/:id
```

### Messages

```
GET   /api/conversations/:id/messages
POST  /api/conversations/:id/messages   { content, model }
      → SSE stream: data: {"delta": "..."} … data: [DONE]
```

### Models

```
GET  /api/models   →  list from lemon.toml
```

### Personas

```
GET    /api/personas           →  global personas + current user's own
POST   /api/personas           { name, description, system_prompt, is_global? }
PATCH  /api/personas/:id       { name?, description?, system_prompt?, is_global? }
DELETE /api/personas/:id
```

`is_global` can only be set/changed by admins.

### Admin (requires `is_admin`)

```
GET    /api/admin/users
POST   /api/admin/users        { username, password?, is_admin? }
PATCH  /api/admin/users/:id    { username?, password?, is_admin? }
DELETE /api/admin/users/:id
```

---

## Configuration

### lemon.toml

```toml
[server]
port    = 8080
db_path = "lemon.db"

# Bootstrap: runs once on first start when the DB has no users.
# Ignored thereafter. Override with env vars for Docker Compose.
#   LEMON_ADMIN_USERNAME, LEMON_ADMIN_PASSWORD
[bootstrap]
admin_username = "admin"
admin_password = ""        # empty = no password required to log in

[[models]]
name         = "llama3.2"
display_name = "Llama 3.2"
api_url      = "http://localhost:11434/api/chat"
default      = true

[[models]]
name         = "mistral"
display_name = "Mistral 7B"
api_url      = "http://localhost:11434/api/chat"
```

### Environment variable overrides

| Variable               | Overrides                     |
|------------------------|-------------------------------|
| `LEMON_ADMIN_USERNAME` | `bootstrap.admin_username`    |
| `LEMON_ADMIN_PASSWORD` | `bootstrap.admin_password`    |
| `LEMON_PORT`           | `server.port`                 |
| `LEMON_DB_PATH`        | `server.db_path`              |

Env vars take precedence over TOML values.

---

## Future / post-MVP

- Search across conversations
- File and image attachments
- Keyboard shortcuts
- Per-conversation title editing
- Conversation export
