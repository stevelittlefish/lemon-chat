# Backend Code Review

## Bugs

### `isDuplicateUsername` checks the wrong table name
**File:** `internal/server/admin.go:15`

```go
func isDuplicateUsername(err error) bool {
    return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: users.username")
}
```

The table is named `user` (singular), so SQLite's actual error message is `UNIQUE constraint failed: user.username`. This check will always return `false`, causing duplicate username errors to return 500 instead of 409. Easy to reproduce: create two users with the same username via the admin API.

---

### `GetCompletion` converts all DB errors to `ErrNotFound`
**File:** `internal/store/completions.go:40-51`

```go
func (s *Store) GetCompletion(id, userID int64) (*Completion, error) {
    var c Completion
    err := s.db.QueryRow(...).Scan(...)
    if err != nil {
        return nil, ErrNotFound  // swallows EVERYTHING
    }
    return &c, nil
}
```

Any real database error (connection problem, schema mismatch, etc.) is silently converted to `ErrNotFound`. This will surface as a 404 in the API when the real cause might be a 500. Compare with `GetConversation` which correctly uses `errors.Is(err, sql.ErrNoRows)`.

---

### `DeleteConversation` doesn't clean up attachments
**File:** `internal/store/conversations.go:66-90`

The transaction deletes messages then the conversation, but the `attachment` table has `conversation_id REFERENCES conversation(id)` with no `ON DELETE CASCADE`. Deleting a conversation leaves attachment DB rows and files on disk. The files in `data/attachments/` are never reclaimed. This will compound over time.

---

### `ForkConversation` drops tool call metadata
**File:** `internal/store/conversations.go:162-173, 201-209`

The fork query selects `role, content, name, character_id, prompt_tokens, completion_tokens, total_time_ms` but omits `tool_calls` and `tool_call_id`. The insert also omits them. Forking a conversation that used tools breaks the assistant/tool message pairing, producing malformed message history that will confuse the model on the next send.

---

### `handleGetMessageContext` silently eats character fetch errors
**File:** `internal/server/messages.go:634-648`

```go
if conv.CharacterID != nil {
    char, err := s.store.GetCharacter(*conv.CharacterID)
    if err == nil {
        // ... apply system prompt
    }
}
```

If the character lookup fails, the character's system prompt and hidden messages are silently omitted. The returned context is then incomplete — the model would have used them when the original message was sent. Compare with `resolveCharacter` which returns an HTTP error. The inconsistency means the context viewer shows a different context than what the model actually received.

---

### Attachment directory is relative to working directory
**File:** `internal/server/tools.go:242, 600`

```go
dir := filepath.Join("data", "attachments", randomID())
```

This path is resolved relative to the process's working directory at startup. The avatar directory (correctly) anchors to the DB path: `filepath.Join(filepath.Dir(s.cfg.Server.DBPath), "avatars")`. If the server is started from a different directory, attachment files are written to (and read from) the wrong location.

---

## Duplicated Code

### Model API request pattern repeated ~4 times

The sequence of: build payload → create HTTP request → set headers → call model → read body → unmarshal choices → extract title text appears in nearly identical form in:

- `internal/tasks/titles.go:276` (`generateTitle`)
- `internal/tasks/titles.go:164` (`generateCompletionTitle`)
- `internal/server/tools.go:650` (`summariseHTML`)
- `internal/server/messages.go` (inlined in `handleSendMessage`)

`generateTitle` and `generateCompletionTitle` are especially close — both build a system-prompt-plus-user message payload, POST to the same endpoint, parse the same response shape, and trim the result. The only difference is the system prompt text and how the user content is sourced. They could share a `callModel(chatURL, modelName, apiKey, msgs []chatMsg, maxTokens int, timeout) (string, error)` helper.

`summariseHTML` in tools.go also defines its own local `chatMsg` struct, duplicating the package-level one in messages.go.

---

### SSE scanner loop repeated verbatim in two handlers

The inner loop in `handleSendMessage` (messages.go:350-413) and `handleRunCompletion` (completions.go:294-335) both:
- Create a 1 MB `bufio.Scanner`
- Strip the `data: ` prefix
- Check for `[DONE]`
- Unmarshal the chunk
- Check `scanner.Err()`
- Close `resp.Body`

The chunk struct shapes differ (chat uses `choices[].delta`, completions uses `choices[].text`), but the framing and error handling are identical and could be shared.

---

### SSE response setup copy-pasted

Three lines appear identically in both `handleSendMessage` and `handleRunCompletion`:

```go
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("X-Accel-Buffering", "no")
flusher := w.(http.Flusher)
```

A `startSSE(w http.ResponseWriter) http.Flusher` helper would remove the duplication and the unsafe cast.

---

### `mimeFromFilename` and `mimeTypeForFilename` do the same thing

- `internal/server/avatars.go:29` — `mimeFromFilename` (handles only jpeg)
- `internal/server/tools.go:193` — `mimeTypeForFilename` (handles 10+ types)

Both exist in the same package. avatars.go should just call the one from tools.go. The avatars version also uses `strings.Split` + last element instead of `filepath.Ext`, which is subtly different for filenames with multiple dots.

---

### `handleUpdateCompletion` triple-repeats the not-found pattern

**File:** `internal/server/completions.go:93-123`

