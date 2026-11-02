# lemon-chat code review: 2026-11-02

**Scope:** whole repo at `2a32329`: Go backend (`cmd/`, `internal/`), vanilla-JS frontend (`static/js`), schema/migrations.
**Companion report:** `docs/code_review_2026-11-02.html` (human-readable version of this file).
**Previous review:** `docs/archive/code_review_2026-06-13.md`.

This file is the working reference. Each finding has an ID, a status box, `file:line` anchors, how it was verified, and a suggested fix. Tick the box and add the commit hash when a finding is fixed.

## Tooling baseline

| Check | Result |
|---|---|
| `go vet ./...` | clean |
| `go test ./...` | all 10 packages pass |
| `go test -race ./...` | clean |
| `staticcheck` / `govulncheck` | **not run** (fetching the tools was blocked in this session) |
| Direct Go deps | 4 (`toml`, `x/crypto`, `x/image`, `modernc sqlite`) |
| Size | ~21k lines Go (28 test files), ~7.5k lines JS |

"Confirmed" below means it was reproduced with a throwaway store-level test, deleted afterwards. "Read" means it was established by reading the code.

## Overall verdict

The codebase is good quality: idiomatic Go, very few dependencies, careful migrations, real tests, and comments that explain *why*. All three High findings from June are fixed: DOMPurify sanitising, notes IDOR, and attachment IDOR. So are the research-SSE race, the WS write deadline, the WS origin check, and the dead `title_prompt`.

The weak spots form a pattern:

1. **Multi-user boundaries:** the WS hub broadcasts every event to every user, and model-authored HTML is served same-origin.
2. **Row lifecycle versus foreign keys:** cleanup, user deletion, and fork were not updated when `attachment.message_id`/uploads and sessions arrived.
3. **Side paths that bypass the provider abstraction:** titles and `fetch_url` summarisation.

None of this points to a rotten foundation. They're localised fixes.

---

## High

