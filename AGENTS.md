# lemon-chat — Claude Code guide

## What this project is

lemon-chat is a local, open-source AI chat UI. Think self-hosted claude.ai or ChatGPT. It talks to locally-running models via HTTP API (e.g. Ollama) and persists everything in a local SQLite database.


## General conventions

- All files must end with a single trailing newline.

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

## Model names and IDs

A `[[model]]`'s `name` is the stable identifier lemon-chat stores — in `character.model`, `conversation.model`, `completion.model`, the research tables and `default_model` — and must be unique. Its optional `id` is the model ID sent to the provider (defaults to `name`). Any code that sends a model to a provider must map it with `s.cfg.ProviderModelID(name)` (or `Model.ProviderID()`); OpenRouter prices are keyed by the provider ID too. Everything else — storage, comparisons, `ServerForModel`, restrictions — uses `name`. The columns that store names are listed in `modelColumns` (`internal/store/model_names.go`); add any new one there so `--rename-model` and `--models-in-use` cover it.

## Model modes

Each `[[model]]` in `lemon.toml` has an optional `modes` array that restricts which interface it appears in:

```toml
[[model]]
name         = "llama3.2"
model_server = "local"
modes        = ["chat"]     # only appears in chat, not completions
```

Valid values are `"chat"` and `"complete"`. Omit `modes` entirely to allow the model in both. Set `modes = []` (an explicitly empty list) to hide the model from both the chat and completions pickers — it remains selectable for research, which lists all configured models regardless of mode. This omitted-vs-empty distinction is why `Model.Modes` is a `*[]string` rather than a plain slice. The frontend fetches `GET /api/models?mode=chat` or `?mode=complete` and only shows the filtered list. When adding a new interface or model picker, pass the correct `mode` parameter.

## Restricted models

`restricted = true` on a `[[model_server]]` (or on a single `[[model]]`, which overrides its server; `restricted = false` opens one model on a restricted server) limits those models to admins and users with `user.can_use_restricted`, toggled as "Restricted models" on the admin users page. It exists so household members can't run up paid API usage. Which models cost money is config; who may use them is per user in the database (usernames never go in the committed config).

Every handler that runs a model on a user's behalf must call `s.denyRestrictedModel(w, user, names...)` (chat send, completions, research start and report regeneration, the Reddit harness), and handlers that store a model choice check it too (conversation create/import, character create, character update when the model changes). `GET /api/models` omits models the user can't use, and `GET /api/characters` sets `model_available` so the chat pickers hide characters on such models. Background work (titles via `default_model`, already-started research) is not re-checked. When adding a new place a user picks or runs a model, add the check.

## Character variable substitution

Character system prompts and first messages support two placeholders that are substituted at runtime:

- `{{char}}` — replaced with the character's name
- `{{user}}` — replaced with the logged-in user's display name

Apply this substitution wherever character text is rendered or sent to the model. The substitution logic lives in `internal/server/messages.go`.

## Code layout

