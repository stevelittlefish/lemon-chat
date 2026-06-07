# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

---

## Critical bugs

- [x] **`isDuplicateUsername` checks the wrong table name** (`internal/server/admin.go:15`)
  The unique-constraint string is `"UNIQUE constraint failed: users.username"` but the table is named `user` (singular). SQLite emits `user.username`, so the check always returns `false`. Creating a duplicate username via the admin API returns 500 instead of 409.

- [x] **`onNewTurn` is missing from `api.js:send()` destructured params** (`static/js/api.js:73`)
  `onNewTurn` is referenced inside the function on line 113 (`if (new_turn) onNewTurn?.(new_turn)`) but is not in the destructured parameter list, so it is always `undefined` inside the function regardless of what the caller passes. The callback never fires, meaning multi-turn tool calls never start a new message bubble — all follow-up responses stream into the same element as the tool-call list.

- [x] **Missing character visibility check in `handleSendMessage`** (`internal/server/messages.go:84`)
  Any authenticated user can invoke a private character they don't own by passing `character_id` in the message body. Add the same visibility check that `handleFirstMessage` already has.

- [x] **`subtle.ConstantTimeCompare` compares string to itself** (`internal/server/auth.go:38`)
  The "belt-and-suspenders" call compares `req.Password` to itself and discards the result — it does nothing. Remove it; bcrypt is already constant-time.

---

## High priority

- [x] **`ForkConversation` is not wrapped in a transaction** (`internal/store/conversations.go:111`)
  Creates the new conversation and inserts messages in separate calls. A failure mid-insert leaves an empty forked conversation in the DB. Wrap the whole thing in a `db.Begin()` / `tx.Commit()`.

- [x] **SQLite busy errors under concurrent writes** (`internal/store/store.go:16`)
  `sql.Open` uses WAL mode but sets no `_busy_timeout` and no `SetMaxOpenConns(1)`. Concurrent writers (SSE, title goroutines, WebSocket broadcasts) immediately get `SQLITE_BUSY` instead of waiting. Add `&_busy_timeout=5000` to the DSN and call `db.SetMaxOpenConns(1)` after opening.

- [x] **Title generation can fire twice concurrently on one request** (`internal/server/messages.go:270–294`)
  The `auto_title` and "3rd assistant response" triggers both evaluate `conv.Title == nil` from the pre-request snapshot. A conversation that satisfies both conditions spawns two goroutines that race to write the title. Add a guard so only one fires.

- [x] **No timeout on model API calls** (`internal/server/messages.go:179`, `internal/tasks/titles.go:161`)
  Both use `http.DefaultClient` which has no timeout. A hung model server holds goroutines and SSE connections indefinitely. Use a custom `http.Client` with a dial timeout, or pass a `context.WithTimeout` for non-streaming calls (title generation especially).

- [x] **Make LLM API timeouts configurable** (`internal/config/config.go:31`, `internal/server/messages.go:27`, `internal/tasks/titles.go:153`)
  The dial timeout (10 s) and title-generation timeout (30 s) are hardcoded. Add optional `dial_timeout_seconds` and `response_timeout_seconds` fields to `ModelServer` in the config so operators can tune them per server; fall back to the current hardcoded values when unset.

- [x] **`bufio.Scanner` 64 KB limit on SSE stream** (`internal/server/messages.go:199`)
  The default scanner buffer is 64 KB. A `data:` line longer than that causes `Scan()` to silently return false mid-response. Set a larger buffer (`scanner.Buffer(make([]byte, 1<<20), 1<<20)`) and check `scanner.Err()` after the loop.

---

## High priority

- [x] **`ForkConversation` drops `tool_calls` and `tool_call_id`** (`internal/store/conversations.go:162`)
  The fork SELECT and INSERT both omit `tool_calls` and `tool_call_id` columns. Forking a conversation that used tools produces messages where the assistant's tool-call fields are NULL and tool-result messages have no `tool_call_id`, making the forked history malformed when the model is asked to continue.

- [x] **`DeleteConversation` doesn't clean up attachments** (`internal/store/conversations.go:66`)
  The transaction deletes messages then the conversation, but the `attachment` table references `conversation_id` with no `ON DELETE CASCADE`. Attachment DB rows and files in `data/attachments/` are never cleaned up when a conversation is deleted. This is a permanent disk and DB leak.

