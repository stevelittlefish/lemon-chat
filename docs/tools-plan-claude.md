# Tool calling — implementation plan

This document covers adding OpenAI-compatible tool calling to lemon-chat, starting with a single `get_time` tool. Tools are opt-in per character.

---

## How OpenAI tool calling works (protocol recap)

When the model wants to call a tool the streaming response looks like this instead of content deltas:

```
data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"get_time","arguments":""}}]},"finish_reason":null}]}
data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{}"}}]},"finish_reason":null}]}
data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}
data: [DONE]
```

After collecting the full tool call(s) you re-send the conversation to the model with two new messages appended:

1. An assistant message that has a `tool_calls` array (not content):
```json
{"role":"assistant","tool_calls":[{"id":"call_abc","type":"function","function":{"name":"get_time","arguments":"{}"}}]}
```

2. A tool result message for each call:
```json
{"role":"tool","tool_call_id":"call_abc","content":"2026-06-05T19:04:22Z"}
```

The model then generates its final natural-language response.

---

## Files touched

| File | Change |
|---|---|
| `internal/store/store.go` | Migration v15: add columns to `character` and `message` |
| `internal/store/characters.go` | `Tools []string` field; update queries |
| `internal/server/tools.go` | New file — tool registry and executors |
| `internal/server/messages.go` | Detect tool calls in stream; execute; loop |
| `internal/server/characters.go` | Accept/persist `tools` in create/update handlers |
| `static/js/settings-character-edit.js` | Tools checkbox section in form |
| `static/js/api.js` | Handle new `tool_call` SSE event (optional, see below) |
| `static/js/thread.js` | Optional: render tool-call indicator on messages |

---

## 1. Database — migration v15

Two schema changes, one migration.

### `character` table — add `tools` column

```sql
ALTER TABLE character ADD COLUMN tools TEXT;
```

Stored as a JSON array of tool IDs, e.g. `["get_time"]`. NULL means no tools. The column lives directly on `character` so it travels with the character through `GetCharacter`, `ListCharacters`, etc.

### `message` table — add `tool_calls` and `tool_call_id` columns

```sql
ALTER TABLE message ADD COLUMN tool_calls  TEXT;
ALTER TABLE message ADD COLUMN tool_call_id TEXT;
```

- `tool_calls` — JSON array; set on assistant messages that contained tool calls (role stays `"assistant"`). Used to reconstruct message history for the model.
- `tool_call_id` — set on tool-result messages (role `"tool"`). Required by the protocol so the model can match results to calls.

These messages are stored in the conversation but **never shown** in the chat thread. `thread.js` already skips non-`user`/`assistant` roles; we just need to also skip `tool`.

---

## 2. `internal/store/characters.go`

Add `Tools []string` to the `Character` struct:

```go
type Character struct {
    // ... existing fields ...
    Tools []string `json:"tools"`
}
```

Serialise as JSON when writing, deserialise when reading. NULL in the DB maps to an empty slice. Update `GetCharacter`, `ListCharacters`, `CreateCharacter`, `UpdateCharacter` to handle the new column (json.Marshal/Unmarshal in the Go layer — no SQL JSON functions needed).

---

## 3. `internal/server/tools.go` — new file

Centralise everything tool-related here.

### Tool definition (sent to the model)

```go
type toolParam struct {
    Type       string         `json:"type"`
    Properties map[string]any `json:"properties"`
    Required   []string       `json:"required"`
}

type toolFunction struct {
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Parameters  toolParam `json:"parameters"`
}

type toolDef struct {
    Type     string       `json:"type"`
    Function toolFunction `json:"function"`
}
```

### Registry

```go
var toolDefs = map[string]toolDef{
    "get_time": {
        Type: "function",
        Function: toolFunction{
            Name:        "get_time",
            Description: "Returns the current UTC date and time.",
            Parameters:  toolParam{Type: "object", Properties: map[string]any{}, Required: []string{}},
        },
    },
}

// ToolDefsForCharacter returns the tool definitions for the given tool IDs.
func ToolDefsForCharacter(toolIDs []string) []toolDef { ... }

// ExecuteTool runs a tool by name and returns its result string or an error.
func ExecuteTool(name, argsJSON string) (string, error) { ... }
```

### Executor

```go
func executorGetTime(_ string) (string, error) {
    return time.Now().UTC().Format(time.RFC3339), nil
}

var executors = map[string]func(string) (string, error){
    "get_time": executorGetTime,
}
```

`ExecuteTool` looks up `executors[name]` and calls it. Unknown tool names return an error; the backend will send that back as the tool result content so the model can handle it gracefully.

### UI metadata (used by the character editor)

```go
type ToolMeta struct {
    ID          string `json:"id"`
    DisplayName string `json:"display_name"`
    Description string `json:"description"`
}

var AllTools = []ToolMeta{
    {"get_time", "Get current time", "Returns the current UTC date and time."},
}
```