```
cmd/lemon-chat/
  main.go              # entry point, config loading, bootstrap, workers, HTTP server
comfyui/
  workflows.toml       # image workflow registry: IDs, node bindings, sizes, timeouts
  workflows/           # ComfyUI API-format workflow JSON files
internal/
  comfyui/
    registry.go        # workflow registry: load/validate bindings, profile selection, per-request graph build
  config/
    config.go          # TOML struct definitions, loading, model name → provider ID mapping
    env.go             # ${VAR} environment substitution in config string values
  debug/
    debug.go           # debug logging flag (debug.Enabled, debug.Log)
  htmltext/
    htmltext.go        # HTML → plain text (snippet strip + structure-preserving)
  llm/
    provider.go        # Provider interface: streaming chat over chat-completions or Responses,
                       #   wire/error transcripts. All model calls go through this.
    responses.go       # Responses API body building and stream → chat-frame translation
    llm.go             # shared types (Usage, Completion), SSE scanner, legacy raw helpers
    models.go          # /models listing
    capabilities.go    # image-input support per model (for the --list-models config check)
  netguard/
    netguard.go        # HTTP client for model-chosen URLs that refuses private/LAN addresses
  openai_auth/         # Codex-style OAuth (PKCE) login, token refresh, shared token store
  redditimport/
    redditimport.go    # browser-extension interchange format for Reddit captures
  research/
    researcher.go      # research engine: phase loop, state, checkpointing
    prompts.go         # all research prompts (ported from docs/deep_research_spec.html)
    llm.go             # LLM calls, think-tag stripping, robust JSON parsing, injection guard
    web.go             # SearXNG search, page fetch, goal-based extraction
  searx/
    searx.go           # small SearXNG client
  server/
    server.go          # router setup, static serving, body limits, providerFor
    auth.go            # login / logout / me / profile / password handlers
    avatars.go         # avatar upload and serve handlers (users + characters)
    conversations.go   # conversation CRUD, fork, import, title, background handlers
    messages.go        # message list + SSE streaming handler + tool loop
    models.go          # model list handler
    characters.go      # character CRUD handlers
    completions.go     # completions CRUD + streaming handlers
    notes.go           # notes API handlers
    attachments.go     # attachment upload + serve handlers
    admin.go           # admin user-management, model listing, note-pack import
    oauth_login.go     # admin "Connect OpenAI account" paste-the-code flow
    middleware.go      # session, auth, admin middleware
    errorlog.go        # always-on model_errors.log sink
    ws.go              # WebSocket hub (per-user routing), upgrade handler
    research.go        # research job manager, handlers, SSE progress, crash recovery
    research_reports.go  # report variants (regenerate, HTML report)
    research_bundle.go   # research download / debug bundles (zip)
    research_runlog.go   # per-job diagnostic run log
    reddit_import_debug.go # debug-only Reddit import harness
    tools.go           # ToolContext, executor dispatch, attachment helpers
    tools_registry.go  # toolRegistry: every selectable tool's schema + metadata
    tools_meta.go      # InitTools, GET /api/tools metadata
    tools_web.go       # fetch_url, SearXNG, Wikipedia tools
    tools_image.go     # generate_image: ComfyUI submit/poll/download + tool schema
    tools_state_notes.go # world-state and notes tools
  store/
    store.go           # open DB, run migrations, bootstrap
    users.go           # user queries (incl. full user deletion)
    sessions.go        # session queries
    conversations.go   # conversation queries (delete, stale cleanup, fork, import)
    messages.go        # message queries
    characters.go      # character queries
    completions.go     # completion queries
    notes.go           # note queries
    state.go           # world-state queries
    research.go        # research job queries (incl. resume checkpoints)
    research_reports.go # research report variant queries
    attachments.go     # attachment queries, unlinked-upload sweep, safe file removal
    oauth.go           # shared OAuth token row
    model_names.go     # stored model-name references: usage listing and --rename-model
    prices.go          # cached model per-token price queries
  tasks/
    titles.go          # Titler: conversation/completion titles via the Provider, with retry backoff
    cleanup.go         # background stale-conversation, session and unsent-upload cleanup
    prices.go          # background daily OpenRouter model-price fetch worker
static/
  index.html           # main chat app shell
  complete.html        # completions app shell
  research.html        # research page shell
  research-help.html   # research help page
  menu.html            # mobile/navigation menu shell
  reddit-import-debug.html # debug-only Reddit import harness
  js/
    app.js             # entry, wires modules together
    api.js             # fetch wrappers for the REST API, SSE consumer
    sidebar.js         # conversation list, navigation
    thread.js          # message display, SSE streaming, artifact panel, lightbox
    composer.js        # text input, send logic, image attach/paste
    markdown.js        # lightweight message rendering
    header.js          # chat header: title display + model/character picker
    icons.js           # SVG icon loader/cache (fetches from /assets/icons/)
    utils.js           # shared frontend utilities (e.g. escapeHtml)
    ws.js              # WebSocket client, auto-reconnect, event dispatch
    complete-app.js    # completions page entry
    research-app.js    # research page entry (job list, live progress, report view)
    modal.js           # shared modal scaffold utility
    reddit-import-debug.js    # debug-only Reddit import harness
    settings-auth.js   # auth guard (requireAuth)
    settings-account.js       # account settings page
    settings-avatar.js        # shared avatar upload UI component
    settings-character-edit.js # character edit page
    settings-characters.js    # characters list page
    settings-import-chat.js   # import chat page
    settings-notes.js         # notes page
    settings-tools.js         # admin tools page
    settings-users.js         # user management page (typed-confirmation delete)
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
    import_chat.html      # import chat page
    notes.html            # notes page
    tools.html            # admin tools page
    users.html            # user management page
  assets/
    icons/             # Lucide SVGs served individually (fetched by icons.js)
    *.svg              # brand SVGs (logo, lemon-slice, sparkle, scribble-underline)
data/
  lemon.toml           # config for Docker volume mount
deploy/
  lemon.toml           # the maintainer's live deployment config (committed; secrets via ${VAR})
  nginx/lemon-chat     # nginx site for that deployment (TLS, websocket, SSE streaming)
Dockerfile
docker-compose.yml
docker-compose.override.yml.example
run.sh                 # shortcut: go run ./cmd/lemon-chat
lemon.toml.example     # documented config template (committed)
lemon.toml             # local config (gitignored)
.env                   # secrets for ${VAR} config references (gitignored; loaded by run.sh)
docs/
  SPEC.md              # feature specification (not actively maintained)
README.md
LICENSE
CLAUDE.md
```

