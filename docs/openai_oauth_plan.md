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

### Phase 3 — Config + transport selection

- Add `api = "responses"` (default `"chat_completions"`) and an auth-mode notion to `ModelServer`.
- Introduce a `Transport` seam so call sites obtain a token from a provider rather than reading a
  static string, and route request-building through either the chat-completions or Responses
  implementation.

### Phase 4 — Responses API translation (the main work)

- `internal/llm/responses.go`: request mapping (`messages[]` → `input[]` + `instructions`,
  `max_tokens` → `max_output_tokens`, tool-schema shape, `/responses` endpoint) and an SSE event
  demuxer mapping Responses events (`response.output_text.delta`, `response.completed`,
  `response.function_call_arguments.delta`, …) onto the existing `onDelta` / usage callbacks.
- Route `messages.go`'s inline builder through the same abstraction so **chat and research share
  one implementation**.

### Phase 5 — Tool-call loop adaptation

- Adapt the `max_tool_loops` loop to the Responses `function_call` output items and feed tool
  results back as `function_call_output` input items.

### Phase 6 — Linking UI + CLI

- `--openai-login` CLI subcommand to run the flow on the host (headless-friendly).
- Admin-only "Connect OpenAI account" control showing connected state / plan / expiry.

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
