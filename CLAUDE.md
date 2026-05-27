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

## Code layout

```
cmd/lemon-chat/
  main.go              # entry point, config loading, server start
internal/
  config/              # TOML struct definitions and loading
  server/              # HTTP handlers and routing
  store/               # SQLite schema and queries
static/
  index.html           # app shell
  js/
    app.js             # entry, wires modules together
    api.js             # fetch wrappers for the REST API
    sidebar.js         # conversation list, navigation
    thread.js          # message display, SSE streaming
    composer.js        # text input, send logic
    markdown.js        # lightweight message rendering
  css/
    colors_and_type.css   # copied from design system
    components.css        # copied from design system
    app.css               # layout and app-specific overrides
  assets/              # SVGs copied from design system
lemon.toml.example     # documented config template (committed)
lemon.toml             # live config (gitignored)
SPEC.md
CLAUDE.md
```

## What to stub for MVP

The following are in `SPEC.md` but not in the MVP build. Stub them out rather than building them:

- Auth and user profiles — single anonymous user for now
- Personas — listed in spec, not yet implemented
- File attachments
- Model management UI — config file only
- Message editing and regeneration
- Conversation search

When something is stubbed, return a `501 Not Implemented` from the API endpoint and leave a `// TODO` comment. Don't build placeholder UI for features that don't exist yet.
