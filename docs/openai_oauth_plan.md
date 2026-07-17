# OpenAI OAuth + Responses API — implementation plan

## Goal

Let lemon-chat authenticate to OpenAI using a Codex-style OAuth login (sign in with a
ChatGPT/OpenAI account) instead of only a static API key, and drive requests through the
**Responses API**. A **single shared token** is stored server-side and used for every user's
requests — lemon-chat is a home-network app, not a true multi-tenant service. Both **chat** and
**research** must work through this path.

## Constraints (from AGENTS.md)

- Backend Go; minimal dependencies — the OAuth flow uses only the standard library.
- SQLite via `modernc.org/sqlite`; singular table names; migrations in `internal/store/store.go`.
- No build step / vanilla JS for any UI.
- Model config stays in TOML. The OAuth *token* mutates at runtime, so it lives in the DB, not
  TOML. Account *linking* is an admin action, not model config, so a small UI surface is allowed.
- Standard `log` logging conventions; one line per infrequent handler.

## Background: how Codex's OAuth works

1. PKCE OAuth 2.0 against `auth.openai.com` with a public `client_id` baked into the client.
2. A loopback HTTP server (`localhost:1455`, path `/auth/callback`) receives the redirect.
3. The `code` is exchanged for `access_token` + `refresh_token` + `id_token` (JWT). The id_token
   carries the ChatGPT account id and plan.
4. Requests hit the **Responses API** with `Authorization: Bearer <access_token>` and a
   `chatgpt-account-id` header. (Codex can alternatively token-exchange for a real API key.)
5. Tokens are persisted and refreshed via the refresh-token grant before expiry.

## Current state of lemon-chat auth

Every `ModelServer` (`internal/config/config.go`) has a static `api_key` injected as
`Authorization: Bearer <key>` at every call site. Two independent code paths speak the
OpenAI-*compatible* `/chat/completions` schema:

1. **Chat** — `internal/server/messages.go` builds the payload inline and parses
   `choices[].delta.content` SSE itself.
2. **`internal/llm`** — `ChatComplete*` used by research (`internal/research/llm.go`),
   completions, titles, and tools; all POST to `apiBase + "/chat/completions"` and parse the same
   delta shape in `ScanSSE`.

Responses support must therefore cover **both** paths.

---

## Phased plan

### LAN hosting and the fixed redirect URI

OpenAI's redirect allow-list pins the OAuth callback to
`http://localhost:1455/auth/callback` — it **cannot** be changed to lemon-chat's LAN address. So
when the server runs on a home-network box but the admin's browser is on a different machine, the
loopback callback can never reach the server. There are therefore two login drivers over the same
PKCE flow (both already implemented in `internal/openai_auth`):

- **Local browser** (`Login`) — binds the loopback callback server on the same machine and
  captures the redirect automatically. Only works when the browser and server share a host (or via
  the `--openai-login` CLI run on the box).
- **Paste-the-code** (`Begin` / `ParsePasted` / `Complete`) — for LAN hosting. `Begin` returns an
  authorize URL to open; after sign-in the browser lands on the dead `localhost:1455` URL, whose
  address bar still carries `?code=…&state=…`; the admin pastes that URL (or the bare code) into
  lemon-chat, and `Complete` exchanges it. The `PendingLogin` (PKCE verifier + state) is held
  in-process, keyed to the admin's session, between the two requests.

This makes **paste-the-code the primary flow** for the typical LAN deployment; the Phase 6 UI
leads with it and offers the local-browser flow as a convenience when co-located.

### Phase 1 — OAuth login core (`internal/openai_auth`)  ← this PR starts here

Self-contained, standard-library-only package, no wiring into the server yet so it can be
developed and unit-tested in isolation.

- `pkce.go` — code verifier/challenge (S256) generation.
- `oauth.go` — endpoint constants (`auth.openai.com` authorize/token), `client_id`, the token
  structs (`Tokens{AccessToken, RefreshToken, IDToken, AccountID, Expiry}`), authorize-URL
  builder, code→token exchange, refresh grant, and id_token (JWT) claim parsing for account id +
  expiry.
- `login.go` — `Login(ctx)` drives the loopback flow: bind `localhost:1455`, open the browser
  (best-effort; print the URL as fallback), block on `/auth/callback`, validate `state`, exchange
  the code, return `Tokens`.
- `login_test.go` / `oauth_test.go` — unit tests for PKCE, authorize-URL shape, JWT parsing, and
  token exchange against an `httptest` server. No network in tests.

Deliverable: a package that can complete a login and produce refreshable `Tokens`, exercised by
tests. Not yet persisted or wired to requests.

### Phase 2 — Token storage + refresh + provider

- Migration **v41** in `store.go`: single-row `oauth_token` table
  (`id` PK check =1, `access_token`, `refresh_token`, `id_token`, `account_id`, `expiry`,
  `updated_at`).
