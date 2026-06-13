# lemon-chat — Claude Code guide

## What this project is

lemon-chat is a local, open-source AI chat UI. Think self-hosted claude.ai or ChatGPT. It talks to locally-running models via HTTP API (e.g. Ollama) and persists everything in a local SQLite database.


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

Tokens live in `static/css/colors_and_type.css`; component styles in `static/css/components.css`. Read these files before touching any UI.

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

## Model modes

Each `[[model]]` in `lemon.toml` has an optional `modes` array that restricts which interface it appears in:

```toml
[[model]]
name         = "llama3.2"
model_server = "local"
modes        = ["chat"]     # only appears in chat, not completions
```

Valid values are `"chat"` and `"complete"`. Omit `modes` entirely to allow the model in both. The frontend fetches `GET /api/models?mode=chat` or `?mode=complete` and only shows the filtered list. When adding a new interface or model picker, pass the correct `mode` parameter.

## Character variable substitution

Character system prompts and first messages support two placeholders that are substituted at runtime:

- `{{char}}` — replaced with the character's name
- `{{user}}` — replaced with the logged-in user's display name

Apply this substitution wherever character text is rendered or sent to the model. The substitution logic lives in `internal/server/messages.go`.

## Code layout

```
cmd/lemon-chat/
  main.go              # entry point, config loading, server start
internal/
  config/
    config.go          # TOML struct definitions and loading
  debug/
    debug.go           # debug logging flag (debug.Enabled, debug.Log)
  research/
    researcher.go      # research engine: phase loop, state, checkpointing
    prompts.go         # all research prompts (ported from docs/deep_research_spec.html)
    llm.go             # LLM calls, think-tag stripping, robust JSON parsing, injection guard
    web.go             # SearXNG search, page fetch, goal-based extraction
  server/
    server.go          # router setup, static serving
    auth.go            # login / logout / me / profile / password handlers
    avatars.go         # avatar upload and serve handlers (users + characters)
    conversations.go   # conversation CRUD handlers
    messages.go        # message list + SSE streaming handler
    models.go          # model list handler
    characters.go      # character CRUD handlers
    completions.go     # completions CRUD + streaming handlers
    research.go        # research job manager, handlers, SSE progress, crash recovery
    tools.go           # tool registry, executors, attachment helpers
    attachments.go     # attachment serve handler
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
    completions.go     # completion queries
    research.go        # research job queries (incl. resume checkpoints)
    attachments.go     # attachment queries
  tasks/
    titles.go          # background title-generation worker
    cleanup.go         # background stale-conversation cleanup worker
static/
  index.html           # main chat app shell
  complete.html        # completions app shell
  research.html        # research page shell
  menu.html            # mobile/navigation menu shell
  js/
    app.js             # entry, wires modules together
    api.js             # fetch wrappers for the REST API
    sidebar.js         # conversation list, navigation
    thread.js          # message display, SSE streaming
    composer.js        # text input, send logic
    markdown.js        # lightweight message rendering
    header.js          # chat header: title display + model/character picker
    icons.js           # SVG icon loader/cache (fetches from /assets/icons/)
    utils.js           # shared frontend utilities (e.g. escapeHtml)
    ws.js              # WebSocket client, auto-reconnect, event dispatch
    complete-app.js    # completions page entry
    research-app.js    # research page entry (job list, live progress, report view)
    modal.js           # shared modal scaffold utility
    settings-auth.js   # auth guard (requireAuth)
    settings-account.js       # account settings page
    settings-avatar.js        # shared avatar upload UI component
    settings-character-edit.js # character edit page
    settings-characters.js    # characters list page
    settings-tools.js         # admin tools page
    settings-users.js         # user management page
    vendor/
      marked.esm.js    # marked.js (vendored, no build step)
      katex.esm.js     # KaTeX math rendering (vendored, no build step)
  css/
    colors_and_type.css   # colour, type, spacing, and motion tokens
    components.css        # buttons, inputs, cards, bubbles, toggles
    app.css               # layout and app-specific overrides
    complete.css          # completions page layout and overrides
    research.css          # research page layout and overrides
    menu.css              # menu page styles
    settings.css          # settings pages layout and overrides
    katex.min.css         # KaTeX math rendering CSS (vendored)
  settings/
    account.html          # account settings page
    character-edit.html   # character edit page
    characters.html       # characters list page
    tools.html            # admin tools page
    users.html            # user management page
  assets/
    icons/             # Lucide SVGs served individually (fetched by icons.js)
    *.svg              # brand SVGs (logo, lemon-slice, sparkle, scribble-underline)
data/
  lemon.toml           # config for Docker volume mount
Dockerfile
docker-compose.yml
docker-compose.override.yml.example
run.sh                 # shortcut: go run ./cmd/lemon-chat
lemon.toml.example     # documented config template (committed)
lemon.toml             # live config (gitignored)
docs/
  SPEC.md              # feature specification (not actively maintained)
README.md
LICENSE
CLAUDE.md
```

## Tool calls

Characters can have a list of tools enabled. When a message is sent to a character with tools, the model may call tools during the response loop. The server runs up to `max_tool_loops` rounds (default 5, configurable in `lemon.toml`) before cutting off and returning the final response.

Tool definitions and executors live in `internal/server/tools.go`. To add a new tool:

1. Add an entry to `toolRegistry` (the `toolDef` sent to the model).
2. Add a matching entry to `executors` (the Go function that runs it).
3. Add the tool to `allTools` in `InitTools` so it appears on `GET /api/tools`.
4. Add the tool ID to the `TOOL_GROUPS` array in `static/js/settings-character-edit.js` — tools not listed there are silently excluded from the character editor UI even though the API returns them.
5. Update the table below.

Available tools and their config requirements:

| Tool ID | Display name | Requires config |
|---|---|---|
| `get_time` | Get current time | — |
| `roll_dice` | Roll dice | — |
| `pick_random` | Pick random | — |
| `random_chance` | Random chance | — |
| `fetch_url` | Fetch URL | — |
| `create_document` | Create document | — |
| `wikipedia_search` | Wikipedia search | — |
| `wikipedia_get_page` | Wikipedia get page | — |
| `searxng` | SearXNG | `[searxng] url` in `lemon.toml` |
| `generate_image_sdxl` | Generate image (SDXL) | `[comfyui] url` + `sdxl_file` in `lemon.toml` |
| `generate_image_flux` | Generate image (Flux Schnell) | `[comfyui] url` + `flux_file` in `lemon.toml` |
| `note_save` | Note: save | — |
| `note_load` | Note: load | — |
| `note_list` | Note: list | — |
| `note_delete` | Note: delete | — |
| `note_append` | Note: append | — |

`InitTools(cfg)` is called once at startup and sets the `Configured` flag on tools that need external services. The frontend reads `GET /api/tools` and shows a config hint for unconfigured tools.

Compound group IDs expand to multiple tools:
- `world_state` → `state_set`, `state_modify`, `state_unset`, `state_list`
- `notes` → `note_save`, `note_load`, `note_list`, `note_delete`, `note_append`

### Attachments

Tools that produce files (`create_document`, `generate_image_sdxl`, `generate_image_flux`) create an `attachment` DB record and write the file under `<data_dir>/attachments/<random-id>/`. They return an `AttachmentResult` JSON struct; `messages.go` detects this shape and emits an `attachment` SSE event so the frontend can render a download card. Attachments are served by `handleGetAttachment` in `attachments.go` — `?download=1` forces a download, otherwise the file is served inline.

## Research

The research feature (reachable from `/menu` → research) runs iterative LLM-driven web research: Plan → Classify → (Think → Search → Extract → Synthesise → Decide)* → Final Report. It was ported from `docs/deep_research_spec.html`; the engine lives in `internal/research/`, handlers in `internal/server/research.go`.

- Requires `[searxng] url` in `lemon.toml`; tuning lives in the optional `[research]` section (model, rounds, timeouts — see `lemon.toml.example`).
- Each job has a `mode` (`research` or `brainstorm`, picked from a dropdown on the form). Brainstorm mode is an ideation-driven variant: it skips classification, each round the model develops ideas and decides for itself whether to search the web (an empty query list means "no search this round" rather than "stop"), and it produces a structured design document instead of a cited report. The mode swaps in the `brainstorm*` prompts in `prompts.go`; the round dispatcher is `runOneRound` in `researcher.go`.
- Jobs are rows in the `research_job` table. The engine checkpoints its full state (plan, evolving report, findings, queries, analyzed URLs, elapsed time) after planning and after every round; `Server.ResumeResearchJobs()` is called at startup and resumes any job left in `pending`/`running` from its last checkpoint, so a server crash mid-job loses at most one round.
- Progress streams over SSE from `GET /api/research/{id}/events`; the run itself is detached from the HTTP request, so closing the page does not stop the job.

## Not yet implemented

The following are not yet built. Stub them out rather than building them:

- User profiles / profile switcher — auth is done; only one active user at a time, no switcher UI
- User-uploaded file attachments (tool-generated attachments are implemented)
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

- Increment the version number (check `store.go` for the current highest version)
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

**Debug logging.** Use `debug.Log()` from `internal/debug` for output that is only useful during development (e.g. title-worker trigger conditions, raw HTTP responses from model servers). `debug.Log` is a no-op unless `debug = true` is set in `lemon.toml`. Never use `debug.Log` for things that should always be visible — use `log.Printf` for those.

## CLI flags

```
lemon-chat [flags]
  --config <path>    path to config file (default: lemon.toml)
  --debug            enable debug logging (overrides config debug = true)
  --token-log        log raw model SSE tokens to <data_dir>/model_tokens.log (overrides config)
  --list-models      query all configured model servers, print their model lists, then exit
```

`--list-models` is useful for finding exact model ID strings to put in `lemon.toml`.

## Debugging flags

Two flags are available for diagnosing runtime issues. Both can be set in `lemon.toml` or passed as CLI flags (the flag overrides the config value):

| Flag | TOML key | Description |
|---|---|---|
| `--debug` | `debug = true` | Enables `debug.Log()` output — title-worker conditions, HTTP details |
| `--token-log` | `token_log = true` | Writes every raw SSE token from the model to `<data_dir>/model_tokens.log`, prefixed with `[loop=N]`. Useful for diagnosing streaming/rendering inconsistencies. |

The token log appends to the file on each request and includes a header line:
```
=== conv=42 model=llama3.2 time=2026-01-01T12:00:00+00:00 ===
[loop=0] data: {"choices":[{"delta":{"content":"Hello"},...}]}
```

## Keeping TODO.md current

Update `TODO.md` as work progresses — mark items `[~]` when started and `[x]` when done. When adding new work that isn't already listed, add it to the relevant section before starting. Don't leave TODO.md stale.