### [ ] H1: Model-authored HTML attachments served inline, same-origin (stored XSS)
- `internal/server/tools.go:72` maps `.html` to `text/html`. `internal/server/attachments.go:132-161` serves attachments `inline` with the stored MIME, with no CSP and no `nosniff`.
- Chain: a prompt-injected page (`fetch_url`/searxng) or a shared character makes the model call `create_document` with `x.html` containing script. The model then writes `[open](/api/attachments/N)` in its reply. DOMPurify keeps relative links, and `markdown.js` opens them in a new tab. Same origin plus a SameSite=Strict cookie sent on same-site navigation means the script runs as the user and can call the whole API (the `X-Requested-With` check doesn't stop same-origin JS).
- The artifact panel itself is safe because it renders as `<pre>`/markdown. Only direct navigation to the URL is the problem.
- **Fix:** in `handleGetAttachment`, set `X-Content-Type-Options: nosniff` and `Content-Security-Policy: sandbox; default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'`, the same approach as `research_reports.go:455`. Alternatively, force `Content-Disposition: attachment` / `text/plain` for anything that isn't `image/*` (excluding SVG) or `text/plain`/`text/markdown`.
- Verified: read.

### [ ] H2: WebSocket hub broadcasts every event to every connected user
- `internal/server/ws.go:48` `broadcast()` sends to all conns. The hub doesn't know which user owns which connection (`handleWS` validates the session at `ws.go:117` but discards the user id).
- Leaks to all other logged-in users: conversation titles (`conversation_titled`, including those from the title worker), completion titles, attachment titles and filenames (`attachment_ready`), background changes, and conversation ids. The README promises "each user's conversations and completions are private to them."
- **Fix:** store `userID` per conn (`map[net.Conn]int64`), and give each `Broadcast*` a `userID` to target. Callers already know the owner: conv/completion owner, `ToolContext.UserID`. The title worker needs the conversation's `user_id`.
- Verified: read.

---

## Medium

### [ ] M1: Stale-conversation cleanup breaks permanently once a stale conversation has an upload
- `internal/store/conversations.go:152` deletes `message` rows, then `conversation` rows, without deleting `attachment` rows. `attachment.conversation_id` and `attachment.message_id` are FKs without cascade, and `foreign_keys(1)` is on.
- Scenario: the user creates a chat, pastes or uploads an image, then leaves without sending. After 30 minutes the cleanup fails with `FOREIGN KEY constraint failed`, every minute, forever. No stale conversation gets cleaned again, because it's one statement for all of them.
- Also: the two DELETEs aren't in a transaction, and unsent uploads (`message_id IS NULL`) and their files are never garbage-collected.
- **Fix:** do it in one tx: collect attachment disk paths, delete attachments, then messages, then conversations. Remove the files after commit (reuse the `DeleteConversation` logic). Add a sweep for unlinked uploads older than N hours.
- Verified: **confirmed** (`FOREIGN KEY constraint failed (787)`).

### [ ] M2: Admin "delete user" fails for any user who has ever logged in
- `internal/store/users.go:63` is a bare `DELETE FROM user`. `session.user_id`, `conversation.user_id` and `character.created_by` reference `user` without cascade, so the delete returns a 500.
- `internal/server/admin.go:235` deletes the avatar file *before* the failing DB call, so the user keeps their account but loses the avatar.
- **Fix:** decide the policy (cascade the user's conversations, messages and attachments; reassign or delete characters). Do it in a tx, then delete the avatar after commit. Ask before adding a destructive migration (per AGENTS.md); this can be done in application code without a schema change.
- Verified: **confirmed** for both the session-only and conversation cases.

### [ ] M3: Forking a conversation drops user-uploaded images
- `internal/store/conversations.go:228-235` copies only attachments whose `tool_call_id` matches a message. Uploads have `tool_call_id = ''` and are keyed by `message_id`.
- The fork shows no thumbnails, and the model loses the vision context on the next turn. The fork also doesn't copy `source`/`status`/`message_id`/`background_attachment_id`.
- **Fix:** build an old-to-new message-id map during the message copy loop (`LastInsertId`). Copy upload attachments with the remapped `message_id` and `source='upload'`, and carry over `status`. Files are shared by `disk_path`, which `DeleteConversation` already handles.
- Verified: **confirmed** (fork had 0 attachments, the source had 1).

### [ ] M4: Title generation and `fetch_url` summarisation bypass the Provider (broken on OAuth/Responses servers)
- `internal/tasks/titles.go:53,264` and `internal/server/tools_web.go:361` POST to `APIBase + "/chat/completions"` with the static `srv.APIKey` via `http.DefaultClient`. They ignore `api = "responses"`, `auth = "oauth"`, and the token source and account id.
- If `default_model` or the chatting model lives on the Codex/OAuth server, auto-titles never work and `fetch_url` returns an error instead of a page.
- Compounding: `generateTitles` retries every eligible untitled conversation **every 30 s forever** with no backoff or failure memory. That spams the model server and the log when titles keep failing.
- **Fix:** route both through `llm.NewProvider(...).Stream` (or add a non-streaming `Complete` to the Provider interface). Pass the server's `tokenSource`/`oauthAccountID`, which means `tasks` needs access to them (inject a function from `server`). Add a per-conversation failure backoff, e.g. an in-memory `map[id]nextAttempt`.
- Verified: read.

### [ ] M5: Avatar upload is vulnerable to decompression bombs
- `internal/server/avatars.go:73` calls `image.Decode` on user input with only a 5 MB *file* limit. A small PNG or GIF that declares e.g. 30000×30000 makes the decoder allocate the full raster (GBs), and any authenticated user can OOM the process.
- **Fix:** call `image.DecodeConfig` first and reject when `Width*Height` exceeds, say, 40 MP. (Vision uploads don't decode, so they're fine.)
- Verified: read.

### [ ] M6: `fetch_url` SSRF to localhost/LAN (unchanged since the June review, S5)
- `internal/server/tools_web.go:45` fetches any model-supplied URL. That covers `http://localhost:*`, the LAN (Home Assistant, NVR, router, ComfyUI) and cloud metadata. A prompt-injected page can make the model read an internal URL and exfiltrate it through a second `fetch_url` with the data in the query string. Same for research page fetches (`internal/research/web.go:41`).
- **Fix:** a custom `net.Dialer.Control` that rejects loopback, private, link-local and CGNAT IPs after DNS resolution (which also handles redirects and DNS rebinding). Add a config allowlist (e.g. `[tools] allow_private_fetch = true`) for people who want it.
- Verified: read.

### [ ] M7: WebSocket writes are unsynchronised, and broadcasts block request handlers
- `ws.go:184` `sendTextFrame` writes the header and payload as two `conn.Write` calls. `broadcast` is called concurrently from many goroutines (handlers, title worker, image goroutines, cleanup worker), and `drainFrames` writes pongs (`ws.go:258`). Concurrent writers can interleave frames and corrupt the stream, causing client reconnect storms.
- `broadcast` runs synchronously in the caller, with a 5 s write deadline *per* stalled client. One dead tab can add 5 s or more to `PATCH /api/conversations/{id}`, the end of a chat stream, and so on.
- **Fix:** give each client a goroutine with a buffered send channel (drop or close on overflow). That fixes both problems, and pairs naturally with H2.
- Verified: read.

### [ ] M8: Mid-stream model failures are treated as success
- `internal/llm/provider.go:504`: if the stream errors *after* any text or tool-call delta, the error is discarded and a normal `Completion` is returned. So:
  - `flushOnError` isn't called. That contradicts AGENTS.md ("every model request that fails … mid-stream error … is appended to `model_errors.log`").
  - `handleSendMessage` persists the truncated text as a complete assistant message, with no indication to the user.
  - The same happens on `response_timeout_seconds` expiry mid-answer.
- **Fix:** return the partial `Completion` **and** the error. In `messageToolLoop.run`, persist the partial content (it's useful), but emit an SSE `error` event, log it, and flush the error log.
- Verified: read.

---

## Low

### [ ] L1: The final tool-loop iteration runs tools and then discards their results
`messages.go:330-342`: on iteration `maxLoops-1` with tool calls, the tools execute (side effects, image generation) and results are persisted, but the model is never called again. `finalContent` is the pre-tool text, often empty, so no assistant message is stored and the user sees nothing. **Fix:** on the last iteration, send the request with `Tools: nil` so the model must answer, or make one extra tool-less call after the limit.

### [ ] L2: Passwordless accounts accept any password
`auth.go:35` skips the bcrypt check when `PasswordHash == nil`, and bootstrap creates a passwordless admin if `admin_password` is unset (`main.go:149`). This is fine on a single-user localhost setup, but dangerous on a LAN. **Fix:** at minimum, log a loud startup warning when any admin has no password. Consider refusing login without a password unless `[server] allow_passwordless = true`.

### [ ] L3: `PUT /api/notes` silently overwrites read-only notes
`notes.go:88` → `UpsertNoteAdmin` ignores the read-only flag (the comment says "handler has already checked permissions", but it hasn't). The UI disables edit for read-only notes, so this is API-only. **Fix:** reject updates to read-only notes unless the request also sets `read_only: false`, which makes unlocking explicit.

### [ ] L4: Message order relies on unspecified tie-breaking
`store/messages.go:32` uses `ORDER BY created_at ASC`, which has second resolution. Tool-call, tool-result and final messages often share a second, and `ForkConversation` writes **every** message with the same timestamp. It works today because SQLite's temp-B-tree sorter happens to preserve rowid order (confirmed with 300 forked messages). **Fix:** `ORDER BY created_at, id` (or just `id`), here and in `conversations.go:205`.

### [ ] L5: No HTTP hardening at the server level
`main.go:75` uses bare `http.ListenAndServe`: no `ReadHeaderTimeout` (slowloris), no graceful shutdown. JSON handlers don't use `http.MaxBytesReader`. Multipart uploads have no body cap (`ParseMultipartForm` spills to temp files, and the size check runs after the full upload). **Fix:** use an `http.Server{ReadHeaderTimeout: 10s}`, plus a body-limit middleware (e.g. 1 MB for JSON, 11 MB for upload routes).

### [ ] L6: Mixed not-found sentinels give 500s
Attachment queries return `sql.ErrNoRows` (`store/attachments.go:169`) where everything else returns `store.ErrNotFound`. `handleSetConversationBackground` uses `notFoundOr500`, so a bad id returns 500. `resolveCharacter` (`messages.go:115`) returns 500 when a character was deleted. **Fix:** normalise attachment queries to `ErrNotFound`.

### [ ] L7: Streaming re-renders the whole message on every token
`static/js/thread.js:573` runs a full `marked` parse, DOMPurify, KaTeX and an `innerHTML` swap per delta. That's O(n²) over a long answer, janky on mobile, and it kills text selection mid-stream. **Fix:** coalesce with `requestAnimationFrame`, rendering at most once per frame.

### [ ] L8: `streamSSE` swallows handler exceptions
`static/js/api.js:22`: the `try/catch` wraps `onEvent(...)` as well as `JSON.parse`, so bugs in thread/composer event handlers vanish silently. **Fix:** parse inside the try, and call the handler outside it.

### [ ] L9: An untrusted tool-call index can force a huge allocation
`llm/provider.go:491` grows `calls` up to `tc.Index`. A buggy or hostile server sending `index: 1e9` causes an OOM. **Fix:** cap the index (e.g. 128) and skip the delta otherwise.

### [ ] L10: Small authorisation and robustness nits
- `handleServeCharacterAvatar` (`characters.go:191`) serves private characters' avatars to anyone.
- `handleCreateConversation` / `handleImportConversation` accept a `character_id` the user can't see (the send path rejects it later; better to validate up front), and an unconfigured `model`.
- `handleDeleteConversation` (`conversations.go:261`) runs `RemoveAll(Dir(disk_path))` with no sanity check that the dir is under `attachments/`. It's safe today, but one bad row would mean `RemoveAll(DataDir)`.
- `w.(http.Flusher)` unchecked assertions (`messages.go:629`, research SSE) will panic if a wrapping middleware is ever added.
- Admin note-pack import isn't transactional, so a partial import on error is possible.

### [ ] L11: Logging convention drift
AGENTS.md says to use gerund lines without a package prefix for handlers. Exceptions: `auth: user %q logged in` / `auth: failed login` (`auth.go`), `characters: uploaded avatar` (`characters.go`), `server: handleAdminListModels —` (`admin.go`). Admin create, update and delete user emit **no** log line at all (the convention requires them for admin actions).

---

## Maintainability and structure

- **Two LLM call paths.** `llm.ChatComplete(url, apiKey)` and `llm.Provider` coexist. The raw path is the root cause of M4. Fold the stragglers (titles, summariseHTML, and anything in `research/llm.go` still using raw URLs) into the Provider, then delete or narrow `ChatComplete`.
- **`http.DefaultClient` in 8 places** (`tools_web.go`, `tools_image.go`, `searx`, `research/web.go`, `titles.go`, `prices.go`). There's no shared transport config, and timeouts depend on every caller remembering a ctx. Pass a client in, and give it the SSRF guard from M6.
- **Large files mixing concerns.** `static/js/thread.js` (1302 lines: artifact panel, lightbox, model picker, streaming, scroll, context modal), `research-app.js` (1179), `internal/research/researcher.go` (1604), `internal/server/research.go` (1060). `thread.js` is the best candidate: split out `lightbox.js` (just touched in `2a32329`), `artifact-panel.js` and `picker.js`, per the project's one-concern-per-file rule.
- **AGENTS.md code layout is stale.** It omits `internal/llm`, `openai_auth`, `searx`, `htmltext`, `redditimport`, `server/notes.go`, `tools_*.go`, `research_*.go`, `errorlog.go`, `oauth_login.go`, plus several JS/settings pages. Agents rely on that map.
- **Migrations aren't transactional per step.** If a multi-statement step fails halfway (e.g. v43 adds `message_id`, then `source` fails), it leaves a column without a version row, and the next start fails with "duplicate column". Wrapping each `if version < N` block in a tx is cheap; SQLite DDL is transactional.
- **Repo weight.** `benchmark/` (4.4 MB, ~100 files of model outputs) and the `docs/choose_your_own` / `docs/adventures` content aren't source. Consider a separate repo, or at least `.gitattributes linguist-generated`.
- **Tests.** Store, llm, research and openai_auth are well covered. The chat send path (`handleSendMessage` + tool loop) has only helper-level tests, and there are no frontend tests at all. Every Medium bug above would have been caught by a store test along the lines of "create user/conv/upload, then delete/fork/cleanup". Add those first.

## What's good (keep doing this)

- Prior review findings were actually fixed, not just triaged.
- Dependency discipline: 4 direct deps, a hand-rolled minimal WS, no frontend build.
- Comments explain non-obvious decisions (nil `*os.File` in an `io.Writer`, lexicographic RFC3339 comparison in session sweep, the Codex `session-id` header).
- Migration hygiene: the `research_job` rebuild uses a tx plus `PRAGMA foreign_key_check`. FKs, WAL and busy_timeout are on.
- Research engine: checkpoint/resume after a crash, bounded extraction concurrency, sandbox CSP on generated HTML reports.
- Provider seam with redacted wire transcripts and an always-on error log.
- Frontend: consistent `escapeHtml` in template literals, DOMPurify on all markdown, progress lines via `textContent`.

## Suggested order

1. H1 (one header block), H2 + M7 together (hub rewrite).
2. M1, M2, M3: store fixes, each with a regression test.
3. M4 (provider consolidation + title backoff), M8.
4. M5, M6, L5 (hardening).
5. The rest as convenient; update the AGENTS.md layout.
