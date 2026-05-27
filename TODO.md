# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

---

## Project scaffolding

- [ ] `go mod init` with module path
- [ ] Create directory structure (`cmd/`, `internal/config`, `internal/server`, `internal/store`, `static/js`, `static/css`, `static/assets`)
- [ ] Copy design system CSS (`colors_and_type.css`, `components.css`) into `static/css/`
- [ ] Copy brand SVGs into `static/assets/`
- [ ] `lemon.toml.example` with all documented fields
- [ ] Add `lemon.toml` and `lemon.db` to `.gitignore`

---

## Backend — config (`internal/config`)

- [ ] TOML struct definitions (`Server`, `Bootstrap`, `Model`)
- [ ] Load config from file path
- [ ] Override `server.port` from `LEMON_PORT` env var
- [ ] Override `server.db_path` from `LEMON_DB_PATH` env var
- [ ] Override `bootstrap.admin_username` from `LEMON_ADMIN_USERNAME` env var
- [ ] Override `bootstrap.admin_password` from `LEMON_ADMIN_PASSWORD` env var

---

## Backend — store (`internal/store`)

- [ ] Open SQLite connection (`modernc.org/sqlite`)
- [ ] Run migrations (create tables if not exist)
  - [ ] `users` table
  - [ ] `personas` table
  - [ ] `conversations` table
  - [ ] `messages` table
- [ ] Bootstrap: create initial admin user on empty DB
- [ ] User queries: get by id, get by username, create, update, delete, list
- [ ] Conversation queries: list by user, get by id, create, delete
- [ ] Message queries: list by conversation, create
- [ ] Persona queries: list (global + by user), get by id, create, update, delete

---

## Backend — server (`internal/server`)

### Infrastructure
- [ ] HTTP router setup (stdlib `net/http` with Go 1.22 method+path routing)
- [ ] Serve `static/` at `/`
- [ ] Session middleware (cookie-based, server-side sessions in SQLite or in-memory)
- [ ] Auth middleware (require valid session for `/api/` routes)
- [ ] Admin middleware (require `is_admin` for `/api/admin/` routes)
- [ ] JSON helper (write response, handle errors consistently)

### Auth handlers
- [ ] `POST /api/auth/login`
- [ ] `POST /api/auth/logout`
- [ ] `GET  /api/auth/me`

### Conversation handlers
- [ ] `GET    /api/conversations`
- [ ] `POST   /api/conversations`
- [ ] `DELETE /api/conversations/:id`

### Message handlers
- [ ] `GET  /api/conversations/:id/messages`
- [ ] `POST /api/conversations/:id/messages` — proxy to model API, stream via SSE

### Model handler
- [ ] `GET /api/models` — return model list from config

### Persona handlers
- [ ] `GET    /api/personas`
- [ ] `POST   /api/personas`
- [ ] `PATCH  /api/personas/:id`
- [ ] `DELETE /api/personas/:id`

### Admin handlers
- [ ] `GET    /api/admin/users`
- [ ] `POST   /api/admin/users`
- [ ] `PATCH  /api/admin/users/:id`
- [ ] `DELETE /api/admin/users/:id`

---

## Backend — entry point (`cmd/lemon-chat/main.go`)

- [ ] Parse config file path from flag/env
- [ ] Load config
- [ ] Open store
- [ ] Run bootstrap
- [ ] Start HTTP server

---

## Frontend — shell

- [ ] `static/index.html` — app shell, imports CSS + `app.js` as module
- [ ] `static/css/app.css` — layout and app-specific styles (sidebar + thread split)
- [ ] Login screen (shown when no active session)

---

## Frontend — JS modules

- [ ] `static/js/api.js` — typed fetch wrappers for every API endpoint, SSE helper
- [ ] `static/js/app.js` — entry point, initialises modules, manages top-level state
- [ ] `static/js/sidebar.js` — conversation list, new chat, delete
- [ ] `static/js/thread.js` — message list render, SSE stream handling, scroll behaviour
- [ ] `static/js/composer.js` — textarea, send on Enter, Shift+Enter newline, disable during stream
- [ ] `static/js/markdown.js` — render paragraphs, `**bold**`, `` `inline code` ``, fenced code blocks

---

## Post-MVP (parked, not forgotten)

- [ ] User profiles — login screen, profile switcher
- [ ] Personas — create/edit/delete in UI, apply to conversation
- [ ] Admin panel — user management, global persona management
- [ ] Conversation search
- [ ] Per-conversation title editing
- [ ] File and image attachments
- [ ] Keyboard shortcuts
- [ ] Conversation export
- [ ] Docker Compose setup