Expose via a new `GET /api/tools` endpoint (unauthenticated is fine, it's static data) so the character editor can populate its checkbox list without hardcoding anything in the frontend.

---

## 4. `internal/server/messages.go` — the agentic loop

### Streaming struct additions

The existing stream parser accumulates `fullContent`. We need to also accumulate tool calls:

```go
type streamToolCall struct {
    Index    int
    ID       string
    Name     string
    ArgsJSON strings.Builder
}
```

### Changes to `handleSendMessage`

After building `chatMsgs` and the payload, the logic becomes a loop:

```
loop:
  POST /chat/completions (streaming)
  scan chunks:
    if delta.content → write to fullContent, stream delta SSE to client (same as now)
    if delta.tool_calls → accumulate into streamToolCall slice
    if finish_reason == "tool_calls" → break inner scan, execute tools
    if finish_reason == "stop" / "[DONE]" → break loop
  if tools were called:
    persist assistant message (role="assistant", content="", tool_calls=JSON)
    for each tool call:
      send `tool_call` SSE event to client (name only, for optional UI indicator)
      result, err = ExecuteTool(name, args)
      persist tool result message (role="tool", tool_call_id=id, content=result)
    append both to chatMsgs in-memory
    continue loop
  else:
    persist assistant message with fullContent (same as now)
    break loop
```

Key points:
- The tool-call messages are persisted to the DB (so they're included in future `ListMessages` calls and thus in subsequent history) but not streamed as message content to the client.
- The `tool_call` SSE event is a new event type: `{"tool_call": {"name": "get_time"}}`. The client can use it to show a subtle "checked the time" annotation. It's optional to render — if the client ignores unknown SSE keys nothing breaks.
- The loop has a hard cap (e.g. 5 iterations) to prevent infinite tool-call loops from a misbehaving model.
- Tool inclusion: only send `"tools"` in the payload when the character has `len(char.Tools) > 0`. Don't send an empty array — some models reject it.

### Payload shape when tools are active

```go
payload := map[string]any{
    "model":    modelName,
    "messages": chatMsgs,
    "stream":   true,
    "stream_options": map[string]any{"include_usage": true},
}
if len(toolDefs) > 0 {
    payload["tools"] = toolDefs
}
```

### Message history reconstruction

`store.ListMessages` returns all messages including role=`tool`. When building `chatMsgs` from history, we need to include them correctly:

```go
for _, m := range history {
    msg := chatMsg{Role: m.Role, Content: m.Content}
    if m.ToolCalls != nil {       // assistant message with tool calls
        msg.ToolCalls = m.ToolCalls  // parsed from JSON
        msg.Content = ""
    }
    if m.ToolCallID != "" {      // tool result message
        msg.ToolCallID = m.ToolCallID
    }
    chatMsgs = append(chatMsgs, msg)
}
```

The `chatMsg` struct needs to grow:

```go
type chatMsg struct {
    Role       string       `json:"role"`
    Content    string       `json:"content,omitempty"`
    ToolCalls  []any        `json:"tool_calls,omitempty"`
    ToolCallID string       `json:"tool_call_id,omitempty"`
}
```

---

## 5. `internal/server/characters.go`

The create/update request structs need a `Tools []string` field. Handlers pass it through to `store.CreateCharacter` / `store.UpdateCharacter`. No other logic needed here.

---

## 6. Frontend — character editor

### New `GET /api/tools` endpoint

The character editor calls this to get the tool list. Response:

```json
[{"id":"get_time","display_name":"Get current time","description":"Returns the current UTC date and time."}]
```

### `settings-character-edit.js`

Add a `tools` variable (array of enabled tool IDs) alongside `hiddenMessages`. Load the tool list from `/api/tools` alongside the existing data fetches.

Add a new form section between "Behaviour" and "Title generation prompt":

```
┌─ Tools ────────────────────────────────────────────┐
│ Tools this character is allowed to call.            │
│                                                     │
│  ☐  Get current time                               │
│     Returns the current UTC date and time.          │
└─────────────────────────────────────────────────────┘
```

Each row is a checkbox (`<input type="checkbox">` styled via the design system's checkbox pattern or a toggle). The checkbox ID is the tool's `id` field.

`readForm()` collects `tools: enabledToolIds` and includes it in the API payload. The API already passes arbitrary fields through — no changes to `api.js` needed.

### `thread.js` — optional tool-call indicator

When the SSE stream emits `{"tool_call": {"name": "get_time"}}`, thread.js can append a small annotation line beneath the streaming assistant bubble, e.g. a muted mono line `called get_time`. This is cosmetic and can be deferred; the feature works correctly without it. The important thing is that `tool` role messages returned from `GET /api/conversations/:id/messages` are silently skipped when rendering the thread.

---

## 7. What does NOT change

- The SSE wire format for content deltas, `[DONE]`, `stats`, and `message_id` events is unchanged.
- Conversations without a character, or characters with no tools configured, follow the exact existing code path — no overhead.
- The completions page is unaffected (tools are chat-only).
- `handleGetMessageContext` already reconstructs history from the DB; once the new message columns are included in the query, tool messages will naturally appear in the context view.

---

## 8. Sequence diagram

```
client          server (handleSendMessage)        model API
  |                        |                          |
  |── POST /messages ──────>|                          |
  |                        |── POST /chat/completions ─>|
  |                        |<── stream (tool_calls) ──|
  |                        |                          |
  |                        | [persist assistant+tool msgs]
  |                        | [execute get_time]
  |<── SSE tool_call ───────|                          |
  |                        |── POST /chat/completions ─>|
  |                        |<── stream (content) ─────|
  |<── SSE delta... ────────|                          |
  |<── SSE [DONE] ──────────|                          |
  |                        | [persist assistant msg]  |
```

---

## 9. Scope decisions / things deliberately deferred

- **Tool errors**: if `ExecuteTool` returns an error, the result content is `"error: <msg>"`. The model will see this and respond accordingly. No special handling.
- **Tool call visibility in thread**: hiding tool messages from the thread is the minimum viable approach. A richer "tool usage" disclosure widget can be added later.
- **Parallel tool calls**: the protocol supports multiple tool calls in a single response (multiple entries in the `tool_calls` array). The loop handles this naturally — execute all, append all results, continue.
- **Tool arguments validation**: `get_time` takes no arguments. Argument validation (for future tools with params) is deferred.
- **`GET /api/tools` route**: add it to `server.go` alongside the other `/api/` routes. It requires authentication (consistent with other API endpoints).