Every model call must go through `llm.Provider` (in `server`, use `s.providerFor(srv)`), so `api = "responses"` and `auth = "oauth"` servers work everywhere. Don't build raw `/chat/completions` requests.

URLs chosen by the model (`fetch_url`, research page fetches) must be fetched with the `netguard` client (`s.fetchClient` / `ToolContext.FetchClient`), which refuses loopback and LAN addresses unless `allow_private_fetch = true` is set in `[server]`. Configured endpoints (model servers, SearXNG, ComfyUI) use ordinary clients.

## Tool calls

Characters can have a list of tools enabled. When a message is sent to a character with tools, the model may call tools during the response loop. The server runs up to `max_tool_loops` rounds of tool calls (default 5, configurable in `lemon.toml`), then makes one more call with tools withheld so the model has to answer with what it has.

If a model stream fails partway through an answer, the partial text is saved and the client gets a non-fatal `warning` SSE event (shown as a note under the reply); tool calls from a broken stream are never run. A stream that fails before producing anything sends an `error` event. A user pressing stop saves the partial text silently.

Tool declarations live in `toolRegistry` in `internal/server/tools_registry.go`; executor implementations are split across the `tools_*.go` files. To add a new selectable tool:

1. Add one `toolRegistry` entry containing its model-facing schema, executor, display metadata, group, and order. Add a `Configure` hook there if availability depends on config.
2. Implement the executor in the appropriate concern file.
3. Update the table below.

`InitTools` derives the executor map and `GET /api/tools` metadata from the registry. The character editor groups the API response by its `group` field, so there are no matching backend/frontend ID lists to maintain.

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
| `generate_image` | Generate image | `[comfyui] url`, `default_profile` + `[comfyui.profile.*]` in `lemon.toml` |
| `view_image` | View image | a `vision = true` model (withheld from other models) |
| `world_state` | World state (compound) | — |
| `state_set` | World state: set | — |
| `state_modify` | World state: modify | — |
| `state_unset` | World state: unset | — |
| `state_list` | World state: list | — |
| `state_clear` | World state: clear | — |
| `notes` | Notes (compound) | — |
| `note_to_self` | Note to self | — |
| `note_save` | Note: save | — |
| `note_load` | Note: load | — |
| `note_list` | Note: list | — |
| `note_delete` | Note: delete | — |
| `note_append` | Note: append | — |

`InitTools(cfg)` is called once at startup and sets the `Configured` flag on tools that need external services. The frontend reads `GET /api/tools` and shows a config hint for unconfigured tools.