- [x] **Attachment directory is relative to working directory** (`internal/server/tools.go:242`)
  `filepath.Join("data", "attachments", randomID())` is resolved relative to the process working directory. The avatar directory correctly anchors to the DB path (`filepath.Join(filepath.Dir(s.cfg.Server.DBPath), "avatars")`). Attachment paths should use the same anchor, otherwise files are written to (and served from) the wrong place if the server is not started from the project root.

- [x] **Finish completions UI: auto-save content and sync model picker on load** (`static/js/complete-app.js:244,352`)
  Typed content is never saved unless Run is pressed — navigating away discards it. `loadCompletion` also never calls `header.setSelection()` with the fetched `comp.model`, so the header picker always shows the default model rather than the completion's stored model.

- [x] **Inline title edit exits immediately when clicking to reposition the cursor** (`static/js/sidebar.js`)
  The `blur` event fires (or the page navigates) when the user clicks inside the inline title input to move the cursor. `stopPropagation` on `mousedown` and `click` did not fix it. The input lives inside an `<a>` element with an `href`; the browser may be following the link before the input can intercept the event. Needs proper investigation — browser devtools event breakpoints recommended.

## Medium priority

- [x] **`GetCompletion` converts all DB errors to `ErrNotFound`** (`internal/store/completions.go:46`)
  `if err != nil { return nil, ErrNotFound }` swallows real database errors (connection failures, schema mismatches). Any underlying error surfaces as a 404 instead of a 500. Compare with `GetConversation` which correctly uses `errors.Is(err, sql.ErrNoRows)`.

- [x] **`handleGetMessageContext` silently ignores character fetch errors** (`internal/server/messages.go:634`)
  Character system prompt and hidden messages are wrapped in `if err == nil { ... }`, so a DB error causes them to be silently omitted. The returned context will differ from what the model actually received. The pattern used in `resolveCharacter` (return an HTTP error on failure) should be used here instead.

- [x] **Context modal shows blank for tool-call messages** (`static/js/thread.js:764`)
  The context viewer renders `pre.textContent = msg.content`. For assistant messages that triggered tool calls, `content` is `""` and the data lives in `msg.tool_calls`. These appear as empty blocks. The modal should also render `msg.tool_calls` (and `msg.tool_call_id` for tool-result messages) when present.

- [x] **No CSRF protection on mutation endpoints**
  Auth is entirely cookie-based. `SameSite: Strict` helps with modern browsers but any state-mutating endpoint is reachable from non-browser clients without a CSRF check. A double-submit cookie or a required custom header (`X-Requested-With`) is conventional for cookie-authed APIs.

- [ ] **Completion page errors are invisible to the user** (`static/js/complete-app.js`)
  Most async failures in the completions page log to `console.error` but show nothing to the user — save failures, model-change failures, run errors, reload failures. Add a toast or status area to surface these errors, matching the inline error display the chat page provides.

- [ ] **`sidebar.addItem` doesn't deduplicate** (`static/js/sidebar.js:275`)
  `state.items.unshift(item)` with no check. If called twice for the same conversation ID (e.g., a race between a local `addItem` and a WebSocket `conversations_changed` reloading the sidebar), duplicate entries appear in `state.items` and in the DOM until the next full `renderList()`.

- [x] **`DeleteConversation` never returns `ErrNotFound`** (`internal/store/conversations.go:66`)
  Uses `db.Exec` without checking `RowsAffected`. Deleting a nonexistent conversation returns 204 instead of 404. Check rows affected and return `ErrNotFound` when zero.

- [x] **Duplicate character-loading block in `handleSendMessage`** (`internal/server/messages.go:84–133`)
  The character fetch + system message + hidden messages block is copy-pasted verbatim twice (once for `req.CharacterID`, once for `conv.CharacterID`). Extract a helper function; this also makes the visibility fix from the critical bug above easier to apply in one place.

- [x] **`UpdateConversationAfterMessage` error is silently ignored** (`internal/server/messages.go:258`)
  `_ = s.store.UpdateConversationAfterMessage(...)` — if this fails, the conversation's model/character metadata silently drifts. At minimum log the error.

- [x] **Missing database indexes on frequently queried columns** (`internal/store/store.go`)
  Add indexes on:
  - `message(conversation_id)` — scanned on every message list and send
  - `conversation(user_id)` — scanned on every conversation list
  - `conversation(created_at)` — used by stale cleanup and untitled eligible queries
  - `session(expires_at)` — scanned on every auth check

