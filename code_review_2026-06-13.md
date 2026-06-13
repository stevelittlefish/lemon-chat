# lemon-chat — code review

**Date:** 2026-06-13
**Scope:** full project — Go backend (`cmd/`, `internal/`), vanilla-JS frontend (`static/js`), CSS, config, migrations.
**Reviewer:** Claude

This is a careful, opinionated review of the whole tree. Findings are grouped by the
categories requested (technical issues, potential bugs, duplication, organisation,
maintainability) with a separate security section up top because that is where the
highest-impact problems are. Each finding has a severity tag and `file:line` references.

Overall the codebase is in good shape: it's idiomatic Go, the migration discipline is
solid, the research engine is well-factored, and the frontend is clean vanilla JS with no
build step as intended. The issues below are real but mostly localised; none of them
indicate a rotten foundation.

---

## Summary of the highest-priority items

| # | Severity | Area | One-liner |
|---|----------|------|-----------|
| S1 | **High** | Security | Stored XSS — assistant/markdown content is rendered with `innerHTML` and **no HTML sanitiser** |
| S2 | **High** | Security | Notes IDOR — note read/delete/read-only-toggle by id with **no ownership check** |
| S3 | **High** | Security | Attachment IDOR — any authenticated user can fetch any attachment by id |
| S4 | Medium | Security | Passwordless accounts are a silent full auth bypass |
| B1 | Medium | Bug | `title_prompt` is stored and editable but **never used** — the feature is dead |
| B2 | Medium | Bug | Research SSE has a subscribe-after-finish race → client hangs with no `[DONE]` |
| B3 | Medium | Bug | WebSocket broadcast has no write deadline → one stuck client stalls all notifications |

---

## 1. Security & authorization

### S1 — Stored XSS via unsanitised markdown rendering — **High**

`static/js/markdown.js:35-44` parses markdown to HTML and assigns it straight to
`innerHTML`:

```js
export function render(text) {
  const div = document.createElement('div');
  div.innerHTML = marked.parse(text);   // <-- no sanitisation
  ...
}
```

`marked` does **not** sanitise by default (the old `sanitize` option was removed), so any
raw HTML in the input — `<img src=x onerror=...>`, `<script>`, etc. — is rendered live.
`grep` confirms there is no DOMPurify or any sanitiser anywhere in `static/js`.

This `render()` is used for all assistant message content (`thread.js:479,550,565,763`),
the artifact panel for `.md` files (`thread.js:41`), and the research/completions report
views. The rendered text is **attacker-influenceable**:

- An assistant reply is model output — and the model's context routinely includes web
  page content (via the `fetch_url`/`searxng`/research tools) and other users' shared
  character system prompts / first messages / hidden messages.
- The artifact panel renders the contents of a `create_document` file, which is entirely
  model-authored.

So a malicious web page or a crafted shared character can get script execution in another
user's session (session cookie is `HttpOnly`, but the attacker can still act as the user
via the same-origin API).

**Fix:** vendor DOMPurify and sanitise the output of `marked.parse` before assigning to
`innerHTML` (or configure a strict allowlist renderer). This is the single most important
change in this review.

### S2 — Notes IDOR: read / delete / toggle any note by id — **High**

The notes REST handlers operate purely on the numeric id with no scope or ownership check:

- `handleGetNote` (`internal/server/notes.go:24-40`) → `store.GetNoteByID(id)` returns the
  full value of **any** note, including another user's `u.` notes and any conversation's
  `c.` notes.
- `handleDeleteNote` (`notes.go:82-110`) deletes any note that isn't read-only — including
  other users' notes and global notes.