Compound group IDs expand to multiple tools:
- `world_state` → `state_set`, `state_modify`, `state_unset`, `state_list`, `state_clear`
- `notes` → `note_save`, `note_load`, `note_list`, `note_delete`, `note_append`

### Image generation

`generate_image` is a general-purpose, ChatGPT-style image tool: the model writes the prompt and picks an aspect ratio; CFG, steps and sampler are fixed inside each workflow. It is backed by a workflow registry (`internal/comfyui/`):

- `comfyui/workflows/*.json` are ComfyUI API-format exports, committed to the repo. `comfyui/workflows.toml` registers each one with an exact ID, explicit `node_id.input_name` bindings for prompt, optional negative prompt, seed and width/height, the output node, a timeout, and per-aspect-ratio sizes. Bindings are validated against the graph at startup; a workflow that fails is logged and reported as unavailable, never used. Nothing is inferred from filenames.
- The prompt input contains `%PROMPT%` exactly once; text around it (e.g. Pony score tags) is part of the workflow's recipe. The bound negative input's value in the file is the default negative; the model may replace it, and the negative actually used is returned in the result.
- Profiles (`[comfyui.profile.<name>]` in `lemon.toml`) map task names to workflow IDs, so swapping the workflow behind a profile is config-only. Only workflows with `exposed = true` can be selected by `workflow_id`; selection is deterministic and never falls back to a different workflow.
- Each request decodes a fresh copy of the graph before setting inputs, so concurrent requests never share state. The result includes the attachment plus workflow ID, workflow hash, profile, seed, size and the negative prompt used.
- Prompt-rewriting (`_llm`) workflows from ComfyTestBed are deliberately not imported; the chat model writes the prompt.

`view_image` lets a vision model look at an image generated in the current conversation (by `image_id`, default the latest; it waits for a pending image). Tool results are text-only on chat-completions, so the executor only validates and returns `{"image_id":N,"status":"shown"}`; the tool loop then appends a user message carrying the image after that round's tool results. `buildChatMsgs` rebuilds the same message from stored `view_image` results on every later turn, so nothing extra is persisted. The tool is filtered out of the tool list for models without `vision = true`, and the loader only serves ready images from the same conversation.

### Attachments

The `attachment` table backs two kinds of file, distinguished by its `source` column:

- **`source = "tool"` (model output).** Tools that produce files (`create_document`, `generate_image`) create an `attachment` record keyed on `tool_call_id` and write the file under `<data_dir>/attachments/<random-id>/`. They return an `AttachmentResult` JSON struct; `messages.go` detects this shape and emits an `attachment` SSE event so the frontend can render a download card.
- **`source = "upload"` (user input / vision).** A user uploads an image via `POST /api/conversations/{id}/attachments` (multipart field `file`, image-only, ≤10 MB), handled by `handleUploadAttachment`. The record is created unlinked (`message_id` NULL); when the user sends the message it was attached to, `LinkAttachmentsToMessage` backfills `message_id`. On send and on history reload, upload attachments for a message are lowered into chat-completions multimodal content (a text part plus one `image_url` data-URL part each) by `buildChatMsgs` / `chatMsg.MarshalJSON`; the Responses surface lowers these to `input_image` in `lowerUserContent` (`internal/llm/responses.go`). Image input is gated on the model's `vision = true` flag in `lemon.toml` (exposed via `GET /api/models`); the composer only shows its attach control for vision models.

Both kinds are served by `handleGetAttachment` in `attachments.go` — `?download=1` forces a download, otherwise the file is served inline.

## Research

The research feature (reachable from `/menu` → research) runs iterative LLM-driven web research: Plan → Classify → (Think → Search → Extract → Synthesise → Decide)* → Final Report. It was ported from `docs/deep_research_spec.html`; the engine lives in `internal/research/`, handlers in `internal/server/research.go`.

