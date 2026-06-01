# lemon-chat — Code Review

> Reviewed against commit `d77fb7a` (main branch). Schema version 9.

---

## Overview

Genuinely solid foundation for a local-first, self-hosted AI chat UI. The backend is clean and idiomatic Go, the frontend stays disciplined (no framework creep), and the overall architecture is well-suited to its purpose. The worst issues are a handful of correctness bugs and one security hole; the rest is polish. This is not slop — it's mostly competent code with a few rough edges.

---

## What's Good

**Architecture and separation of concerns are well done.** The store/server/tasks/config split is clean and each package has a narrow responsibility. The store layer never leaks SQL into the server layer; response helpers (`writeJSON`, `writeError`, `internalError`) are used consistently across all handlers.

**No SQL injection anywhere.** Every query uses parameterised arguments. Consistent and correct.

**bcrypt password hashing with server-side sessions.** The auth model is solid for a local app. Sessions are stored in SQLite with a 30-day TTL enforced at read time. The HttpOnly + SameSite=Strict cookie is correct.

**The custom WebSocket implementation is a good call.** Adding a full WebSocket library for a one-directional notification channel would be overkill. The RFC 6455 implementation correctly handles handshake, frame framing (all three length encodings), masking/unmasking, ping→pong, and close frames.

**SSE streaming is clean.** The separation between API layer (SSE events) and streaming controller (`thread.js startStreaming()`) is well thought out. AbortController for the stop button is the right approach.

**Avatar processing is properly defensive.** Content-type sniffing from the first 512 bytes before reading the full body, size limit, decode + re-encode through a known format (eliminating metadata and parser confusion), CatmullRom resampling — all correct.

**DB constraint enforces the model XOR character invariant.** `CHECK ((model IS NOT NULL) != (character_id IS NOT NULL))` on the conversation table is a nice belt-and-suspenders. This invariant could otherwise drift silently.

**Nil-safe array returns.** Every list endpoint that could return `nil` instead converts it to `[]T{}` before serialisation. Prevents `null` vs `[]` confusion on the client side.

**Migration system is clean.** Sequential, numbered, idempotent version gates. Each migration inserts a `schema_version` row for auditability.

**WAL mode and FK enforcement are set at open time.** Both are the right defaults for SQLite in a web application context.

**`userResponse()` helper hides sensitive fields.** The password hash never makes it out of the server layer; the response includes `has_password` (bool) rather than the hash.

**`ForkConversation` uses a transaction for hidden messages.** `ReplaceCharacterHiddenMessages` (characters.go:144) wraps the delete+insert in a transaction. Correct.

---

## Issues

### Critical

**1. `subtle.ConstantTimeCompare` is comparing a string to itself — it does nothing**
`internal/server/auth.go:38–39`

```go
_ = subtle.ConstantTimeCompare([]byte(req.Password), []byte(req.Password))
```

The "belt-and-suspenders" comment says this is a constant-time belt on top of bcrypt. It isn't. It compares `req.Password` to itself, so it always returns 1 and its result is discarded. The bcrypt comparison above it is already constant-time; this line is dead code that looks like a security control.

**2. `handleSendMessage` does not check character visibility when the client passes `character_id`**
`internal/server/messages.go:84–104`

When a request body includes `character_id`, the handler fetches and uses the character without checking whether the calling user is allowed to access it:

```go
if req.CharacterID != nil {
    char, err := s.store.GetCharacter(*req.CharacterID)
    // ← no visibility check here
```

Any authenticated user can send messages using a private character they don't own by passing its ID directly. The conversation does not need to reference the character — they just include it in the message body. The same pattern is used for the conversation's character (lines 109–133) which is also unchecked. Compare with `handleFirstMessage` at line 418, which does check visibility correctly.

Fix: after fetching the character, add:
```go
if char.Visibility == "private" && char.CreatedBy != user.ID && !user.IsAdmin {
    writeError(w, http.StatusForbidden, "forbidden")
    return
}
```

---

### High

**3. `ForkConversation` is not wrapped in a transaction**
`internal/store/conversations.go:111–179`

The fork creates a new conversation (via `CreateConversation`) and then inserts messages in a separate loop. If the process dies or the DB errors partway through message insertion, the database is left with an empty forked conversation that has no messages.

Fix: open a transaction at the top of `ForkConversation`, do all work inside it, and commit at the end.

**4. Title generation can fire twice for the same conversation on a single request**
`internal/server/messages.go:270–294`