- `internal/store/oauth.go` — get/upsert the single row.
- `internal/openai_auth/provider.go` — mutex-guarded in-memory cache wrapping the store; returns a
  live bearer token, refreshing (refresh-token grant) when within N seconds of expiry and
  persisting the rotated tokens.

### Phase 3 — Config + transport selection  ✅ done

- `ModelServer` gains `api` (`chat_completions` default | `responses`) and `auth`
  (`api_key` default | `oauth`), with `UsesResponses()`, `UsesOAuth()`, and `Endpoint()` helpers
  and validation. Documented in `lemon.toml.example`.
- `config.TokenSource` (`func(context.Context) (string, error)`) is the seam: `Server.tokenSource`
  returns the shared OAuth provider's `Token` for oauth servers, else a `StaticToken` of the
  api_key. `Server.bearerToken` resolves it at call time.
- Wired into the **chat** path (`messages.go`, incl. the `chatgpt-account-id` header for oauth),
  **completions**, and **research** — `research.Config` now takes `APIToken`/`WorkerAPIToken`
  token sources instead of static keys, so detached jobs pick up refreshed tokens. Report
  generation (`research_reports.go`) and the reddit-import debug extractor resolve the same way.
- **Deferred to a later phase:** the background title worker (`internal/tasks`) and the
  `summariseHTML` fetch-url tool helper are free functions without provider access; they stay on
  static api_key for now (only matters if a default/tool model is pointed at an oauth server).
  The actual Responses request/response translation is Phase 4 — until then an `api = "responses"`
  server still receives chat-completions-shaped requests.

### Phase 4 — Responses API translation (the main work)  ✅ done

Implemented as **translation at both ends**, so there is no second parser and chat + research
share one implementation:

- `internal/llm/responses.go`:
  - `BuildResponsesBody` lowers chat-completions messages + tools into a Responses request body —
    system prompts → `instructions`, messages → item-oriented `input` (`input_text` /
    `output_text` / `function_call` / `function_call_output`), tools flattened to
    `{type,name,description,parameters}`, `max_tokens` → `max_output_tokens`, `temperature`
    dropped (Codex rejects it), `store:false`, `include:["reasoning.encrypted_content"]`.
  - `ResponsesToChatSSE` converts the `response.*` SSE grammar back into chat-completions
    `data:{choices:[{delta}]}` frames (text deltas, tool-call start/args deltas, terminal
    finish_reason + usage), via an `io.Pipe`. Reasoning deltas are not forwarded as content.
  - `ResponsesStreamWithUsage` + a shared `readChatCompletionsStream` helper so both surfaces use
    one parser.
- **Chat** (`messages.go`): builds the Responses body when the server uses it and wraps the
  response with `ResponsesToChatSSE` before its existing inline parser — otherwise unchanged.
- **Research** (`research/llm.go`, `research_reports.go`): `modelEndpoint`/`Config` carry a
  Responses flag + account id; streaming and non-streaming calls (and the HTML report writer)
  route through `ResponsesStreamWithUsage` when the endpoint is a Responses server.

### Phase 5 — Tool-call loop adaptation  ✅ largely covered by Phase 4

Because `ResponsesToChatSSE` emits chat-completions tool-call frames (with `finish_reason:
tool_calls`) and `BuildResponsesBody` lowers replayed `tool_calls` / tool results into
`function_call` / `function_call_output` items, the **existing** `max_tool_loops` loop in
`messages.go` drives Responses tool calls unchanged. Remaining: live end-to-end verification
against the real Codex endpoint (multi-round tool loops, reasoning items).

### Phase 6 — Linking UI + CLI

- `--openai-login` CLI subcommand to run the local-browser flow on the host (headless-friendly).
- Admin-only "Connect OpenAI account" control that leads with the **paste-the-code** flow (open
  authorize URL → paste the redirected URL back) for LAN deployments, showing connected state /
  account / expiry. The server holds the `PendingLogin` in memory between Begin and Complete.

---

## Risks / open questions

- **Responses translation is the schedule driver** and the main risk; chat and research both
  exercise it.
- Loopback redirect assumes the browser and server share a host; the `--openai-login` CLI path
  covers headless installs.
- **Resolved:** target the ChatGPT-subscription Responses surface directly (no token-exchange to
  an API key). This is an accepted, documented pattern used by other tools (e.g. Pi agent,
  opencode) and confirmed acceptable by OpenAI.

## Effort estimate

| Phase | Effort | Risk |
|---|---|---|
| 1. OAuth login core | ~1 day | low |
| 2. Token store + refresh | ~0.5 day | low |
| 3. Config + transport seam | ~0.5 day | low |
| 4. Responses translation | ~2–3 days | high |
| 5. Tool-call loop | ~1 day | medium |
| 6. UI + CLI | ~0.5 day | low |