- [x] **WebSocket pong does not validate ping payload ≤ 125 bytes** (`internal/server/ws.go:198`)
  RFC 6455 §5.5: control frame payloads must not exceed 125 bytes. The pong is built as `append([]byte{0x8A, byte(length)}, payload...)` without checking length. Add `if length > 125 { return }` before building the pong.

- [x] **`defer resp.Body.Close()` inside loop in `listModels`** (`cmd/lemon-chat/main.go:87`)
  `defer` inside a loop defers until the function returns, not the end of each iteration. All response bodies stay open until `listModels` exits. Call `resp.Body.Close()` explicitly at the end of each loop body.

---

## Low priority / polish

- [x] **O(n²) string concatenation in streaming loop** (`internal/server/messages.go:235`)
  `fullContent += text` in a loop allocates on every iteration. Use `strings.Builder`.

- [x] **`isImageExt()` is defined but never called** (`internal/server/avatars.go:115`)
  Dead code — remove it.

- [x] **`mimeFromFilename` non-JPEG cases are unreachable** (`internal/server/avatars.go:27`)
  `receiveAvatar` always returns `".jpg"`, so only the JPEG branch is ever hit. Remove the dead cases or add a comment explaining the function is currently JPEG-only.

- [x] **`go.mod` marks direct dependencies as `// indirect`** (`go.mod`)
  `BurntSushi/toml`, `golang.org/x/crypto`, and `golang.org/x/image` are imported directly. Run `go mod tidy` to fix.

- [x] **Title generation uses `http.NewRequest` without a context** (`internal/tasks/titles.go:153`)
  Use `http.NewRequestWithContext` with a timeout context so a slow model server doesn't hang the worker goroutine indefinitely.

- [x] **`time.Parse` error ignored for `prevUpdatedAt`** (`internal/server/messages.go:256`)
  If `conv.UpdatedAt` is malformed, `prevUpdatedAt` is the zero time and `BroadcastConversationListChanged` fires on every message. Handle or at least log the parse error.

- [x] **`prompt()` / `confirm()` dialogs in sidebar** (`static/js/sidebar.js:47, 62`)
  Native browser dialogs are blocking and visually inconsistent with the rest of the UI. The title edit would be better as an inline editable field; the delete confirm should use the same modal pattern as the fork dialog.

- [x] **Full `innerHTML` re-render of sidebar on every state change** (`static/js/sidebar.js:132`)
  `render()` rebuilds the entire sidebar and re-attaches all event listeners on `setActive()`, `addConversation()`, etc. Targeted DOM updates (as already done in `updateTitle()`) would be better.

- [x] **`escHtml` vs `escapeHtml` — same function, two names** (`static/js/thread.js:112`, `static/js/sidebar.js:195`)
  Identical implementations with inconsistent names. Extract to a shared module or at least pick one name.

- [x] **`sendMessage` in `app.js` is missing a try/catch** (`static/js/app.js:185`)
  If `convApi.create()` throws, the exception propagates with no user feedback and the composer is left in a broken state.

- [x] **`initApp` does not handle `Promise.all` rejection** (`static/js/app.js:93`)
  If model or character list fetch fails on startup, the whole app silently fails to initialise. Add a catch that shows the user a meaningful error.

- [x] **`applyFirstMessage` swallows all errors** (`static/js/app.js:254`)
  The catch block is intentionally silent for expected 409/400 responses, but also hides network errors and server errors. Narrow the suppression to expected status codes.

- [x] **Improve auto-scroll stop sensitivity during streaming** (`static/js/thread.js:52-58`)
  The `scroll` listener only sets `userScrolledDuringStream` when the user is more than 40 px from the bottom, and the `programmaticScroll` flag is cleared only after a `requestAnimationFrame` — so a user scroll that lands during that window is silently ignored and the auto-scroll fights back. Consider using a `wheel` or `pointerdown` event to detect intent before the position changes, and raise or remove the `isNearBottom` threshold guard.

- [ ] **`mimeFromFilename` (avatars.go) and `mimeTypeForFilename` (tools.go) are duplicate MIME helpers** (`internal/server/avatars.go:29`, `internal/server/tools.go:193`)
  Both exist in the same package. The avatars version uses `strings.Split` instead of `filepath.Ext` and only handles JPEG. Remove `mimeFromFilename` and call `mimeTypeForFilename` from avatars.go instead.

- [ ] **`noCacheMiddleware` is defined but never used** (`internal/server/middleware.go:56`)
  Dead code — remove it. The individual route handlers set `Cache-Control: no-cache` inline where needed.