Both the `auto_title` trigger (line 278) and the "3rd assistant response" trigger (line 292) can pass their conditions and each spawn a goroutine that calls `GenerateTitleForConversation`. Since both evaluate `conv.Title == nil` using the pre-request snapshot of the conversation, a conversation on its third exchange with `auto_title=true` will spawn two concurrent title generation goroutines. Both will make a model API call and both will call `UpdateConversationTitle`, with a race on which result persists.

Fix: add a mutual exclusion check (e.g., a `sync.Map` of in-flight conv IDs, or an early-return if `auto_title` already fired for this message count), or collapse both trigger paths into one.

**5. `http.DefaultClient` is used for all model API calls with no timeout**
`internal/server/messages.go:179` and `internal/tasks/titles.go:161`

```go
resp, err := http.DefaultClient.Do(httpReq)
```

`http.DefaultClient` has no timeout. A model server that hangs or returns data very slowly will hold the goroutine — and the client's SSE connection — indefinitely. For streaming this is partly intentional, but it should at least have a connection timeout for the initial response. Title generation has no justification for being unbounded.

Fix: create a package-level `http.Client` with a `Timeout` (or at minimum a `DialContext` timeout), or use `context.WithTimeout` on the request context before the model call.

**6. `bufio.Scanner` default buffer is too small for large SSE lines**
`internal/server/messages.go:199`

```go
scanner := bufio.NewScanner(resp.Body)
```

The default scanner buffer is 64 KB. A single `data:` line longer than that (possible with very long model responses in a single chunk — some models emit large JSON payloads) will cause `scanner.Scan()` to return false with `ErrTooLong`, silently terminating the stream mid-response. The error is never checked.

Fix: `scanner.Buffer(make([]byte, 1<<20), 1<<20)` to set a 1 MB buffer, and check `scanner.Err()` after the loop.

---

### Medium

**7. `DeleteConversation` never returns `ErrNotFound`**
`internal/store/conversations.go:66–69` and `internal/server/conversations.go:133`

The store's `DeleteConversation` uses `db.Exec` and never checks `RowsAffected()`. It returns `nil` whether the conversation existed or not. The server handler does `errors.Is(err, store.ErrNotFound)` — that branch is never reachable. Deleting an already-deleted (or never-existed) conversation returns 204 instead of 404.

**8. `listModels` in `main.go` defers `resp.Body.Close()` inside a loop**
`cmd/lemon-chat/main.go:87`

```go
for _, srv := range cfg.ModelServers {
    ...
    defer resp.Body.Close()  // ← fires when main() returns, not each iteration
```