- Requires `[searxng] url` in `lemon.toml`; tuning lives in the optional `[research]` section (model, rounds, timeouts — see `lemon.toml.example`).
- Each job has a `mode` (`research` or `brainstorm`, picked from a dropdown on the form). Brainstorm mode is an ideation-driven variant: it skips classification, each round the model develops ideas and decides for itself whether to search the web (an empty query list means "no search this round" rather than "stop"), and it produces a structured design document instead of a cited report. The mode swaps in the `brainstorm*` prompts in `prompts.go`; the round dispatcher is `runOneRound` in `researcher.go`.
- Jobs are rows in the `research_job` table. The engine checkpoints its full state (plan, evolving report, findings, queries, analyzed URLs, elapsed time) after planning and after every round; `Server.ResumeResearchJobs()` is called at startup and resumes any job left in `pending`/`running` from its last checkpoint, so a server crash mid-job loses at most one round.
- Progress streams over SSE from `GET /api/research/{id}/events`; the run itself is detached from the HTTP request, so closing the page does not stop the job.

## Not yet implemented

The following are not yet built. Stub them out rather than building them:

- User profiles / profile switcher — auth is done; only one active user at a time, no switcher UI
- User-uploaded *text-file* attachments — image uploads (vision input) and tool-generated attachments are implemented; other file types are not
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
  --openai-login     run the OpenAI (Codex) account login on this host, store the token, then exit
  --models-in-use    list model names stored in the DB with counts, flagging ones not in the config, then exit
  --rename-model OLD NEW [OLD NEW ...]
                     rewrite stored model-name references (NEW must be configured), then exit
```

`--list-models` is useful for finding exact model ID strings to put in a model's `id`. After each server's list it checks that server's configured models: an `id` the server doesn't offer, or a `vision` flag that disagrees with the server's reported image support (`llm.ImageInputSupport`: Codex catalog, OpenRouter `architecture`, Ollama `/api/show`; vLLM reports nothing). `vision` stays a config flag — this is a check, not runtime detection.

`--openai-login` runs the local-browser PKCE flow (loopback on `localhost:1455`) and persists a
single shared OAuth token used by every `auth = "oauth"` model server. For a remote/LAN install
where no browser runs on the host, use the admin Tools page "Connect OpenAI account" paste-the-code
flow instead. See `docs/openai_oauth_plan.md` and `docs/provider_abstraction.md`.

## Debugging flags

Two flags are available for diagnosing runtime issues. Both can be set in `lemon.toml` or passed as CLI flags (the flag overrides the config value):

| Flag | TOML key | Description |
|---|---|---|
| `--debug` | `debug = true` | Enables `debug.Log()` output — title-worker conditions, HTTP details |
| `--token-log` | `token_log = true` | Writes replay-oriented model HTTP transcripts to `<data_dir>/model_tokens.log`: request URL, safe headers, exact JSON body, response metadata, and raw SSE body. OAuth credentials and account IDs are redacted. Prompt and response content is recorded. |

The token log appends to the file on each request and includes a header line followed by the request and response transcript:
```
=== conv=42 model=llama3.2 time=2026-01-01T12:00:00+00:00 ===
--- request ---
POST https://example.test/v1/responses
Authorization: [redacted]

{"model":"llama3.2",...}
--- response ---
200 OK
Content-Type: text/event-stream

data: {"type":"response.output_text.delta",...}
```

### Model error log (always on)

Separately from the opt-in token log, every model request that **fails** — a
transport error, a non-2xx response, or a mid-stream error event — has its
redacted HTTP transcript appended to `<data_dir>/model_errors.log`. No flag
enables it; successful requests write nothing, so the file only ever contains
failures. This is what to check first when a chat or research call errors.

The transcript is buffered in memory per request (capped at 1 MiB, keeping the
tail where the failing frame is) and flushed on failure only. Redaction matches
the token log. It's wired via `llm.Handler.ErrorLog`; the shared sink lives in
`internal/server/errorlog.go`.

## Keeping TODO.md current

Use `TODO.md` to track planned work, not as a changelog for every request.

- If the requested work already has a TODO item, mark it `[~]` when started and `[x]` when done.
- If the user asks to add or track a TODO item, add it without also implementing the work unless they explicitly ask for both.
- If the user asks to perform a task that is not already listed, just do the task. Do not create a TODO item for it.

Don't leave existing TODO items stale.
