# lemon-chat — Claude Code guide

## What this project is

lemon-chat is a local, open-source AI chat UI. Think self-hosted claude.ai or ChatGPT. It talks to locally-running models via HTTP API (e.g. Ollama) and persists everything in a local SQLite database.

See `SPEC.md` for the full feature specification.

## Technical constraints

These apply to every change. Don't add packages, build steps, or abstractions without a clear reason.

- **Backend language:** Go
- **Dependencies:** minimal — justify every new package
- **Frontend:** no build step, no bundler, no transpilation
- **JS framework:** vanilla JS with native ES modules (`<script type="module">`) — no React, no Vue, no Svelte
- **Model/server config:** TOML files only, not the UI
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGo required)
- **Database table names:** singular (`user`, `message`, `conversation` — not `users`, `messages`, etc.)
- **TOML parsing:** `github.com/BurntSushi/toml`

## Frontend conventions

- Use `<script type="module">` — browsers resolve imports natively, no bundler needed
- Split JS into focused files, one concern per file (see layout below)
- Render lists with template literals; update streaming text with direct DOM mutation
- No CSS frameworks — use the design system tokens (see below)

## Design system

The design system lives at `.claude/skills/lemon-ai-design/`. Read it before touching any UI.

Key files:
- `colors_and_type.css` — all colour, type, spacing, and motion tokens
- `components.css` — buttons, inputs, cards, bubbles, toggles
- `assets/` — brand SVGs (logo, lemon-slice, sparkle, scribble-underline)
- `ui_kits/chat/` — reference implementation for the chat UI
- `ui_kits/settings/` — reference implementation for the settings pane

Non-negotiable design rules:
- Paper backgrounds, never pure white (`--bg` from tokens)
- Warm near-black ink (`--fg`), never `#000`
- One lemon yellow element per viewport maximum
- Sentence case everywhere; eyebrow labels in small mono caps
- No emoji in chrome; no exclamation points outside genuine errors
- Lucide icons (1.8px stroke) for UI; hand-drawn SVGs for brand decoration only
- No glass/backdrop-blur, no bluish-purple gradients, no left-border-accent cards

## Icons

Icons are Lucide SVGs served from `static/assets/icons/` and loaded by `static/js/icons.js`.

**Whenever you use a new icon name, you must add it to the `ICONS` array in `icons.js`.** Icons not in that list are preloaded at startup; any name missing from the list will return an empty string from `icon()` and render as nothing.

## Code layout

```
cmd/lemon-chat/
  main.go              # entry point, config loading, server start
internal/
  config/
    config.go          # TOML struct definitions and loading
  server/
    server.go          # router setup, static serving
    auth.go            # login / logout / me handlers
    conversations.go   # conversation CRUD handlers
    messages.go        # message list + SSE streaming handler
    models.go          # model list handler
    characters.go      # character CRUD handlers
    admin.go           # admin user-management handlers
    middleware.go      # session, auth, admin middleware
    ws.go              # WebSocket hub, upgrade handler, broadcast
  store/
    store.go           # open DB, run migrations, bootstrap
    users.go           # user queries
    sessions.go        # session queries
    conversations.go   # conversation queries
    messages.go        # message queries
    characters.go      # character queries
  tasks/
    titles.go          # background title-generation worker
static/
  index.html           # main app shell
  settings.html        # settings page shell
  js/
    app.js             # entry, wires modules together
    api.js             # fetch wrappers for the REST API
    sidebar.js         # conversation list, navigation
    thread.js          # message display, SSE streaming
    composer.js        # text input, send logic
    markdown.js        # lightweight message rendering
    header.js          # chat header: title display + model/character picker
    icons.js           # SVG icon loader/cache (fetches from /assets/icons/)
    settings.js        # settings page: account, characters, admin panels
    ws.js              # WebSocket client, auto-reconnect, event dispatch
    vendor/
      marked.esm.js    # marked.js (vendored, no build step)
  css/
    colors_and_type.css   # copied from design system
    components.css        # copied from design system
    app.css               # layout and app-specific overrides
    settings.css          # settings page layout and overrides
  assets/
    icons/             # Lucide SVGs served individually (fetched by icons.js)
    *.svg              # brand SVGs copied from design system
data/
  lemon.toml           # config for Docker volume mount
Dockerfile
docker-compose.yml
docker-compose.override.yml.example
run.sh                 # shortcut: go run ./cmd/lemon-chat
lemon.toml.example     # documented config template (committed)
lemon.toml             # live config (gitignored)
README.md
LICENSE
SPEC.md
CLAUDE.md
```

## Not yet implemented

The following are in `SPEC.md` but not built. Stub them out rather than building them:

- User profiles / profile switcher — auth is done; only one active user at a time, no switcher UI
- Personas — API endpoints stubbed (`501 Not Implemented`), no UI
- File attachments
- Model management UI — config file only, no settings panel for it
- Message editing and regeneration
- Conversation search

When something is stubbed, return a `501 Not Implemented` from the API endpoint and leave a `// TODO` comment. Don't build placeholder UI for features that don't exist yet.

## Database schema changes

All migrations live in `internal/store/store.go` in the `migrate()` function. You should be able to work out the current schema version by looking in that file.

### Adding a new table

Add it to the v0→v1 block (the initial schema). New tables don't need their own migration version — they'll be created on first run for fresh databases, and existing databases already have the full schema.

### Modifying an existing table or any other schema change

Add a new numbered migration block after the last one. The pattern:

```go
if version < N {
    log.Println("store: migrating vN-1 → vN (short description)")
    // ALTER TABLE ..., CREATE INDEX ..., etc.
    if _, err := s.db.Exec(`...`); err != nil {
        return err
    }
    // If you need a timestamp or other Go value, use a second Exec with ? params.
    if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (N, ?)`, now()); err != nil {
        return err
    }
    version = N
    log.Println("store: migration vN-1 → vN complete")
}
```

- Increment the version number (next is **4**)
- Always insert a row into `schema_version` with the new version and `now()` as the timestamp
- Ask the user before doing anything destructive (dropping columns, dropping tables, data transforms)
- Never delete or edit old migrations unless the user has specifically requested it

## Logging conventions

Use Go's standard `log` package throughout the backend. Follow these rules:

**Single log line for non-frequent request handlers.** Any handler that isn't called on a hot path should emit one `log.Printf` line when it runs. This includes: creating or deleting a conversation, sending a message, creating/updating/deleting a character, login/logout, any admin action, forking a conversation. Do not log: individual SSE token chunks, WebSocket pings, static file serving, or the `/api/models` poll.

Start with a capital letter and use a gerund (present participle). No "server:" prefix — it's redundant. Key IDs and usernames follow inline:
```
Deleting conversation id=12 user_id=3
Creating conversation user_id=3
Sending message conversation_id=5 user_id=3
Login attempt username="alice"
```

**Verbose logging for infrequent tools and admin operations.** Any endpoint or store function that runs rarely (admin tools, migrations, one-off maintenance) should log every significant step — what it found, what it's about to do, and what it did. Individual records being affected should each get their own log line.

Format: `<package>: <function> — <detail>` — e.g.:
```
store: DeleteOrphanedMessages — found 3 orphaned message(s)
store:   message id=42 conversation_id=7 role=assistant content="..."
store: DeleteOrphanedMessages — deleted 3 message(s) successfully
```

## Keeping TODO.md current

Update `TODO.md` as work progresses — mark items `[~]` when started and `[x]` when done. When adding new work that isn't already listed, add it to the relevant section before starting. Don't leave TODO.md stale.