`defer` in a loop defers until the surrounding function returns, not until the end of the loop body. All response bodies accumulate open until `listModels` exits. This is not a memory leak in practice (it's a one-shot utility function), but it is incorrect and can cause "too many open files" if there are many model servers.

Fix: call `resp.Body.Close()` at the end of each iteration, or `defer` inside an anonymous function.

**9. `handleSendMessage` duplicates the character-loading block verbatim**
`internal/server/messages.go:84–133`

The character fetching and system message building (including hidden messages) is copy-pasted twice — once for `req.CharacterID != nil` and once for `conv.CharacterID != nil`. They are identical modulo which ID they use. If the logic changes (e.g., adding the visibility check from issue #2), the change needs to be made in two places.

Fix: extract a `loadCharacterContext(charID int64) (modelName, assistantName string, msgs []chatMsg, err error)` helper.

**10. `drainFrames` does not validate ping frame payload length**
`internal/server/ws.go:198–200`

RFC 6455 §5.5 requires that control frames (ping, pong, close) must not exceed 125 bytes. The pong response is built as:

```go
pong := append([]byte{0x8A, byte(length)}, payload...)
```

If a ping arrives with a payload > 125 bytes (which is technically a protocol violation from the client, but should be handled gracefully), the response's length byte will overflow and the pong will be malformed. This could cause some clients to terminate the connection.

Fix: add `if length > 125 { return }` before building the pong, or send a close frame.

**11. `UpdateConversationAfterMessage` is called without checking its error return value**
`internal/server/messages.go:258`

```go
_ = s.store.UpdateConversationAfterMessage(convID, usedModel, usedCharacterID)
```

This is the call that updates the conversation's `model`/`character_id` and touches `updated_at`. If it fails silently, the conversation's metadata drifts from reality. The error should at least be logged.

**12. No missing database indexes**
`internal/store/store.go`

The schema has no explicit indexes on columns used in frequent WHERE clauses:
- `message(conversation_id)` — queried on every message list and send
- `conversation(user_id)` — every conversation list query
- `conversation(created_at)` — used in `ListUntitledEligible` and `DeleteStaleConversations`
- `session(expires_at)` — scanned on every auth check

SQLite will full-scan these tables on every request. For a local app with small data sets this is fine, but it should be addressed before the data grows.

**13. `go.mod` marks all dependencies as `// indirect`**
`go.mod`

```
github.com/BurntSushi/toml v1.6.0 // indirect
golang.org/x/crypto v0.52.0       // indirect
golang.org/x/image v0.41.0        // indirect
```

These are direct dependencies — they are imported in the module's own source files. Marking them as indirect is technically wrong and can confuse tooling. Run `go mod tidy` to fix.

**14. Title generation HTTP request does not use a context**
`internal/tasks/titles.go:153`

```go
httpReq, err := http.NewRequest("POST", chatURL, bytes.NewReader(payload))
```

Uses `http.NewRequest` instead of `http.NewRequestWithContext`. There is no way to cancel this request or have it time out. If the model server is slow, the title worker goroutine hangs indefinitely.

Fix: pass a context with a timeout (e.g., `context.WithTimeout(context.Background(), 30*time.Second)`).

---

### Low / Polish

**15. String concatenation in streaming loop is O(n²)**
`internal/server/messages.go:235`

```go
fullContent += text
```

Appending to a string in a loop creates a new allocation on each iteration. For a long model response this is wasteful. Use `strings.Builder` instead.

**16. `isImageExt()` function is defined but never called**
`internal/server/avatars.go:115–121`

Dead code. Should be deleted.

**17. `mimeFromFilename()` cases other than JPEG are unreachable**
`internal/server/avatars.go:27–43`

`receiveAvatar` always returns the extension `".jpg"` regardless of the input format. So `mimeFromFilename` will only ever match the `jpg/jpeg` case. The PNG/GIF/WebP arms are dead code. Either remove them or make the function honest about what it actually receives.

**18. `prevUpdatedAt` ignores parse error**
`internal/server/messages.go:256`

```go
prevUpdatedAt, _ := time.Parse(time.RFC3339, conv.UpdatedAt)
```

If `conv.UpdatedAt` is malformed, `prevUpdatedAt` becomes the zero value of `time.Time` and `time.Since(prevUpdatedAt)` is ~2000 years — always broadcasting `BroadcastConversationListChanged`. This can't happen with well-formed data but is a silent logic bug if something corrupts the timestamp.

**19. `sidebar.js` uses `prompt()` and `confirm()` for user input**
`static/js/sidebar.js:47, 62`

The native browser dialogs are blocking, look out of place next to the otherwise custom UI, and cannot be styled. The title edit in particular would be much better as an inline editable field in the sidebar item itself.

**20. `sidebar.js` full re-render on every state change**
`static/js/sidebar.js:132`

`render()` sets `sidebarEl.innerHTML = ...` and re-attaches all event listeners on every call. It's called on `setActive()` (every conversation click), `addConversation()`, `removeConversation()`, etc. For a local single-user app with a modest conversation list this is fine, but it's not scalable and causes unnecessary DOM thrash. The `updateTitle()` function already shows the right pattern — targeted DOM update.

**21. Function name inconsistency: `escHtml` vs `escapeHtml`**
`static/js/thread.js:112` vs `static/js/sidebar.js:195`

Both files define the identical HTML-escaping function but with different names. One of them should import from the other or both should import from a shared utils module.

**22. `sendMessage` in `app.js` is not wrapped in try/catch**
`static/js/app.js:185`

```js
const conv = await convApi.create(null, model, charId);
```

If `convApi.create` throws (network error, server error), the exception propagates to the caller with no UI feedback. The user is left with a non-functional composer and no error message.

**23. `initApp` does not handle `Promise.all` rejection**
`static/js/app.js:93–98`

```js
const [models, chars] = await Promise.all([
    modelApi.list(),
    characterApi.list(),
    ...
]);
```

If either API call fails (e.g., on first start when the server is still initialising), the entire `initApp` function throws and the app shows nothing with no error message. Should be wrapped with a catch that shows the user a meaningful error.

**24. `applyFirstMessage` silently swallows all errors**
`static/js/app.js:254`

```js
} catch {
    // Character has no first message, or already has messages — silent
}
```

The comment describes expected conditions but the catch also silently swallows network errors, server errors, etc. Only the specific expected 409/400 responses should be suppressed; other errors should surface.

**25. `window.location.reload()` after logout is coarser than necessary**
`static/js/sidebar.js:191`

A redirect to `/` would be cleaner and would also clear the URL state (<code>?c=...</code>), preventing a 401 redirect loop if the conversation ID is embedded in the URL.

**26. `noCacheMiddleware` applies to all static assets including vendored libraries**
`internal/server/server.go:87`

```go
mux.Handle("/", noCacheMiddleware(http.FileServer(http.Dir("static"))))
```

`Cache-Control: no-cache` forces a revalidation request for every asset on every page load — including large vendored files like `marked.esm.js` and `katex.min.css` that never change between requests. These should be served with long-lived cache headers (or at least `Cache-Control: no-store` only for HTML) to reduce redundant network traffic. Only the HTML pages and API routes truly need `no-cache`.

**27. `_ctxModal._messages` is added as a dynamic property on the modal object**
`static/js/thread.js:461, 492`

```js
modal._messages = messages;
```

A property is monkey-patched onto the modal object after construction. This works but is surprising. The messages should be stored in a closure variable or a proper state variable adjacent to the modal.

**28. Context modal error leaks raw error messages to the DOM via template literal**
`static/js/thread.js:518`

```js
modal.body.innerHTML = `<p class="ctx-modal-loading">Failed to load context: ${err.message}</p>`;
```

`err.message` is set as innerHTML without escaping. If the error message contains `<`, `>`, or `&` characters (possible from HTTP error bodies), this could cause display issues or minor XSS in the error UI itself.

---

## Summary Table

| # | Severity | Location | Issue |
|---|----------|----------|-------|
| 1 | Critical | `auth.go:38` | `subtle.ConstantTimeCompare` compares string to itself — does nothing |
| 2 | Critical | `messages.go:84` | No visibility check when `character_id` passed in message body |
| 3 | High | `store/conversations.go:111` | `ForkConversation` not wrapped in a transaction |
| 4 | High | `messages.go:270–294` | Title generation can fire twice concurrently per request |
| 5 | High | `messages.go:179`, `titles.go:161` | `http.DefaultClient` with no timeout |
| 6 | High | `messages.go:199` | `bufio.Scanner` 64KB limit + unchecked error |
| 7 | Medium | `store/conversations.go:66` | `DeleteConversation` never returns `ErrNotFound` |
| 8 | Medium | `main.go:87` | `defer resp.Body.Close()` inside loop |
| 9 | Medium | `messages.go:84–133` | Duplicate character-loading block |
| 10 | Medium | `ws.go:198` | Ping pong response does not validate payload ≤125 bytes |
| 11 | Medium | `messages.go:258` | `UpdateConversationAfterMessage` error silently ignored |
| 12 | Medium | `store/store.go` | Missing DB indexes on FK columns |
| 13 | Low | `go.mod` | Direct deps marked `// indirect` |
| 14 | Low | `titles.go:153` | `http.NewRequest` without context/timeout |
| 15 | Low | `messages.go:235` | O(n²) string concat in streaming loop |
| 16 | Low | `avatars.go:115` | `isImageExt()` defined but never called |
| 17 | Low | `avatars.go:27` | `mimeFromFilename` non-JPEG cases unreachable |
| 18 | Low | `messages.go:256` | `time.Parse` error ignored |
| 19 | Low | `sidebar.js:47,62` | `prompt()` / `confirm()` native dialogs |
| 20 | Low | `sidebar.js:132` | Full innerHTML re-render on every state change |
| 21 | Low | `thread.js:112` / `sidebar.js:195` | `escHtml` vs `escapeHtml` inconsistency |
| 22 | Low | `app.js:185` | `convApi.create` not in try/catch |
| 23 | Low | `app.js:93` | `initApp` `Promise.all` rejection unhandled |
| 24 | Low | `app.js:254` | `applyFirstMessage` catch swallows all errors |
| 25 | Low | `sidebar.js:191` | `location.reload()` after logout |
| 26 | Low | `server.go:87` | `no-cache` applied to all static assets |
| 27 | Low | `thread.js:461` | Dynamic property on modal object |
| 28 | Low | `thread.js:518` | `err.message` used as innerHTML without escaping |
