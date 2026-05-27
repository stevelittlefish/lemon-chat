# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

---

## Project scaffolding

- [x] `go mod init` with module path
- [x] Create directory structure (`cmd/`, `internal/config`, `internal/server`, `internal/store`, `static/js`, `static/css`, `static/assets`)
- [x] Copy design system CSS (`colors_and_type.css`, `components.css`) into `static/css/`
- [x] Copy brand SVGs into `static/assets/`
- [x] `lemon.toml.example` with all documented fields
- [x] Add `lemon.toml` and `lemon.db` to `.gitignore`

---

## Backend — config (`internal/config`)

- [x] TOML struct definitions (`Server`, `Bootstrap`, `Model`)
- [x] Load config from file path
- [x] Override `server.port` from `LEMON_PORT` env var
- [x] Override `server.db_path` from `LEMON_DB_PATH` env var
- [x] Override `bootstrap.admin_username` from `LEMON_ADMIN_USERNAME` env var
- [x] Override `bootstrap.admin_password` from `LEMON_ADMIN_PASSWORD` env var

---

## Backend — store (`internal/store`)

- [x] Open SQLite connection (`modernc.org/sqlite`)
- [x] Run migrations (create tables if not exist)
  - [x] `users` table
  - [x] `personas` table
  - [x] `conversations` table
  - [x] `messages` table
  - [x] `sessions` table
- [x] Bootstrap: create initial admin user on empty DB
- [x] User queries: get by id, get by username, create, update, delete, list
- [x] Session queries: create, lookup (with expiry check), delete
- [x] Conversation queries: list by user, get by id, create, delete, touch (update timestamp)
- [x] Message queries: list by conversation, create
- [x] Persona queries: list (global + by user), get by id, create, update, delete

---

## Backend — server (`internal/server`)

### Infrastructure
- [x] HTTP router setup (stdlib `net/http` with Go 1.22 method+path routing)
- [x] Serve `static/` at `/`
- [x] Session middleware (cookie-based, server-side sessions in SQLite)
- [x] Auth middleware (require valid session for `/api/` routes)
- [x] Admin middleware (require `is_admin` for `/api/admin/` routes)
- [x] JSON helper (write response, handle errors consistently)

### Auth handlers
- [x] `POST /api/auth/login`
- [x] `POST /api/auth/logout`
- [x] `GET  /api/auth/me`

### Conversation handlers
- [x] `GET    /api/conversations`
- [x] `POST   /api/conversations`
- [x] `DELETE /api/conversations/:id`

### Message handlers
- [x] `GET  /api/conversations/:id/messages`
- [x] `POST /api/conversations/:id/messages` — proxy to model API, stream via SSE

### Model handler
- [x] `GET /api/models` — return model list from config

### Persona handlers
- [x] `GET    /api/personas`
- [x] `POST   /api/personas`
- [x] `PATCH  /api/personas/:id`
- [x] `DELETE /api/personas/:id`

### Admin handlers
- [x] `GET    /api/admin/users`
- [x] `POST   /api/admin/users`
- [x] `PATCH  /api/admin/users/:id`
- [x] `DELETE /api/admin/users/:id`

---

## Backend — entry point (`cmd/lemon-chat/main.go`)

- [x] Parse config file path from flag/env
- [x] Load config
- [x] Open store
- [x] Run bootstrap
- [x] Start HTTP server

---

## Frontend — shell

- [x] `static/index.html` — app shell, imports CSS + `app.js` as module
- [x] `static/css/app.css` — layout and app-specific styles (sidebar + thread split)
- [x] Login screen (shown when no active session)

---

## Frontend — JS modules

- [x] `static/js/api.js` — typed fetch wrappers for every API endpoint, SSE helper
- [x] `static/js/app.js` — entry point, initialises modules, manages top-level state
- [x] `static/js/sidebar.js` — conversation list, new chat, delete
- [x] `static/js/thread.js` — message list render, SSE stream handling, scroll behaviour
- [x] `static/js/composer.js` — textarea, send on Enter, Shift+Enter newline, disable during stream
- [x] `static/js/markdown.js` — render paragraphs, `**bold**`, `` `inline code` ``, fenced code blocks

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
