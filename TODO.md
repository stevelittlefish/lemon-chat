# lemon-chat — TODO

Status markers: `[ ]` not started · `[~]` in progress · `[x]` done

---

## Critical bugs

- [x] **Missing character visibility check in `handleSendMessage`** (`internal/server/messages.go:84`)
  Any authenticated user can invoke a private character they don't own by passing `character_id` in the message body. Add the same visibility check that `handleFirstMessage` already has.

- [ ] **`subtle.ConstantTimeCompare` compares string to itself** (`internal/server/auth.go:38`)
  The "belt-and-suspenders" call compares `req.Password` to itself and discards the result — it does nothing. Remove it; bcrypt is already constant-time.

---

## High priority

- [ ] **`ForkConversation` is not wrapped in a transaction** (`internal/store/conversations.go:111`)
  Creates the new conversation and inserts messages in separate calls. A failure mid-insert leaves an empty forked conversation in the DB. Wrap the whole thing in a `db.Begin()` / `tx.Commit()`.

- [ ] **Title generation can fire twice concurrently on one request** (`internal/server/messages.go:270–294`)
  The `auto_title` and "3rd assistant response" triggers both evaluate `conv.Title == nil` from the pre-request snapshot. A conversation that satisfies both conditions spawns two goroutines that race to write the title. Add a guard so only one fires.

- [ ] **No timeout on model API calls** (`internal/server/messages.go:179`, `internal/tasks/titles.go:161`)
  Both use `http.DefaultClient` which has no timeout. A hung model server holds goroutines and SSE connections indefinitely. Use a custom `http.Client` with a dial timeout, or pass a `context.WithTimeout` for non-streaming calls (title generation especially).

- [ ] **`bufio.Scanner` 64 KB limit on SSE stream** (`internal/server/messages.go:199`)
  The default scanner buffer is 64 KB. A `data:` line longer than that causes `Scan()` to silently return false mid-response. Set a larger buffer (`scanner.Buffer(make([]byte, 1<<20), 1<<20)`) and check `scanner.Err()` after the loop.

---

## Medium priority

- [ ] **`DeleteConversation` never returns `ErrNotFound`** (`internal/store/conversations.go:66`)
  Uses `db.Exec` without checking `RowsAffected`. Deleting a nonexistent conversation returns 204 instead of 404. Check rows affected and return `ErrNotFound` when zero.

- [ ] **Duplicate character-loading block in `handleSendMessage`** (`internal/server/messages.go:84–133`)
  The character fetch + system message + hidden messages block is copy-pasted verbatim twice (once for `req.CharacterID`, once for `conv.CharacterID`). Extract a helper function; this also makes the visibility fix from the critical bug above easier to apply in one place.

- [ ] **`UpdateConversationAfterMessage` error is silently ignored** (`internal/server/messages.go:258`)
  `_ = s.store.UpdateConversationAfterMessage(...)` — if this fails, the conversation's model/character metadata silently drifts. At minimum log the error.

- [ ] **Missing database indexes on frequently queried columns** (`internal/store/store.go`)
  Add indexes on:
  - `message(conversation_id)` — scanned on every message list and send
  - `conversation(user_id)` — scanned on every conversation list
  - `conversation(created_at)` — used by stale cleanup and untitled eligible queries
  - `session(expires_at)` — scanned on every auth check

- [ ] **WebSocket pong does not validate ping payload ≤ 125 bytes** (`internal/server/ws.go:198`)
  RFC 6455 §5.5: control frame payloads must not exceed 125 bytes. The pong is built as `append([]byte{0x8A, byte(length)}, payload...)` without checking length. Add `if length > 125 { return }` before building the pong.

- [ ] **`defer resp.Body.Close()` inside loop in `listModels`** (`cmd/lemon-chat/main.go:87`)
  `defer` inside a loop defers until the function returns, not the end of each iteration. All response bodies stay open until `listModels` exits. Call `resp.Body.Close()` explicitly at the end of each loop body.

---

## Low priority / polish

- [ ] **O(n²) string concatenation in streaming loop** (`internal/server/messages.go:235`)
  `fullContent += text` in a loop allocates on every iteration. Use `strings.Builder`.

- [ ] **`isImageExt()` is defined but never called** (`internal/server/avatars.go:115`)
  Dead code — remove it.

- [ ] **`mimeFromFilename` non-JPEG cases are unreachable** (`internal/server/avatars.go:27`)
  `receiveAvatar` always returns `".jpg"`, so only the JPEG branch is ever hit. Remove the dead cases or add a comment explaining the function is currently JPEG-only.

- [ ] **`go.mod` marks direct dependencies as `// indirect`** (`go.mod`)
  `BurntSushi/toml`, `golang.org/x/crypto`, and `golang.org/x/image` are imported directly. Run `go mod tidy` to fix.

- [ ] **Title generation uses `http.NewRequest` without a context** (`internal/tasks/titles.go:153`)
  Use `http.NewRequestWithContext` with a timeout context so a slow model server doesn't hang the worker goroutine indefinitely.

- [ ] **`time.Parse` error ignored for `prevUpdatedAt`** (`internal/server/messages.go:256`)
  If `conv.UpdatedAt` is malformed, `prevUpdatedAt` is the zero time and `BroadcastConversationListChanged` fires on every message. Handle or at least log the parse error.

- [ ] **`prompt()` / `confirm()` dialogs in sidebar** (`static/js/sidebar.js:47, 62`)
  Native browser dialogs are blocking and visually inconsistent with the rest of the UI. The title edit would be better as an inline editable field; the delete confirm should use the same modal pattern as the fork dialog.

- [ ] **Full `innerHTML` re-render of sidebar on every state change** (`static/js/sidebar.js:132`)
  `render()` rebuilds the entire sidebar and re-attaches all event listeners on `setActive()`, `addConversation()`, etc. Targeted DOM updates (as already done in `updateTitle()`) would be better.

- [ ] **`escHtml` vs `escapeHtml` — same function, two names** (`static/js/thread.js:112`, `static/js/sidebar.js:195`)
  Identical implementations with inconsistent names. Extract to a shared module or at least pick one name.

- [ ] **`sendMessage` in `app.js` is missing a try/catch** (`static/js/app.js:185`)
  If `convApi.create()` throws, the exception propagates with no user feedback and the composer is left in a broken state.

- [ ] **`initApp` does not handle `Promise.all` rejection** (`static/js/app.js:93`)
  If model or character list fetch fails on startup, the whole app silently fails to initialise. Add a catch that shows the user a meaningful error.

- [ ] **`applyFirstMessage` swallows all errors** (`static/js/app.js:254`)
  The catch block is intentionally silent for expected 409/400 responses, but also hides network errors and server errors. Narrow the suppression to expected status codes.

- [ ] **`window.location.reload()` after logout** (`static/js/sidebar.js:191`)
  Redirect to `/` instead to clear the `?c=…` query param and avoid a 401 redirect loop on reload.

- [ ] **`noCacheMiddleware` applied to all static assets including vendored libraries** (`internal/server/server.go:87`)
  Vendored files like `marked.esm.js` and `katex.min.css` never change between requests and should be cached aggressively. Apply `no-cache` only to HTML pages; serve static assets with `Cache-Control: max-age=…` or at least conditionally by path.

- [ ] **`err.message` used as `innerHTML` without escaping** (`static/js/thread.js:518`)
  Error messages from the server may contain `<` or `>` characters. Escape before inserting into the DOM, or use `.textContent`.