Three separate `if req.X != nil { if err := s.store.UpdateX(...); errors.Is(err, ErrNotFound) { ... } }` blocks for title, content, and model. Each has its own full error-handling branch. The store could have a single `UpdateCompletion(id, userID int64, updates CompletionUpdate)` that applies all three in one statement.

---

## Confusing / Poorly Structured Code

### `handleGetMessageContext` re-implements `resolveCharacter` differently

**File:** `internal/server/messages.go:632-648`

The function manually fetches the character, checks `if err == nil`, and applies system prompt and hidden messages. This is the same logic as `resolveCharacter()` (lines 50-73) but with silent-failure semantics instead of HTTP-error semantics. Two code paths that should behave identically now diverge: a character DB error causes a 500 in `handleSendMessage` and a silently incomplete context in `handleGetMessageContext`.

---

### Title trigger logic is convoluted and operates on stale data

**File:** `internal/server/messages.go:558-587`

The title trigger counts messages from `history`, which was fetched *before* the new assistant message was saved. This means the comment "trigger on 3rd assistant response" is off by one from what you'd expect — it fires when `assistantMsgCount >= 2` (i.e., the current response is the 3rd). The `titleTriggered` flag and the dual-path structure (character auto-title vs. generic 3rd-response) make this hard to reason about. The conditions could be extracted into clearly named functions.

---

### Tool loop `continue/break` logic is non-obvious

**File:** `internal/server/messages.go:333-524`

The `for loop := 0; loop < maxToolLoops; loop++` has an unusual control flow:
- `continue` is used inside the `if finishReason == "tool_calls"` block to re-loop
- `break` exits when there's final content
- But the final `finalContent = fullContent.String(); break` at line 522-524 is *outside* the `if finishReason == "tool_calls"` block

This means the break at line 523 fires for every non-tool response (correct), but also fires when `finishReason == "tool_calls"` AND the loop limit is hit on the last iteration (because the `continue` at line 518 isn't reached). The code is correct but requires careful reading to verify — an early return out of a helper would be clearer than loop-and-break.

---

### `UpdateConversationAfterMessage` can violate the CHECK constraint

**File:** `internal/store/conversations.go:99-105`

The DB has `CHECK ((model IS NOT NULL) != (character_id IS NOT NULL))`. `UpdateConversationAfterMessage` accepts `model *string` and `characterID *int64` and just writes them. Nothing enforces that exactly one is non-nil at the call site. Callers must get this right manually; a type-level approach (e.g., a `ConvOwner` union type) would make violations impossible.

---

### `character_hidden_message` appears in both v0→v1 and v8→v9 migrations

**File:** `internal/store/store.go:99-107, 227-245`

The table is created in the initial v1 schema block AND in a separate v9 migration (`CREATE TABLE IF NOT EXISTS`). For fresh installs v9 is a no-op. For older databases that predate both, v9 would have created it before v1 was updated to include it. The migration history is inconsistent — it's unclear which version "owns" the table, and the v9 block is now dead code for any install that ran v1 with the current schema.

---

## Maintenance Concerns

### Hand-rolled WebSocket implementation

**File:** `internal/server/ws.go`

The WebSocket protocol (handshake, frame parsing, ping/pong, masking) is implemented from scratch. This is a known-hard protocol to implement correctly. Current issues:
- Frame length with `length == 127` reads only 4 bytes of the 8-byte extended length (lines 180-182): `int(ext[4])<<24 | int(ext[5])<<16 | int(ext[6])<<8 | int(ext[7])`. This truncates payloads larger than 2³¹ bytes, but more importantly, the upper 4 bytes are simply read and discarded — which is correct behavior to avoid int overflow on 32-bit systems but confusing. A standard library (`golang.org/x/net/websocket` or `nhooyr.io/websocket`) would eliminate this class of bugs.
- `drainFrames` doesn't handle fragmented frames.
- `sendTextFrame` writes header and payload in two separate `conn.Write` calls, which may be split into two TCP packets under load.

---

### No CSRF protection

Auth is entirely cookie-based. `SameSite: Strict` provides protection in modern browsers but not all clients (curl, older browsers, native mobile apps). Any state-mutating endpoint (`POST /api/conversations`, `DELETE /api/characters/{id}`, etc.) is callable by a page the user happens to visit if the browser doesn't enforce SameSite. A double-submit cookie or custom header check (`X-Requested-With`) is conventional for cookie-authed APIs.

---

### `noCacheMiddleware` is defined but never used

**File:** `internal/server/middleware.go:56-61`

Dead code. The per-route `w.Header().Set("Cache-Control", "no-cache")` calls are used instead.

---

### Title generation uses `http.DefaultClient` for model calls but `s.modelClient` for chat

**File:** `internal/tasks/titles.go:195, 335`

```go
resp, err := http.DefaultClient.Do(httpReq)
```

Chat and completion streaming use the custom `s.modelClient` with a configurable dial timeout. Title generation bypasses this and uses the default client (no dial timeout). A hung title request can block indefinitely. The same issue exists in the `web_search`, `fetch_url`, and `summariseHTML` tools — they all use `http.DefaultClient`.

---

### Session cookie lacks `Secure` flag

**File:** `internal/server/auth.go:51-58`

For local/self-hosted use this may be intentional, but if the server is ever proxied through HTTPS, the cookie should be `Secure` to prevent transmission over plaintext connections. This is a deployment concern but worth noting.