- `handleSetNoteReadOnly` (`notes.go:112-149`) flips the read-only flag on **any** note,
  including imported global note packs (an attacker can un-protect a read-only global note
  and then delete it, or lock notes they don't own).

`GetNoteByID`/`DeleteNoteByID`/`SetNoteReadOnly` in `internal/store/notes.go:423-520` are
all "by id, no scope" by design, and the handlers never re-check that the note belongs to
the current user / their conversations. Contrast with `ListUserVisibleNotes`, which *does*
scope correctly — the read-by-id path bypasses exactly that logic.

The whole point of the `g.`/`u.`/`c.` scoping is per-user/per-conversation visibility, and
these three endpoints defeat it. It only looks benign today because the product is
"one active user at a time", but the user table, admin role, and per-user scoping all
exist, so this is a genuine authorization bug.

**Fix:** in each handler, after `GetNoteByID`, verify the note is visible to
`currentUser(r)` (global, owned by the user, or in one of the user's conversations) before
returning/mutating it.

### S3 — Attachment IDOR — **High**

`handleGetAttachment` (`internal/server/attachments.go:11-35`) looks up the attachment by
id and serves the file with **no ownership/conversation check**:

```go
att, err := s.store.GetAttachment(id)
...
http.ServeFile(w, r, filepath.Join(s.cfg.Server.DataDir, att.DiskPath))
```

Any authenticated user can enumerate `/api/attachments/{id}` and read every other user's
generated documents and images. Same root cause as S2.

**Fix:** join through `conversation` and require `conversation.user_id == currentUser.ID`
(or admin) in the query.

### S4 — Passwordless accounts are a silent auth bypass — Medium

`handleLogin` (`internal/server/auth.go:36-42`) skips the password comparison entirely when
the stored hash is nil:

```go
if user.PasswordHash != nil {
    if err := bcrypt.CompareHashAndPassword(...); err != nil { ... }
}
// else: logged in with any (or no) password
```

`bootstrap` (`cmd/lemon-chat/main.go:127-135`) creates the admin user with a nil hash when
`admin_password` is empty, and `handleChangePassword`/`handleAdminUpdateUser` both allow
setting a nil password. So a default deployment with the default username `admin` and no
password is a full, unauthenticated admin login for anyone who can reach the port.

This is arguably intentional for a localhost-only tool (`has_password` is surfaced in the
UI), but it deserves a loud warning at minimum. Recommend logging a prominent warning at
startup when any account has no password, and/or refusing passwordless login unless an
explicit `allow_passwordless` config flag is set.

### S5 — SSRF via `fetch_url` and research fetching — Low/Medium

`fetch_url` (`internal/server/tools.go:802-861`) and the research fetcher
(`internal/research/web.go:81-128`) issue server-side GETs to arbitrary
model/user-supplied URLs with no host allowlist or private-IP filtering. A model (steered
by a malicious web page or character) can make the server fetch `http://localhost:…`,
`http://169.254.169.254/…` (cloud metadata), or other internal services. For a
self-hosted local tool the blast radius is small, but if deployed on a network it's a
classic SSRF. Consider blocking RFC-1918 / loopback / link-local targets for these tools.

### S6 — Content-Disposition header built from filename without escaping — Low

`attachments.go:29-31` interpolates `att.Filename` directly into the `Content-Disposition`
header inside quotes. `create_document` sanitises with `filepath.Base` but that does not
strip `"` or newlines, so a filename like `a".png` corrupts the header. Use
`mime.FormatMediaType` or strip/encode the filename.

### S7 — No WebSocket Origin check — Low

`handleWS` (`internal/server/ws.go:80-131`) authenticates the session cookie but never
validates the `Origin` header (cross-site WebSocket hijacking). Mitigated in practice by
the `SameSite=Strict` session cookie, but an explicit Origin check is cheap defence in
depth.

### Note: login endpoint has no CSRF guard

The `X-Requested-With: XMLHttpRequest` CSRF mitigation in `requireAuth`
(`middleware.go:17-23`) is sensible, but `POST /api/auth/login` is registered without
`requireAuth` (`server.go:39`), so it has neither the header check nor a token. Login CSRF
is low severity, but worth noting.

---

## 2. Potential bugs

### B1 — `title_prompt` is dead code — Medium

Migration v8 added `character.title_prompt`; it's editable in the character editor, sent on
create/update, and stored (`characters.go:70,87,132,166`). The store even exposes
`GetConversationTitlePrompt` (`internal/store/conversations.go:345-357`). **But the title
worker never uses any of it** — both `generateTitle` and `generateCompletionTitle`
(`internal/tasks/titles.go:299-383, 162-229`) use a hardcoded system prompt and never call
`GetConversationTitlePrompt`. So a user setting a custom title prompt on a character gets
no effect whatsoever, and `GetConversationTitlePrompt` is unused. Either wire it in or
remove the field/method and the UI control.

### B2 — Research SSE: subscribe-after-finish race → client hangs — Medium

In `handleResearchEvents` (`internal/server/research.go:499-529`) there's a window between
`run := s.research.get(id)` returning non-nil and `run.subscribe()`. If the engine calls
`run.finish()` (`research.go:56-68`) in that window, `finish` closes all current
subscribers and resets `run.subs` to an empty map. The subsequent `subscribe()` then adds a
channel that will **never be closed** (finish already ran), so the handler's
`for { select { case data, ok := <-ch ... } }` loop only ever delivers the replayed
`run.last` terminal event and then blocks until the client disconnects — the client never
receives `[DONE]`. Low frequency, but it leaves a hung SSE connection. Consider having
`subscribe()` detect the already-finished state (e.g. a `finished` bool) and immediately
return a closed channel after sending `last`.

### B3 — WebSocket broadcast has no write deadline — Medium

`Hub.broadcast` (`internal/server/ws.go:43-55`) copies the connection set, then calls
`sendTextFrame` for each. `sendTextFrame` (`ws.go:139-156`) does raw `conn.Write` with no
deadline. A single slow or half-dead client will block the broadcast loop, delaying (and on
TCP backpressure, effectively stalling) title/conversation-changed notifications for every
other client. Set a write deadline on each `conn.Write`, or push to a per-client buffered
channel with a drop policy (as the research hub already does).

### B4 — User message is persisted before the model is known to be reachable — Low

`handleSendMessage` writes the user message (`messages.go:274`) before the first
`doRequest()` (`messages.go:338`). If the model server is unreachable, the handler returns
502 but the user message is already stored with no assistant reply. On the next load the
conversation shows a dangling user turn. Either persist after a successful response start,
or clean up on early failure.

### B5 — Streaming continues after client disconnect — Low

During the SSE loop in `handleSendMessage`, writes to `w` ignore errors
(`fmt.Fprintf(w, ...)`), and the per-tool executors run on `context.Background()` with their
own timeouts (`tools.go`), not on the request context. So if the user closes the tab
mid-stream, the request `ctx` cancellation reaches the *model* HTTP calls but not the
*tool* executions, and the loop keeps running, executing tools and writing to a dead
`ResponseWriter` until it finishes. Wasted work and possible duplicate tool side-effects
(e.g. image generation). Consider checking `r.Context().Err()` between loop iterations and
threading the request context into `ToolContext`.

### B6 — `note_list` prefix semantics disagree between model and UI — Low

The model-facing `ListNotes` treats a bare term as `g.foo%`/`u.foo%`/`c.foo%`
(`internal/store/notes.go:198-200`) — so `foo` matches `foobar`. The settings UI's
`ListUserVisibleNotes` uses `g.foo.%`/… with a dot (`notes.go:382-384`) — so `foo` only
matches `foo` and `foo.*`, not `foobar`. The same prefix returns different result sets in
the two surfaces. Pick one rule and share it.

### B7 — Migration version-label gap — Low (confusing, not broken)

`store.go:228` jumps straight from the v8 block to `if version < 10` labelled
`"migrating v9 → v10"`. There is no v8→v9 block, so a DB at version 8 jumps to 10. It works
(no DB ever sits at 9), but the label is misleading and a future reader will assume a v9
migration exists. Relabel or add an explicit comment.

### Minor

- `parseJSONObject` shadows the builtin with `close := strings.LastIndex(...)`
  (`internal/research/llm.go:266`). Legal, but a lint smell; rename to `closeIdx`.
- `research.go:266` recomputes `elapsedMS` from `state.ElapsedMS` independently of the
  checkpoint bookkeeping in `Researcher.elapsedMS()` — two sources of truth for the same
  number. Benign today, fragile under change.

---

## 3. Duplicated code

The backend re-implements the same OpenAI-style chat-completions plumbing in at least five
places, each with its own copy of: build payload map → marshal → `NewRequestWithContext` →
set `Authorization` → scan/decode `choices[0]`.

- **D1 — SSE chat-completion call + scanner loop** is duplicated in
  `internal/server/messages.go:311-446`, `internal/server/completions.go:246-335`, and
  `internal/research/llm.go:78-150`. Each re-implements the `bufio.Scanner` with the same
  `1<<20` buffer, the `"data: "` prefix test, and the `[DONE]` sentinel. A shared
  `streamChatCompletion(ctx, url, payload, onDelta, ...)` helper would remove ~150 lines
  and one class of subtle divergence.
- **D2 — Non-streaming chat call + `choices[0].Message.Content` decode** appears in
  `tools.go:summariseHTML` (1320-1376), `research/llm.go:llmCall` (23-72), and twice in
  `tasks/titles.go` (`generateTitle`, `generateCompletionTitle`). Near-identical.
- **D3 — SearXNG search** is implemented twice with separate response structs:
  `tools.go:862-938` (the `searxng` tool) and `internal/research/web.go:23-59`.
- **D4 — HTML stripping** is implemented twice with two regex sets: `tools.go:stripHTML`
  (1305-1318) and `research/web.go:fetchWebpage` (67-128).
- **D5 — Conversation vs completion title generation** (`tasks/titles.go`) is essentially
  the same function pair duplicated for two entity types — the batch worker, the
  fire-and-forget variant, and the HTTP call all exist in near-duplicate form.
- **D6 — Handler boilerplate**: "parse `{id}` → `GetX(id, user.ID)` → `ErrNotFound`→404 →
  err→500" is repeated dozens of times across `conversations.go`, `characters.go`,
  `completions.go`, `research.go`, `notes.go`. A small generic helper would shrink each
  handler noticeably.

None of these are bugs, but D1–D4 are the kind of duplication where a fix applied to one
copy silently misses the others.

---

## 4. Poor organisation / abstractions

- **O1 — `internal/server/tools.go` is 1637 lines** and mixes: tool JSON schemas, all
  executors, three external-service HTTP clients (SearXNG, Wikipedia, ComfyUI), HTML
  stripping, image generation/polling, and attachment writing. This should be split by
  concern (e.g. `tools_registry.go`, `tools_web.go`, `tools_image.go`, `tools_notes_state.go`).
- **O2 — Four parallel tool lists must be hand-synced**: `toolRegistry`, `executors`,
  `allTools` (`tools.go`), and `TOOL_GROUPS` in `settings-character-edit.js`. CLAUDE.md
  documents this as a 5-step manual process, which is itself a sign the abstraction is
  wrong — adding a tool should be one declaration. Evidence it's already drifting: the
  CLAUDE.md tool table omits `note_to_self` and `state_clear`, and documents `world_state`
  as expanding to four tools when the code (`tools.go:555`) includes `state_clear` as a
  fifth.
- **O3 — `handleSendMessage` is a ~430-line function** (`messages.go:195-623`) doing model
  resolution, tool-loop orchestration, SSE framing, DB persistence, and title-trigger
  logic. It's the hardest thing in the repo to reason about or test. Extract the tool loop
  and the title-trigger block at minimum.
- **O4 — Ad-hoc anonymous request structs** are declared inline in nearly every handler.
  Fine individually, but it makes the API surface impossible to see in one place and leads
  to subtle inconsistencies (e.g. the `(req.Model == nil) == (req.CharacterID == nil)`
  XOR check is written three different ways across `conversations.go` and `messages.go`).
- **O5 — Single-connection DB**: `store.Open` sets `SetMaxOpenConns(1)`
  (`internal/store/store.go:22`), so the WAL journal mode is effectively unused for
  concurrency and all DB access is serialised. Correct and safe for SQLite, but it means a
  long write transaction (`ForkConversation`, `ImportConversation`) blocks *all* reads,
  including the auth lookup on every request. Worth a comment explaining the deliberate
  trade-off, and worth keeping those transactions short.

---

## 5. Maintainability / smaller issues

- **U1 — Six Go files are not `gofmt`-clean**: `internal/server/{admin,avatars,messages,
  notes,research}.go` and `internal/store/research.go` (verified with `gofmt -l`). E.g.
  `research.go:346-350` and the `admin.go:151-199` struct literal are misaligned. CI should
  run `gofmt -l` / `go vet` and fail on diff. (`go vet` and `go build` are currently clean.)
- **U2 — Pervasive ignored errors**: `json.Marshal`/`json.Unmarshal`/`fmt.Fprintf(w, …)`
  results are dropped throughout. Most are deliberate and harmless, but the marshalled
  payloads (e.g. `messages.go:313 p, _ := json.Marshal(payloadMap)`) silently produce `nil`
  on failure rather than erroring. A few `//nolint:errcheck` are annotated; the rest are
  inconsistent.
- **U3 — `math/rand` for "binding" game rolls**: `roll_dice`/`random_chance`/`pick_random`
  and image seeds use the global `math/rand` (`tools.go`). Functionally fine (auto-seeded
  since Go 1.20), but the `random_chance` description claims the result is "binding" and
  "server-side random and cannot be influenced" — worth a one-line comment that this is
  PRNG, not crypto, to set expectations.
- **U4 — Magic timeouts** scattered as literals: 120s ComfyUI poll, 30s/15s/5s fetch
  timeouts (`tools.go`), 10s/15s research fetch (`web.go`). The research engine nicely
  centralises its tuning in config — the tool layer should too, or at least name the
  constants.
- **U5 — CLAUDE.md tool documentation is out of date** (see O2): missing `note_to_self`,
  `state_clear`; `world_state` expansion description wrong.
- **U6 — `serveFile` uses relative paths** (`server.go:169-173`, `http.ServeFile(w, r,
  "static/index.html")`) and the static `FileServer` uses `http.Dir("static")`. Both depend
  on the process CWD being the repo root. Fine under the Dockerfile/`run.sh`, but fragile
  for any other launch method; consider resolving against an absolute asset dir from config.
- **U7 — No periodic session cleanup**: expired sessions are deleted lazily on access
  (`sessions.go:41-44`), but a session that's never hit again lingers forever. There's an
  index on `expires_at` (migration v10) but no sweeper. The cleanup worker only handles
  stale conversations. Low impact; the table just grows.
- **U8 — Static directory listing**: the catch-all `http.FileServer(http.Dir("static"))`
  (`server.go:149`) will serve directory listings for any subfolder without an index file.
  Minor information disclosure (it's all client code anyway), but tidy to disable.

---

## What's good (worth preserving)

- Migration discipline is exemplary: numbered, logged, append-only, with data-transform
  migrations (v17, v19) done carefully and idempotently.
- The research engine (`internal/research`) is genuinely well-factored: clear phase
  separation, resumable checkpointed state, robust LLM-output parsing
  (`parseJSONStringArray`), and a real prompt-injection guard (`untrustedContextMessage`).
- Foreign-key + cascade design plus the explicit shared-attachment check in
  `DeleteConversation` (`conversations.go:85-103`) shows care about fork semantics.
- The frontend honours the no-build-step / vanilla-ES-modules constraint cleanly, and the
  streaming/tool-call rendering in `thread.js` is careful about scroll intent and avatar
  consecutiveness.
- Logging conventions from CLAUDE.md are followed consistently across handlers.

---

## Suggested priority order

1. **S1** (XSS — sanitise markdown). Highest impact, contained fix.
2. **S2 / S3** (notes & attachment IDOR — add ownership checks). Same pattern, do together.
3. **S4** (warn/guard on passwordless accounts).
4. **B1** (decide: wire up or remove `title_prompt`).
5. **B2 / B3** (research SSE race; WS write deadline).
6. `gofmt` the six files and add `gofmt`/`vet` to CI (**U1**).
7. Refactor the duplicated chat-completion plumbing (**D1/D2**) and split `tools.go`
   (**O1/O2**) when next touching that area.