- [ ] **SSE streaming logic copy-pasted between `messages.send` and `completions.run`** (`static/js/api.js:79, 152`)
  Both functions contain identical boilerplate: getReader → TextDecoder → buffer → split on newlines → strip `data: ` prefix → check `[DONE]` → parse JSON. Only the extracted fields differ. Extract a shared `consumeSSE(res, onEvent)` helper that calls `onEvent(parsed)` for each chunk.

- [ ] **Modal construction pattern repeated four times** (`static/js/thread.js`, `static/js/sidebar.js`)
  `getMdModal()`, `getForkModal()`, `getCtxModal()`, and `getDeleteModal()` each implement the same lazy-singleton: create overlay, wire backdrop-click, add Escape listener, append to body, cache result. The four Escape-key listeners also accumulate permanently on `document`. A small `createModal({ title, body, actions })` factory would remove the duplication and fix the listener leak.

- [ ] **`handleUndo` and `handleRedo` in complete-app.js are 95% identical** (`static/js/complete-app.js:569, 590`)
  Both functions differ only in which API is called and which direction the `undone` flag goes. Steps 3–10 are identical (reset state, set textarea, setMode, renderControls, updateUndoBtn, scroll). Merge into a shared `performUndoRedo(forward)` function.

- [ ] **`character_hidden_message` appears in both v0→v1 and v8→v9 migrations** (`internal/store/store.go:99, 227`)
  The table is created in the initial v1 schema block AND in a separate v9 migration using `CREATE TABLE IF NOT EXISTS`. The v9 block is now a no-op for any install that ran the current v1. Remove the v9 block (or add a comment explaining why it exists) to clarify migration ownership.

- [ ] **`header.js` drops the open model-picker dropdown on every WebSocket title update** (`static/js/header.js:80`)
  `render()` writes `headerEl.innerHTML = ...`, destroying all child elements including any open dropdown. `updateTitle()` already has a targeted path for title-only changes — `setConversation()` should call `updateTitle()` when only the title changes rather than calling `render()`.

- [ ] **Session cookie lacks `Secure` flag** (`internal/server/auth.go:51`)
  If the server is ever accessed over HTTP (common for local/self-hosted use), the session cookie is transmitted in plaintext. Add `Secure: true` and document that it requires HTTPS, or make it configurable.

- [ ] **Settings pages each independently check auth** (`static/js/settings-account.js`, `static/js/settings-character-edit.js`, etc.)
  Every settings page calls `auth.me()` and redirects to `/` on failure. A shared `requireAuth()` module that throws/redirects would centralize this and ensure consistency if the redirect target changes.

- [ ] **Show per-message timestamps in the thread** (`static/js/thread.js:116,252,535`)
  Add a subtle timestamp (e.g. "2:34 pm") to each message using `msg.created_at`. When the date changes mid-conversation, insert a date separator (e.g. "Tuesday 3 June") between messages so the day boundary is visible. Should be unobtrusive — small, muted text — but legible without hovering.

- [x] **`window.location.reload()` after logout** (`static/js/sidebar.js:191`)
  Redirect to `/` instead to clear the `?c=…` query param and avoid a 401 redirect loop on reload.

- [x] **`noCacheMiddleware` applied to all static assets including vendored libraries** (`internal/server/server.go:87`)
  Vendored files like `marked.esm.js` and `katex.min.css` never change between requests and should be cached aggressively. Apply `no-cache` only to HTML pages; serve static assets with `Cache-Control: max-age=…` or at least conditionally by path.

- [x] **`err.message` used as `innerHTML` without escaping** (`static/js/thread.js:518`)
  Error messages from the server may contain `<` or `>` characters. Escape before inserting into the DOM, or use `.textContent`.

- [ ] **Add request-level logging to all non-frequent handlers** (`internal/server/conversations.go`, `internal/server/messages.go`, `internal/server/auth.go`)
  Most handlers (create/delete conversation, send message, logout, update profile, change password, fork, first-message, etc.) emit no log line. Per the logging conventions in CLAUDE.md, each should log one `log.Printf` line on entry with the relevant IDs and username.

- [x] **Sidebar footer invisible until first conversation is selected** (`static/js/app.js:93`)
  `sidebar.init()` renders the sidebar before `preloadIcons()` resolves, so icon SVGs are empty strings and the footer buttons are invisible. A re-render only happens when `setActive()` is called after selecting a conversation. Fix by re-rendering the sidebar after the `Promise.all` resolves, or by awaiting icon preload before calling `sidebar.init()`.
